// Package testreuse runs one of make preflight's test stages so that what Go's
// test cache holds for unchanged inputs is replayed (itd-2611011325421304).
//
// Go's test cache keys a package's result on its test binary, its cacheable
// flags and the files and environment variables its tests read. A fresh
// worktree defeats all of it: its paths change the binary, its checkout gives
// every file a new time, and a file a test opens is hashed under its absolute
// path. So a stage runs from one fixed check directory per repository and
// account, ~/.abcd.noindex/tests/<root-sha>/, whose tree is the commit's and
// whose file times derive from file content, under -trimpath and one lock.
//
// The cache cannot see what a test reads through a subprocess, a binary it
// builds, the clock or the network. What reruns on every check:
//
//   - The always-rerun list on the default branch (ReadList) names a package or
//     a test. It is read from the branch, so a change's own diff does not move
//     it. A list that is missing, does not read, or names what does not exist
//     means nothing is reused.
//   - The classifier (ClassifyFiles) reads every test's source on every check.
//     A test it finds evidence against reruns whether declared or not, and the
//     report names it as undeclared. A file it cannot read reruns its package.
//   - The check directory holds no .git and git may not walk above it. An
//     always-rerun test runs in the worktree, where git answers, with -count=1.
//
// The hosted run is the backstop: it passes -count=1 on every test step, on every
// pull request and merge-queue entry.
//
// The sync reads every file of the tree back against the commit before a line
// runs and again after, and a receipt is minted only from a stage's record of that
// exact tree. One check holds the lock at a time, for the lines that use the tree;
// the rerun lines run outside it.
//
// A race stage's binary is built differently, so race results reuse race
// results only, never the unit stage's, and the unit stage's never a race run's.
package testreuse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Exit codes beyond go test's own: a recipe mistake, and a check that gave up
// waiting for another to leave the directory.
const (
	ExitUsage = 2
	ExitBusy  = 3
)

// Config is one stage's run.
type Config struct {
	Stage    string
	Args     []string // what follows `go test` in the recipe
	Worktree string
	Home     string
	Stdout   io.Writer
	Stderr   io.Writer
	// Fresh is a release cut: nothing is reused and nothing is synced, and the
	// report's mode says so.
	Fresh bool
	// CachedTimeout is the -timeout of a line whose results are to be reused.
	// go test's cache key holds the timeout, and a stage's own takes one of two
	// values with the load, so a fixed one keeps a result reusable however busy
	// the machine is. Empty keeps the stage's.
	CachedTimeout string
	// LockWait bounds the wait for another check on this directory.
	LockWait time.Duration
	// Uid is the account that must own the check directory.
	Uid int
	// AfterWait, when a check has waited for the lock, takes -p and -timeout again.
	// An error stops the stage.
	AfterWait func() (p, timeout string, err error)
	// RaceGate is the race gate, which a race stage that waited passes again.
	RaceGate func() error
}

// goTestArgs is a recipe's go test command, read closely: a flag the check does
// not know could change what is being checked, so it refuses the recipe.
type goTestArgs struct {
	race            bool
	p, timeout, run string
	patterns        []string
}

func parseGoTest(args []string) (goTestArgs, error) {
	var g goTestArgs
	for i := 0; i < len(args); i++ {
		a := args[i]
		value := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s has no value", a)
			}
			i++
			return args[i], nil
		}
		var err error
		switch {
		case a == "-race":
			g.race = true
		case a == "-p":
			g.p, err = value()
		case a == "-timeout":
			g.timeout, err = value()
		case a == "-run":
			g.run, err = value()
		case strings.HasPrefix(a, "-"):
			err = fmt.Errorf("this check does not know the go test flag %s", a)
		default:
			g.patterns = append(g.patterns, a)
		}
		if err != nil {
			return g, err
		}
	}
	if len(g.patterns) == 0 {
		return g, errors.New("the go test command names no packages")
	}
	return g, nil
}

// flags spells the command with the timeout and any extra flags given.
func (g goTestArgs) flags(timeout string, extra ...string) []string {
	out := []string{"test", "-trimpath"}
	if g.race {
		out = append(out, "-race")
	}
	if g.p != "" {
		out = append(out, "-p", g.p)
	}
	if timeout != "" {
		out = append(out, "-timeout", timeout)
	}
	if g.run != "" {
		out = append(out, "-run", g.run)
	}
	return append(out, extra...)
}

