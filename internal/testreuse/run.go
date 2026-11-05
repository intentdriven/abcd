// Package testreuse runs one of make preflight's test stages so that a test
// whose inputs have not changed is not run again (itd-2611011325421304).
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
// builds, the clock or the network, so a result must not be reused for those.
// Three things hold that, in this order:
//
//   - The always-rerun list on the default branch (ReadList) names a package or
//     a test that reads what the cache cannot see. It is read from the branch,
//     so a change cannot take its own tests off it; a list that is missing or
//     does not read means nothing is reused.
//   - The classifier (ClassifyFiles) reads every test's source on every check.
//     A test it finds evidence against reruns whether declared or not, and the
//     report names it as undeclared. A file it cannot read reruns its package.
//   - The check directory holds no .git and git may not walk above it, so a
//     test that asks git about the repository from there fails or skips rather
//     than reading state the cache cannot see. An always-rerun test runs in the
//     worktree, where git answers, and is never reused.
//
// What none of the three sees is a test the classifier finds no evidence
// against, whose subprocess reads the tree by a path or a working directory
// its source does not spell, or that compares the clock with a date in code it
// links. That is reused here and caught by the hosted full run, which reuses
// nothing, on every pull request and merge-queue entry.
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
	// Fresh is a release cut: nothing is reused and nothing is synced.
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
		return runFresh(ctx, cfg, g, "this is not a git checkout", "", "")
	}
	cfg.Worktree = worktree
	head, tree := headAndTree(ctx, worktree)
	if cfg.Fresh {
		return runFresh(ctx, cfg, g, "a release cut reuses nothing", head, tree)
	}
	if why := treeNotCommit(ctx, worktree); why != "" {
		return runFresh(ctx, cfg, g, why, head, tree)
	}
	list, _, err := ReadList(ctx, worktree)
	if err != nil {
		return runFresh(ctx, cfg, g, err.Error(), head, tree)
	}
	root, err := gitutil.RootCommit(worktree)
	if err != nil || root == "" {
		return runFresh(ctx, cfg, g, "no root commit names this repository", head, tree)
	}
	dir, err := checkDir(cfg.Home, root, cfg.Uid)
	if err != nil {
		return runFresh(ctx, cfg, g, err.Error(), head, tree)
	}

	code := -1
	var fallback string
	locked := func() error {
		noteHolder(dir, worktree)
		c, why := runReusing(ctx, cfg, g, dir, list, head, tree)
		code, fallback = c, why
		return nil
	}
	lock := filepath.Join(dir, lockName)
	err = fsutil.WithFileLock(lock, time.Second, locked)
	if errors.Is(err, fsutil.ErrLockContention) {
		fmt.Fprintf(cfg.Stdout, "check-tests: waiting for another check of this repository on this account to finish: %s\n", holder(dir))
		err = fsutil.WithFileLock(lock, cfg.LockWait, locked)
	}
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		fmt.Fprintf(cfg.Stderr, "check-tests: another check still holds %s after %s: %s. Nothing ran.\n", dir, cfg.LockWait, holder(dir))
		return ExitBusy
	case err != nil:
		return runFresh(ctx, cfg, g, "the check directory's lock: "+err.Error(), head, tree)
	case fallback != "":
		return runFresh(ctx, cfg, g, fallback, head, tree)
	}
	return code
}

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
	_ = os.WriteFile(filepath.Join(dir, holderName), fmt.Appendf(nil, "pid %d in %s", os.Getpid(), worktree), 0o600)
}

func holder(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, holderName))
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
// every path where reuse cannot be shown safe.
func runFresh(ctx context.Context, cfg Config, g goTestArgs, why, head, tree string) int {
	fmt.Fprintf(cfg.Stdout, "check-tests: %s stage reuses nothing: %s\n", cfg.Stage, why)
	var tally Tally
	r := lineRunner{ctx: ctx, cfg: cfg, tally: &tally}
	args := g.flags(g.timeout, "-count=1")
	failed := r.run(goLine{dir: cfg.Worktree, args: append(args, g.patterns...)})
	return finish(cfg, g, &tally, "fresh", head, tree, nil, failed)
}

