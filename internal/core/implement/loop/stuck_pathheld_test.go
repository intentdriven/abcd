package loop

import (
	"strings"
	"testing"
	"time"
)

// TestTwoLanesWaitingOnEachOthersHoldAreHandedBackPastTheMinutesBound: the
// lane's branch changes README.txt, which another copy's lane holds live and
// keeps holding, as two lanes each changing a path the other holds do. Its
// wait is never counted toward the refusals bound, but it is counted toward
// the minutes bound, and once that passes the lane is handed back as stuck,
// the reason naming the path and the hold's record, and its reservation is
// released.
func TestTwoLanesWaitingOnEachOthersHoldAreHandedBackPastTheMinutesBound(t *testing.T) {
	f := newNarrowFixture(t)
	f.reserveAs(t, "iss-99", "quinn", "README.txt", time.Now().Add(-time.Minute))
	path := f.walledStage(t, "docs/page.md", "README.txt")
	if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
		t.Fatalf("the receipt is taken: %v", err)
	}
	b := stuckBounds{refusals: PaceValue{Value: BundledStuckRefusals}, minutes: PaceValue{Value: BundledStuckMinutes}}
	base := time.Now()
	for i := range 8 {
		at := base.Add(time.Duration(i) * 10 * time.Minute)
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return at }})
		if rf, ok := AsRefusal(err); !ok || !rf.pathHeld {
			t.Fatalf("step %d: the lane waits on the other hold: %v", i, err)
		}
		run, err := ReadState(f.repo.Root(), f.runID)
		if err != nil {
			t.Fatal(err)
		}
		l := run.Lanes[run.current()]
		if l.Refused == nil || l.Refused.Count != 0 {
			t.Fatalf("step %d: the wait is logged against the minutes bound alone: %+v", i, l.Refused)
		}
		why := stuckReason(run, b)
		if i < 6 && why != "" {
			t.Fatalf("step %d, %d minutes in: the lane is not stuck yet: %s", i, i*10, why)
		}
		if i == 7 {
			if why == "" || !strings.Contains(why, "README.txt") || !strings.Contains(why, "iss-99") {
				t.Fatalf("past the minutes bound the lane is stuck, naming the path and the hold's record: %q", why)
			}
		}
	}
	at := base.Add(70 * time.Minute)
	run, err := handBackStuck(f.repo.Root(), f.runID, b, at)
	if err != nil {
		t.Fatal(err)
	}
	l := run.Lanes[len(run.Lanes)-1]
	if l.HandBack == nil || l.HandBack.Kind != HandBackStuck || !strings.Contains(l.HandBack.Reason, "README.txt") || !strings.Contains(l.HandBack.Reason, "iss-99") {
		t.Fatalf("the lane is handed back as stuck, naming the path and the hold's record: %+v", l.HandBack)
	}
	if tip := f.tip(t); !tip.released {
		t.Fatalf("the hand-back releases the lane's reservation, ending the other lane's wait: %+v", tip)
	}
}

// TestAWaitOnAnotherHoldKeepsItsStartAcrossLooksFarApart: a drive's looks at
// a lane waiting on another copy's hold may come further apart than the gap
// that starts a refusal's time afresh; the wait keeps the time it began, so
// it still reaches the minutes bound.
func TestAWaitOnAnotherHoldKeepsItsStartAcrossLooksFarApart(t *testing.T) {
	f := newNarrowFixture(t)
	f.reserveAs(t, "iss-99", "quinn", "README.txt", time.Now().Add(-time.Minute))
	path := f.walledStage(t, "docs/page.md", "README.txt")
	if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
		t.Fatalf("the receipt is taken: %v", err)
	}
	b := stuckBounds{refusals: PaceValue{Value: BundledStuckRefusals}, minutes: PaceValue{Value: BundledStuckMinutes}}
	base := time.Now()
	apart := stuckGap + 5*time.Minute
	var why string
	for i := range 4 {
		at := base.Add(time.Duration(i) * apart)
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return at }}); err == nil {
			t.Fatalf("step %d: the lane waits on the other hold", i)
		}
		run, err := ReadState(f.repo.Root(), f.runID)
		if err != nil {
			t.Fatal(err)
		}
		if l := run.Lanes[run.current()]; l.Refused == nil || l.Refused.Since.Sub(base).Abs() > time.Second {
			t.Fatalf("step %d: the wait keeps the time it began: %+v", i, l.Refused)
		}
		why = stuckReason(run, b)
	}
	if why == "" || !strings.Contains(why, "README.txt") {
		t.Fatalf("looks %s apart still reach the minutes bound: %q", apart, why)
	}
}