// Run runs the stage and returns its exit code.
func Run(ctx context.Context, cfg Config) int {
	g, err := parseGoTest(cfg.Args)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "check-tests: %v\n", err)
		return ExitUsage
	}
	worktree, err := gitutil.Toplevel(cfg.Worktree)
	if err != nil {
		return runFresh(ctx, cfg, g, modeFallback, "this is not a git checkout", "", "")
	}
	cfg.Worktree = worktree
	head, tree := headAndTree(ctx, worktree)
	if why := treeNotCommit(ctx, worktree); why != "" {
		// The tests read the edits, so the report vouches for no commit's tree.
		return runFresh(ctx, cfg, g, modeDirty, why, head, "")
	}
	if cfg.Fresh {
		return runFresh(ctx, cfg, g, modeCut, "a release cut reuses nothing", head, tree)
	}
	list, _, err := ReadList(ctx, worktree)
	if err != nil {
		return runFresh(ctx, cfg, g, modeFallback, err.Error(), head, tree)
	}
	root, err := gitutil.RootCommit(worktree)
	if err != nil || root == "" {
		return runFresh(ctx, cfg, g, modeFallback, "no root commit names this repository", head, tree)
	}
	dir, err := checkDir(cfg.Home, root, cfg.Uid)
	if err != nil {
		return runFresh(ctx, cfg, g, modeFallback, err.Error(), head, tree)
	}

	var (
		waited   bool
		toGate   bool // a race stage that waited owes the gate before it runs
		atGate   bool // set when it found that holding the lock, to go and pass it
		plan     Plan
		ran      bool
		failed   bool
		fallback string
		refused  error
		tally    Tally
	)
	locked := func() error {
		if toGate {
			atGate = true
			return nil
		}
		noteHolder(dir, worktree)
		if waited && cfg.AfterWait != nil {
			p, timeout, err := cfg.AfterWait()
			if err != nil {
				refused = err
				return nil
			}
			g.p, g.timeout = p, timeout
		}
		plan, failed, fallback = runReusedLines(ctx, cfg, g, dir, list, head, &tally)
		ran = fallback == ""
		return nil
	}
	lock := filepath.Join(dir, lockName)
	deadline := time.Now().Add(cfg.LockWait)
	for {
		err = fsutil.WithFileLock(lock, lockSlice, locked)
		// The race gate can wait for hours, so a stage passes it holding nothing,
		// and every other stage of the repository stays free to run. Losing the
		// lock to one meanwhile means the gate again.
		if atGate {
			atGate, toGate = false, false
			if err := cfg.RaceGate(); err != nil {
				fmt.Fprintf(cfg.Stderr, "check-tests: after the wait: %v. Nothing ran.\n", err)
				return 1
			}
			continue
		}
		if !errors.Is(err, fsutil.ErrLockContention) {
			break
		}
		if !waited {
			waited = true
			fmt.Fprintf(cfg.Stdout, "check-tests: waiting for another check of this repository on this account to finish: %s\n", holder(dir))
		}
		toGate = g.race && cfg.RaceGate != nil
		if ctx.Err() != nil {
			fmt.Fprintf(cfg.Stderr, "check-tests: cancelled while waiting for %s. Nothing ran.\n", dir)
			return ExitBusy
		}
		if time.Now().After(deadline) {
			fmt.Fprintf(cfg.Stderr, "check-tests: another check still holds %s after %s: %s. Nothing ran.\n", dir, cfg.LockWait, holder(dir))
			return ExitBusy
		}
	}
	switch {
	case err != nil:
		return runFresh(ctx, cfg, g, modeFallback, "the check directory's lock: "+err.Error(), head, tree)
	case refused != nil:
		fmt.Fprintf(cfg.Stderr, "check-tests: after the wait: %v. Nothing ran.\n", refused)
		return 1
	case !ran:
		return runFresh(ctx, cfg, g, modeFallback, fallback, head, tree)
	}
	r := lineRunner{ctx: ctx, cfg: cfg, tally: &tally}
	failed = runRerunLines(r, g, plan, cfg) || failed
	return finish(cfg, g, &tally, modeReuse, "", head, tree, &plan, failed)
}

// lockSlice is how long one try at the lock lasts, so a wait of hours still hears a
// cancellation.
const lockSlice = 500 * time.Millisecond

// The modes a stage's report names.
const (
	modeReuse    = "reuse"    // from the check directory
	modeCut      = "cut"      // a release cut: nothing reused, by asking
	modeFallback = "fallback" // nothing reused, for the reason the report gives
	modeDirty    = "dirty"    // the working tree's edits: vouches for no commit
)

