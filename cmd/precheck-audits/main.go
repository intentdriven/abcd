// Command precheck-audits runs, in every package `make precheck` does not
// already cover, the tests that audit the committed tree, so that a lane meets
// them in minutes rather than most of an hour into a preflight
// (iss-2611061656036069, iss-2610230048490215). There is no list of tests to
// keep: a new audit is chosen by what it does.
//
// A test is an audit here by testreuse.Audits: it reaches the repository tree
// from its package directory, in its own body, in a same-package helper it names
// or in a package-level value, and not only through a helper that starts a process.
// A literal naming an ancestor directory ("../../..") handed to a helper counts,
// so a test whose helper runs go list there is chosen (the drive's git table in
// internal/core/implement/loop).
//
// What it cannot see: a tree reached through a helper in another package, other
// than fsutil.ModuleRoot, and a climb assembled at run time. The command and
// site packages stay on the name pattern in the Makefile (PRECHECK_AUDIT_TESTS),
// where hundreds of tests reach the root and a whole selection would not be
// minutes. TestPrecheckRunsEveryAuditNamedTest (coverageGaps) checks the whole
// of precheck's choice with a wider detector of its own.
//
// Usage, from the repository root:
//
//	go run ./cmd/precheck-audits ./internal/already/run ./internal/also/run
//
// Each argument is a package the recipe runs another way, left out here. It
// exits with go test's own failure, and 2 when it cannot read a test file, is
// given something that is not a package path, or finds nothing at all, which
// would mean the detector stopped working rather than that there are no audits.
//
// Beside a command, for make preflight (iss-2611232155447855):
//
//	go run ./cmd/precheck-audits -whole '<packages>' -named '<packages>' -run '<pattern>' -- <command>...
//
// runs precheck's three test lines (the whole packages in full, the named
// packages' tests the pattern matches, and the audits chosen everywhere else) as
// narrow runs, each a go test of at most machineload.MaxNarrowPackages packages
// through `abcd implement load --wait --narrow`, while the command runs (the
// whole run's gate and its stages). It is beside the whole run's wait, not
// before it, because a make preflight that has no waiter queued beneath it is a
// running preflight to every other run's first gate (machineload's queue), so
// audits run ahead of the wait would hold every other preflight back for as
// long as they took. The first audit run that fails stops the command and every
// other run, each through the process group it leads, and the exit is that
// run's, so make mints no receipt. Each group's leader is this binary (lead),
// which kills its group once this process is gone, so a kill of make's own
// group (abcd land's bound, a drive's lane) leaves no run behind. When the command ends first, its verdict is
// the exit and the audit runs still going stop: its unit and race stages run
// every one of these tests. An effective GOFLAGS, which makes no run narrow,
// runs the command alone and says so.
//
// It is developer tooling, never part of a release.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/testreuse"
)

// timeout is the budget of precheck's other test lines.
const timeout = "20m"

