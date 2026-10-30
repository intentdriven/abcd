package lint_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// The load check (itd-2609231434459890, spc-2609231542463113) runs once at the
// start of `make preflight` and once at the start of the eval harness, never
// once per test package, and on CI it runs only as the step that says it is
// skipped. These tests hold that placement, beside the preflight gate pins.

const (
	loadCheckPreflightCall = "go run ./cmd/abcd implement load --site preflight"
	loadCheckCIStep        = "go run ./cmd/abcd implement load --site eval-harness"
	loadCheckMarker        = "preflight: export ABCD_LOAD_CHECKED := preflight"
)

// TestLoadCheckIsPreflightsFirstPrerequisite: the check is the first thing
// preflight runs once the load gate admits it, its failure to build is ignored
// (a broken build is the build gate's to fail), and the marker the eval
// harness reads is exported from a line after the stages' prerequisite line:
// set on `preflight`, it reaches the stages' make through the environment.
func TestLoadCheckIsPreflightsFirstPrerequisite(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	prereqs := preflightPrereqs(t, root)
	if len(prereqs) == 0 || prereqs[0] != "load-check" {
		t.Fatalf("preflight's prerequisites are %v; the load check must be the first", prereqs)
	}
	makefile := readRepoFile(t, root, "Makefile")
	recipe, ok := makeRecipe(makefile, "load-check")
	if !ok || !strings.Contains(recipe, "\t-"+loadCheckPreflightCall) {
		t.Fatalf("the load-check recipe does not run %q with its failure ignored:\n%s", loadCheckPreflightCall, recipe)
	}
	lines := strings.Split(makefile, "\n")
	prereqLine, markerLine := -1, -1
	for i, l := range lines {
		if prereqLine < 0 && strings.HasPrefix(l, preflightStages+":") {
			prereqLine = i
		}
		if l == loadCheckMarker {
			markerLine = i
		}
	}
	if markerLine < 0 || markerLine <= prereqLine {
		t.Fatalf("the Makefile does not carry %q after the prerequisite line (prerequisites at %d, marker at %d)",
			loadCheckMarker, prereqLine+1, markerLine+1)
	}
}

// testMainBody returns the source of a file's TestMain body, or ok=false.
func testMainBody(t *testing.T, path string) (string, bool) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		// A file that does not parse is the compiler's concern, not this test's.
		return "", false
	}
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "TestMain" || fn.Recv != nil || fn.Body == nil {
			continue
		}
		return string(src[fset.Position(fn.Body.Pos()).Offset:fset.Position(fn.Body.End()).Offset]), true
	}
	return "", false
}

// loadCallRe matches an invocation of the verb or the core function in Go
// source.
var loadCallRe = regexp.MustCompile(`"implement",\s*"load"|CheckLoad\(`)

// TestEvalHarnessRunsTheLoadCheckOnce: the harness's TestMain runs the verb
// exactly once, at the eval-harness site, before m.Run().
func TestEvalHarnessRunsTheLoadCheckOnce(t *testing.T) {
	body, ok := testMainBody(t, filepath.Join("..", "..", "..", "evals", "harness_test.go"))
	if !ok {
		t.Fatal("evals/harness_test.go has no TestMain")
	}
	calls := loadCallRe.FindAllStringIndex(body, -1)
	if len(calls) != 1 {
		t.Fatalf("the harness's TestMain runs the load check %d times, want exactly once", len(calls))
	}
	if !strings.Contains(body[calls[0][0]:], `"--site", "eval-harness"`) {
		t.Fatal("the harness's load check does not name the eval-harness site")
	}
	run := strings.Index(body, "m.Run()")
	if run < 0 || calls[0][0] > run {
		t.Fatal("the harness's load check does not run before m.Run()")
	}
}

// TestNoOtherTestMainRunsTheLoadCheck: no other package's TestMain runs the
// check, so `go test ./...` never runs it once per package. The verb's own unit
// tests call it from ordinary test functions, which run on demand.
func TestNoOtherTestMainRunsTheLoadCheck(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	harness := filepath.Join(root, "evals", "harness_test.go")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".abcd", "node_modules", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") || path == harness {
			return nil
		}
		if body, ok := testMainBody(t, path); ok && loadCallRe.MatchString(body) {
			t.Errorf("%s: its TestMain runs the load check, which then runs once per package", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// taggedGoTestRe is a `go test` that selects a build tag: the eval harness run
// directly rather than through its make target.
var taggedGoTestRe = regexp.MustCompile(`go test[^\n]*-tags`)

// harnessJobs returns every workflow job that starts the eval harness, keyed
// "workflow:job", with its block.
func harnessJobs(t *testing.T, root string) map[string]string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no workflows: %v", err)
	}
	jobRe := regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)
	out := map[string]string{}
	for _, p := range paths {
		wf := readRepoFile(t, root, filepath.ToSlash(strings.TrimPrefix(p, root+string(filepath.Separator))))
		if strings.Contains(wf, "make preflight") {
			t.Errorf("%s runs make preflight; CI runs the gates directly, and the load check is skipped there by a step", filepath.Base(p))
		}
		inJobs := false
		for _, l := range strings.Split(wf, "\n") {
			if strings.HasPrefix(l, "jobs:") {
				inJobs = true
				continue
			}
			m := jobRe.FindStringSubmatch(l)
			if !inJobs || m == nil {
				continue
			}
			block, _ := workflowJobBlock(wf, m[1])
			if strings.Contains(block, "make smoke") || strings.Contains(block, "make evals-cold-reading") ||
				taggedGoTestRe.MatchString(block) {
				out[filepath.Base(p)+":"+m[1]] = block
			}
		}
	}
	return out
}

