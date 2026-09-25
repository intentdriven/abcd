package intent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// steppedSpecNaming is a spec naming intentID that lists three steps, the
// first marked landed.
func steppedSpecNaming(id, slug, intentID string) string {
	return specNaming(id, slug, intentID) + "\n## Summary\n\nWritten.\n\n## Steps\n\n" +
		"1. The parser\n   - packages: internal/core/spec\n   - landed: #101\n" +
		"2. The loop\n   - packages: internal/core/implement\n   - tests: the loop advances after merge\n" +
		"3. The brief\n   - tests: the brief names the step\n"
}

// `abcd intent plan` mints a spec whose stub carries an empty `## Steps`
// section, and that spec is built as one step (criterion 1).
func TestPlanMintsASpecWithAnEmptyStepsSection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	pr, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, pr.Spec.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n## Steps\n") {
		t.Fatalf("the minted stub must carry `## Steps`:\n%s", data)
	}
	steps, err := spec.Steps(string(data))
	if err != nil || len(steps) != 1 || !steps[0].Implicit {
		t.Fatalf("a freshly planned spec is built as one step: %+v, %v", steps, err)
	}
}

// A close with --remainder carries the steps not marked landed into the
// remainder spec, renumbered, and says which it carried (criterion 3).
func TestReconcileRemainderCarriesTheUnlandedSteps(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", steppedSpecNaming("spc-1", "alpha", "itd-10"))

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "the-rest"})
	if err != nil {
		t.Fatalf("the remainder close must succeed: %v", err)
	}
	if len(res.RemainderSteps) != 2 || res.RemainderSteps[0].Title != "The loop" || res.RemainderSteps[1].Title != "The brief" ||
		res.RemainderSteps[0].Number != 1 || res.RemainderSteps[1].Number != 2 {
		t.Fatalf("the result must name the two unlanded steps carried: %+v", res.RemainderSteps)
	}
	data, err := os.ReadFile(filepath.Join(root, res.Remainder.Path))
	if err != nil {
		t.Fatal(err)
	}
	steps, err := spec.ParseSteps(string(data))
	if err != nil {
		t.Fatalf("the remainder's Steps must parse: %v\n%s", err, data)
	}
	if len(steps) != 2 || steps[0].Number != 1 || steps[0].Title != "The loop" ||
		steps[0].Tests != "the loop advances after merge" || steps[1].Title != "The brief" {
		t.Fatalf("the remainder must carry the unlanded steps, renumbered: %+v\n%s", steps, data)
	}
	if strings.Contains(string(data), "The parser") {
		t.Fatalf("a landed step must stay with the closed spec:\n%s", data)
	}
	if !spec.BodyIsStub(string(data)) {
		t.Fatal("the remainder's summary is still the author's to write")
	}
}

// A remainder minted from a spec that lists no steps carries the empty
// placeholder: there is no piece to carry.
func TestReconcileRemainderOfAnUnsteppedSpecCarriesNoSteps(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "the-rest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.RemainderSteps) != 0 {
		t.Fatalf("no step to carry: %+v", res.RemainderSteps)
	}
	data, err := os.ReadFile(filepath.Join(root, res.Remainder.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n## Steps\n") {
		t.Fatalf("the remainder still carries the empty section:\n%s", data)
	}
}

// A malformed Steps section on the closing spec refuses a --remainder close
// before anything is written: the copy would be a guess.
func TestReconcileRemainderRefusesAMalformedStepsSection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md",
		specNaming("spc-1", "alpha", "itd-10")+"\n## Steps\n\nFirst the parser, then the loop.\n")

	_, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "the-rest"})
	if err == nil {
		t.Fatal("a malformed Steps section must refuse the remainder close")
	}
	if !strings.Contains(err.Error(), "## Steps") || !strings.Contains(err.Error(), "nothing was minted") {
		t.Fatalf("the refusal must name the section and say nothing was written: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, specsOpen))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("the refusal must mint nothing and close nothing: %d specs in open/", len(entries))
	}
	// Without --remainder the close does not read the section and proceeds.
	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err != nil {
		t.Fatalf("a plain close does not depend on the Steps section: %v", err)
	}
}

// An unclosed span above the closing spec's `## Steps` masks the section, and
// read as absent the remainder would carry the placeholder and drop the
// unlanded steps. A --remainder close refuses instead, naming the opener's
// line, with nothing minted and nothing moved.
func TestReconcileRemainderRefusesAnUnclosedSpan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	body := specNaming("spc-1", "alpha", "itd-10") + "\n## Approach\n\n<!-- a draft nobody closed\n" +
		strings.TrimPrefix(steppedSpecNaming("spc-1", "alpha", "itd-10"), specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", body)
	opener := 0
	for i, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "<!--") {
			opener = i + 1
		}
	}

	_, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "the-rest"})
	if err == nil {
		t.Fatal("an unclosed span above ## Steps must refuse the remainder close")
	}
	for _, want := range []string{fmt.Sprintf("line %d", opener), "HTML comment", "nothing was minted"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %q: %v", want, err)
		}
	}
	entries, rerr := os.ReadDir(filepath.Join(root, specsOpen))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(entries) != 1 {
		t.Fatalf("the refusal must mint nothing and close nothing: %d specs in open/", len(entries))
	}
	if _, serr := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); serr != nil {
		t.Fatalf("the intent must stay in planned/: %v", serr)
	}
}
