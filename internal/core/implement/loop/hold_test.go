package loop

// hold_test.go is DR6c's armed case (spc-2609202134341288, "The `held`
// state"): a sibling already armed when a lane is handed back is disarmed
// through the forge client and held, or, where the forge refuses the
// withdrawal, the step refuses naming the pull request; a lane the forge
// reports merged had landed before the hand-back. And the person's discard of a
// held lane with a pull request, which a refused worktree removal leaves
// retryable.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// armedSibling stands up a run whose lane 2 has landed as far as its armed
// merge while lane 1 is still at work: steps 2 (and, with third, 3) need none,
// and every implementer is out before lane 2's receipt.
func armedSibling(t *testing.T, third bool) *parFixture {
	t.Helper()
	steps, lanes := "1. One\n2. Two\n   - needs: none\n", 2
	if third {
		steps, lanes = steps+"3. Three\n   - needs: none\n", 3
	}
	f := newParFixture(t, steps, Options{SubAgents: strp("5"), FixRounds: strp("1")})
	f.stepUntil(t, "every implementer is out", func(st State) bool {
		if len(st.Lanes) != lanes {
			return false
		}
		for _, l := range st.Lanes {
			if len(l.Awaits) != 1 {
				return false
			}
		}
		return true
	})
	f.implement(t, "lane-2", "two.txt")
	f.roundPassed(t, "lane-2")
	for range 20 {
		l := f.lane(t, "lane-2")
		if l.Landing != nil && l.Landing.Armed {
			return f
		}
		if l.Landing != nil && l.Landing.RecordsDone && l.Landing.Pushed == "" {
			preflighted(t, l, l.HeadSHA)
		}
		f.step(t)
	}
	t.Fatalf("lane 2 never armed: %+v", f.lane(t, "lane-2"))
	return nil
}

// handedBack takes lane 1 through its fix round to its hand-back, and stops
// there: the next step is the first after the hand-back.
func (f *parFixture) handedBack(t *testing.T) {
	t.Helper()
	f.implement(t, "lane-1", "one.txt")
	for round := 1; round <= 2; round++ {
		f.stepUntil(t, "lane-1's reviewers are out", func(st State) bool { return len(st.Lanes[0].Awaits) == 2 })
		f.ret(t, "lane-1", RoleRuthless, "FIX FIRST")
		f.ret(t, "lane-1", RoleSecurity, "APPROVE")
		if round == 1 {
			f.stepUntil(t, "lane-1's fix implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
			f.receipt(t, "lane-1", f.await(t, "lane-1", RoleImplementer), laneCommit(t, f.repo, f.lane(t, "lane-1"), "fix.txt"))
		}
	}
	f.stepUntil(t, "lane-1 is handed back", func(st State) bool { return st.Lanes[0].Stage == StageHandedBack })
	if l := f.lane(t, "lane-2"); l.Stage != StageLand || l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("lane 2 is still armed at the hand-back: %+v", l)
	}
}

func (f *parFixture) touchGH(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.gh, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A landing waiting on the forge's merge holds only its own lane: the call
// moves another, and names the wait under blocked and in its next move.
func TestALandingWaitingOnItsMergeIsNamedBesideTheMoveTaken(t *testing.T) {
	f := armedSibling(t, false)
	f.implement(t, "lane-1", "one.txt")
	res := f.step(t)
	if res.Lane != "lane-1" || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the call hands out lane 1's reviewer while lane 2's merge waits: %+v", res)
	}
	if len(res.Blocked) != 1 || res.Blocked[0].Lane != "lane-2" || !res.Blocked[0].Contention {
		t.Fatalf("the result names lane 2's wait under blocked: %+v", res.Blocked)
	}
	if !strings.Contains(res.Next, "lane-2") || !strings.Contains(res.Next, "pull request #7 is not merged yet") {
		t.Fatalf("the next move names lane 2's wait: %s", res.Next)
	}
}

