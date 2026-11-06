package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestALandingRefusesToShipAnIntentNamingAnIssueNoLaneResolves: the closing lane
// runs the close `abcd spec close` runs, so it meets the same refusal for an
// open issue the intent names and no receipt of this lane resolves, before the
// spec moves (iss-2611060603312990).
func TestALandingRefusesToShipAnIntentNamingAnIssueNoLaneResolves(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	const intentRel = ".abcd/development/intents/planned/itd-10-alpha.md"
	body, err := os.ReadFile(filepath.Join(l.Worktree, intentRel))
	if err != nil {
		t.Fatal(err)
	}
	named := strings.Replace(string(body), "\nslug:", "\nrelated_issues: [iss-77]\nslug:", 1)
	if named == string(body) {
		t.Fatalf("premise: no slug field to anchor on:\n%s", body)
	}
	if err := os.WriteFile(filepath.Join(l.Worktree, intentRel), []byte(named), 0o644); err != nil {
		t.Fatal(err)
	}
	issueRel := ".abcd/work/issues/open/iss-77-another.md"
	issue := "---\nschema_version: 1\nid: \"iss-77\"\nslug: \"another\"\nseverity: \"minor\"\ncategory: \"ux\"\nsource: \"agent-observation\"\nfound_during: \"t\"\nrelated_intents: [itd-10]\n---\n\nAnother observation.\n"
	if err := os.WriteFile(filepath.Join(l.Worktree, issueRel), []byte(issue), 0o644); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "name another issue")
	head := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[st.current()].HeadSHA = head
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	var refusal *Refusal
	for range 2 {
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
			refusal = mustRefusal(t, err)
			break
		}
	}
	if refusal == nil {
		t.Fatal("the landing shipped an intent naming an open issue no lane resolves")
	}
	if refusal.Stage != string(StageLand) || !strings.Contains(refusal.Reason, "iss-77") || !strings.Contains(refusal.Reason, "open/") {
		t.Fatalf("the refusal names the issue and its folder: %+v", refusal)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, intentRel)); err != nil {
		t.Fatalf("the intent moved on a refused landing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, ".abcd/development/specs/open/spc-1-alpha.md")); err != nil {
		t.Fatalf("the spec moved on a refused landing: %v", err)
	}
}
