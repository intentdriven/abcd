package loop

// hold_approval_test.go is where DR6c's hold meets AM1's arm decision: a held
// lane is never armed, and a released lane lands through the arm decision, so
// it is armed only where the ruleset requires a person's approval and is left
// open for a person where none does.

import (
	"strings"
	"testing"
)

// releaseHeldLane2 steps until lane 2 is held before its arm, checks it holds
// no arming, releases it and steps until its landing has decided its merge.
func releaseHeldLane2(t *testing.T, f *parFixture) Lane {
	t.Helper()
	f.handBackLane1(t)
	f.stepUntil(t, "lane-2 is held", func(st State) bool { return st.Lanes[1].Stage == StageHeld })
	if l := f.lane(t, "lane-2"); l.Hold == nil || l.Hold.Before != HoldBeforeArm || l.Landing.Armed || l.Landing.Merge != "" {
		t.Fatalf("lane 2 is held before its arm, holding no merge decision: %+v", l)
	}
	if _, err := Release(f.repo.Root(), f.runID, "lane-2", f.opts()); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if l := f.lane(t, "lane-2"); l.Landing.Merge != "" {
			return l
		}
		f.step(t)
	}
	t.Fatalf("the released lane never reached its arm decision: %+v", f.lane(t, "lane-2"))
	return Lane{}
}

func TestAReleasedLaneIsLeftOpenWhereTheRulesetRequiresNoApproval(t *testing.T) {
	f := siblingAtArm(t, false, unreviewedQueueRuleset("MERGE"), func(ld *Landing) bool { return ld.Merge != "" })
	if l := f.lane(t, "lane-2"); l.Landing.Armed || !strings.HasPrefix(l.Landing.Merge, leftOpenForAPerson) {
		t.Fatalf("lane 2 is left open before the hand-back: %+v", l.Landing)
	}
	l := releaseHeldLane2(t, f)
	if l.Landing.Armed || !strings.HasPrefix(l.Landing.Merge, leftOpenForAPerson) {
		t.Fatalf("the released lane is left open for a person: %+v", l.Landing)
	}
	if strings.Contains(f.ghLog(t), "--auto") {
		t.Fatalf("no lane of a run without a required approval is ever armed:\n%s", f.ghLog(t))
	}
}

func TestAReleasedLaneIsArmedWhereTheRulesetRequiresApproval(t *testing.T) {
	f := armedSibling(t, false)
	l := releaseHeldLane2(t, f)
	if !l.Landing.Armed {
		t.Fatalf("the released lane is armed again: %+v", l.Landing)
	}
	log := f.ghLog(t)
	if strings.Count(log, "pr merge 7 --auto") != 2 || strings.Count(log, "pr merge 7 --disable-auto") != 1 {
		t.Fatalf("armed, disarmed at the hold, armed again at the release:\n%s", log)
	}
}