// DR6c, the armed sibling: where the forge refuses to withdraw the arming, the
// step refuses naming the pull request and moves no other lane; once the forge
// withdraws it, the lane is held before arming, its merge disarmed.
func TestAnArmedSiblingIsDisarmedOrTheStepRefusesNamingItsPullRequest(t *testing.T) {
	f := armedSibling(t, true)
	f.handedBack(t)
	// Lane 3 has a reviewer to hand out: a step that moved on would move it.
	f.implement(t, "lane-3", "three.txt")
	f.touchGH(t, "refuse-disarm", "")
	before := stateBytes(t, f.repo.Root(), f.runID)
	res, err := Advance(f.repo.Root(), f.runID, f.stages, f.opts())
	r := mustRefusal(t, err)
	if r.Contention || r.Lane != "lane-2" || !strings.Contains(r.Reason, "pull request #7") || !strings.Contains(r.Remedy, "pull request #7") {
		t.Fatalf("the step refuses naming the armed pull request: %+v %+v", r, res)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatalf("a refused disarm moves no other lane: %+v", f.lane(t, "lane-3"))
	}

	if err := os.Remove(filepath.Join(f.gh, "refuse-disarm")); err != nil {
		t.Fatal(err)
	}
	res = f.step(t)
	l2 := f.lane(t, "lane-2")
	if res.Lane != "lane-2" || l2.Stage != StageHeld || l2.Hold == nil || l2.Hold.Before != HoldBeforeArm || l2.Hold.Cause != "lane-1" || l2.Landing.Armed || l2.Landing.Merge != "" {
		t.Fatalf("the disarmed lane is held before arming: %+v %+v", res, l2)
	}
	if strings.Count(f.ghLog(t), "pr merge 7 --disable-auto") != 2 {
		t.Fatalf("the forge was asked twice to withdraw the arming:\n%s", f.ghLog(t))
	}
}

// DR6c, landed before the hand-back: an armed lane the forge reports merged is
// recorded as landed, not disarmed, though the local tracking ref, never
// fetched for the hold, has not caught up.
func TestAnArmedSiblingTheForgeReportsMergedIsRecordedAsLanded(t *testing.T) {
	f := armedSibling(t, false)
	f.handedBack(t)
	l2 := f.lane(t, "lane-2")
	stale := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main"))
	f.repo.Git("push", "-q", "origin", l2.Landing.Pushed+":refs/heads/main")
	f.repo.Git("update-ref", "refs/remotes/origin/main", stale)
	f.touchGH(t, "state", "MERGED\n")
	f.step(t)
	l2 = f.lane(t, "lane-2")
	if l2.Stage != StageDone || l2.Hold != nil || l2.Landing.Merged == "" {
		t.Fatalf("the merged lane is recorded as landed: %+v", l2)
	}
	if log := f.ghLog(t); strings.Contains(log, "--disable-auto") {
		t.Fatalf("a merged pull request is never disarmed:\n%s", log)
	}
}

// A discard removes the lane's worktree and branch before it closes the pull
// request: a worktree git refuses to remove leaves the pull request open and
// the lane held, and the retry closes it once.
func TestADiscardRemovesTheLaneLocallyBeforeItClosesItsPullRequest(t *testing.T) {
	f := armedSibling(t, false)
	f.handedBack(t)
	f.step(t)
	l2 := f.lane(t, "lane-2")
	if l2.Stage != StageHeld || l2.PR != 7 {
		t.Fatalf("lane 2 is held with its pull request open: %+v", l2)
	}
	stray := filepath.Join(l2.Worktree, "stray.txt")
	if err := os.WriteFile(stray, []byte("work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, f.repo.Root(), f.runID)
	_, err := Discard(f.repo.Root(), f.runID, "lane-2", f.opts())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "worktree") {
		t.Fatalf("a worktree with changes refuses the discard: %+v", r)
	}
	if n := strings.Count(f.ghLog(t), "pr close 7"); n != 0 || !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatalf("a refused discard closes no pull request (%d) and leaves the lane held", n)
	}
	if err := os.Remove(stray); err != nil {
		t.Fatal(err)
	}
	res, err := Discard(f.repo.Root(), f.runID, "lane-2", f.opts())
	if err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the retry discards the lane: %+v %v", res, err)
	}
	if n := strings.Count(f.ghLog(t), "pr close 7"); n != 1 {
		t.Fatalf("the pull request is closed once: %d\n%s", n, f.ghLog(t))
	}
	if _, err := os.Stat(l2.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is gone: %v", err)
	}
	if gitErr(f.repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+l2.Branch) == nil {
		t.Fatal("the lane's branch is gone")
	}
}
