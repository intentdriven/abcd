package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A drain lane merges origin's default branch in its sync (spc-2610161044461448
// step 13; iss-2610161015410527; dcn-2610161040384068): an issue lane whose
// default branch moved since its base merges it before its records and its
// push, the merge head judged by a fresh round; a conflict confined to files
// origin's default branch declares linguist-generated goes back to an
// implementer, any other conflict hands the lane back naming the files with
// nothing pushed, and a pull request that starts to conflict while it is
// armed or waiting re-enters the sync rather than being refused for a person.

// genAttributes declares gen.txt generated, as abcd declares its CLI
// reference.
const genAttributes = "gen.txt linguist-generated\n"

// syncFixture is drainLandFixture with gen.txt and plain.txt at the base, the
// former declared generated on origin's default branch, and the drain's
// documentation lane opened.
func syncFixture(t *testing.T) *landFixture {
	t.Helper()
	f := drainLandFixture(t)
	f.repo.Write(".gitattributes", genAttributes)
	f.repo.Write("gen.txt", "generated at the base\n")
	f.repo.Write("plain.txt", "written at the base\n")
	f.repo.Commit("a generated file and a plain one")
	f.repo.Git("push", "-q", "origin", "main")
	f.repo.Git("fetch", "-q", "origin")
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	return f
}

// passToLand drives the lane through its implementer, who changes names, and
// a passing round, to its landing's first step.
func (f *landFixture) passToLand(t *testing.T, names ...string) Lane {
	t.Helper()
	repo := f.repo
	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := commitIn(t, repo, l, names...)
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: f.issue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	return stepTo(t, repo, f.runID, f.stages, StageLand)
}

// moveMain lands another account's change of names on origin's default
// branch, as a merge there does, and returns the new tip.
func (f *landFixture) moveMain(t *testing.T, content string, names ...string) string {
	t.Helper()
	for _, name := range names {
		f.repo.Write(name, content)
	}
	f.repo.Commit("another account's change of " + strings.Join(names, ", "))
	f.repo.Git("push", "-q", "origin", "main")
	return strings.TrimSpace(f.repo.Git("rev-parse", "HEAD"))
}

// TestAnIssueLaneMergesTheMovedDefaultBranchInItsSync: origin's default
// branch moved while the lane was validated, so its landing's first step
// fetches it and merges it into the lane's branch with a merge commit whose
// parents are the judged head and the default branch's tip, and a fresh round
// judges that head before anything is recorded or pushed.
func TestAnIssueLaneMergesTheMovedDefaultBranchInItsSync(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLand(t, "fix.txt").HeadSHA
	tip := f.moveMain(t, "another account's file\n", "other.txt")

	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageValidate || len(l.Syncs) != 1 || l.Landing != nil {
		t.Fatalf("the sync merges the default branch and sends the lane to a fresh round: %+v", l)
	}
	s := l.Syncs[0]
	if s.Merged != tip || s.Conflicted || s.Head != l.HeadSHA || len(s.Siblings) != 0 {
		t.Fatalf("the sync names the default branch's tip it merged and the head it made: %+v (tip %s)", s, tip)
	}
	if parents := strings.Fields(f.repo.Git("rev-list", "--parents", "-n", "1", l.HeadSHA)); len(parents) != 3 || parents[1] != judged || parents[2] != tip {
		t.Fatalf("the merge commit's parents are the judged head and the default branch's tip: %q", parents)
	}
	if !l.syncRoundDue() {
		t.Fatalf("a fresh round judges the merge head: %+v", l)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("nothing is pushed before the fresh round passes")
	}

	passRound(t, f.repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, f.repo, f.runID, f.stages, StageLand)
	f.step(t)
	if l := currentLane(t, f.repo, f.runID); l.Landing == nil || len(l.Syncs) != 1 {
		t.Fatalf("with the default branch merged, the landing goes on without a second sync: %+v", l)
	}
}

