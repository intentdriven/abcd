package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A commit in the lane's worktree moves the branch off the judged head, and the
// landing refuses a moved head whatever the commit holds, so no landing remedy can
// send a person there. These tests walk each remedy's own route to the far side.

// wantsRefusedStep takes the step that must refuse and hands back its refusal.
func (f *landFixture) wantsRefusedStep(t *testing.T) *Refusal {
	t.Helper()
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err == nil {
		t.Fatal("the landing step went through; it should refuse")
	}
	return mustRefusal(t, err)
}

func (f *landFixture) wantsRecordsCommit(t *testing.T) Lane {
	t.Helper()
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if l.Landing == nil || !l.Landing.RecordsDone {
		t.Fatalf("the landing's records are committed once the remedy is followed: %+v", l.Landing)
	}
	return l
}

func wantsUncommittedRoute(t *testing.T, r *Refusal) {
	t.Helper()
	if !strings.Contains(r.Remedy, "in the lane's worktree and leave it uncommitted") {
		t.Fatalf("the remedy names the route the landing's head check lets through: %+v", r)
	}
}

// wantsRemoveRoute: a change this check refuses is one the landing cannot carry,
// so the remedy offers only its removal.
func wantsRemoveRoute(t *testing.T, r *Refusal) {
	t.Helper()
	if strings.Contains(r.Remedy, "commit or remove") || !strings.Contains(r.Remedy, "remove") {
		t.Fatalf("the remedy offers only what gets through: %+v", r)
	}
}

// TestAChangeBeforeThePrepareStepIsRemovedAndTheLandingGoesOn: the first step takes
// no change at all, markdown included.
func TestAChangeBeforeThePrepareStepIsRemovedAndTheLandingGoesOn(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	stray := filepath.Join(l.Worktree, "stray.md")
	if err := os.WriteFile(stray, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantsRemoveRoute(t, f.wantsRefusedStep(t))

	if err := os.Remove(stray); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	if l = currentLane(t, f.repo, f.runID); l.Landing == nil {
		t.Fatal("the landing is prepared once the change is removed")
	}
}

// TestAStrayFileAtTheRecordsStepIsRemovedAndTheLandingGoesOn: only markdown can stay
// for the records commit, so anything else is removed.
func TestAStrayFileAtTheRecordsStepIsRemovedAndTheLandingGoesOn(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validatedOnce(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	stray := filepath.Join(l.Worktree, "stray.txt")
	if err := os.WriteFile(stray, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantsRemoveRoute(t, f.wantsRefusedStep(t))

	if err := os.Remove(stray); err != nil {
		t.Fatal(err)
	}
	f.wantsRecordsCommit(t)
}

// TestASpecTheMarkCannotReadIsSettledUncommitted: the mark refuses a spec that no
// longer lists the step, and the same spec, put right in the worktree, lands.
func TestASpecTheMarkCannotReadIsSettledUncommitted(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n"),
	})
	f.validated(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	path := filepath.Join(l.Worktree, specRel)
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(specWithSteps("")), 0o644); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "could not be marked landed") {
		t.Fatalf("the mark refuses: %+v", r)
	}
	wantsUncommittedRoute(t, r)

	if err := os.WriteFile(path, good, 0o644); err != nil {
		t.Fatal(err)
	}
	f.wantsRecordsCommit(t)
}

// TestAnIntentTheCloseRefusesIsSettledUncommitted: the close refuses an intent with
// no impact, and the same intent, given one in the worktree, ships.
func TestAnIntentTheCloseRefusesIsSettledUncommitted(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validatedOnce(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	path := filepath.Join(l.Worktree, plannedIntentRel)
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	without := strings.Replace(string(good), "impact: additive\n", "", 1)
	if without == string(good) {
		t.Fatalf("premise: the intent declares no impact:\n%s", good)
	}
	if err := os.WriteFile(path, []byte(without), 0o644); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "refused in the lane's worktree") {
		t.Fatalf("the close refuses: %+v", r)
	}
	wantsUncommittedRoute(t, r)

	if err := os.WriteFile(path, good, 0o644); err != nil {
		t.Fatal(err)
	}
	l = f.wantsRecordsCommit(t)
	files := f.repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", l.HeadSHA)
	if !strings.Contains(files, ".abcd/development/intents/shipped/itd-10-alpha.md") {
		t.Fatalf("the intent ships:\n%s", files)
	}
}

// TestAnIssueTheResolutionCannotFindIsSettledUncommitted: the close has run by the
// time the resolution refuses, so the retry goes on over the closed spec.
func TestAnIssueTheResolutionCannotFindIsSettledUncommitted(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validatedOnce(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	found, err := filepath.Glob(filepath.Join(l.Worktree, ".abcd/work/issues/open", f.issue+"-*.md"))
	if err != nil || len(found) != 1 {
		t.Fatalf("premise: the lane's issue is open: %v %v", found, err)
	}
	open := found[0]
	aside := strings.Replace(open, "/issues/open/", "/issues/wontfix/", 1)
	if err := os.MkdirAll(filepath.Dir(aside), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(open, aside); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "`capture resolve "+f.issue+"` refused") {
		t.Fatalf("the resolution refuses: %+v", r)
	}
	wantsUncommittedRoute(t, r)

	if err := os.Rename(aside, open); err != nil {
		t.Fatal(err)
	}
	l = f.wantsRecordsCommit(t)
	files := f.repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", l.HeadSHA)
	for _, want := range []string{".abcd/development/intents/shipped/itd-10-alpha.md", ".abcd/work/issues/resolved/" + f.issue} {
		if !strings.Contains(files, want) {
			t.Fatalf("the records commit carries %s:\n%s", want, files)
		}
	}
}
