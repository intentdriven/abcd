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
		m := regexp.MustCompile(`^\$\(CHECK_TESTS\) (unit|race) go test `).FindStringSubmatch(l)
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
	if !regexp.MustCompile(`(?m)^CHECK_TESTS = go run \./cmd/check-tests `).MatchString(makefile) {
		t.Error("CHECK_TESTS is not cmd/check-tests")
	}
	if !strings.Contains(makefile, "$(if $(PREFLIGHT_FRESH), -fresh,)") {
		t.Error("a release cut has no way to reuse nothing: CHECK_TESTS does not read PREFLIGHT_FRESH")
	}
}

// TestTheReusedLinesTimeoutIsTheStagesLargest: go test's cache key holds the
// timeout, so the lines whose results are to be reused take one fixed value, and
// it must be no less than the most a stage's own budget scales to, or a package
// slowed by the load is killed with no test hung (iss-2610241911052191).
func TestTheReusedLinesTimeoutIsTheStagesLargest(t *testing.T) {
	makefile := readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile")
	m := regexp.MustCompile(`(?m)^CHECK_TESTS = .* -cached-timeout (\S+?)(\$|\s|$)`).FindStringSubmatch(makefile)
	if m == nil {
		t.Fatal("CHECK_TESTS sets no -cached-timeout")
	}
	fixed, err := time.ParseDuration(m[1])
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
	if strings.Contains(rule, "CHECK_TESTS") {
		t.Errorf("the records-only rule runs through the check:\n%s", rule)
	}
	if !strings.Contains(rule, "go test ") {
		t.Errorf("the records-only rule runs no tests:\n%s", rule)
	}
}

// TestAReleaseWorkflowReusesNothing: a release cut's hosted gates run every test
// on the tree being tagged, in the workflow and in the template it is rendered
// from (itd-2611011325421304, criterion 2).
func TestAReleaseWorkflowReusesNothing(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, rel := range []string{".github/workflows/release.yml", "internal/core/launch/scaffold/templates/release.yml.tmpl"} {
		found := 0
		for _, l := range strings.Split(readRepoFile(t, root, rel), "\n") {
			l = strings.TrimSpace(l)
			if !strings.HasPrefix(l, "run: go test ") {
				continue
			}
			found++
			if !strings.Contains(l, " -count=1 ") {
				t.Errorf("%s: a release gate can reuse a result: %s", rel, l)
			}
		}
		if found == 0 {
			t.Errorf("%s runs no go test; the parser or the workflow changed shape", rel)
		}
	}
}
