package loop

import (
	"encoding/json"
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

// A first step that takes a spec record's change takes only the lane's own record:
// anything else in the store stays refused, as it does on a lane with no spec.
func TestTheFirstStepTakesOnlyTheLanesOwnSpecRecord(t *testing.T) {
	const other = ".abcd/development/specs/open/spc-2-other.md"
	otherBody := "---\nid: spc-2\nslug: other\nintent: itd-11\n---\n# other\n\n## Summary\n\nAnother record.\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		other: otherBody,
	})
	l := f.validated(t)
	own := filepath.Join(l.Worktree, specRel)
	write := func(rel, body string) func() {
		return func() {
			abs := filepath.Join(l.Worktree, rel)
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	remove := func(rel string) func() {
		return func() {
			if err := os.Remove(filepath.Join(l.Worktree, rel)); err != nil {
				t.Fatal(err)
			}
		}
	}
	const dir = ".abcd/development/specs/"
	for name, change := range map[string]func(){
		"another spec's record changed":   write(other, otherBody+"\nA line more.\n"),
		"another spec's record deleted":   remove(other),
		"a new record of another spec":    write(dir+"open/spc-3-new.md", otherBody),
		"a file that is no spec's record": write(dir+"open/notes.md", "notes\n"),
		"the own id outside a bucket":     write(dir+"spc-1-alpha.md", otherBody),
		"the own id nested in a bucket":   write(dir+"open/nested/spc-1-alpha.md", otherBody),
		"the own record deleted":          remove(specRel),
	} {
		t.Run(name, func(t *testing.T) {
			change()
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, "uncommitted") && !strings.Contains(r.Reason, "links") {
				t.Fatalf("the first step refuses it: %+v", r)
			}
			if l := currentLane(t, f.repo, f.runID); l.Landing != nil {
				t.Fatalf("no landing is prepared over it: %+v", l.Landing)
			}
			f.repo.Git("-C", l.Worktree, "reset", "-q", "--hard", "HEAD")
			f.repo.Git("-C", l.Worktree, "clean", "-q", "-fd")
		})
	}
	if _, err := os.Stat(own); err != nil {
		t.Fatalf("the own record stands after the cases: %v", err)
	}

	// The own record, changed, is the one thing it takes.
	body, err := os.ReadFile(own)
	if err != nil {
		t.Fatal(err)
	}
	write(specRel, string(body)+"\nA line more.\n")()
	f.step(t)
	if l := currentLane(t, f.repo, f.runID); l.Landing == nil {
		t.Fatal("the lane's own record is taken")
	}
}

func TestTheFirstStepTakesNoSpecRecordOnAnIssueLane(t *testing.T) {
	f := syncFixture(t)
	l := f.passToLandOnce(t, "fix.txt")
	p := filepath.Join(l.Worktree, ".abcd/development/specs/open/spc-1-alpha.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "uncommitted") {
		t.Fatalf("an issue lane has no spec record to change: %+v", r)
	}
}

// twoStepLandWithNoResolutions is a lane at its landing that neither closes the
// spec, a second step being still to build, nor fixed a capture.
func (f *landFixture) twoStepLandWithNoResolutions(t *testing.T) Lane {
	t.Helper()
	repo, id := f.repo, f.runID
	stepTo(t, repo, id, f.stages, StageImplement)
	if res, err := advance(repo.Root(), id, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, id)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	sha := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, sha)
	path := writeReceipt(t, dir, map[string]any{
		"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report, "model": f.model,
	})
	if _, err := Receipt(repo.Root(), id, path, f.stages, Options{}); err != nil {
		t.Fatalf("the receipt verifies: %v", err)
	}
	passRound(t, repo, id, f.stages, RoleRuthless, RoleSecurity)
	return stepTo(t, repo, id, f.stages, StageLand)
}

