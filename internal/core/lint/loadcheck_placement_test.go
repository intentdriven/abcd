package lint_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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
// preflight runs, its failure to build is ignored (a broken build is the build
// gate's to fail), and the marker the eval harness reads is exported to the
// prerequisites from a line after the prerequisite line.
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
		if prereqLine < 0 && strings.HasPrefix(l, "preflight:") {
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