// checkDir makes and vets the account's check directory. Two accounts on one
// machine each have their own under their own home, and each refuses one it
// does not own or that anyone else can write: a tree another account could
// write is a tree whose tests this account would run.
func checkDir(home, root string, uid int) (string, error) {
	if home == "" {
		return "", errors.New("no home directory to keep the check directory in")
	}
	tests := abcdhome.Path(home, "tests")
	dir := filepath.Join(tests, root)
	if err := os.MkdirAll(abcdhome.Path(home), abcdhome.DirMode); err != nil {
		return "", err
	}
	for _, d := range []string{tests, dir} {
		if err := os.Mkdir(d, abcdhome.DirMode); err != nil && !errors.Is(err, os.ErrExist) {
			return "", err
		}
		if err := ownedDir(d, uid); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func ownedDir(d string, uid int) error {
	info, err := os.Lstat(d)
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	switch {
	case info.Mode()&os.ModeSymlink != 0 || !info.IsDir():
		return fmt.Errorf("%s is not a directory", d)
	case !ok || int(st.Uid) != uid:
		return fmt.Errorf("%s belongs to another account", d)
	case info.Mode().Perm()&0o022 != 0:
		return fmt.Errorf("%s can be written by another account", d)
	}
	return nil
}

// noteHolder says who holds the lock, for a waiter's message. It is a courtesy:
// the lock is the flock, which a crash releases.
func noteHolder(dir, worktree string) {
	f, err := openNoFollow(filepath.Join(dir, holderName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "pid %d in %s", os.Getpid(), worktree)
}

func holder(dir string) string {
	f, err := openNoFollow(filepath.Join(dir, holderName), os.O_RDONLY)
	if err != nil {
		return "holder not recorded"
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 4096))
	if err != nil {
		return "holder not recorded"
	}
	return string(bytes.TrimSpace(raw))
}

func headAndTree(ctx context.Context, worktree string) (head, tree string) {
	if out, err := gitOutput(ctx, worktree, "rev-parse", "HEAD"); err == nil {
		head = strings.TrimSpace(string(out))
	}
	if out, err := gitOutput(ctx, worktree, "rev-parse", "HEAD^{tree}"); err == nil {
		tree = strings.TrimSpace(string(out))
	}
	return head, tree
}

// treeNotCommit says why the working tree is not the commit, reading the state
// the receipt reads, so the check and the receipt judge one tree. Only a clean
// tree goes to the check directory, which holds the commit: a run with edits in
// it must test the edits.
func treeNotCommit(ctx context.Context, worktree string) string {
	script := filepath.Join(worktree, "scripts", "preflight-receipt.sh")
	cmd := exec.CommandContext(ctx, "bash", script, "state")
	cmd.Dir = worktree
	cmd.Env = gitEnv()
	out, err := cmd.Output()
	if err != nil {
		return "the tree's state could not be read"
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 || fields[1] != "clean" {
		return "the working tree differs from HEAD, so its edits are what to test"
	}
	return ""
}

// runFresh runs the stage as written, in the worktree, reusing nothing. It is
// every path where reuse cannot be shown safe, and a release cut.
func runFresh(ctx context.Context, cfg Config, g goTestArgs, mode, why, head, tree string) int {
	fmt.Fprintf(cfg.Stdout, "check-tests: %s stage reuses nothing: %s\n", cfg.Stage, why)
	var tally Tally
	r := lineRunner{ctx: ctx, cfg: cfg, tally: &tally}
	args := g.flags(g.timeout, "-count=1")
	failed := r.run(goLine{dir: cfg.Worktree, args: append(args, g.patterns...)})
	return finish(cfg, g, &tally, mode, why, head, tree, nil, failed)
}

// runReusedLines brings the check directory to the commit, plans the stage and
// runs the lines that use the tree, holding its lock. A non-empty last result is a
// reason to run fresh instead, with nothing yet run.
func runReusedLines(ctx context.Context, cfg Config, g goTestArgs, dir string, list List, head string, tally *Tally) (Plan, bool, string) {
	stats, snap, err := syncTree(ctx, cfg.Worktree, head, dir)
	if err != nil {
		return Plan{}, false, "the check directory could not be brought to the commit: " + err.Error()
	}
	fmt.Fprintf(cfg.Stdout, "check-tests: %s stage from %s at %.12s: %d files written, %d removed, %d directories retimed, %d files already right, in %s\n",
		cfg.Stage, snap.tree, head, stats.Written, stats.Removed, stats.Retimed, stats.Kept, stats.Took.Round(time.Millisecond))

	env := reusedLineEnv(dir, snap.tree)
	all, err := listPackages(ctx, snap.tree, env, g.race, "./...")
	if err != nil {
		return Plan{}, false, err.Error()
	}
	named, err := listPackages(ctx, snap.tree, env, g.race, g.patterns...)
	if err != nil {
		return Plan{}, false, err.Error()
	}
	rels := map[string]string{}
	known := map[string][]string{}
	for _, p := range all {
		rel, err := filepath.Rel(snap.tree, p.Dir)
		if err != nil {
			return Plan{}, false, err.Error()
		}
		rels[p.ImportPath] = filepath.ToSlash(rel)
		known[filepath.ToSlash(rel)] = testNames(p)
	}
	if bad := unresolved(list, known); len(bad) > 0 {
		return Plan{}, false, ListPath + " on the default branch names what this tree does not hold: " + strings.Join(firstFew(bad), ", ") +
			"; merging origin's default branch clears it when this branch is behind it"
	}
	pkgs := make([]string, len(named))
	for i, p := range named {
		pkgs[i] = p.ImportPath
	}
	plan := MakePlan(pkgs, rels, Survey(all), list, g.run != "")

	r := lineRunner{ctx: ctx, cfg: cfg, tally: tally}
	failed := runReused(r, g, plan, snap.tree, env, cfg)
	if changed := snap.changedTracked(); len(changed) > 0 {
		failed = true
		fmt.Fprintf(cfg.Stdout, "check-tests: a test rewrote tracked files in the check directory, so what ran after it was not the commit: %s\n", strings.Join(firstFew(changed), ", "))
	}
	if left := snap.leftBehind(); len(left) > 0 {
		fmt.Fprintf(cfg.Stdout, "check-tests: a test left files in the check directory, so a test that walks the tree will not replay: %s\n", strings.Join(firstFew(left), ", "))
	}
	return plan, failed, ""
}

// reusedLineEnv is the environment of a line whose results are to be reused. It
// says nothing of the worktree the check started in: go test's key holds a
// variable a test reads, at the value the go command itself was given, so PWD and
// OLDPWD would tie a result to one worktree. Git may not walk above the check
// directory, so a test asking about the repository gets no answer, and go reads no
// go.work from anywhere above the module.
func reusedLineEnv(dir, tree string) []string {
	var env []string
	for _, kv := range gitEnv() {
		name, _, _ := strings.Cut(kv, "=")
		if name != "PWD" && name != "OLDPWD" && name != "GOWORK" {
			env = append(env, kv)
		}
	}
	return append(env, "PWD="+tree, "GIT_CEILING_DIRECTORIES="+dir, "GOWORK=off")
}

func firstFew(names []string) []string {
	if len(names) > 10 {
		return append(slices.Clone(names[:10]), fmt.Sprintf("and %d more", len(names)-10))
	}
	return names
}

// runReused runs a plan's lines that use the tree, every one whatever the one
// before did, as go test runs every package whatever the one before did. The lines
// run one after another, or within -p of each other, so the stage keeps to its -p.
func runReused(r lineRunner, g goTestArgs, plan Plan, tree string, env []string, cfg Config) bool {
	cachedTimeout := cfg.CachedTimeout
	if cachedTimeout == "" {
		cachedTimeout = g.timeout
	}
	failed := false
	if len(plan.Cached) > 0 {
		failed = r.run(goLine{dir: tree, env: env, args: append(g.flags(cachedTimeout), plan.Cached...)}) || failed
	}

	// A split package reuses everything but its always-rerun tests, under a
	// -skip of those alone, so one package's list does not change another's
	// cache key. One go test per package, within the stage's -p.
	limit := 1
	if n, err := fmt.Sscan(g.p, &limit); err != nil || n != 1 || limit < 1 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, route := range plan.Split {
		wg.Add(1)
		sem <- struct{}{}
		go func(route PackageRoute) {
			defer wg.Done()
			defer func() { <-sem }()
			own := g
			own.p = "1"
			args := append(own.flags(cachedTimeout, "-skip", exactly(route.testNames())), route.Path)
			if r.buffered(goLine{dir: tree, env: env, args: args}) {
				mu.Lock()
				failed = true
				mu.Unlock()
			}
		}(route)
	}
	wg.Wait()
	return failed
}

// runRerunLines runs a plan's always-rerun lines. They run in the worktree with
// -count=1 and touch nothing of the check directory, so they run outside its lock.
func runRerunLines(r lineRunner, g goTestArgs, plan Plan, cfg Config) bool {
	failed := false
	if len(plan.Rerun) > 0 {
		var pkgs []string
		for _, route := range plan.Rerun {
			pkgs = append(pkgs, route.Path)
		}
		failed = r.run(goLine{dir: cfg.Worktree, args: append(g.flags(g.timeout, "-count=1"), pkgs...)}) || failed
	}
	if len(plan.Split) > 0 {
		var pkgs, names []string
		for _, route := range plan.Split {
			pkgs = append(pkgs, route.Path)
			names = append(names, route.testNames()...)
		}
		slices.Sort(names)
		names = slices.Compact(names)
		// One pattern for every split package: a name two packages share runs
		// in both, which only reruns a test that had no need to.
		failed = r.run(goLine{dir: cfg.Worktree, args: append(g.flags(g.timeout, "-count=1", "-run", exactly(names)), pkgs...)}) || failed
	}
	return failed
}

// goLine is one go command.
type goLine struct {
	dir  string
	env  []string
	args []string
}

// lineRunner runs go commands, feeding a tally.
type lineRunner struct {
	ctx   context.Context
	cfg   Config
	tally *Tally
}

// run runs one line with its output streamed, and reports a failure.
func (r lineRunner) run(l goLine) bool {
	w := &lineWriter{tally: r.tally, out: r.cfg.Stdout}
	failed := r.exec(l, w, r.cfg.Stderr)
	w.flush()
	return failed
}

var outputMu sync.Mutex

// buffered runs one line of several at once, writing its output whole when it
// ends so two packages' output never interleave.
func (r lineRunner) buffered(l goLine) bool {
	out := &lockedBuffer{}
	w := &lineWriter{tally: r.tally, out: out}
	failed := r.exec(l, w, out)
	w.flush()
	outputMu.Lock()
	defer outputMu.Unlock()
	_, _ = r.cfg.Stdout.Write(out.buf.Bytes())
	return failed
}

// lockedBuffer takes a command's standard output and error at once, each of
// which go copies on a goroutine of its own.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (r lineRunner) exec(l goLine, stdout, stderr io.Writer) bool {
	cmd := exec.CommandContext(r.ctx, "go", l.args...)
	cmd.Dir = l.dir
	cmd.Env = l.env
	if cmd.Env == nil {
		cmd.Env = gitEnv()
	}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run() != nil
}

// finish writes the stage's report and the receipt's record and returns its
// exit code.
func finish(cfg Config, g goTestArgs, tally *Tally, mode, why, head, tree string, plan *Plan, failed bool) int {
	c := tally.Counts()
	fmt.Fprintf(cfg.Stdout, "check-tests: %s stage reused %d of %d packages (%d with no test run, %d with only their always-rerun tests run); %d ran\n",
		cfg.Stage, c.Reused, c.Packages, c.Whole, c.Reused-c.Whole, len(c.Ran))
	for _, name := range c.Ran {
		fmt.Fprintf(cfg.Stdout, "check-tests:   ran %s: %s\n", name, ranWhy(plan, name, mode))
	}
	if head != "" {
		report := StageReport{Commit: head, Tree: tree, Stage: cfg.Stage, Mode: mode, Why: why, Counts: c, Passed: !failed}
		if err := report.write(cfg.Worktree, append([]string{g.run, fmt.Sprint(g.race)}, g.patterns...)); err != nil {
			fmt.Fprintf(cfg.Stderr, "check-tests: the stage's report was not written, so no receipt will be minted: %v\n", err)
			return 1
		}
	}
	if failed {
		return 1
	}
	return 0
}

// ranWhy says why a package ran.
func ranWhy(plan *Plan, name, mode string) string {
	if plan == nil {
		return "nothing is reused on this run (" + mode + ")"
	}
	for _, route := range plan.Rerun {
		if route.Path == name {
			return route.Rerun
		}
	}
	for _, route := range plan.Split {
		if route.Path == name {
			names := route.testNames()
			return fmt.Sprintf("%d always-rerun tests, as %s %s", len(names), names[0], route.Tests[names[0]])
		}
	}
	return "its inputs changed, or this stage has no result for them yet"
}
