package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
)

const plannedIntentRel = ".abcd/development/intents/planned/itd-10-alpha.md"

// fileAnotherIssue writes an open issue no receipt of the lane resolves into the
// lane's worktree, for laneNamesIssue to commit.
func (f *landFixture) fileAnotherIssue(t *testing.T, l Lane) {
	t.Helper()
	issue := "---\nschema_version: 1\nid: \"iss-77\"\nslug: \"another\"\nseverity: \"minor\"\ncategory: \"ux\"\nsource: \"agent-observation\"\nfound_during: \"t\"\nrelated_intents: [itd-10]\n---\n\nAnother observation.\n"
	if err := os.WriteFile(filepath.Join(l.Worktree, ".abcd/work/issues/open/iss-77-another.md"), []byte(issue), 0o644); err != nil {
		t.Fatal(err)
	}
}

// laneNamesIssue has the intent name issue, committed at the head the validators
// judged, which is where a lane stands when its intent named the issue all along.
func (f *landFixture) laneNamesIssue(t *testing.T, l Lane, issue string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(l.Worktree, plannedIntentRel))
	if err != nil {
		t.Fatal(err)
	}
	named := strings.Replace(string(body), "\nslug:", "\nrelated_issues: ["+issue+"]\nslug:", 1)
	if named == string(body) {
		t.Fatalf("premise: no slug field to anchor on:\n%s", body)
	}
	if err := os.WriteFile(filepath.Join(l.Worktree, plannedIntentRel), []byte(named), 0o644); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "name an issue")
	head := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[st.current()].HeadSHA = head
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestALandingRefusesToShipAnIntentNamingAnIssueNoLaneResolves: the closing lane
// runs the close `abcd spec close` runs, so it meets the same refusal for an
// open issue the intent names and no receipt of this lane resolves, before the
// spec moves (iss-2611060603312990).
func TestALandingRefusesToShipAnIntentNamingAnIssueNoLaneResolves(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.fileAnotherIssue(t, l)
	f.laneNamesIssue(t, l, "iss-77")

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
	if _, err := os.Stat(filepath.Join(l.Worktree, plannedIntentRel)); err != nil {
		t.Fatalf("the intent moved on a refused landing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, ".abcd/development/specs/open/spc-1-alpha.md")); err != nil {
		t.Fatalf("the spec moved on a refused landing: %v", err)
	}
}

// TestFollowingTheShipLinkRemedyLands walks the refusal's own remedy: the link is
// settled in the lane's worktree and left uncommitted, and the landing goes on to
// arm. A commit there would move the branch off the judged head, which the landing
// refuses, and restoring that head undoes the fix.
func TestFollowingTheShipLinkRemedyLands(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.fileAnotherIssue(t, l)
	f.laneNamesIssue(t, l, "iss-77")

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
	if !strings.Contains(refusal.Remedy, "lane's worktree") || !strings.Contains(refusal.Remedy, "leave it uncommitted") {
		t.Fatalf("the remedy names what gets through the landing's head check: %+v", refusal)
	}

	if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: l.Worktree, ID: "iss-77", Resolution: "settled for this landing", Impact: "fix",
		ByCommit: currentLane(t, f.repo, f.runID).HeadSHA}); err != nil {
		t.Fatalf("settling the link as the remedy says: %v", err)
	}

	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if l.Landing == nil || !l.Landing.RecordsDone {
		t.Fatalf("the landing's records are committed once the link is settled: %+v", l.Landing)
	}
	files := f.repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", l.HeadSHA)
	for _, want := range []string{".abcd/development/intents/shipped/itd-10-alpha.md", ".abcd/work/issues/resolved/iss-77-another.md"} {
		if !strings.Contains(files, want) {
			t.Fatalf("the records commit carries %s:\n%s", want, files)
		}
	}
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if got := f.remoteBranch(t, l.Branch); got != l.HeadSHA {
		t.Fatalf("the lane's head is pushed: remote %q, head %s", got, l.HeadSHA)
	}
	if !strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("the merge is armed:\n%s", f.ghLog(t))
	}
}

// TestALandingShipsAnIntentNamingTheIssueItsReceiptsResolve: the lane's own
// resolution lands in the commit that closes the spec, so the link is settled by
// the time the intent ships, and the guard lets it through.
func TestALandingShipsAnIntentNamingTheIssueItsReceiptsResolve(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.laneNamesIssue(t, l, f.issue)

	for range 2 {
		f.step(t)
	}
	l = currentLane(t, f.repo, f.runID)
	files := f.repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", l.HeadSHA)
	if !strings.Contains(files, ".abcd/development/intents/shipped/itd-10-alpha.md") || !strings.Contains(files, ".abcd/work/issues/resolved/"+f.issue) {
		t.Fatalf("the records commit ships the intent and resolves the issue it names:\n%s", files)
	}
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	if !strings.Contains(msg, "Resolves: "+f.issue) || !strings.Contains(msg, "Delivers: itd-10") {
		t.Fatalf("the records commit declares both:\n%s", msg)
	}
}