// selection is what one run hands to go test.
type selection struct {
	names    []string
	packages []string
	// byPackage is each selected package's audit tests.
	byPackage map[string][]string
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == leadFlag {
		os.Exit(lead(os.Args[2:]))
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// leadFlag starts this binary as the leader of one run's process group
// (lead), the first argument alone; nothing else spells it.
const leadFlag = "-lead-run-group"

// lead runs argv, after a `--`, as the child of a process that leads its own
// group (the runner started it so), and kills that whole group, itself
// included, once its standard input ends: the runner holds the only write end,
// so that is the runner gone, whatever stopped it. A make preflight stopped by
// a kill of its own process group (abcd land's bound, a drive's lane) kills
// the runner with it but not the groups the runner's runs lead, which would
// otherwise outlive it. It exits with argv's exit as exitCode reads it, and
// refuses (2) when it does not lead its group, since a kill of the group it is
// in would then reach the runner's.
func lead(args []string) int {
	if len(args) < 2 || args[0] != "--" {
		fmt.Fprintln(os.Stderr, "precheck-audits: "+leadFlag+" takes -- and a command")
		return 2
	}
	if syscall.Getpgrp() != os.Getpid() {
		fmt.Fprintln(os.Stderr, "precheck-audits: "+leadFlag+" does not lead its process group, so it runs nothing")
		return 2
	}
	go func() {
		_, _ = io.Copy(io.Discard, os.Stdin)
		_ = syscall.Kill(0, syscall.SIGKILL)
	}()
	cmd := exec.Command(args[1], args[2:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return exitCode(cmd.Run())
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fset := flag.NewFlagSet("precheck-audits", flag.ContinueOnError)
	fset.SetOutput(stderr)
	whole := fset.String("whole", "", "beside a command: the packages precheck runs in full")
	named := fset.String("named", "", "beside a command: the packages precheck runs the -run pattern's tests of")
	pattern := fset.String("run", "", "beside a command: the pattern of the named packages' audit tests")
	if err := fset.Parse(args); err != nil {
		return 2
	}
	rest := fset.Args()
	dash := slices.Contains(args, "--")
	if dash {
		if len(rest) == 0 {
			fmt.Fprintln(stderr, "precheck-audits: nothing to run after --")
			return 2
		}
		plan, err := preflightPlan(".", strings.Fields(*whole), strings.Fields(*named), *pattern)
		if err != nil {
			fmt.Fprintf(stderr, "precheck-audits: %v\n", err)
			return 2
		}
		return beside(ctx, plan, rest, stdout, stderr, besideOptions{})
	}
	if *whole != "" || *named != "" || *pattern != "" {
		fmt.Fprintln(stderr, "precheck-audits: -whole, -named and -run are for a run beside a command, after --")
		return 2
	}
	for _, arg := range rest {
		if !strings.HasPrefix(arg, "./") {
			fmt.Fprintf(stderr, "precheck-audits: %q is not a package path starting ./\n", arg)
			return 2
		}
	}
	got, err := selectAudits(".", rest)
	if err == nil && len(got.names) == 0 {
		err = errors.New("found no audit test at all, so the detector has stopped working")
	}
	if err != nil {
		fmt.Fprintf(stderr, "precheck-audits: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "precheck-audits: %d audit tests in %d packages\n", len(got.names), len(got.packages))
	goArgs := []string{"test", "-timeout", timeout, "-run", "^(" + strings.Join(got.names, "|") + ")$"}
	cmd := exec.CommandContext(ctx, "go", append(goArgs, got.packages...)...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			return exit.ExitCode()
		}
		fmt.Fprintf(stderr, "precheck-audits: %v\n", err)
		return 2
	}
	return 0
}

// testFiles maps each package directory under root's internal and cmd trees,
// outside testdata, to its test files.
func testFiles(root string) (map[string][]string, error) {
	dirs := map[string][]string{}
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir() && d.Name() == "testdata":
				return filepath.SkipDir
			case !d.IsDir() && strings.HasSuffix(path, "_test.go"):
				dirs[filepath.Dir(path)] = append(dirs[filepath.Dir(path)], path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return dirs, nil
}

// packagePath is dir's package as go test names it from root: ./internal/x.
func packagePath(root, dir string) (string, error) {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return "", err
	}
	return "./" + filepath.ToSlash(rel), nil
}

// selectAudits walks root's internal and cmd trees and returns the audit tests
// of every package outside skip, names and packages both sorted.
func selectAudits(root string, skip []string) (selection, error) {
	dirs, err := testFiles(root)
	if err != nil {
		return selection{}, err
	}
	got := selection{byPackage: map[string][]string{}}
	for dir, files := range dirs {
		pkg, err := packagePath(root, dir)
		if err != nil {
			return selection{}, err
		}
		if slices.Contains(skip, pkg) {
			continue
		}
		names, err := testreuse.Audits(files)
		if err != nil {
			return selection{}, fmt.Errorf("%s: %w", pkg, err)
		}
		if len(names) > 0 {
			got.names = append(got.names, names...)
			got.packages = append(got.packages, pkg)
			got.byPackage[pkg] = names
		}
	}
	slices.Sort(got.names)
	got.names = slices.Compact(got.names)
	slices.Sort(got.packages)
	return got, nil
}

// auditRun is one go test of the plan: a -run pattern, or none, over at most
// machineload.MaxNarrowPackages packages.
type auditRun struct {
	run      string
	packages []string
}

// goTest is the run's go test command, its program first, with precheck's own
// -timeout, so a result precheck left in go's test cache is one this replays.
func (r auditRun) goTest() []string {
	cmd := []string{"go", "test", "-timeout", timeout}
	if r.run != "" {
		cmd = append(cmd, "-run", r.run)
	}
	return append(cmd, r.packages...)
}

// preflightPlan is precheck's three test lines in narrow runs: whole's
// packages in full, named's with pattern, and the audits selectAudits chooses
// everywhere else, each run's -run naming its own packages' audits alone.
// Every run is one the narrow gate reads as narrow, or the plan is refused.
func preflightPlan(root string, whole, named []string, pattern string) ([]auditRun, error) {
	if len(whole) == 0 || len(named) == 0 || pattern == "" {
		return nil, errors.New("beside a command, -whole, -named and -run are each required")
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return nil, fmt.Errorf("-run is not a pattern go test takes: %w", err)
	}
	for _, pkg := range append(slices.Clone(whole), named...) {
		if !strings.HasPrefix(pkg, "./") {
			return nil, fmt.Errorf("%q is not a package path starting ./", pkg)
		}
	}
	got, err := selectAudits(root, append(slices.Clone(whole), named...))
	if err == nil && len(got.names) == 0 {
		err = errors.New("found no audit test at all, so the detector has stopped working")
	}
	if err != nil {
		return nil, err
	}
	var plan []auditRun
	for _, group := range chunk(whole) {
		plan = append(plan, auditRun{packages: group})
	}
	for _, group := range chunk(named) {
		plan = append(plan, auditRun{run: pattern, packages: group})
	}
	for _, group := range chunk(got.packages) {
		var names []string
		for _, pkg := range group {
			names = append(names, got.byPackage[pkg]...)
		}
		slices.Sort(names)
		plan = append(plan, auditRun{run: "^(" + strings.Join(slices.Compact(names), "|") + ")$", packages: group})
	}
	for _, r := range plan {
		if !machineload.IsNarrowCommand(r.goTest()) {
			return nil, fmt.Errorf("the run %v is not one the narrow gate reads as narrow", r.goTest())
		}
	}
	return plan, nil
}

// chunk splits packages into groups of at most machineload.MaxNarrowPackages.
func chunk(packages []string) [][]string {
	var out [][]string
	for len(packages) > 0 {
		n := min(len(packages), machineload.MaxNarrowPackages)
		out = append(out, packages[:n])
		packages = packages[n:]
	}
	return out
}

// workers is how many audit runs wait or run at once. The narrow gate bounds
// them again, by its own limit and its account's hold between openings.
const workers = 2

// grace bounds the wait for a stopped run's output.
const grace = 10 * time.Second

// besideOptions are what a test replaces: how an audit run and the go flags
// probe are started.
type besideOptions struct {
	// gate is the command an audit run starts, its program first.
	gate func(r auditRun) []string
	// goFlags reads the effective GOFLAGS.
	goFlags func() (string, error)
	// self is the binary that leads each run (lead): this one.
	self string
}

// gateCommand waits for the run at the narrow gate, through the checkout's own
// abcd, for as long as make preflight's whole run may wait.
func gateCommand(r auditRun) []string {
	return append([]string{"go", "run", "./cmd/abcd", "implement", "load", "--wait", "--narrow", "--timeout", "12h", "--"}, r.goTest()...)
}

// effectiveGoFlags is go env GOFLAGS, with go found as the gate finds it.
func effectiveGoFlags() (string, error) {
	wd, _ := os.Getwd()
	goBin, err := tools.LookPath("go", wd)
	if err != nil {
		return "", err
	}
	out, err := exec.Command(goBin, "env", "GOFLAGS").Output()
	return strings.TrimSpace(string(out)), err
}

// event is one started process ending: an audit run (index into the plan) or
// the command (index -1).
type event struct {
	index int
	code  int
	out   []byte
}

// beside runs plan's audit runs while command runs, and returns the exit the
// first failure or the command's end decides (the package comment).
func beside(ctx context.Context, plan []auditRun, command []string, stdout, stderr io.Writer, o besideOptions) int {
	if o.gate == nil {
		o.gate = gateCommand
	}
	if o.goFlags == nil {
		o.goFlags = effectiveGoFlags
	}
	if o.self == "" {
		self, err := os.Executable()
		if err != nil {
			fmt.Fprintf(stderr, "precheck-audits: cannot name its own binary to lead the runs: %v\n", err)
			return 2
		}
		o.self = self
	}
	// Deferred in this order, so every run still going is stopped (cancel, which
	// kills the group it leads) before the wait that reaps it.
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := make(chan event, len(plan)+1)

	// Each run is argv beneath a leader of its own group (lead), which this
	// process stops through the group's handle (procgroup.Bound) and which
	// stops the group itself when this process goes, its standard input this
	// process's alone.
	start := func(index int, argv []string, out io.Writer) error {
		cmd := exec.CommandContext(ctx, o.self, append([]string{leadFlag, "--"}, argv...)...)
		procgroup.Bound(cmd, grace)
		cmd.Env = withoutClock(os.Environ())
		var buf bytes.Buffer
		if out == nil {
			cmd.Stdout, cmd.Stderr = &buf, &buf
		} else {
			cmd.Stdout, cmd.Stderr = out, stderr
			cmd.Env = os.Environ()
		}
		watch, held, err := os.Pipe()
		if err != nil {
			return err
		}
		cmd.Stdin = watch
		err = cmd.Start()
		_ = watch.Close()
		if err != nil {
			_ = held.Close()
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			code := exitCode(cmd.Wait())
			procgroup.KillLeft(cmd)
			_ = held.Close()
			events <- event{index: index, code: code, out: buf.Bytes()}
		}()
		return nil
	}

	if flags, err := o.goFlags(); err != nil || flags != "" {
		why := fmt.Sprintf("the effective GOFLAGS is %q", flags)
		if err != nil {
			why = "go env GOFLAGS could not be read (" + err.Error() + ")"
		}
		fmt.Fprintf(stdout, "precheck-audits: %s, so no audit run is narrow; the preflight runs without its early audits\n", why)
		plan = nil
	}
	if err := start(-1, command, stdout); err != nil {
		fmt.Fprintf(stderr, "precheck-audits: the command could not be started: %v\n", err)
		return 127
	}
	fmt.Fprintf(stdout, "precheck-audits: %d audit runs beside the whole run, %d at a time; the first to fail stops the preflight\n", len(plan), workers)

	next, running, passed := 0, 0, 0
	launch := func() {
		for running < workers && next < len(plan) {
			if err := start(next, o.gate(plan[next]), nil); err != nil {
				events <- event{index: next, code: 127, out: []byte(err.Error() + "\n")}
			}
			next++
			running++
		}
	}
	launch()
	for {
		var e event
		select {
		case e = <-events:
		case <-ctx.Done():
		}
		// Only an interrupt ends the context while this loop runs, and a run it
		// killed can end before the loop sees it, so it is asked after either.
		if ctx.Err() != nil {
			// The deferred cancel and wait stop and reap every run.
			fmt.Fprintln(stderr, "precheck-audits: interrupted; stopping the preflight and its audit runs")
			return 130
		}
		if e.index < 0 {
			if left := len(plan) - passed; left > 0 && e.code == 0 {
				fmt.Fprintf(stdout, "precheck-audits: the whole run passed with %d audit runs not finished; its stages ran their tests, so they stop\n", left)
			}
			return e.code
		}
		running--
		r := plan[e.index]
		if e.code != 0 {
			_, _ = stdout.Write(e.out)
			fmt.Fprintf(stdout, "precheck-audits: FAILED: %s; stopping the preflight, which mints no receipt\n", strings.Join(r.goTest(), " "))
			return e.code
		}
		passed++
		fmt.Fprintf(stdout, "precheck-audits: passed %d of %d: %s\n", passed, len(plan), strings.Join(r.packages, " "))
		if passed == len(plan) {
			fmt.Fprintln(stdout, "precheck-audits: every audit run passed; the preflight goes on")
		}
		launch()
	}
}

// withoutClock is env without make preflight's clock, which no test reads and
// which would change go's test cache key with every run (internal/testreuse
// strips it the same way).
func withoutClock(env []string) []string {
	return slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		return strings.HasPrefix(kv, testreuse.ClockEnv+"=")
	})
}

// exitCode is a finished process's exit: its own code, 128 plus a signal that
// stopped it, and 1 for anything else that is not success.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if code := exit.ExitCode(); code > 0 {
			return code
		}
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
	}
	return 1
}

