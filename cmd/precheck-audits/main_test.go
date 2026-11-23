package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/procgroup"
)

// TestMain lets the test binary lead a run's group as the command's binary
// does (lead), since the runner starts its own binary for that.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == leadFlag {
		os.Exit(lead(os.Args[2:]))
	}
	os.Exit(m.Run())
}

// TestSelectsWhatLostPreflights: each of these failed a preflight 42 to 91
// minutes in on 6 and 7 November after precheck had passed
// (iss-2611061656036069), and each reaches the root another way: through a
// package helper around fsutil.ModuleRoot, a local walk up to go.mod, and a
// filepath.Join of parent directories. It reads the live tree, so renaming one
// of them means changing this list too.
func TestSelectsWhatLostPreflights(t *testing.T) {
	got, err := selectAudits(filepath.Join("..", ".."), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"TestCommandSurfaceResolvesBinaryFromPluginRoot",
		"TestTestGitCallsAreHermetic",
		"TestNoPrivateDelimiterCompare",
		// Two that failed preflights on 23 November (iss-2611232155463657): the
		// first was chosen already, through its own walk up to go.mod; the
		// second reaches the tree only through go list, run in an ancestor its
		// helper is handed as a literal, as its siblings do.
		"TestNoTestMakesABareRepositoryByHand",
		"TestEveryGitCommandADriveRunsIsInTheTable",
		"TestEveryIsolatedCallADriveMakesIsJudgedForWhatItCanStart",
		"TestTheExemptConstructorIsBuiltOnlyByTheComparison",
		// A walk from the parent directory, found by the coverage check.
		"TestNoInlineBaseNameDisplayRule",
	} {
		if !slices.Contains(got.names, name) {
			t.Errorf("%s is not selected", name)
		}
	}
	for _, pkg := range []string{"./internal/core/launch", "./internal/gittest", "./internal/core/frontmatter", "./internal/core/implement/loop", "./internal/fsutil"} {
		if !slices.Contains(got.packages, pkg) {
			t.Errorf("%s is not selected", pkg)
		}
	}
}

