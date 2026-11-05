package lint_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/testreuse"
)

// rawRecipe is a target's recipe lines as the Makefile spells them, which
// makeRecipe does not give for a test stage.
func rawRecipe(t *testing.T, makefile, target string) []string {
	t.Helper()
	var lines []string
	in := false
	for _, l := range strings.Split(makefile, "\n") {
		if strings.HasPrefix(l, target+":") {
			in = true
			continue
		}
		if in {
			if !strings.HasPrefix(l, "\t") {
				break
			}
			lines = append(lines, strings.TrimSpace(l))
		}
	}
	if len(lines) == 0 {
		t.Fatalf("the Makefile has no recipe for %s", target)
	}
	return lines
}

// TestBothTestStagesRunThroughTheCheck: preflight's test stages go through
// cmd/check-tests, each with its name, and no go test in the recipe runs around
// it, so every stage reports its reuse and the receipt reads it
// (itd-2611011325421304).
func TestBothTestStagesRunThroughTheCheck(t *testing.T) {
	makefile := readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile")
	stages := map[string]int{}
	for _, l := range rawRecipe(t, makefile, preflightStages) {
		if !strings.Contains(l, "go test ") {
			continue
		}
		m := regexp.MustCompile(`^go run \./cmd/check-tests \$\(CHECK_TESTS_FLAGS\) (unit|race) go test `).FindStringSubmatch(l)
		if m == nil {
			t.Errorf("a test line runs around the check: %s", l)
			continue
		}
		stages[m[1]]++
		if (m[1] == "race") != strings.Contains(l, " -race ") {
			t.Errorf("the %s stage's line and its -race disagree: %s", m[1], l)
		}
	}
	if stages["unit"] == 0 || stages["race"] == 0 {
		t.Fatalf("the recipe has stages %v; want unit and race", stages)
	}
	if !strings.Contains(makefile, "$(if $(PREFLIGHT_FRESH), -fresh,)") {
		t.Error("a release cut has no way to reuse nothing: CHECK_TESTS_FLAGS does not read PREFLIGHT_FRESH")
	}
	// PREFLIGHT_FRESH on a records-only commit must not take the shorter rule,
	// whose test lines the check does not run and whose results replay.
	if !strings.Contains(makefile, "PREFLIGHT_RECORDS_ONLY := $(if $(PREFLIGHT_FRESH),,$(shell ") {
		t.Error("PREFLIGHT_FRESH does not hold a records-only commit to the full rule")
	}
}

// TestTheReusedLinesTimeoutIsTheStagesLargest: go test's cache key holds the
// timeout, so the lines whose results are to be reused take one fixed value, and
// it must be no less than the most a stage's own budget scales to, or a package
// slowed by the load is killed with no test hung (iss-2610241911052191).
func TestTheReusedLinesTimeoutIsTheStagesLargest(t *testing.T) {
	makefile := readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile")
	m := regexp.MustCompile(`(?m)^CHECK_TESTS_FLAGS = -cached-timeout (\S+?)(\$|\s|$)`).FindStringSubmatch(makefile)
	if m == nil {
		t.Fatal("CHECK_TESTS_FLAGS sets no -cached-timeout")
	}
	fixed, err := time.ParseDuration(m[1])
	if err != nil {
		t.Fatal(err)
	}
	b := regexp.MustCompile(`(?m)^CHECK_TESTS_FLAGS = .* -budget (\S+?)(\$|\s|$)`).FindStringSubmatch(makefile)
	if b == nil {
		t.Fatal("CHECK_TESTS_FLAGS sets no -budget, so a stage that waited keeps a -timeout as old as the wait")
	}
	budget, err := time.ParseDuration(b[1])
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range rawRecipe(t, makefile, preflightStages) {
		if b := timeoutCall.FindStringSubmatch(l); b != nil {
			base, err := time.ParseDuration(b[1])
			if err != nil {
				t.Fatal(err)
			}
			if want := base * machineload.TestTimeoutMaxFactor; fixed != want {
				t.Errorf("-cached-timeout is %s; a stage with the base %s scales to %s", fixed, base, want)
			}
			if base != budget {
				t.Errorf("-budget is %s; the stage's base is %s", budget, base)
			}
		}
	}
}

// TestTheAlwaysRerunListReadsAndNamesRealPackages: the list decides what is
// never reused, and a line that does not read refuses all reuse, so the tree's
// copy, the next default branch's, must read; and a line naming a package that
// is gone holds nothing to rerun, which is a test leaving the list unnoticed.
func TestTheAlwaysRerunListReadsAndNamesRealPackages(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	list, err := testreuse.ParseList(readRepoFile(t, root, testreuse.ListPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range list {
		dir := filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(d.Pattern, "/...")))
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("%s names no package: %v", d.Pattern, err)
			continue
		}
		sources, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
		if err != nil {
			t.Fatal(err)
		}
		var text strings.Builder
		for _, f := range sources {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			text.Write(raw)
		}
		for _, name := range d.Tests {
			if !strings.Contains(text.String(), "func "+name+"(") {
				t.Errorf("%s names the test %s, which it does not declare", d.Pattern, name)
			}
		}
	}
}

// TestARecordsOnlyPreflightStillTakesItsOwnRoute: the shorter path for a change
// to records alone stays as it was (spc-2611012047503720, criterion 3), its test
// lines precheck's, outside the check.
func TestARecordsOnlyPreflightStillTakesItsOwnRoute(t *testing.T) {
	makefile := readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile")
	_, records, ok := strings.Cut(makefile, "else\npreflight: ")
	if !ok {
		t.Fatal("the Makefile has no records-only preflight rule")
	}
	rule, _, _ := strings.Cut(records, "\nendif")
	if strings.Contains(rule, "check-tests") {
		t.Errorf("the records-only rule runs through the check:\n%s", rule)
	}
	if !strings.Contains(rule, "go test ") {
		t.Errorf("the records-only rule runs no tests:\n%s", rule)
	}
}

// TestEveryHostedTestStepReusesNothing: every go test line a workflow runs passes
// -count=1, so the hosted full run the reuse stands behind is a flag and not the
// accident of a fresh checkout's file times, and a release cut's hosted gates run
// every test on the tree being tagged (itd-2611011325421304, criterion 2).
func TestEveryHostedTestStepReusesNothing(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	rels := []string{"internal/core/launch/scaffold/templates/release.yml.tmpl"}
	for _, w := range workflows {
		rel, err := filepath.Rel(root, w)
		if err != nil {
			t.Fatal(err)
		}
		rels = append(rels, filepath.ToSlash(rel))
	}
	found := 0
	for _, rel := range rels {
		for _, l := range strings.Split(readRepoFile(t, root, rel), "\n") {
			l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "run:"))
			if !strings.HasPrefix(l, "go test ") {
				continue
			}
			found++
			if !strings.Contains(l, " -count=1 ") {
				t.Errorf("%s: a hosted test step can reuse a result: %s", rel, l)
			}
		}
	}
	if found < 8 {
		t.Errorf("found %d go test steps; the parser or the workflows changed shape", found)
	}
}
