package scaffold

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// setupGoStepRe, goVersionFileRe and goVersionLiteralRe read the three shapes a
// setup-go pin can take. "go-version:" is not a substring of "go-version-file:",
// so the literal check needs no lookahead (RE2 has none).
var (
	setupGoStepRe      = regexp.MustCompile(`uses:\s*['"]?[^'"\n]*actions/setup-go`)
	goVersionFileRe    = regexp.MustCompile(`go-version-file:\s*['"]?go\.mod['"]?`)
	goVersionLiteralRe = regexp.MustCompile(`go-version:`)
)

// TestEverySetupGoResolvesTheToolchainFromGoMod holds every setup-go step abcd
// ships — and every one it scaffolds into a managed repo — to reading the
// toolchain out of go.mod rather than restating it (iss-2609090951291799).
//
// The predecessor of this test, TestWorkflowGoVersionsMatchSubstitutions,
// coupled each literal pin to AbcdSubstitutions().GoVersion after the d594511
// incident, where CI scanned green on a patched toolchain while the release
// built on the unpatched one and shipped four known stdlib CVEs. The coupling
// worked, but it kept nine literals alive and made the go directive the tenth
// spelling. That is the same defect the format gate goes to some trouble to
// remove from the Makefile: bump the go directive and forget one of the nine,
// and the lane compiles on the old toolchain while `make fmt-check` resolves and
// fetches the new one, so CI's own two halves disagree about which gofmt is
// correct.
//
// go-version-file deletes the second spelling instead of policing it, and it is
// strictly safer in the scaffold direction too: the template used to interpolate
// a version derived from the adopter's go.mod at scaffold time — a snapshot that
// goes stale silently, and a value that had to be validated against an
// injection-safe allowlist before it could be written into YAML. A fixed
// `go.mod` needs neither.
//
// It fails CLOSED on both edges: a file the sweep cannot see contributes no
// setup-go step, and a sweep that sees no setup-go step at all is a fatal rather
// than a pass.
func TestEverySetupGoResolvesTheToolchainFromGoMod(t *testing.T) {
	root := repoRoot(t)

	var roster []string
	for _, dir := range []string{
		filepath.Join(".github", "workflows"),
		filepath.Join("internal", "core", "launch", "scaffold", "templates"),
	} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			switch filepath.Ext(e.Name()) {
			case ".yml", ".yaml", ".tmpl":
				roster = append(roster, filepath.Join(dir, e.Name()))
			}
		}
	}
	if len(roster) == 0 {
		t.Fatal("the workflow/template sweep matched no files; it would pass by finding nothing")
	}

	steps, files := 0, 0
	for _, rel := range roster {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		src := string(data)

		n := len(setupGoStepRe.FindAllString(src, -1))
		if n == 0 {
			continue
		}
		steps += n
		files++

		if pins := len(goVersionLiteralRe.FindAllString(src, -1)); pins > 0 {
			t.Errorf("%s restates the Go version in %d setup-go step(s).\n\n"+
				"go.mod's `go` directive is the declaration; a literal beside it is a second "+
				"spelling to bump, and the one that gets forgotten leaves the lane compiling on "+
				"a different toolchain from the one the format gate resolves. Use "+
				"`go-version-file: go.mod`.", rel, pins)
		}
		if got := len(goVersionFileRe.FindAllString(src, -1)); got != n {
			t.Errorf("%s has %d setup-go step(s) and %d `go-version-file: go.mod` line(s); "+
				"every setup-go step must read the version from go.mod, or the ones that do not "+
				"resolve whatever the runner has preinstalled", rel, n, got)
		}
	}
	if steps == 0 {
		t.Fatal("no setup-go step found under .github/workflows or the scaffold templates — " +
			"the sweep matched nothing, which cannot be right")
	}
	if !strings.Contains(strings.Join(roster, " "), "release.yml.tmpl") {
		t.Fatal("the scaffold templates are not in the sweep; a managed repo would receive an " +
			"unchecked setup-go pin")
	}
}

// TestPullRequestTargetWorkflowsHaveNoCheckout arms the invariant external-review.yml
// states in prose ("the PR's code is never checked out, never built, never
// executed, which is what makes the trigger safe to use here"): a workflow
// triggered by pull_request_target must not check out PR code, or it becomes a
// pwn request. zizmor's dangerous-triggers audit is (necessarily) suppressed on
// that file, so nothing else catches a later checkout being added — this test does.
func TestPullRequestTargetWorkflowsHaveNoCheckout(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	triggerRe := regexp.MustCompile(`(?m)^\s*pull_request_target:`)
	checkoutRe := regexp.MustCompile(`(?m)uses:\s*['"]?[^'"\n]*actions/checkout`)
	checked := 0
	for _, e := range entries {
		if ext := filepath.Ext(e.Name()); e.IsDir() || (ext != ".yml" && ext != ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !triggerRe.Match(data) {
			continue
		}
		checked++
		if checkoutRe.Match(data) {
			t.Errorf("%s is triggered by pull_request_target and checks out code — a pwn request; keep PR code out of this workflow", e.Name())
		}
	}
	if checked == 0 {
		t.Fatal("no pull_request_target workflow found — the sweep matched nothing; external-review.yml should trip this")
	}
}