// TestSelectionFollowsWhatATestDoes holds the detector to both sides of the
// line: what reaches the committed tree is chosen, and a test that only runs a
// script, or writes a go.mod of its own, is not.
func TestSelectionFollowsWhatATestDoes(t *testing.T) {
	const head = "package x\nimport (\n\t\"os\"\n\t\"os/exec\"\n\t\"path/filepath\"\n\t\"testing\"\n\t\"github.com/intentdriven/abcd/internal/fsutil\"\n)\nvar _, _, _ = os.Getwd, exec.Command, filepath.Join\n"
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(head+body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/a/a_test.go", `
func TestJoinClimb(t *testing.T) { _ = filepath.Join("..", "..", "commands") }
func TestLiteralClimb(t *testing.T) { _, _ = os.ReadFile("../../AGENTS.md") }
func TestModuleRoot(t *testing.T) { _, _ = fsutil.ModuleRoot(".") }
func TestThroughHelper(t *testing.T) { _ = rootHelper(t) }
func TestThroughTwoHelpers(t *testing.T) { _ = outerHelper(t) }
func TestWalkUp(t *testing.T) { _ = walkUp() }
func TestInAClosure(t *testing.T) { t.Run("x", func(t *testing.T) { _ = rootHelper(t) }) }
func TestThroughAPackageValue(t *testing.T) { _ = treeRoot }
func TestSpawnsItselfToListFiles(t *testing.T) {
	_ = exec.Command("git", "-C", rootHelper(t), "ls-files").Run()
}
func TestAnAncestorHandedToAHelper(t *testing.T) { _ = listAt("../../..") }
func TestWalkFromTheParent(t *testing.T) { _ = filepath.WalkDir("..", nil) }
func TestATraversalCaseHandedToTheCode(t *testing.T) { _ = refuse("../..") }
func TestATraversalCaseInATable(t *testing.T) { for _, c := range []string{"../.."} { _ = c } }
func listAt(dir string) string {
	c := exec.Command("go", "list", "./...")
	c.Dir = dir
	_ = c.Run()
	return dir
}
func testlowercaseIsNotATest(t *testing.T) { _ = rootHelper(t) }
var treeRoot = filepath.Join("..", "..", "..")
func rootHelper(t *testing.T) string { r, _ := fsutil.ModuleRoot("."); return r }
func outerHelper(t *testing.T) string { return rootHelper(t) }
func walkUp() string {
	d, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		d = filepath.Dir(d)
	}
}
`)
	write("internal/b/b_test.go", `
func TestRunsAHook(t *testing.T) { runHook(t) }
func TestWritesItsOwnGoMod(t *testing.T) {
	_ = os.WriteFile(filepath.Join(t.TempDir(), "go.mod"), nil, 0o644)
}
func TestMain(m *testing.M) { _ = filepath.Join("..", "..") }
func runHook(t *testing.T) {
	_ = exec.Command("bash", filepath.Join(rootHelper(t), "hooks", "x")).Run()
}
func rootHelper(t *testing.T) string { r, _ := fsutil.ModuleRoot("."); return r }
`)
	write("cmd/c/c_test.go", `func TestInACommand(t *testing.T) { _ = filepath.Join("..", "..") }`)
	write("internal/skipped/s_test.go", `func TestSkippedPackage(t *testing.T) { _ = filepath.Join("..", "..") }`)
	write("internal/a/testdata/fixture_test.go", `func TestInTestdata(t *testing.T) { _ = filepath.Join("..", "..") }`)
	write("internal/a/a.go", `func TestNotInATestFile() { _ = filepath.Join("..", "..") }
func refuse(s string) error { return nil }`)

	got, err := selectAudits(root, []string{"./internal/skipped"})
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{
		"TestAnAncestorHandedToAHelper",
		"TestInAClosure", "TestInACommand", "TestJoinClimb", "TestLiteralClimb", "TestModuleRoot", "TestSpawnsItselfToListFiles",
		"TestThroughAPackageValue", "TestThroughHelper", "TestThroughTwoHelpers", "TestWalkFromTheParent", "TestWalkUp",
	}
	if !slices.Equal(got.names, wantNames) {
		t.Errorf("names = %v\nwant    %v", got.names, wantNames)
	}
	if want := []string{"./cmd/c", "./internal/a"}; !slices.Equal(got.packages, want) {
		t.Errorf("packages = %v, want %v", got.packages, want)
	}
}

// TestAFileThatDoesNotParseIsAnError: a test file the detector cannot read is
// one whose audits it would silently stop choosing.
func TestAFileThatDoesNotParseIsAnError(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "a")
	for _, d := range []string{dir, filepath.Join(root, "cmd")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package x\nfunc {"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := selectAudits(root, nil); err == nil {
		t.Fatal("a test file that does not parse was read past")
	}
}

// makefileList is the value of one of the Makefile's precheck variables.
func makefileList(t *testing.T, variable string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^` + variable + ` := (.+)$`).FindSubmatch(raw)
	if m == nil {
		t.Fatalf("the Makefile declares no %s", variable)
	}
	return string(m[1])
}

// precheckLists are the Makefile's three precheck lists: the packages run in
// full, the packages run by pattern, and the pattern.
func precheckLists(t *testing.T) (whole, named []string, pattern string) {
	t.Helper()
	return strings.Fields(makefileList(t, "PRECHECK_AUDIT_PACKAGES")),
		strings.Fields(makefileList(t, "PRECHECK_NAMED_PACKAGES")),
		makefileList(t, "PRECHECK_AUDIT_TESTS")
}

// TestPrecheckRunsEveryAuditNamedTest (iss-2611232155463657): every test whose
// name says it audits (TestNo..., TestEvery...) and whose reach touches the
// repository tree, through a process or a climbing literal included, is one
// precheck runs, or is named in notAudits with what it does instead. Watched
// failing before the selection read an ancestor handed to a helper and a walk
// from the parent: it named the drive's git table in
// internal/core/implement/loop, which had failed a preflight that day, and
// fsutil's display rule.
func TestPrecheckRunsEveryAuditNamedTest(t *testing.T) {
	whole, named, pattern := precheckLists(t)
	gaps, err := coverageGaps(filepath.Join("..", ".."), whole, named, pattern, notAudits)
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) > 0 {
		t.Fatalf("make precheck does not run these audit tests; make the selection see them (testreuse.Audits), "+
			"add their package or a word of their name to the Makefile's lists, or, for a test that is no audit, "+
			"say what it does in notAudits:\n  %s", strings.Join(gaps, "\n  "))
	}
}