// TestAConflictOnADeclaredGeneratedFileGoesToTheImplementer: the default
// branch changed gen.txt, which origin declares generated, and so did the
// lane; the merge is aborted with the branch unchanged, and the conflict goes
// to an implementer with a brief that says the file is generated, nothing
// pushed.
func TestAConflictOnADeclaredGeneratedFileGoesToTheImplementer(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLand(t, "fix.txt", "gen.txt").HeadSHA
	f.moveMain(t, "generated again on main\n", "gen.txt")

	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if len(l.Syncs) != 1 || !l.Syncs[0].Conflicted || !slices.Equal(l.Syncs[0].Paths, []string{"gen.txt"}) || l.HeadSHA != judged {
		t.Fatalf("the conflict on the generated file is recorded, the branch unchanged: %+v", l)
	}
	if l.Stage != StageValidate || l.pendingSync() == nil {
		t.Fatalf("the conflict goes to an implementer: %+v", l)
	}
	brief, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(l.Syncs[0].Brief)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(brief), "generated") || !strings.Contains(string(brief), "gen.txt") {
		t.Fatalf("the brief names the generated file and says to regenerate it:\n%s", brief)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("nothing is pushed")
	}
}

// TestAConflictOnAnUndeclaredFileHandsBackAndPushesNothing: a conflict on
// plain.txt, which nothing declares generated, refuses the step naming it,
// in a way that cannot change, so the next drain move hands the lane back
// as stuck naming the file; nothing is pushed.
func TestAConflictOnAnUndeclaredFileHandsBackAndPushesNothing(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLand(t, "fix.txt", "plain.txt").HeadSHA
	f.moveMain(t, "rewritten on main\n", "plain.txt")

	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "plain.txt") || r.final == "" {
		t.Fatalf("the step is refused naming the conflicting file, in a way that cannot change: %+v", r)
	}
	l := currentLane(t, f.repo, f.runID)
	if l.HeadSHA != judged || len(l.Syncs) != 0 || f.repo.Git("rev-parse", "refs/heads/"+l.Branch) != judged {
		t.Fatalf("the refused sync leaves the branch at the judged head: %+v", l)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("nothing is pushed")
	}

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	route := routeOf(t, res.Routed, secondIssue)
	if route.Kind != claim.HandBackStuck || !strings.Contains(route.Reason, "plain.txt") {
		t.Fatalf("the next drain move hands the lane back naming the file: %+v", route)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("the hand-back pushes nothing")
	}
}

