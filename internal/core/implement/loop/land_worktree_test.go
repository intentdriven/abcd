package loop

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plantWorktree rewrites the run's state so its current lane names other as
// its worktree, as a hand-written state file under the gitignored local tier
// can.
func plantWorktree(t *testing.T, f *landFixture, other string) {
	t.Helper()
	err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		if i < 0 {
			t.Fatal("no lane is in progress")
		}
		st.Lanes[i].Worktree = other
		return true, nil
	})
	if err != nil {
		t.Fatalf("planting the worktree path: %v", err)
	}
}

// TestALandingRefusesAWorktreeTheLoopDidNotDerive is iss-2610090821552801: the
// brief stage refuses a worktree that is not the path the loop derives for the
// run and lane, but the land stage checked only that the string was non-empty,
// and with a landing recorded it skipped the sync, so a state file naming
// another directory had the close lay `.abcd/.work.local` there. Every stage
// that acts on the lane's worktree holds it to the derived path, so a planted
// path refuses before anything is made in it, whether the landing is recorded
// or not.
func TestALandingRefusesAWorktreeTheLoopDidNotDerive(t *testing.T) {
	t.Run("with the landing recorded", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.validated(t)
		if res := f.step(t); res.Stage != StageLand {
			t.Fatalf("the landing's first step leaves the lane at land: %+v", res)
		}
		l := currentLane(t, f.repo, f.runID)
		if l.Landing == nil || !l.Landing.Closes || l.Landing.RecordsDone {
			t.Fatalf("premise: a closing landing recorded and its records not yet done: %+v", l.Landing)
		}
		other := t.TempDir()
		f.repo.Git("init", "-q", other)
		plantWorktree(t, f, other)
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		if err == nil || !strings.Contains(err.Error(), "worktree") {
			t.Fatalf("a planted worktree path was landed in: %v", err)
		}
		if _, serr := os.Stat(filepath.Join(other, ".abcd")); serr == nil {
			t.Fatalf("the land stage made %s/.abcd in a directory the loop did not derive", other)
		}
	})
	t.Run("before the landing is recorded", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.validated(t)
		other := t.TempDir()
		f.repo.Git("init", "-q", other)
		plantWorktree(t, f, other)
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		if err == nil || !strings.Contains(err.Error(), "worktree") {
			t.Fatalf("a planted worktree path was prepared for landing: %v", err)
		}
		if _, serr := os.Stat(filepath.Join(other, ".abcd")); serr == nil {
			t.Fatalf("the land stage made %s/.abcd in a directory the loop did not derive", other)
		}
		if l := currentLane(t, f.repo, f.runID); l.Landing != nil {
			t.Fatalf("a landing was recorded for a planted worktree: %+v", l.Landing)
		}
	})
}

// TestStagesThatHandOutTheWorktreeRefuseAPlantedPath is the same rule at the
// implement and validate stages: each awaits an agent the driver starts in the
// lane's worktree, so a planted path would start one in a directory the loop
// did not make.
func TestStagesThatHandOutTheWorktreeRefuseAPlantedPath(t *testing.T) {
	for _, stage := range []Stage{StageImplement, StageValidate} {
		t.Run(string(stage), func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			if stage == StageValidate {
				implemented(t, f.repo, f.runID, f.stages, "one.txt")
			} else {
				stepTo(t, f.repo, f.runID, f.stages, stage)
			}
			other := t.TempDir()
			plantWorktree(t, f, other)
			res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if err == nil || !strings.Contains(err.Error(), "worktree") {
				t.Fatalf("the %s stage handed out a planted worktree path: %+v %v", stage, res, err)
			}
		})
	}
}