// TestCoverageGapsNamesAnAuditPrecheckSkips holds the check to both shapes that
// failed preflights on 23 November: an audit that walks up to go.mod itself (the
// hand-made bare repository check) and one whose helper runs go list in an
// ancestor (the drive's git table). Each is named when precheck runs neither its
// package nor its name, and not when it does; a test that reaches the tree only
// through a script it runs is named until notAudits says what it is, and an
// entry of notAudits that excuses nothing is named too.
func TestCoverageGapsNamesAnAuditPrecheckSkips(t *testing.T) {
	const head = "package x\nimport (\n\t\"os\"\n\t\"os/exec\"\n\t\"path/filepath\"\n\t\"testing\"\n\t\"github.com/intentdriven/abcd/internal/fsutil\"\n)\nvar _, _, _ = os.Getwd, exec.Command, filepath.Join\n"
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(head+body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	write("internal/bare/bare_test.go", `
func TestNoTestMakesABareRepositoryByHand(t *testing.T) { _ = moduleRoot(t) }
func moduleRoot(t *testing.T) string {
	d, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		d = filepath.Dir(d)
	}
}
`)
	write("internal/drive/drive_test.go", `
func TestEveryGitCommandADriveRunsIsInTheTable(t *testing.T) { _ = walkOnce() }
func TestThePlainHelperIsNoAudit(t *testing.T) { _ = walkOnce() }
func TestEveryHookRuns(t *testing.T) { runHook(t) }
func walkOnce() string { return walk("../../../..") }
func walk(root string) string {
	c := exec.Command("go", "list", "./...")
	c.Dir = root
	_ = c.Run()
	return root
}
func runHook(t *testing.T) {
	r, _ := fsutil.ModuleRoot(".")
	_ = exec.Command("bash", filepath.Join(r, "hooks", "x")).Run()
}
`)
	hook := "./internal/drive.TestEveryHookRuns"
	for _, c := range []struct {
		name         string
		whole, named []string
		pattern      string
		exempt       map[string]string
		want         []string
	}{
		{
			name: "both chosen by what they do", pattern: "Surface",
			exempt: map[string]string{hook: "runs a hook script"},
		},
		{
			name: "both packages run by a pattern neither matches", named: []string{"./internal/bare", "./internal/drive"}, pattern: "Surface",
			exempt: map[string]string{hook: "runs a hook script"},
			want:   []string{"./internal/bare.TestNoTestMakesABareRepositoryByHand", "./internal/drive.TestEveryGitCommandADriveRunsIsInTheTable"},
		},
		{
			name: "both packages run by a pattern both match", named: []string{"./internal/bare", "./internal/drive"}, pattern: "Bare|Table",
			exempt: map[string]string{hook: "runs a hook script"},
		},
		{
			name: "both packages run in full", whole: []string{"./internal/bare", "./internal/drive"}, pattern: "Surface",
		},
		{
			name: "a script runner nothing excuses", pattern: "Surface",
			want: []string{hook},
		},
		{
			name: "an excuse that names nothing", pattern: "Surface",
			exempt: map[string]string{hook: "runs a hook script", "./internal/bare.TestGone": "was removed"},
			want:   []string{"./internal/bare.TestGone (named in notAudits, but no longer a test the detector finds outside precheck)"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := coverageGaps(root, c.whole, c.named, c.pattern, c.exempt)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("gaps = %q\nwant   %q", got, c.want)
			}
		})
	}
}