// TestAPullRequestThatStartsToConflictReentersTheSync: a lane's pull request
// that the forge reports conflicting on a generated file goes back through
// the sync, its push and its arming to be made again over the merge, rather
// than being refused for a person: at its arming, from the run's own step,
// and while it waits armed, from the drain's next move.
func TestAPullRequestThatStartsToConflictReentersTheSync(t *testing.T) {
	conflicting := func(t *testing.T, f *landFixture) {
		t.Helper()
		f.moveMain(t, "generated again on main\n", "gen.txt")
		if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(`{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reentered := func(t *testing.T, f *landFixture, pushed string) {
		t.Helper()
		l := currentLane(t, f.repo, f.runID)
		if len(l.Syncs) != 1 || !l.Syncs[0].Conflicted || !slices.Equal(l.Syncs[0].Paths, []string{"gen.txt"}) ||
			l.Stage != StageValidate || l.pendingSync() == nil {
			t.Fatalf("the lane re-enters its sync, the generated file's conflict going to an implementer: %+v", l)
		}
		if ld := l.Landing; ld == nil || ld.Pushed != "" || ld.Merge != "" || ld.Armed || l.PR != 7 || !ld.RecordsDone {
			t.Fatalf("the push and the arming are to be made again, the records and the pull request kept: %+v", l.Landing)
		}
		if got := f.remoteBranch(t, l.Branch); got != pushed {
			t.Fatalf("nothing more is pushed until a fresh round passes: %s, pushed %s", got, pushed)
		}
	}

	t.Run("at its arming", func(t *testing.T) {
		f := syncFixture(t)
		f.passToLand(t, "fix.txt", "gen.txt")
		for range 4 {
			f.step(t)
			if l := currentLane(t, f.repo, f.runID); l.Landing != nil && l.Landing.Pushed == "" && l.Landing.RecordsDone {
				preflighted(t, l, l.HeadSHA)
			}
			if l := currentLane(t, f.repo, f.runID); l.PR == 7 && l.Landing.BodyChecked {
				break
			}
		}
		l := currentLane(t, f.repo, f.runID)
		if l.PR != 7 || l.Landing.Merge != "" {
			t.Fatalf("the pull request is open and not yet armed: %+v", l.Landing)
		}
		conflicting(t, f)
		f.step(t)
		reentered(t, f, l.Landing.Pushed)
		if strings.Contains(f.ghLog(t), "--auto") {
			t.Fatalf("a conflicting pull request is not armed:\n%s", f.ghLog(t))
		}
	})

	t.Run("and lands again once a fresh round judges the merge", func(t *testing.T) {
		f := syncFixture(t)
		f.passToLand(t, "fix.txt")
		f.step(t)
		f.step(t)
		l := currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		f.step(t)
		f.step(t)
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if !l.Landing.Armed {
			t.Fatalf("the pull request is armed: %+v", l.Landing)
		}
		first := l.Landing.Pushed
		// The default branch moves with no conflict here, and the forge
		// reports the pull request conflicting all the same.
		tip := f.moveMain(t, "another account's file\n", "other.txt")
		if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(`{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if l.Stage != StageValidate || len(l.Syncs) != 1 || l.Syncs[0].Merged != tip || l.Syncs[0].Conflicted || !l.syncRoundDue() {
			t.Fatalf("the lane merges the default branch and a fresh round judges it: %+v", l)
		}
		if err := os.Remove(filepath.Join(f.gh, "mergeable.json")); err != nil {
			t.Fatal(err)
		}
		passRound(t, f.repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
		stepTo(t, f.repo, f.runID, f.stages, StageLand)
		l = currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		// The push, then the arming: the pull request is kept.
		f.step(t)
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if l.Landing.Pushed != l.HeadSHA || l.Landing.Pushed == first || !l.Landing.Armed || l.PR != 7 {
			t.Fatalf("the merge head is pushed to the same pull request and armed again: %+v", l.Landing)
		}
		if got := f.remoteBranch(t, l.Branch); got != l.HeadSHA {
			t.Fatalf("origin holds the merge head: %s, want %s", got, l.HeadSHA)
		}
	})

	t.Run("while it waits armed", func(t *testing.T) {
		f := syncFixture(t)
		f.passToLand(t, "fix.txt", "gen.txt")
		f.step(t)
		f.step(t)
		l := currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		f.step(t)
		f.step(t)
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if !l.Landing.Armed || l.PR != 7 {
			t.Fatalf("the pull request is armed: %+v", l.Landing)
		}
		conflicting(t, f)
		f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "CONFLICTING", MergeStateStatus: "DIRTY"})

		res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		reentered(t, f, l.Landing.Pushed)
		if !strings.Contains(f.ghLog(t), "--disable-auto") {
			t.Fatalf("the armed merge is withdrawn before the branch moves:\n%s", f.ghLog(t))
		}
		if p := parkedOf(t, res, f.runID); p.State != ParkedSyncing {
			t.Fatalf("the drain reads the lane as syncing again: %+v", p)
		}
		i := slices.IndexFunc(res.Lanes, func(dl DrainLane) bool { return dl.RunID == f.runID })
		if i < 0 || res.Lanes[i].Outcome != DrainLaneInProgress {
			t.Fatalf("the drain drives the lane again: %+v", res.Lanes)
		}
		if st, err := ReadState(f.repo.Root(), f.runID); err != nil || st.Lanes[0].HandBack != nil {
			t.Fatalf("the lane is not handed back: %+v %v", st.Lanes[0], err)
		}
	})
}