// TestCIRunsTheSkipStepBeforeEachHarness: both of ci.yml's harness jobs run the
// load check first, where it prints why it is skipped into the job log; the
// harness's own call is hidden by package-list mode. The release workflow's
// verify job also starts the harness, and carries no step: that workflow is
// rendered from the scaffold abcd writes into every managed repository, where
// no `cmd/abcd` exists to run, and its harness takes the same skipped branch.
func TestCIRunsTheSkipStepBeforeEachHarness(t *testing.T) {
	jobs := harnessJobs(t, filepath.Join("..", "..", ".."))
	checked := 0
	for name, block := range jobs {
		if !strings.HasPrefix(name, "ci.yml:") {
			continue
		}
		checked++
		step := strings.Index(block, loadCheckCIStep)
		harness := len(block)
		for _, h := range []string{"make smoke", "make evals-cold-reading"} {
			if i := strings.Index(block, h); i >= 0 && i < harness {
				harness = i
			}
		}
		if loc := taggedGoTestRe.FindStringIndex(block); loc != nil && loc[0] < harness {
			harness = loc[0]
		}
		if step < 0 || step > harness {
			t.Errorf("%s starts the eval harness without first running %q", name, loadCheckCIStep)
		}
	}
	if checked != 2 {
		t.Fatalf("found %d harness jobs in ci.yml, want the smoke and cold-reading-evals jobs", checked)
	}
}

// TestHarnessJobsRunOnHostedRunners: the check is skipped on CI because a fresh
// runner carries no strays, which holds only on a GitHub-hosted runner. A job
// that starts the harness on a self-hosted runner fails this.
func TestHarnessJobsRunOnHostedRunners(t *testing.T) {
	hosted := regexp.MustCompile(`(?m)^    runs-on: (ubuntu|macos|windows)-[a-z0-9.-]+\s*$`)
	for name, block := range harnessJobs(t, filepath.Join("..", "..", "..")) {
		if !hosted.MatchString(block) || strings.Contains(block, "self-hosted") {
			t.Errorf("%s starts the eval harness on a runner that is not GitHub-hosted; the load check's CI exemption assumes a fresh runner", name)
		}
	}
}

// loadGateCall is the load gate make preflight waits at as one whole run
// (iss-2610102020442624, iss-2610140638074587, itd-2610262127138979).
const loadGateCall = "go run ./cmd/abcd implement load --wait"