// A landing with nothing of its own to record still carries a spec the first step
// took: the change reaches the branch in the records commit, under the model the
// lane's receipts name.
func TestALandingWithNoRecordsCommitsTheSpecTheFirstStepTook(t *testing.T) {
	const second = "2. The loop\n   - packages: internal/core/implement\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n" + second),
	})
	l := f.twoStepLandWithNoResolutions(t)
	judged := f.commitAtTheJudgedHead(t, l, func() {
		if err := os.WriteFile(filepath.Join(l.Worktree, specRel), []byte(specWithSteps("1. Another step\n"+second)), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "does not list step") {
		t.Fatalf("the first step refuses the title: %+v", r)
	}
	settled := specWithSteps("1. The parser\n   - packages: internal/core/spec\n   - landed: #1\n" + second)
	if err := os.WriteFile(filepath.Join(l.Worktree, specRel), []byte(settled), 0o644); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if l.Landing == nil || l.Landing.MarkStep != 0 || l.Landing.Closes || l.Landing.RecordsDone {
		t.Fatalf("the prepared landing still has the spec to carry: %+v", l.Landing)
	}
	l = f.wantsRecordsCommit(t)
	if l.HeadSHA == judged {
		t.Fatal("the records commit carries the spec")
	}
	if got := f.repo.Git("-C", l.Worktree, "show", l.HeadSHA+":"+specRel); !strings.Contains(got, "1. The parser") {
		t.Fatalf("the branch holds the settled spec:\n%s", got)
	}
	if status := f.repo.Git("-C", l.Worktree, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Fatalf("nothing stays uncommitted: %s", status)
	}
	if msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA); !strings.Contains(msg, "Assisted-by: Claude:claude-test-5") {
		t.Fatalf("the commit names the lane's model:\n%s", msg)
	}
}

// A resolution written on this call that leaves nothing staged means the branch
// already holds what the landing would write, so the step finishes with no commit.
func TestAResolutionIdenticalToTheBranchsFinishesWithNoCommit(t *testing.T) {
	f := syncFixture(t)
	l := f.passToLandOnce(t, "fix.txt")
	if len(l.Resolves) != 1 {
		t.Fatalf("premise: the lane declares one resolution: %+v", l.Resolves)
	}
	r := l.Resolves[0]
	found, err := filepath.Glob(filepath.Join(l.Worktree, ".abcd/work/issues/open", f.issue+"-*.md"))
	if err != nil || len(found) != 1 {
		t.Fatalf("premise: the lane's issue is open: %v %v", found, err)
	}
	open := found[0]
	body, err := os.ReadFile(open)
	if err != nil {
		t.Fatal(err)
	}
	judged := f.commitAtTheJudgedHead(t, l, func() {
		if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: l.Worktree, ID: r.Issue, Resolution: r.Note, Impact: r.Impact, ByCommit: r.Commit, Grounds: r.Grounds}); err != nil {
			t.Fatal(err)
		}
	})
	f.step(t)
	if refusal := f.wantsRefusedStep(t); !strings.Contains(refusal.Reason, "changed nothing") {
		t.Fatalf("a resolution found on the branch writes nothing: %+v", refusal)
	}
	if err := os.Remove(strings.Replace(open, "/issues/open/", "/issues/resolved/", 1)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(open, body, 0o644); err != nil {
		t.Fatal(err)
	}

	l = f.wantsRecordsCommit(t)
	if l.HeadSHA != judged || l.Landing.Records != "" {
		t.Fatalf("the step finishes with no commit: head %s, judged %s, %+v", l.HeadSHA, judged, l.Landing)
	}
}

func TestARecordsRoundIsToldAboutASpecChange(t *testing.T) {
	scope := recordsRoundScope("base", "head")
	for _, want := range []string{"`- landed:`", "its folder", "named and justified"} {
		if !strings.Contains(scope, want) {
			t.Errorf("the records round's scope lacks %q:\n%s", want, scope)
		}
	}
}

// A sibling's landing syncs a lane before its first step, and a sync merges only into
// a clean worktree, so a spec left for the landing comes out first and goes back when
// the first step takes its turn after the sync's round.
func TestASyncAndASettledSpecTakeTheirTurns(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.bothImplementersOut(t)
	f.implement(t, "lane-1", "one.txt")
	f.implement(t, "lane-2", "two.txt")
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	f.landed(t, "lane-1")
	l := f.lane(t, "lane-2")
	path := filepath.Join(l.Worktree, specRel)
	edit := func() {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(string(body), "A written design record.", "A written design record, and more.", 1)
		if changed == string(body) {
			t.Fatalf("premise: the spec's summary is not there:\n%s", body)
		}
		if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	edit()
	_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "uncommitted changes") {
		t.Fatalf("the sync is refused over the spec: %+v", r)
	}
	wantsRemoveRoute(t, r)
	if !strings.Contains(r.Remedy, "settle that spec again") {
		t.Fatalf("the remedy says what follows the sync: %+v", r)
	}

	f.repo.Git("-C", l.Worktree, "checkout", "-q", "--", specRel)
	judged := l.HeadSHA
	f.stepUntil(t, "lane 2's sync", func(st State) bool {
		for _, cand := range st.Lanes {
			if cand.ID == "lane-2" {
				return cand.HeadSHA != judged
			}
		}
		return false
	})
	f.roundPassed(t, "lane-2")
	if l = f.lane(t, "lane-2"); l.Stage != StageLand || l.Landing != nil {
		t.Fatalf("the sync's round returns the lane to its first step: %+v", l)
	}

	edit()
	f.step(t)
	if l = f.lane(t, "lane-2"); l.Landing == nil || !l.Landing.SpecSettled {
		t.Fatalf("the first step takes the spec once the sync is done: %+v", l.Landing)
	}
}