// isAuditName reports whether a test's name says it holds every case of
// something to a rule: TestNo<Word> or TestEvery<Word>.
func isAuditName(name string) bool {
	for _, prefix := range []string{"TestNo", "TestEvery"} {
		if rest, ok := strings.CutPrefix(name, prefix); ok && rest != "" && rest[0] >= 'A' && rest[0] <= 'Z' {
			return true
		}
	}
	return false
}

// notAudits are the tests coverageGaps would name that are not audits of the
// committed tree, each with what it does instead. A key is package and name.
var notAudits = map[string]string{
	"./internal/core/landing.TestEveryWorktreesReceiptIsRead": "reads the receipts of worktrees it adds to a fixture repository",
	"./internal/testreuse.TestNoListMeansNoReuse":             "runs the check against a fixture repository with no list",
}

// coverageGaps names, as package.name, each test precheck does not run whose
// name says it is an audit (isAuditName) and whose reach touches the
// repository tree by testreuse.ReachesTree, which is wider than the selection's
// detector: it follows helpers that start a process and counts any literal
// naming an ancestor directory. Precheck runs a test when its package is in
// whole, when its package is in named and pattern matches its name, or when
// selectAudits chooses it; notAudits names the rest that are not audits. An
// entry of exempt that names no test the detector finds is named too, so the
// list cannot outlive what it excuses.
//
// What it cannot see: an audit whose name does not start TestNo or TestEvery,
// unless the selection chooses it anyway, and a tree reached in a way neither
// detector reads (a helper in another package, a path built at run time).
func coverageGaps(root string, whole, named []string, pattern string, exempt map[string]string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	dirs, err := testFiles(root)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	var gaps []string
	for dir, files := range dirs {
		pkg, err := packagePath(root, dir)
		if err != nil {
			return nil, err
		}
		if slices.Contains(whole, pkg) {
			continue
		}
		reaching, err := testreuse.ReachesTree(files)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pkg, err)
		}
		var chosen []string
		if !slices.Contains(named, pkg) {
			if chosen, err = testreuse.Audits(files); err != nil {
				return nil, fmt.Errorf("%s: %w", pkg, err)
			}
		}
		for _, name := range reaching {
			key := pkg + "." + name
			switch {
			case !isAuditName(name):
			case slices.Contains(named, pkg) && re.MatchString(name):
			case slices.Contains(chosen, name):
			case exempt[key] != "":
				used[key] = true
			default:
				gaps = append(gaps, key)
			}
		}
	}
	for key := range exempt {
		if !used[key] {
			gaps = append(gaps, key+" (named in notAudits, but no longer a test the detector finds outside precheck)")
		}
	}
	slices.Sort(gaps)
	return gaps, nil
}