// runReusing runs the stage from the check directory, holding its lock. A
// non-empty second result is a reason to run fresh instead, with nothing yet run.
func runReusing(ctx context.Context, cfg Config, g goTestArgs, dir string, list List, head, tree string) (int, string) {
	stats, snap, err := syncTree(ctx, cfg.Worktree, head, dir)
	if err != nil {
		return 0, "the check directory could not be brought to the commit: " + err.Error()
	}
	fmt.Fprintf(cfg.Stdout, "check-tests: %s stage from %s at %.12s: %d files written, %d removed, %d directories retimed, %d files already right, in %s\n",
		cfg.Stage, snap.tree, head, stats.Written, stats.Removed, stats.Retimed, stats.Kept, stats.Took.Round(time.Millisecond))

	env := reusedLineEnv(dir, snap.tree)
	all, err := listPackages(ctx, snap.tree, env, g.race, "./...")
	if err != nil {
		return 0, err.Error()
	}
	named, err := listPackages(ctx, snap.tree, env, g.race, g.patterns...)
	if err != nil {
		return 0, err.Error()
	}
	rels := map[string]string{}
	for _, p := range all {
		rel, err := filepath.Rel(snap.tree, p.Dir)
		if err != nil {
			return 0, err.Error()
		}
		rels[p.ImportPath] = filepath.ToSlash(rel)
	}
	pkgs := make([]string, len(named))
	for i, p := range named {
		pkgs[i] = p.ImportPath
	}
	plan := MakePlan(pkgs, rels, Survey(all), list, g.run != "")

	var tally Tally
	r := lineRunner{ctx: ctx, cfg: cfg, tally: &tally}
	failed := runPlan(r, g, plan, snap.tree, env, cfg)
	changed := snap.changedTracked()
	if len(changed) > 0 {
		failed = true
		fmt.Fprintf(cfg.Stdout, "check-tests: a test rewrote tracked files in the check directory, so what ran after it was not the commit: %s\n", strings.Join(firstFew(changed), ", "))
	}
	if left := snap.leftBehind(); len(left) > 0 {
		fmt.Fprintf(cfg.Stdout, "check-tests: a test left files in the check directory, so a test that walks the tree will not replay: %s\n", strings.Join(firstFew(left), ", "))
	}
	return finish(cfg, g, &tally, "reuse", head, tree, &plan, failed), ""
}

// reusedLineEnv is the environment of a line whose results are to be reused. It
// says nothing of the worktree the check started in: go test's key holds a
// variable a test reads, at the value the go command itself was given, so PWD and
// OLDPWD would tie a result to one worktree. Git may not walk above the check
// directory, so a test asking about the repository gets no answer.
func reusedLineEnv(dir, tree string) []string {
	var env []string
	for _, kv := range gitEnv() {
		name, _, _ := strings.Cut(kv, "=")
		if name != "PWD" && name != "OLDPWD" {
			env = append(env, kv)
		}
	}
	return append(env, "PWD="+tree, "GIT_CEILING_DIRECTORIES="+dir)
}

func firstFew(names []string) []string {
	if len(names) > 10 {
		return append(slices.Clone(names[:10]), fmt.Sprintf("and %d more", len(names)-10))
	}
	return names
}

// runPlan runs a plan's lines in order, every one whatever the one before did,
// as go test runs every package whatever the one before did. The lines run one
// after another, or within -p of each other, so the stage keeps to its -p.
func runPlan(r lineRunner, g goTestArgs, plan Plan, tree string, env []string, cfg Config) bool {
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
func finish(cfg Config, g goTestArgs, tally *Tally, mode, head, tree string, plan *Plan, failed bool) int {
	c := tally.Counts()
	fmt.Fprintf(cfg.Stdout, "check-tests: %s stage reused %d of %d packages (%d with no test run, %d with only their always-rerun tests run); %d ran\n",
		cfg.Stage, c.Reused, c.Packages, c.Whole, c.Reused-c.Whole, len(c.Ran))
	for _, name := range c.Ran {
		fmt.Fprintf(cfg.Stdout, "check-tests:   ran %s: %s\n", name, ranWhy(plan, name, mode))
	}
	if head != "" {
		report := StageReport{Commit: head, Tree: tree, Stage: cfg.Stage, Mode: mode, Counts: c, Passed: !failed}
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
