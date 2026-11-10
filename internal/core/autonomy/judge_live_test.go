package autonomy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
)

// judge_live_test.go: an owed item whose whole test run is live is waiting,
// not stuck. A wake releases its unchanged set however many wakes have passed,
// the wake count does not advance, and the wait is bounded.

// running is item with a whole run live for it since since.
func running(item OwedItem, since time.Time) OwedItem {
	item.Live = &LiveWork{What: "a whole test run (abcd process 4242) for " + item.Branch, Since: since}
	return item
}

// TestLiveWorkReleasesEveryWake: a preflight live for the owed head over
// five wakes releases the stop each time, and the wake count stays at none.
func TestLiveWorkReleasesEveryWake(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	wake := StopInput{SessionCrons: crons("*/10 * * * *")}
	for _, item := range []OwedItem{unpushedBr, stackedBr, behindPR, failingPR} {
		s := newSequence(t, awayRec(now, false), now)
		owed := owedOf(running(item, now.Add(-time.Minute)))
		if v := s.stop(owed, wake); v.Row != 2 {
			t.Fatalf("%s: the first stop is row %d, want the first block", item.Kind, v.Row)
		}
		for i := 1; i <= 5; i++ {
			v := s.stop(owed, wake)
			if v.Block || v.Row != 3 || v.State.Wakes != 0 {
				t.Errorf("%s: wake %d: block %v row %d wakes %d, want released at row 3 with the count at none", item.Kind, i, v.Block, v.Row, v.State.Wakes)
			}
			if v.State.WaitingSince.IsZero() {
				t.Errorf("%s: wake %d: no waiting span recorded", item.Kind, i)
			}
		}
	}
}

// TestLiveWorkEndedResumesTheCount: once the run ends with no receipt the
// item is read with no live work, and the count resumes where it was:
// today's rule, two more wakes.
func TestLiveWorkEndedResumesTheCount(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	wake := StopInput{SessionCrons: crons("*/10 * * * *")}
	s := newSequence(t, awayRec(now, false), now)
	s.stop(owedOf(unpushedBr), wake)
	if v := s.stop(owedOf(unpushedBr), wake); v.Block || v.State.Wakes != 1 {
		t.Fatalf("first wake: %+v", v)
	}
	for i := 0; i < 3; i++ {
		if v := s.stop(owedOf(running(unpushedBr, now)), wake); v.Block || v.State.Wakes != 1 {
			t.Fatalf("a live wake moved the count: block %v wakes %d", v.Block, v.State.Wakes)
		}
	}
	if v := s.stop(owedOf(unpushedBr), wake); v.Block || v.State.Wakes != 2 {
		t.Fatalf("the second wake with no live work: block %v wakes %d, want released at two", v.Block, v.State.Wakes)
	}
	if v := s.stop(owedOf(unpushedBr), wake); !v.Block || v.Row != 4 {
		t.Fatalf("the third wake with no live work: block %v row %d, want row 4", v.Block, v.Row)
	}
}

// TestLiveWorkOnEveryCappedItem: a set with one capped item live and another
// not keeps today's rule.
func TestLiveWorkOnEveryCappedItem(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	wake := StopInput{SessionCrons: crons("*/10 * * * *")}
	s := newSequence(t, awayRec(now, false), now)
	owed := owedOf(running(unpushedBr, now), behindPR)
	s.stop(owed, wake)
	for i, want := range []bool{false, false, true} {
		if v := s.stop(owed, wake); v.Block != want {
			t.Errorf("wake %d: block %v, want %v (row %d)", i+1, v.Block, want, v.Row)
		}
	}
	// A renewal due, which is not capped, needs no live work of its own.
	s = newSequence(t, awayRec(now, false), now)
	due := renewalDue
	owed = Owed{Items: []OwedItem{running(unpushedBr, now), due}, NextRenewal: now.Add(-time.Hour), WakeBy: now.Add(3 * time.Hour)}
	s.stop(owed, wake)
	for i := 0; i < 4; i++ {
		if v := s.stop(owed, wake); v.Block {
			t.Errorf("wake %d with a renewal due beside a live run: blocked (row %d)", i+1, v.Row)
		}
	}
}

// TestLiveWorkKeyIgnoresTheRun: the live work is not part of an item's
// identity, so the fingerprints do not change as the run ages or ends.
func TestLiveWorkKeyIgnoresTheRun(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	plain, live, older := owedOf(unpushedBr), owedOf(running(unpushedBr, now)), owedOf(running(unpushedBr, now.Add(-time.Hour)))
	for _, o := range []Owed{live, older} {
		if o.Fingerprint() != plain.Fingerprint() || o.CappedFingerprint() != plain.CappedFingerprint() {
			t.Fatal("the live work changed the owed set's fingerprint")
		}
	}
}

// TestLiveWorkBoundIsOnePreflight: the bound is landing's on one preflight,
// its gate waits included, eighteen hours.
func TestLiveWorkBoundIsOnePreflight(t *testing.T) {
	if LiveWorkBound != landing.PreflightBound {
		t.Fatalf("LiveWorkBound = %s, landing.PreflightBound = %s; want them one", LiveWorkBound, landing.PreflightBound)
	}
	if LiveWorkBound != 18*time.Hour {
		t.Fatalf("LiveWorkBound = %s, want 18h (12h + 2h + 4h)", LiveWorkBound)
	}
}