// TestPreflightPlanRunsPrechecksLinesNarrowly: the early audits preflight runs
// beside its wait are precheck's three test lines, every package and audit of
// them, in runs the narrow gate admits: at most three packages each, with
// precheck's own -timeout.
func TestPreflightPlanRunsPrechecksLinesNarrowly(t *testing.T) {
	root := filepath.Join("..", "..")
	whole, named, pattern := precheckLists(t)
	plan, err := preflightPlan(root, whole, named, pattern)
	if err != nil {
		t.Fatal(err)
	}
	chosen, err := selectAudits(root, append(slices.Clone(whole), named...))
	if err != nil {
		t.Fatal(err)
	}
	var full, byPattern, selected []string
	for _, r := range plan {
		cmd := r.goTest()
		if !machineload.IsNarrowCommand(cmd) || len(r.packages) > machineload.MaxNarrowPackages {
			t.Errorf("%q is not a narrow run", cmd)
		}
		if !slices.Equal(cmd[:4], []string{"go", "test", "-timeout", timeout}) {
			t.Errorf("%q does not take precheck's -timeout", cmd)
		}
		switch r.run {
		case "":
			full = append(full, r.packages...)
		case pattern:
			byPattern = append(byPattern, r.packages...)
		default:
			re := regexp.MustCompile(r.run)
			for _, pkg := range r.packages {
				selected = append(selected, pkg)
				for _, name := range chosen.byPackage[pkg] {
					if !re.MatchString(name) {
						t.Errorf("the run over %v does not run %s's %s", r.packages, pkg, name)
					}
				}
			}
		}
	}
	if !slices.Equal(full, whole) || !slices.Equal(byPattern, named) || !slices.Equal(selected, chosen.packages) {
		t.Fatalf("the plan runs in full %v, by pattern %v and by selection %v;\nwant %v, %v and %v", full, byPattern, selected, whole, named, chosen.packages)
	}
	for _, bad := range [][3]string{
		{"", "./internal/x", "X"},
		{"./internal/x", "", "X"},
		{"./internal/x", "./internal/y", ""},
		{"./internal/x", "./internal/y", "("},
		{"internal/x", "./internal/y", "X"},
	} {
		if _, err := preflightPlan(root, strings.Fields(bad[0]), strings.Fields(bad[1]), bad[2]); err == nil {
			t.Errorf("preflightPlan(%q) was not refused", bad)
		}
	}
}

// lockedBuffer is a buffer two goroutines can write.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// sh is a command running script in sh, after writing its pid to pidFile when
// one is named, so a test can prove it was stopped.
func sh(script, pidFile string) []string {
	if pidFile != "" {
		script = "echo $$ > '" + pidFile + "'; " + script
	}
	return []string{"sh", "-c", script}
}

// gone waits a moment for the process whose pid pidFile holds to be gone.
func gone(t *testing.T, pidFile string) {
	t.Helper()
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the process never wrote its pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("process %d is still running (%v)", pid, err)
	}
}

// waitForFile waits until path exists, so a stop is tested on a started process.
func waitForFile(t *testing.T, path string) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(path); err == nil {
			return
		}
	}
	t.Fatalf("%s was never written", path)
}

func noGoFlags([]string) (string, error) { return "", nil }

// TestAFailingAuditRunStopsTheWholeRun (iss-2611232155447855): an audit run
// that fails stops the command beside it, the whole run's gate and stages, at
// once, and the exit is the audit run's, so make mints no receipt.
func TestAFailingAuditRunStopsTheWholeRun(t *testing.T) {
	dir := t.TempDir()
	commandPid := filepath.Join(dir, "command")
	plan := []auditRun{{packages: []string{"./a"}}, {packages: []string{"./b"}}}
	gate := func(r auditRun) []string {
		if r.packages[0] == "./a" {
			waitForFile(t, commandPid)
			return sh(`printf -- '--- FAIL: TestX (0.00s)\nFAIL\n'; printf 'FAIL\t./a\t0.1s\n'; exit 1`, "")
		}
		return sh("exec sleep 60", filepath.Join(dir, "other"))
	}
	var out lockedBuffer
	began := time.Now()
	code := beside(context.Background(), plan, sh("exec sleep 60", commandPid), &out, &out, besideOptions{gate: gate, goFlags: noGoFlags})
	if code != 1 {
		t.Fatalf("exit %d, want the audit run's 1:\n%s", code, out.String())
	}
	if took := time.Since(began); took > 30*time.Second {
		t.Fatalf("the failure took %s to stop the run", took)
	}
	gone(t, commandPid)
	if s := out.String(); !strings.Contains(s, "--- FAIL: TestX") || !strings.Contains(s, "FAILED: go test -timeout 20m ./a") {
		t.Fatalf("the failing run's output and the stop are not said:\n%s", s)
	}
}

