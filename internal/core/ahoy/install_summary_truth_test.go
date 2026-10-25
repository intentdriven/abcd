package ahoy

import (
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/identity"
)

// TestIdentityRecordActionSaysHowToCorrectIt is iss-2609260057117878's
// first sentence, carried to the person's own record: a re-run never rewrites
// a record that exists, so the summary item must say how a wrong one is
// corrected (edit it, or remove it and be asked again).
func TestIdentityRecordActionSaysHowToCorrectIt(t *testing.T) {
	dir := idGitRepo(t, "Right Person", "right@example.com")
	idWritePin(t, dir, `{"name":"Wrong Person","email":"wrong@example.com"}`)
	gapPresent := map[string]bool{}
	for _, g := range detectGitIdentity(dir) {
		gapPresent[g.ID] = true
	}
	p := &identityPrompter{terminal: true, answer: true}
	a := &applyCtx{cwd: dir, gapPresent: gapPresent, prompter: p}
	a.stepIdentityRecord()
	rec, _, ok, err := identity.LoadRecordForTest(os.Getenv("HOME"), "")
	if err != nil || !ok || rec.Name != "Wrong Person" || len(p.asked) != 0 {
		t.Fatalf("precondition: a re-run leaves a record as it is; got %+v ok=%v err=%v asked=%q", rec, ok, err, p.asked)
	}
	action := writeKindHelp[writeIdentityRecord].Action
	for _, want := range []string{"edit", "remove"} {
		if !strings.Contains(action, want) {
			t.Errorf("the identity-record item does not say %q: %q", want, action)
		}
	}
}

// TestSkippedRecordActionNamesTheQuestion is the second sentence: the item
// must name the question that is actually asked, and where.
func TestSkippedRecordActionNamesTheQuestion(t *testing.T) {
	action := optionalSkippedHelp[UnrecordedGapID].Action
	for _, want := range []string{"terminal", "identity question", "without --yes"} {
		if !strings.Contains(action, want) {
			t.Errorf("the skipped-record item does not name %q: %q", want, action)
		}
	}
}

// TestVisibilityMeaningNamesEveryFencedPath is the third: each visibility's
// meaning must name every path its .gitignore block keeps out of git, so the
// public meaning cannot again omit the memory/ fence the block writes.
func TestVisibilityMeaningNamesEveryFencedPath(t *testing.T) {
	for _, c := range promptHelp["visibility"].Choices {
		entries, ok := visibilityEntries[c.Value]
		if !ok {
			t.Fatalf("visibility choice %q has no entry set", c.Value)
		}
		for _, e := range entries {
			bare := strings.Trim(e, "/")
			if !strings.Contains(c.Meaning, bare) {
				t.Errorf("the %s meaning does not name %s, which its .gitignore block keeps out of git: %q", c.Value, e, c.Meaning)
			}
		}
	}
}