// TestShortDuration: a bound is shown as a person reads it.
func TestShortDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{18 * time.Hour: "18h", 18*time.Hour + 30*time.Minute: "18h30m", 90 * time.Second: "1m30s", 20 * time.Minute: "20m"} {
		if got := shortDuration(d); got != want {
			t.Errorf("shortDuration(%s) = %q, want %q", d, got, want)
		}
	}
}

// TestLiveWorkPastTheBound:a session that has waited on live work for one
// unchanged capped set longer than the bound is held to today's rule: with
// the count at two the wake is refused, and the reason says the wait is
// older than the bound. The span is measured from the first waiting turn,
// not from the run's age, so a run restarted at each wake cannot renew it,
// and a change in the capped set starts it again.
func TestLiveWorkPastTheBound(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	rec := awayRec(now, false)
	wake := StopInput{SessionCrons: crons("*/10 * * * *")}
	young := owedOf(running(unpushedBr, now.Add(-time.Minute)))
	seen := SessionState{Fingerprint: young.Fingerprint(), WakeFingerprint: young.CappedFingerprint(), Wakes: MaxWakesUnchanged}

	st := seen
	st.WaitingSince = now.Add(-LiveWorkBound + time.Minute)
	if v := judgeAt(t, rec, young, wake, st, now); v.Block || v.Row != 3 || !v.State.WaitingSince.Equal(st.WaitingSince) {
		t.Fatalf("inside the bound: block %v row %d since %s, want released, the span kept", v.Block, v.Row, v.State.WaitingSince)
	}
	st.WaitingSince = now.Add(-LiveWorkBound - time.Minute)
	v := judgeAt(t, rec, young, wake, st, now)
	if !v.Block || v.Row != 4 {
		t.Fatalf("past the bound: block %v row %d, want row 4", v.Block, v.Row)
	}
	for _, w := range []string{"longer than", "18h", "bound"} {
		if !strings.Contains(v.Reason, w) {
			t.Errorf("past the bound, the reason misses %q: %q", w, v.Reason)
		}
	}
	if strings.Contains(v.Reason, "has not moved over two wakes") {
		t.Errorf("past the bound, the reason still says the set has not moved: %q", v.Reason)
	}
	// Past the bound with the count below two: today's rule, the count moves.
	st.Wakes = 0
	if v := judgeAt(t, rec, young, wake, st, now); v.Block || v.State.Wakes != 1 {
		t.Fatalf("past the bound with no wakes counted: block %v wakes %d, want released with the count at one", v.Block, v.State.Wakes)
	}
	// A change in the capped set starts the span again.
	moved := owedOf(running(stackedBr, now))
	st.WaitingSince = now.Add(-LiveWorkBound - time.Minute)
	if v := judgeAt(t, rec, moved, wake, st, now); !v.State.WaitingSince.IsZero() && !v.State.WaitingSince.Equal(now) {
		t.Fatalf("a new capped set kept the old span: %s", v.State.WaitingSince)
	}
}

// TestLiveWorkWithoutAWakeNamesIt: a waiting turn with no qualifying wake
// (row 5) names what is running and since when before asking for a look; a
// waiting turn with a wake is allowed silently.
func TestLiveWorkWithoutAWakeNamesIt(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	rec := awayRec(now, false)
	since := at(t, "2026-10-18 20:31")
	owed := owedOf(running(unpushedBr, since))
	st := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint()}
	v := judgeAt(t, rec, owed, StopInput{}, st, now)
	if !v.Block || v.Row != 5 {
		t.Fatalf("no wake: block %v row %d, want row 5", v.Block, v.Row)
	}
	for _, w := range []string{"a whole test run (abcd process 4242) for feat/unpushed", "since 20:31", "schedule a look"} {
		if !strings.Contains(v.Reason, w) {
			t.Errorf("the reason misses %q: %q", w, v.Reason)
		}
	}
	if v := judgeAt(t, rec, owed, StopInput{SessionCrons: crons("*/10 * * * *")}, st, now); v.Block || v.Reason != "" {
		t.Errorf("with a wake: block %v reason %q, want a silent allow", v.Block, v.Reason)
	}
}

// TestLiveWorkCutShortLeavesTheState: a reading the deadline cut short is
// allowed as today, the session's state left as it was, whatever live work
// it found before the cut.
func TestLiveWorkCutShortLeavesTheState(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	owed := owedOf(running(unpushedBr, now))
	owed.Incomplete = true
	owed.Notes = []string{"the check ran out of its time budget before reading the whole test runs"}
	v := judgeAt(t, awayRec(now, false), owed, StopInput{SessionCrons: crons("*/10 * * * *")}, SessionState{Wakes: 2}, now)
	if v.Block || v.Save || v.Stop == nil || v.Stop.Kind != StopCheckIncomplete {
		t.Fatalf("a cut-short reading: %+v, want allowed, nothing saved, the stop recorded", v)
	}
}

// TestWaitingSinceSurvivesTheStateFile: the span is kept between hook runs.
func TestWaitingSinceSurvivesTheStateFile(t *testing.T) {
	root, since := t.TempDir(), time.Date(2026, 10, 18, 20, 0, 0, 0, time.UTC)
	if err := SaveSession(root, "sess-1", SessionState{WaitingSince: since}); err != nil {
		t.Fatal(err)
	}
	st, err := LoadSession(context.Background(), root, "sess-1")
	if err != nil || !st.WaitingSince.Equal(since) {
		t.Fatalf("loaded %+v, %v; want the span kept", st, err)
	}
}