// TestTheWholeRunsVerdictEndsTheAudits: when the command ends first, its exit is
// the line's, and the audit runs still going stop, since its stages ran every
// test they would.
func TestTheWholeRunsVerdictEndsTheAudits(t *testing.T) {
	for _, want := range []int{0, 5} {
		dir := t.TempDir()
		auditPid := filepath.Join(dir, "audit")
		gate := func(auditRun) []string { return sh("exec sleep 60", auditPid) }
		command := sh("while [ ! -e '"+auditPid+"' ]; do sleep 0.05; done; exit "+strconv.Itoa(want), "")
		var out lockedBuffer
		code := beside(context.Background(), []auditRun{{packages: []string{"./a"}}}, command, &out, &out, besideOptions{gate: gate, goFlags: noGoFlags})
		if code != want {
			t.Fatalf("exit %d, want the command's %d:\n%s", code, want, out.String())
		}
		gone(t, auditPid)
	}
}

// TestPassingAuditsLeaveTheVerdictToTheWholeRun: audit runs that all pass say
// so, and the line waits for the command and takes its exit.
func TestPassingAuditsLeaveTheVerdictToTheWholeRun(t *testing.T) {
	plan := []auditRun{{packages: []string{"./a"}}, {packages: []string{"./b"}}, {packages: []string{"./c"}}}
	var started []string
	var mu sync.Mutex
	gate := func(r auditRun) []string {
		mu.Lock()
		started = append(started, r.packages[0])
		mu.Unlock()
		return sh("exit 0", "")
	}
	var out lockedBuffer
	code := beside(context.Background(), plan, sh("sleep 2; exit 4", ""), &out, &out, besideOptions{gate: gate, goFlags: noGoFlags})
	if code != 4 {
		t.Fatalf("exit %d, want the command's 4:\n%s", code, out.String())
	}
	if len(started) != 3 || !strings.Contains(out.String(), "every audit run passed") {
		t.Fatalf("started %v, want every run, and the pass said:\n%s", started, out.String())
	}
}

// TestAnEffectiveGoFlagsRunsTheWholeRunAlone: with a flag in GOFLAGS that no
// narrow run may carry, each would wait as a whole run behind this one, so none
// starts.
func TestAnEffectiveGoFlagsRunsTheWholeRunAlone(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=mod")
	gate := func(auditRun) []string { t.Error("an audit run started"); return sh("exit 0", "") }
	var out lockedBuffer
	code := beside(context.Background(), []auditRun{{packages: []string{"./a"}}}, sh("exit 0", ""), &out, &out,
		besideOptions{gate: gate, goFlags: envGoFlags})
	if code != 0 || !strings.Contains(out.String(), `the effective GOFLAGS "-mod=mod" holds a flag no narrow run may carry`) {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
}

// TestAnInterruptStopsEverything: the runs lead groups of their own, out of
// the terminal's, so an interrupt reaches them through the line alone.
func TestAnInterruptStopsEverything(t *testing.T) {
	dir := t.TempDir()
	commandPid, auditPid := filepath.Join(dir, "command"), filepath.Join(dir, "audit")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			_, a := os.Stat(commandPid)
			_, b := os.Stat(auditPid)
			if a == nil && b == nil {
				return
			}
		}
	}()
	var out lockedBuffer
	code := beside(ctx, []auditRun{{packages: []string{"./a"}}}, sh("exec sleep 60", commandPid), &out, &out,
		besideOptions{gate: func(auditRun) []string { return sh("exec sleep 60", auditPid) }, goFlags: noGoFlags})
	if code != 130 {
		t.Fatalf("exit %d, want 130:\n%s", code, out.String())
	}
	gone(t, commandPid)
	gone(t, auditPid)
}

// TestALeaderStopsItsGroupWhenTheRunnerGoes: a make preflight stopped by a kill
// of its own group (abcd land's bound, a drive's lane) kills the runner, not the
// groups its runs lead; each leader sees its standard input end and takes its
// group with it.
func TestALeaderStopsItsGroupWhenTheRunnerGoes(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(t.TempDir(), "child")
	watch, held, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self, append([]string{leadFlag, "--"}, sh("exec sleep 60", pidFile)...)...)
	procgroup.Lead(cmd)
	cmd.Stdin = watch
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = watch.Close()
	waitForFile(t, pidFile)
	_ = held.Close() // what the runner's death does to its write end
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		procgroup.Kill(cmd)
		t.Fatal("the leader outlived its runner")
	}
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		raw, _ := os.ReadFile(pidFile)
		pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the run (pid %d) outlived its leader", pid)
		}
	}
}