// A closing lane's own record names what the close ships, so a change to that list
// is never one a first step takes: a bundle's close ships every planned member that
// carries the bundle's name, and a spec's intent: names the one a plain close ships.
func TestAChangeToWhatTheSpecLinksIsRefused(t *testing.T) {
	const second = ".abcd/development/intents/planned/itd-11-beta.md"
	body := strings.Replace(strings.Replace(readyIntent("impact: additive\nbundle: pair\n", settledQuestions), "id: itd-10\n", "id: itd-11\n", 1), "slug: alpha\n", "slug: beta\n", 1)
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		second: body,
	})
	l := f.validated(t)
	f.commitAtTheJudgedHead(t, l, func() {
		abs := filepath.Join(l.Worktree, plannedIntentRel)
		own, err := os.ReadFile(abs)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(strings.Replace(string(own), "\nslug:", "\nbundle: pair\nslug:", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	path := filepath.Join(l.Worktree, specRel)
	was, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string][2]string{
		"a bundle's members": {"intent: itd-10\n", "intent: itd-10\nbundle: pair\nintents: [itd-10, itd-11]\n"},
		"another intent":     {"intent: itd-10\n", "intent: itd-11\n"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.Replace(string(was), change[0], change[1], 1)
			if changed == string(was) {
				t.Fatal("premise: the record did not change")
			}
			if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if err == nil {
				_, next := advance(f.repo.Root(), f.runID, f.stages, Options{})
				cur := currentLane(t, f.repo, f.runID)
				files := f.repo.Git("-C", cur.Worktree, "ls-tree", "-r", "--name-only", cur.HeadSHA)
				t.Fatalf("the first step took the change; next step %v; itd-11 shipped in the records commit: %v", next, strings.Contains(files, "intents/shipped/itd-11-beta.md"))
			}
			if r := mustRefusal(t, err); !strings.Contains(r.Reason, "links") || !strings.Contains(r.Remedy, "as the judged head") {
				t.Fatalf("the refusal names the links: %+v", r)
			}
			if l := currentLane(t, f.repo, f.runID); l.Landing != nil {
				t.Fatalf("no landing is prepared over it: %+v", l.Landing)
			}
			if err := os.WriteFile(path, was, 0o644); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// A close reads the spec record the store resolves for the lane's spec id and the
// intent records the corpus resolves for its links, so the records step holds those
// to be the lane's own, as the judged head links them, and the only ones with the id.
func TestAClosingLaneReadsOnlyItsOwnRecords(t *testing.T) {
	const (
		secondIntent = ".abcd/development/intents/planned/itd-11-beta.md"
		otherSpec    = ".abcd/development/specs/open/spc-2-other.md"
		links        = "intent: itd-10\nbundle: pair\nintents: [itd-10, itd-11]\n"
	)
	write := func(t *testing.T, wt, rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(wt, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(t *testing.T, wt, rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(wt, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	linked := func(t *testing.T, wt string) string {
		t.Helper()
		return strings.Replace(read(t, wt, specRel), "intent: itd-10\n", links, 1)
	}
	beta := strings.Replace(strings.Replace(readyIntent("impact: additive\nbundle: pair\n", settledQuestions), "id: itd-10\n", "id: itd-11\n", 1), "slug: alpha\n", "slug: beta\n", 1)
	for name, c := range map[string]struct {
		extra  map[string]string
		change func(t *testing.T, wt string)
		reason string
	}{
		"the lane's record with its links changed": {change: func(t *testing.T, wt string) {
			t.Helper()
			write(t, wt, specRel, linked(t, wt))
		}, reason: "changes what it links"},
		"a new spec record with the id": {change: func(t *testing.T, wt string) {
			t.Helper()
			write(t, wt, ".abcd/development/specs/open/spc-0-second.md", linked(t, wt))
		}, reason: "the id of spc-1"},
		"an existing spec record given the id": {
			extra: map[string]string{otherSpec: "---\nid: spc-2\nslug: other\nintent: itd-11\n---\n# other\n\n## Summary\n\nAnother record.\n"},
			change: func(t *testing.T, wt string) {
				t.Helper()
				write(t, wt, otherSpec, linked(t, wt))
			}, reason: "the id of spc-1"},
		"the lane's record given another id": {change: func(t *testing.T, wt string) {
			t.Helper()
			write(t, wt, specRel, strings.Replace(read(t, wt, specRel), "id: spc-1\n", "id: spc-3\n", 1))
		}, reason: "the id of spc-1"},
		"a new intent record with the id": {change: func(t *testing.T, wt string) {
			t.Helper()
			write(t, wt, ".abcd/development/intents/planned/itd-10-aaa.md", read(t, wt, plannedIntentRel))
		}, reason: "the id of itd-10"},
		"an existing intent record given the id": {change: func(t *testing.T, wt string) {
			t.Helper()
			write(t, wt, secondIntent, strings.Replace(read(t, wt, secondIntent), "id: itd-11\n", "id: itd-10\n", 1))
		}, reason: "the id of itd-10"},
		"the lane's intent record's id taken by another": {change: func(t *testing.T, wt string) {
			t.Helper()
			write(t, wt, plannedIntentRel, strings.Replace(read(t, wt, plannedIntentRel), "id: itd-10\n", "id: itd-12\n", 1))
			write(t, wt, secondIntent, strings.Replace(read(t, wt, secondIntent), "id: itd-11\n", "id: itd-10\n", 1))
		}, reason: "the id of itd-10"},
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				secondIntent: beta,
			}
			for rel, body := range c.extra {
				files[rel] = body
			}
			f := newLandFixtureWith(t, files)
			l := f.validated(t)
			head := f.commitAtTheJudgedHead(t, l, func() {
				write(t, l.Worktree, plannedIntentRel, strings.Replace(read(t, l.Worktree, plannedIntentRel), "\nslug:", "\nbundle: pair\nslug:", 1))
			})
			f.step(t)
			if l := currentLane(t, f.repo, f.runID); l.Landing == nil || l.Landing.SpecSettled {
				t.Fatalf("premise: the first step ran on a clean tree: %+v", l.Landing)
			}
			c.change(t, l.Worktree)

			_, stepErr := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if _, err := os.Stat(filepath.Join(l.Worktree, ".abcd/development/intents/shipped/itd-11-beta.md")); err == nil {
				t.Fatalf("the close moved a record the lane's own does not link; step: %v", stepErr)
			}
			if stepErr == nil {
				t.Fatal("the records step went through")
			}
			if r := mustRefusal(t, stepErr); !strings.Contains(r.Reason, c.reason) {
				t.Fatalf("the refusal names the records: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != head {
				t.Fatalf("no records commit is made: %s, %s", got, head)
			}
			if _, err := os.Stat(filepath.Join(l.Worktree, filepath.FromSlash(plannedIntentRel))); err != nil {
				t.Fatalf("the lane's own intent stays planned: %v", err)
			}
		})
	}
}

// An audit's verdict is on the record it was asked of: another intent record parking
// the receipt does not take the ingest.
func TestAVerdictIngestsOnlyOnTheRecordItWasAskedOf(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE")})
	l := f.validated(t)
	f.step(t)
	raw, err := os.ReadFile(abs(f.repo.Root(), passingAudit(l)))
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		ReceiptID string `json:"receipt_id"`
	}
	if err := json.Unmarshal(raw, &v); err != nil || v.ReceiptID == "" {
		t.Fatalf("premise: the lane's audit names its receipt: %v %q", err, v.ReceiptID)
	}
	other := strings.Replace(strings.Replace(readyIntent("impact: additive\n", settledQuestions), "id: itd-10\n", "id: itd-0\n", 1), "slug: alpha\n", "slug: first\n", 1) +
		"\n## Audit Notes\n\n<!-- abcd-review: OWED receipt=" + v.ReceiptID + " -->\n"
	if err := os.WriteFile(filepath.Join(l.Worktree, ".abcd/development/intents/planned/itd-0-first.md"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stepErr := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if stepErr == nil {
		t.Fatal("the records step went through")
	}
	if r := mustRefusal(t, stepErr); !strings.Contains(r.Reason, "did not ingest") {
		t.Fatalf("the refusal names the ingest: %+v", r)
	}
	if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != l.HeadSHA {
		t.Fatalf("no records commit is made: %s, %s", got, l.HeadSHA)
	}
}

// A landing prepared over a change to the lane's spec holds it no longer once it is
// put back, so the first step runs again on the tree as it stands.
func TestASpecPutBackAfterTheFirstStepPreparesTheLandingAgain(t *testing.T) {
	const steps = "1. The parser\n   - packages: internal/core/spec\n2. The loop\n   - packages: internal/core/implement\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps(steps),
	})
	l := f.twoStepLandWithNoResolutions(t)
	path := filepath.Join(l.Worktree, specRel)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(before), "A written design record.", "A written design record, and more.", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	if cur := currentLane(t, f.repo, f.runID); cur.Landing == nil || !cur.Landing.SpecSettled {
		t.Fatalf("premise: the first step took the change: %+v", cur.Landing)
	}
	if err := os.WriteFile(path, before, 0o644); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	if cur := currentLane(t, f.repo, f.runID); cur.Landing != nil {
		t.Fatalf("no landing stands over the change put back: %+v", cur.Landing)
	}
	f.step(t)
	if cur := currentLane(t, f.repo, f.runID); cur.Landing == nil || cur.Landing.SpecSettled || cur.Landing.MarkStep != 1 {
		t.Fatalf("the first step prepares the landing again: %+v", cur.Landing)
	}
	l = f.wantsRecordsCommit(t)
	if msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA); !strings.Contains(msg, composedAssistedBy()) {
		t.Fatalf("a commit of the mark alone declares abcd:\n%s", msg)
	}
}

// A close's check on what it ships reads the lane's record in whichever bucket it
// stands, so a record already in closed/ meets the check an open one meets.
func TestAClosedRecordMeetsTheSameCheckOnWhatItShips(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.fileAnotherIssue(t, l)
	f.laneNamesIssue(t, l, "iss-77")
	f.step(t)
	if cur := currentLane(t, f.repo, f.runID); cur.Landing == nil || !cur.Landing.Closes {
		t.Fatalf("premise: the lane closes: %+v", cur.Landing)
	}
	closedRel := strings.Replace(specRel, "/open/", "/closed/", 1)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(l.Worktree, closedRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(l.Worktree, specRel), filepath.Join(l.Worktree, closedRel)); err != nil {
		t.Fatal(err)
	}
	_, stepErr := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if stepErr == nil {
		t.Fatal("the records step went through with a link the lane leaves unresolved")
	}
	if r := mustRefusal(t, stepErr); !strings.Contains(r.Reason, "iss-77") {
		t.Fatalf("the refusal names the link: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, plannedIntentRel)); err != nil {
		t.Fatalf("the lane's own intent stays planned: %v", err)
	}
}

// A commit declares abcd's own label only when it holds the mark abcd computed; the
// label is chosen from what is staged, so anything else in it names the lane's model.
func TestARecordsCommitIsLabelledFromWhatItHolds(t *testing.T) {
	const steps = "1. The parser\n   - packages: internal/core/spec\n2. The loop\n   - packages: internal/core/implement\n"
	for name, c := range map[string]struct {
		change  func(t *testing.T, f *landFixture, l Lane)
		abcdHas bool
	}{
		"the mark alone": {func(*testing.T, *landFixture, Lane) {}, true},
		"a spec edited after the mark refused": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			path := filepath.Join(l.Worktree, specRel)
			if err := os.WriteFile(path, []byte(specWithSteps("")), 0o644); err != nil {
				t.Fatal(err)
			}
			if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "could not be marked landed") {
				t.Fatalf("the mark refuses: %+v", r)
			}
			edited := strings.Replace(specWithSteps(steps), "A written design record.", "A written design record, and more.", 1)
			if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
				t.Fatal(err)
			}
		}, false},
		"a markdown file beside the mark": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(l.Worktree, "note.md"), []byte("a note\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, false},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				specRel: specWithSteps(steps),
			})
			f.twoStepLandWithNoResolutions(t)
			f.step(t)
			l := currentLane(t, f.repo, f.runID)
			if l.Landing == nil || l.Landing.MarkStep != 1 || l.Landing.Closes {
				t.Fatalf("premise: a landing that only marks its step: %+v", l.Landing)
			}
			c.change(t, f, l)
			l = f.wantsRecordsCommit(t)
			msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
			if got := strings.Contains(msg, composedAssistedBy()); got != c.abcdHas {
				t.Errorf("the abcd label present = %v, want %v:\n%s", got, c.abcdHas, msg)
			}
			if got := strings.Contains(msg, "Assisted-by: Claude:claude-test-5"); got == c.abcdHas {
				t.Errorf("the model's label present = %v, want %v:\n%s", got, !c.abcdHas, msg)
			}
		})
	}
}
