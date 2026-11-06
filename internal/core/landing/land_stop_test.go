package landing

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runstop"
)

// aStop is a stop in force, as the read reports one.
var aStop = runstop.Reading{Stop: &runstop.Stop{SetBy: "alice", SetAt: time.Date(2026, 11, 6, 7, 0, 0, 0, time.UTC), Path: "stop.json"}}

// wantStopHolds is a refusal held by a stop, naming who set it and how it lifts.
func wantStopHolds(t *testing.T, err error) {
	t.Helper()
	r := wantHeld(t, err)
	if !strings.Contains(r.Reason, "a stop set by alice at 2026-11-06T07:00:00Z holds") || !strings.Contains(r.Reason, "pushes, arms and queues nothing") ||
		!strings.Contains(r.Fix, "abcd implement stop --lift") {
		t.Fatalf("the refusal does not name the stop: %v", r)
	}
}

// TestAStopHoldsALandingBeforeAnythingRuns: with a stop in force nothing is gated, asked of
// the forge, pushed or armed.
func TestAStopHoldsALandingBeforeAnythingRuns(t *testing.T) {
	for _, tc := range []struct {
		name  string
		apply func(*Options)
	}{
		{"both acts on", func(*Options) {}},
		{"the push alone", func(o *Options) { o.NoEnqueue = true }},
		{"the arming alone", func(o *Options) { o.NoPush = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, queueRuleset)
			f.forge.prs = f.prFor(f.head())
			f.forge.views = []string{viewBlocked, viewQueued}
			o := f.options()
			tc.apply(&o)
			o.StopRead = func() runstop.Reading { return aStop }
			_, err := f.land(o)
			wantStopHolds(t, err)
			if len(f.gates) != 0 || len(f.pushes()) != 0 || f.forge.arms != 0 || len(f.forge.calls) != 0 || f.onOrigin("feat/x") != "" {
				t.Fatalf("something ran: gates %v, git %v, forge %v", f.gates, f.gitCalls, f.forge.calls)
			}
		})
	}
}

// TestAStopSetDuringTheGateHoldsThePush: the preflight can run for hours, so the read is made
// again after it, before anything is announced or pushed.
func TestAStopSetDuringTheGateHoldsThePush(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	o := f.options()
	o.StopRead = func() runstop.Reading {
		if len(f.gates) > 0 {
			return aStop
		}
		return runstop.Reading{}
	}
	_, err := f.land(o)
	wantStopHolds(t, err)
	if len(f.gates) != 1 || len(f.pushes()) != 0 || f.forge.arms != 0 || f.onOrigin("feat/x") != "" {
		t.Fatalf("gates %v, git %v, arms %d", f.gates, f.gitCalls, f.forge.arms)
	}
}

// TestAStopSetAfterThePushHoldsTheArming: the read is made again at the arming, the merge
// gate's own read answered, and the steps done before it are in the result.
func TestAStopSetAfterThePushHoldsTheArming(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.asPat()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	o := f.options()
	o.StopRead = func() runstop.Reading {
		if len(f.pushes()) > 0 {
			return aStop
		}
		return runstop.Reading{}
	}
	res, err := f.land(o)
	wantStopHolds(t, err)
	if f.forge.arms != 0 {
		t.Fatalf("the move armed under a stop: %v", f.forge.calls)
	}
	if f.onOrigin("feat/x") != f.head() || step(t, res, "push").Status != StatusDone {
		t.Fatalf("the push before the stop is not reported: %+v", res.Steps)
	}
}

// TestAReadThatCannotTellHoldsALanding: a read that cannot be trusted is neither a stop nor
// its lift, and says what the person puts back.
func TestAReadThatCannotTellHoldsALanding(t *testing.T) {
	f := newFixture(t, queueRuleset)
	o := f.options()
	o.StopRead = func() runstop.Reading { return runstop.Reading{Held: "the shared-folder setting cannot be read"} }
	_, err := f.land(o)
	r := wantHeld(t, err)
	if !strings.Contains(r.Reason, "the shared-folder setting cannot be read") || !strings.Contains(r.Fix, "abcd ahoy shared-folder") {
		t.Fatalf("the refusal does not say why: %v", r)
	}
	if len(f.gates) != 0 || len(f.pushes()) != 0 || len(f.forge.calls) != 0 {
		t.Fatalf("something ran: gates %v, git %v, forge %v", f.gates, f.gitCalls, f.forge.calls)
	}
}

// TestALandingNamesWhatTheReadRefusedAndGoesOn: a file the read refused and a stop another
// account left are no stop: each is named once, and the landing goes on.
func TestALandingNamesWhatTheReadRefusedAndGoesOn(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.asPat()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	o := f.options()
	o.StopRead = func() runstop.Reading {
		return runstop.Reading{Refused: []string{"the stop file is refused"}, Foreign: []string{"a stop from another account stands"}}
	}
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	if f.onOrigin("feat/x") != f.head() {
		t.Fatalf("the branch was not pushed: %+v", res.Steps)
	}
	said := strings.Join(res.Notices, "\n")
	for _, want := range []string{"the stop file is refused", "a stop from another account stands"} {
		if strings.Count(said, want) != 1 {
			t.Fatalf("%q is not named once: %q", want, said)
		}
	}
}

// TestALandingWithBothActsOffNeedsNoRead: it would land nothing, so a stop has nothing to hold.
func TestALandingWithBothActsOffNeedsNoRead(t *testing.T) {
	f := newFixture(t, queueRuleset)
	o := f.options()
	o.NoPush, o.NoEnqueue = true, true
	o.StopRead = func() runstop.Reading { return aStop }
	if _, err := f.land(o); err != nil {
		t.Fatalf("a landing that does nothing is refused: %v", err)
	}
}