// TestALeaderOutsideItsOwnGroupRunsNothing: a kill of the group it is in would
// reach whoever started it.
func TestALeaderOutsideItsOwnGroupRunsNothing(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "ran")
	out, err := exec.Command(self, leadFlag, "--", "sh", "-c", "touch '"+marker+"'").CombinedOutput()
	if code := exitCode(err); code != 2 || !strings.Contains(string(out), "does not lead its process group") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the command ran")
	}
}

// envGoFlags reads GOFLAGS the way go env does for the environment alone.
func envGoFlags(env []string) (string, error) {
	flags := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "GOFLAGS="); ok {
			flags = v
		}
	}
	return flags, nil
}

// TestTheMakefilesTrimpathKeepsTheAudits: make exports GOFLAGS=-trimpath to
// every recipe (Makefile), a flag a narrow run may carry, so the audit runs go
// ahead with it on their command line and GOFLAGS empty, which is what the
// narrow gate admits. Watched failing while any GOFLAGS dropped the plan.
func TestTheMakefilesTrimpathKeepsTheAudits(t *testing.T) {
	t.Setenv("GOFLAGS", "-trimpath")
	dir := t.TempDir()
	seen := filepath.Join(dir, "goflags")
	var words []string
	gate := func(r auditRun) []string {
		words = r.goTest()
		return sh(`printf '[%s]' "$GOFLAGS" > '`+seen+`'`, "")
	}
	var out lockedBuffer
	code := beside(context.Background(), []auditRun{{packages: []string{"./a"}}}, sh("sleep 1", ""), &out, &out,
		besideOptions{gate: gate, goFlags: envGoFlags})
	if code != 0 || strings.Contains(out.String(), "no audit run is narrow") {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
	if !slices.Equal(words, []string{"go", "test", "-trimpath", "-timeout", timeout, "./a"}) || !machineload.IsNarrowCommand(words) {
		t.Fatalf("the audit run is %q, want -trimpath on its command line, narrow", words)
	}
	if raw, err := os.ReadFile(seen); err != nil || string(raw) != "[]" {
		t.Fatalf("the audit run saw GOFLAGS %q (%v), want it empty", raw, err)
	}
}

// TestTheMakefilesOwnGOFLAGSLeavesEveryRunNarrow: the GOFLAGS the Makefile
// gives make preflight, read from make's own database rather than written
// here, moves onto every run of the live plan through go's own go env, so no
// stub can stand between the plan and what make sets.
func TestTheMakefilesOwnGOFLAGSLeavesEveryRunNarrow(t *testing.T) {
	env, flags := makePreflightEnv(t)
	root := filepath.Join("..", "..")
	whole, named, pattern := precheckLists(t)
	plan, err := preflightPlan(root, whole, named, pattern)
	if err != nil {
		t.Fatal(err)
	}
	got, why := narrowed(plan, env, effectiveGoFlags)
	if why != "" || len(got) != len(plan) {
		t.Fatalf("under make preflight's GOFLAGS %q the plan keeps %d of %d runs: %s", flags, len(got), len(plan), why)
	}
	for _, r := range got {
		if !machineload.IsNarrowCommand(r.goTest()) || !slices.Equal(r.flags, strings.Fields(flags)) {
			t.Errorf("%q is not narrow, or does not carry GOFLAGS %q", r.goTest(), flags)
		}
	}

	// What the narrow gate judges is the waiter's own `go env GOFLAGS`
	// (narrowRefused): beside starts each run with it empty.
	seen := filepath.Join(t.TempDir(), "seen")
	gate := func(auditRun) []string { return sh(`go env GOFLAGS > '`+seen+`'`, "") }
	var out lockedBuffer
	if code := beside(context.Background(), plan[:1], sh("sleep 30", ""), &out, &out, besideOptions{gate: gate, env: env}); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
	if raw, err := os.ReadFile(seen); err != nil || strings.TrimSpace(string(raw)) != "" {
		t.Fatalf("an audit run's waiter reads GOFLAGS %q (%v), which the narrow gate refuses:\n%s", raw, err, out.String())
	}
}

// makePreflightEnv is this test's environment with the GOFLAGS the Makefile
// exports to make preflight's recipe, read from make's own database, and those
// flags; it skips where go env -w sets GOFLAGS, which no run can shed.
func makePreflightEnv(t *testing.T) ([]string, string) {
	t.Helper()
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make unavailable")
	}
	var env []string
	for _, kv := range os.Environ() {
		// A make above this test (make precheck, make preflight) passes its own
		// flags down, which a second make would read as its own.
		if k, _, _ := strings.Cut(kv, "="); k != "MAKEFLAGS" && k != "MFLAGS" && k != "MAKELEVEL" && k != "GOFLAGS" {
			env = append(env, kv)
		}
	}
	dry := exec.Command("make", "-pn", "preflight", "PREFLIGHT_RECORDS_ONLY=")
	dry.Dir, dry.Env = filepath.Join("..", ".."), env
	raw, err := dry.Output()
	if err != nil {
		t.Fatalf("make -pn preflight: %v", err)
	}
	m := regexp.MustCompile(`(?m)^GOFLAGS := (.*)$`).FindSubmatch(raw)
	if m == nil {
		t.Fatal("make's database for preflight holds no GOFLAGS")
	}
	flags := strings.TrimSpace(string(m[1]))
	if !strings.Contains(flags, "-trimpath") {
		t.Fatalf("the Makefile gives preflight GOFLAGS %q; this test holds the -trimpath it exports", flags)
	}
	if left, err := effectiveGoFlags(withGoFlags(env, "")); err != nil || left != "" {
		t.Skipf("go env -w sets GOFLAGS %q on this machine (%v), which no run can shed", left, err)
	}
	return withGoFlags(env, flags), flags
}