// TestADiscardRefusesToRemoveAWorktreeTheLaneDidNotMake: the discard removes
// the worktree git lists at the lane's path on the lane's branch, and both come
// from the state. A state naming a peer's worktree of the same repository, on
// its branch, had the discard remove that worktree and delete its branch.
func TestADiscardRefusesToRemoveAWorktreeTheLaneDidNotMake(t *testing.T) {
	f := armedSibling(t, false)
	f.handedBack(t)
	f.step(t)
	if l2 := f.lane(t, "lane-2"); l2.Stage != StageHeld {
		t.Fatalf("premise: lane 2 is held: %+v", l2)
	}
	peer := filepath.Join(t.TempDir(), "peer")
	f.repo.Git("worktree", "add", "-q", "-b", BranchPrefix+"peer", peer, "main")
	err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		for i := range st.Lanes {
			if st.Lanes[i].ID == "lane-2" {
				st.Lanes[i].Worktree, st.Lanes[i].Branch = peer, BranchPrefix+"peer"
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Discard(f.repo.Root(), f.runID, "lane-2", f.opts()); err == nil || !strings.Contains(err.Error(), "worktree") {
		t.Fatalf("a discard removed a worktree the lane did not make: %v", err)
	}
	if _, err := os.Stat(peer); err != nil {
		t.Fatalf("the peer's worktree is gone: %v", err)
	}
	if gitErr(f.repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+BranchPrefix+"peer") != nil {
		t.Fatal("the peer's branch is gone")
	}
}

// TestADiscardRefusesToDeleteABranchTheLoopDidNotMake: the hold discard
// deleted the lane's branch from the state alone, without the prefix refusal
// the hand-back discard applies, so a state naming `main` as a held lane's
// branch had `implement step --discard` delete the default branch. The discard
// refuses a branch outside the loop's prefix before it removes anything.
func TestADiscardRefusesToDeleteABranchTheLoopDidNotMake(t *testing.T) {
	f := armedSibling(t, false)
	f.handedBack(t)
	f.step(t)
	if l2 := f.lane(t, "lane-2"); l2.Stage != StageHeld {
		t.Fatalf("premise: lane 2 is held: %+v", l2)
	}
	mainTip := strings.TrimSpace(f.repo.Git("rev-parse", "refs/heads/main"))
	err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		for i := range st.Lanes {
			if st.Lanes[i].ID == "lane-2" {
				st.Lanes[i].Worktree, st.Lanes[i].Branch = "", "main"
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, f.repo.Root(), f.runID)
	_, err = Discard(f.repo.Root(), f.runID, "lane-2", f.opts())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "branch") {
		t.Fatalf("a discard of a lane naming main refuses on its branch: %+v", r)
	}
	if got := strings.TrimSpace(f.repo.Git("rev-parse", "--verify", "--quiet", "refs/heads/main")); got != mainTip {
		t.Fatalf("the discard deleted or moved main: %q, want %q", got, mainTip)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused discard leaves the lane held")
	}
	if n := strings.Count(f.ghLog(t), "pr close"); n != 0 {
		t.Fatalf("a refused discard closes no pull request: %d", n)
	}
}

// TestALandingRefusesABranchTheLoopDidNotMake is the same rule at the land
// stage, which pushes the lane's branch and deletes it once it has landed: a
// state naming `main`, with main's tip as the judged head, is refused before
// the landing pushes, records or deletes anything.
func TestALandingRefusesABranchTheLoopDidNotMake(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validated(t)
	mainTip := strings.TrimSpace(f.repo.Git("rev-parse", "refs/heads/main"))
	err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		if i < 0 {
			t.Fatal("no lane is in progress")
		}
		// main's own tip as the judged head, so the head check that would
		// otherwise refuse a branch at another commit is passed.
		st.Lanes[i].Branch, st.Lanes[i].HeadSHA = "main", mainTip
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "branch") {
		t.Fatalf("a landing of a lane naming main refuses on its branch: %+v", r)
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing != nil {
		t.Fatalf("a landing was recorded for a planted branch: %+v", l.Landing)
	}
	if got := strings.TrimSpace(f.repo.Git("rev-parse", "--verify", "--quiet", "refs/heads/main")); got != mainTip {
		t.Fatalf("the landing moved or deleted main: %q, want %q", got, mainTip)
	}
}
