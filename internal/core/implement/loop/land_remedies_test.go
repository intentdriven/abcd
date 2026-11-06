package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
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
// no change beside a spec record's, markdown included.
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

// TestAChangeBeforeASyncIsRemovedAndTheSyncGoesOn: a sync merges only into a clean
// worktree, so the remedy offers the removal, and the merge follows it.
func TestAChangeBeforeASyncIsRemovedAndTheSyncGoesOn(t *testing.T) {
	f := syncFixture(t)
	l := f.passToLandOnce(t, "fix.txt")
	judged := l.HeadSHA
	f.moveMain(t, "another account's file\n", "other.txt")
	stray := filepath.Join(l.Worktree, "stray.txt")
	if err := os.WriteFile(stray, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "uncommitted changes") {
		t.Fatalf("the sync is refused over the change: %+v", r)
	}
	wantsRemoveRoute(t, r)

	if err := os.Remove(stray); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	if l = currentLane(t, f.repo, f.runID); l.HeadSHA == judged {
		t.Fatalf("the sync merges once the change is removed: head still %s", judged)
	}
}

// commitAtTheJudgedHead makes what edit changes part of the head the validators
// judged, which is where a lane stands when its own commits made the change.
func (f *landFixture) commitAtTheJudgedHead(t *testing.T, l Lane, edit func()) string {
	t.Helper()
	edit()
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "a change at the judged head")
	head := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[st.current()].HeadSHA = head
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return head
}

// wantsACommitRefused commits what settle changes, finds the landing refuse the
// moved head, and puts the branch back: the route no remedy may name.
func (f *landFixture) wantsACommitRefused(t *testing.T, l Lane, judged string, settle func()) {
	t.Helper()
	settle()
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "the fix, committed")
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, l.Branch) || !strings.Contains(r.Remedy, "restore the branch") {
		t.Fatalf("a commit moves the branch off the judged head: %+v", r)
	}
	f.repo.Git("-C", l.Worktree, "reset", "-q", "--hard", judged)
}

// A spec the judged head leaves unfit to mark: neither a commit (the head check) nor
// any change the first step refuses gets past it, so the one route is the spec put
// right in the worktree and left uncommitted, which the records commit carries.
func TestASpecUnfitToMarkAtTheFirstStepIsSettledUncommitted(t *testing.T) {
	const good = "1. The parser\n   - packages: internal/core/spec\n"
	closedRel := strings.Replace(specRel, "/open/", "/closed/", 1)
	cases := []struct {
		name   string
		reason string
		breaks func(t *testing.T, wt string)
		settle func(t *testing.T, wt string)
	}{
		{"a spec no longer open", "holds no open",
			func(t *testing.T, wt string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(filepath.Join(wt, closedRel)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(wt, specRel), filepath.Join(wt, closedRel)); err != nil {
					t.Fatal(err)
				}
			},
			func(t *testing.T, wt string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(filepath.Join(wt, specRel)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(wt, closedRel), filepath.Join(wt, specRel)); err != nil {
					t.Fatal(err)
				}
			}},
		{"steps that cannot be read", "cannot be read",
			func(t *testing.T, wt string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(wt, specRel), []byte(specWithSteps("the parser, then the loop\n")), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			func(t *testing.T, wt string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(wt, specRel), []byte(specWithSteps(good)), 0o644); err != nil {
					t.Fatal(err)
				}
			}},
		{"a step listed under another title", "does not list step",
			func(t *testing.T, wt string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(wt, specRel), []byte(specWithSteps("1. Another step\n")), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			func(t *testing.T, wt string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(wt, specRel), []byte(specWithSteps(good)), 0o644); err != nil {
					t.Fatal(err)
				}
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				specRel: specWithSteps(good),
			})
			l := f.validated(t)
			judged := f.commitAtTheJudgedHead(t, l, func() { c.breaks(t, l.Worktree) })

			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, c.reason) {
				t.Fatalf("the first step refuses the spec: %+v", r)
			}
			wantsUncommittedRoute(t, r)
			f.wantsACommitRefused(t, l, judged, func() { c.settle(t, l.Worktree) })

			c.settle(t, l.Worktree)
			f.step(t)
			l = currentLane(t, f.repo, f.runID)
			if l.Landing == nil || l.HeadSHA != judged {
				t.Fatalf("the landing is prepared at the judged head: %+v", l.Landing)
			}
			l = f.wantsRecordsCommit(t)
			files := f.repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", l.HeadSHA)
			if !strings.Contains(files, closedRel) {
				t.Fatalf("the records commit closes the settled spec:\n%s", files)
			}
		})
	}
}

// A record the lane's own commits already moved leaves the landing nothing to write,
// so the records commit has nothing to make until it stands where the landing
// expects it, in the worktree, for the landing's own resolution to write.
func TestARecordTheBranchAlreadyMovedIsSettledUncommitted(t *testing.T) {
	f := syncFixture(t)
	l := f.passToLandOnce(t, "fix.txt")
	found, err := filepath.Glob(filepath.Join(l.Worktree, ".abcd/work/issues/open", f.issue+"-*.md"))
	if err != nil || len(found) != 1 {
		t.Fatalf("premise: the lane's issue is open: %v %v", found, err)
	}
	openRel, err := filepath.Rel(l.Worktree, found[0])
	if err != nil {
		t.Fatal(err)
	}
	openBody, err := os.ReadFile(found[0])
	if err != nil {
		t.Fatal(err)
	}
	resolved := strings.Replace(found[0], "/issues/open/", "/issues/resolved/", 1)
	judged := f.commitAtTheJudgedHead(t, l, func() {
		if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: l.Worktree, ID: f.issue, Resolution: "resolved on the branch", Impact: "fix", ByCommit: l.HeadSHA}); err != nil {
			t.Fatal(err)
		}
	})
	f.step(t)

	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "changed nothing") {
		t.Fatalf("the records commit has nothing to make: %+v", r)
	}
	wantsUncommittedRoute(t, r)

	settle := func() {
		if err := os.Remove(resolved); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(l.Worktree, openRel), openBody, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	settle()
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "the record, put back and committed")
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Remedy, "restore the branch") {
		t.Fatalf("a commit moves the branch off the judged head: %+v", r)
	}
	f.repo.Git("-C", l.Worktree, "reset", "-q", "--hard", judged)

	settle()
	l = f.wantsRecordsCommit(t)
	if msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA); !strings.Contains(msg, "Resolves: "+f.issue) {
		t.Fatalf("the records commit resolves the record:\n%s", msg)
	}
}