// parallelismCall derives each test stage's go test -p.
const parallelismCall = `p=$$(go run ./cmd/test-parallelism) && \`

// wholeRunTimeout is the least the whole run's gate may wait: the queue on a
// shared machine has run hours deep, and a gate that gives up first fails a
// preflight that did nothing wrong.
const wholeRunTimeout = 12 * time.Hour

// stagesMakeVar is the variable preflight's gate runs the stages' make
// through (TestPreflightWaitsAsOneWholeRun).
const stagesMakeVar = "PREFLIGHT_STAGES_MAKE"

// TestPreflightWaitsAsOneWholeRun: `make preflight` has no prerequisite and
// runs nothing before the load gate's command form, which runs every stage
// (preflightStages) as its command, so a queued preflight does no lint, eval,
// build, vet or test work and its turn holds until the run ends; the only
// other line mints the receipt once the run has passed. CI never runs make
// preflight (harnessJobs refuses a workflow that does), and the gate opens at
// once on a runner.
func TestPreflightWaitsAsOneWholeRun(t *testing.T) {
	makefile := readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile")
	// The first `preflight:` rule is the full run's; the records-only rule,
	// the else branch after it, waits at no gate and runs the same gates as
	// prerequisites (TestPreflightRecordsRuleRunsPrechecksTests).
	for _, l := range strings.Split(makefile, "\n") {
		rest, ok := strings.CutPrefix(l, "preflight:")
		if !ok || strings.HasPrefix(strings.TrimSpace(rest), "export ") {
			continue
		}
		if strings.TrimSpace(rest) != "" {
			t.Errorf("preflight declares prerequisites %q; each would run before the load gate admits the run", strings.TrimSpace(rest))
		}
		break
	}
	recipe, ok := makeRecipe(makefile, "preflight")
	if !ok {
		t.Fatal("Makefile declares no `preflight:` recipe")
	}
	var lines []string
	for _, l := range strings.Split(recipe, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "@scripts/preflight-receipt.sh mint ") {
		t.Fatalf("preflight's recipe is %q; want the gate's command form, then the receipt's mint, and nothing else", lines)
	}
	gate := strings.Fields(lines[0])
	cmd := slices.Index(gate, "--")
	if !strings.HasPrefix(lines[0], loadGateCall+" ") || cmd < 0 || !slices.Equal(gate[cmd+1:], []string{"$(" + stagesMakeVar + ")"}) {
		t.Fatalf("preflight's first line is %q; want %s ... -- $(%s)", lines[0], loadGateCall, stagesMakeVar)
	}
	// make runs a recipe line that names MAKE itself even under -n, -q and
	// -t, so a dry run would queue at the gate for hours; named through a
	// variable, the line is an ordinary one, which those forms never run.
	if strings.Contains(recipe, "$(MAKE)") || strings.Contains(recipe, "${MAKE}") {
		t.Errorf("preflight's recipe names MAKE directly, so make -n, -q and -t run its gate:\n%s", recipe)
	}
	if want := stagesMakeVar + " = $(MAKE) --no-print-directory " + preflightStages; !slices.Contains(strings.Split(makefile, "\n"), want) {
		t.Errorf("the Makefile does not declare %q", want)
	}
	at := slices.Index(gate[:cmd], "--timeout")
	if at < 0 || at+1 >= cmd {
		t.Fatalf("preflight's gate %q sets no --timeout; the default gives up after half an hour", lines[0])
	}
	if d, err := time.ParseDuration(gate[at+1]); err != nil || d < wholeRunTimeout {
		t.Fatalf("preflight's gate waits at most %s (%v); want at least %s", gate[at+1], err, wholeRunTimeout)
	}
}

// TestPreflightStagesHaveNoGateOfTheirOwn: the stages run inside the whole
// run's turn, so they wait at no first gate: a second waiter inside a held
// turn could deadlock against a run queued meanwhile, each shut by the other
// until its timeout. Each go test stage takes the -p derived as it starts, so
// one run cannot saturate the machine.
func TestPreflightStagesHaveNoGateOfTheirOwn(t *testing.T) {
	recipe, ok := makeRecipe(readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile"), preflightStages)
	if !ok {
		t.Fatalf("Makefile declares no `%s:` recipe", preflightStages)
	}
	lines := strings.Split(recipe, "\n")
	tests := 0
	for i, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, loadGateCall) {
			t.Errorf("%s waits at the first gate (%q) inside the whole run's turn", preflightStages, l)
		}
		if !strings.HasPrefix(l, "go test ") {
			continue
		}
		tests++
		if i == 0 || strings.TrimSpace(lines[i-1]) != parallelismCall || !strings.Contains(l, ` -p "$$p" `) {
			t.Errorf("%s's %q does not take the -p derived as it starts (%s)", preflightStages, l, parallelismCall)
		}
	}
	if tests != 2 {
		t.Fatalf("%s's recipe has %d go test stages, want 2:\n%s", preflightStages, tests, recipe)
	}
}

// raceGateCall is the race gate make preflight runs its race stage behind, so
// one race stage runs at a time whatever load.preflights admits to the test
// stages (iss-2610161639025702).
const raceGateCall = "go run ./cmd/abcd implement load --race-gate"

// TestPreflightGatesItsRaceStage: the race stage waits at the race gate after
// the unit stage and before it derives its -p, so the -p reflects the load as
// the race stage starts. It stays inside the whole run: above one slot it is
// what keeps race stages apart.
func TestPreflightGatesItsRaceStage(t *testing.T) {
	recipe, ok := makeRecipe(readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile"), preflightStages)
	if !ok {
		t.Fatalf("Makefile declares no `%s:` recipe", preflightStages)
	}
	lines := strings.Split(recipe, "\n")
	unit, gate, race := -1, -1, -1
	for i, l := range lines {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, raceGateCall):
			gate = i
		case strings.HasPrefix(l, "go test -race "):
			race = i
		case strings.HasPrefix(l, "go test "):
			unit = i
		}
	}
	if unit < 0 || gate < 0 || race < 0 || !(unit < gate && gate < race) || strings.TrimSpace(lines[race-1]) != parallelismCall || gate != race-2 {
		t.Fatalf("preflight's recipe has the unit stage at line %d, the race gate %q at %d and the race stage at %d; want the gate straight before the race stage's -p:\n%s",
			unit, raceGateCall, gate, race, recipe)
	}
}