// gateE2E names the variable that runs TestAnAuditRunOpensAtTheNarrowGate. It
// waits at the machine's real narrow gate, so it runs only when asked, under a
// whole-run gate: inside a narrow run it would wait for a second narrow place
// its own run may be holding.
const gateE2E = "ABCD_PRECHECK_AUDITS_GATE_E2E"

// TestAnAuditRunOpensAtTheNarrowGate runs one planned audit run through the
// real gate command, beside a command, under make preflight's GOFLAGS, and
// expects the narrow gate to open for it. Run it from the repository root:
//
//	ABCD_PRECHECK_AUDITS_GATE_E2E=1 scripts/abcd.sh implement load --wait --timeout 30m -- go test -trimpath -count=1 -run TestAnAuditRunOpensAtTheNarrowGate ./cmd/precheck-audits/
func TestAnAuditRunOpensAtTheNarrowGate(t *testing.T) {
	if os.Getenv(gateE2E) != "1" {
		t.Skipf("set %s=1 to wait at the real narrow gate", gateE2E)
	}
	env, flags := makePreflightEnv(t)
	t.Chdir(filepath.Join("..", ".."))
	plan, why := narrowed([]auditRun{{run: "^TestALeaderOutsideItsOwnGroupRunsNothing$", packages: []string{"./cmd/precheck-audits"}}}, env, effectiveGoFlags)
	if why != "" {
		t.Fatalf("under make preflight's GOFLAGS %q: %s", flags, why)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out lockedBuffer
	done := make(chan int, 1)
	go func() { done <- beside(ctx, plan, sh("sleep 1800", ""), &out, &out, besideOptions{env: env}) }()
	for deadline := time.Now().Add(20 * time.Minute); ; time.Sleep(time.Second) {
		s := out.String()
		if strings.Contains(s, "passed 1 of 1") || strings.Contains(s, "could not run") || strings.Contains(s, "FAILED") ||
			strings.Contains(s, "no audit run is narrow") || time.Now().After(deadline) {
			break
		}
	}
	cancel()
	<-done
	if s := out.String(); !strings.Contains(s, "passed 1 of 1") || !strings.Contains(s, "precheck-audits:   narrow gate open after") {
		t.Fatalf("the audit run did not pass through the narrow gate under GOFLAGS %q:\n%s", flags, s)
	}
	t.Log(out.String())
}

// TestAGoFlagsSetByGoEnvRunsTheWholeRunAlone: GOFLAGS that stay set with the
// variable empty (go env -w) cannot move onto the command line, and the narrow
// gate would refuse every run under them.
func TestAGoFlagsSetByGoEnvRunsTheWholeRunAlone(t *testing.T) {
	gate := func(auditRun) []string { t.Error("an audit run started"); return sh("exit 0", "") }
	var out lockedBuffer
	code := beside(context.Background(), []auditRun{{packages: []string{"./a"}}}, sh("exit 0", ""), &out, &out,
		besideOptions{gate: gate, goFlags: func([]string) (string, error) { return "-trimpath", nil }})
	if code != 0 || !strings.Contains(out.String(), "go env -w") {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
}

// TestAGateRefusalLeavesTheWholeRunAlone: an audit run that ends without go
// test's own verdict (the gate timed out, refused at the disk floor or a bad
// setting, or could not start) says why and stops the audits, and the whole
// run decides; only a test that failed stops it.
func TestAGateRefusalLeavesTheWholeRunAlone(t *testing.T) {
	for name, script := range map[string]string{
		"a timeout":   "echo 'LOAD GATE TIMED OUT after 12h: still load'; exit 1",
		"a refusal":   "echo 'abcd implement load --wait: the disk is below its floor'; exit 1",
		"a bad start": "echo 'abcd implement load --wait: the command could not be started'; exit 127",
	} {
		t.Run(name, func(t *testing.T) {
			var started int
			gate := func(auditRun) []string { started++; return sh(script, "") }
			var out lockedBuffer
			plan := []auditRun{{packages: []string{"./a"}}, {packages: []string{"./b"}}, {packages: []string{"./c"}}}
			code := beside(context.Background(), plan, sh("sleep 1; exit 0", ""), &out, &out, besideOptions{gate: gate, goFlags: noGoFlags})
			if code != 0 {
				t.Fatalf("exit %d, want the whole run's 0:\n%s", code, out.String())
			}
			if s := out.String(); !strings.Contains(s, "could not run") || strings.Contains(s, "FAILED") || started != 1 {
				t.Fatalf("started %d audit runs, want the first alone, and the reason said:\n%s", started, s)
			}
		})
	}
}

// TestAStopLetsTheRunsCleanUp: a stop, for a failed audit or an interrupt,
// interrupts each group first, as Ctrl-C at the terminal did, so go test removes
// its work directory, and kills it only after the grace.
func TestAStopLetsTheRunsCleanUp(t *testing.T) {
	dir := t.TempDir()
	cleaned, ready := filepath.Join(dir, "cleaned"), filepath.Join(dir, "ready")
	command := sh(`trap 'echo yes > `+cleaned+`; exit 0' INT; touch `+ready+`; while :; do sleep 0.05; done`, "")
	gate := func(auditRun) []string {
		waitForFile(t, ready)
		return sh(`printf 'FAIL\t./a\t0.1s\n'; exit 1`, "")
	}
	var out lockedBuffer
	_ = beside(context.Background(), []auditRun{{packages: []string{"./a"}}}, command, &out, &out, besideOptions{gate: gate, goFlags: noGoFlags})
	if raw, err := os.ReadFile(cleaned); err != nil || strings.TrimSpace(string(raw)) != "yes" {
		t.Fatalf("the whole run was not interrupted before it was killed (%v):\n%s", err, out.String())
	}
}

// TestOneAuditRunAtATime: the audits leave the account's other narrow places
// free, so a drive lane's short narrow check is not crowded out behind them.
func TestOneAuditRunAtATime(t *testing.T) {
	dir := t.TempDir()
	busy, overlap := filepath.Join(dir, "busy"), filepath.Join(dir, "overlap")
	gate := func(auditRun) []string {
		return sh(`mkdir `+busy+` 2>/dev/null || echo x >> `+overlap+`; sleep 0.3; rmdir `+busy+` 2>/dev/null; :`, "")
	}
	plan := []auditRun{{packages: []string{"./a"}}, {packages: []string{"./b"}}, {packages: []string{"./c"}}}
	var out lockedBuffer
	if code := beside(context.Background(), plan, sh("sleep 3", ""), &out, &out, besideOptions{gate: gate, goFlags: noGoFlags}); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
	if _, err := os.Stat(overlap); err == nil {
		t.Fatalf("two audit runs ran at once:\n%s", out.String())
	}
}
