package loop

// terminal_wait.go is how long the terminal loop waits (spc-2610230813348813
// step 2): a wait a move names, and the outage schedule a run of failures
// enters.

import (
	"context"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
)

// TerminalWaitFloor is the shortest wait the terminal loop makes: no wait a
// move names, and no retry, comes sooner.
const TerminalWaitFloor = 30 * time.Second

// terminalFailuresToOutage is how many failures in a row enter the schedule.
const terminalFailuresToOutage = 3

// terminalOutageLimit is how long the loop waits through an outage before the
// run ends.
const terminalOutageLimit = 8 * time.Hour

// terminalRenewEvery is the longest a wait goes without renewing what the run
// holds: half the bundled lease, the point a renewal falls due.
const terminalRenewEvery = time.Hour

// What a wait under the person's bound (--until) keeps to.
const (
	// terminalProbeCap is the longest between two probes of a limit that named
	// no reset.
	terminalProbeCap = 30 * time.Minute
	// terminalIdleCap is the longest between two looks at something someone
	// else must do.
	terminalIdleCap = 15 * time.Minute
	// terminalProbeMisses is how many probes in a row may get no answer before
	// the lane's own launch finds out instead.
	terminalProbeMisses = 3
	// terminalResetSane is the furthest a reset may be and be read as one, so a
	// field in another unit reads as no reset rather than a wait until the bound.
	terminalResetSane = 35 * 24 * time.Hour
	// terminalNamedProbeEvery is how often a wait for a named reset asks whether
	// the limit still holds. A real limit is expected to reject the asking.
	terminalNamedProbeEvery = 4 * time.Hour
	// terminalFarReset is how far off a named reset is worth doubting sooner, and
	// terminalFarResetFirstLook how soon. The figure encodes no host's window.
	terminalFarReset          = 8 * 24 * time.Hour
	terminalFarResetFirstLook = 30 * time.Minute
	// terminalPoisonLaunches is how many launches of one role in a row may fail in a
	// way that is the work's before the lane goes back.
	terminalPoisonLaunches = 3
	// terminalNoMoveLimit is how many steps of one lane in a row may start no agent
	// and move no stage before it goes back: about 17 minutes on the outage's schedule
	// from a fresh clock.
	terminalNoMoveLimit = 6
	// terminalSpentUnanswered is how many launches of one role in a row may spend and
	// fail with no probe answered before the lane goes back or the issue is held.
	terminalSpentUnanswered = 6
	// terminalNetRepeats is how many times in a row the same failure may read as a
	// lost network while origin answers before the lane goes back.
	terminalNetRepeats = 3
)

// terminalCallSayEvery is how often an agent's call or a preflight says it is still
// going.
const terminalCallSayEvery = 30 * time.Minute

// terminalSayEvery is how often a wait that has not changed says it still stands.
const terminalSayEvery = 30 * time.Minute

// wallLookEvery is how often the production sleep looks at the wall clock.
const wallLookEvery = time.Minute

// terminalJudgeAsks is how many unanswered judgements of one issue a run makes
// before it holds that issue back and goes on.
const terminalJudgeAsks = 3

// terminalRenewTick is how often what a lane holds is looked at while an agent
// is in its call; a variable so a test can shorten it.
var terminalRenewTick = 5 * time.Minute

// moveWait is how long the loop sleeps after a move that returned a wait: until
// the earliest time the move named, at least the floor. ok is false when it
// named none.
func moveWait(res DrainResult, now time.Time) (time.Duration, bool) {
	var earliest time.Time
	note := func(t time.Time) {
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	if res.NextEligibleAt != nil {
		note(*res.NextEligibleAt)
	}
	if res.WaitingUntil != nil {
		note(*res.WaitingUntil)
	}
	if res.OriginWait != nil {
		note(res.OriginWait.NextProbeAt)
	}
	if earliest.IsZero() {
		return 0, false
	}
	return max(earliest.Sub(now), TerminalWaitFloor), true
}

// terminalOutageClock counts failures in a row and, once they enter the
// schedule, how far along it the loop is.
type terminalOutageClock struct {
	failures int
	waits    int
	since    time.Time
	// noLimit keeps the schedule going for as long as the person's bound lasts.
	noLimit bool
}

// fail counts one failure at now: how long to wait before trying again, and
// whether the outage has run its limit. A judge's failure enters the schedule
// at once.
func (c *terminalOutageClock) fail(now time.Time, judge bool) (time.Duration, bool) {
	c.failures++
	if c.since.IsZero() {
		if !judge && c.failures < terminalFailuresToOutage {
			return TerminalWaitFloor, false
		}
		c.since = now
	}
	if !c.noLimit && now.Sub(c.since) >= terminalOutageLimit {
		return 0, true
	}
	s := implement.DefaultSchedule
	wait := s.Hourly
	if c.waits < len(s.Waits) {
		wait = s.Waits[c.waits]
	}
	c.waits++
	return wait, false
}

// inOutage reports whether the failures have entered the schedule.
func (c *terminalOutageClock) inOutage() bool { return !c.since.IsZero() }

// succeed ends the count.
func (c *terminalOutageClock) succeed() { *c = terminalOutageClock{noLimit: c.noLimit} }

// sleepOr is the production sleep: d, or until the stop or ctx ends it, or until
// the wall clock says d has passed.
func sleepOr(ctx context.Context, d time.Duration, stop <-chan struct{}) {
	look := time.NewTicker(wallLookEvery)
	defer look.Stop()
	wallSleep(ctx, d, stop, time.Now, look.C)
}

// wallSleep waits d by a timer, which stops while the machine sleeps, and at each
// look by the wall clock, which does not, and whichever passes d first ends it.
func wallSleep(ctx context.Context, d time.Duration, stop <-chan struct{}, now func() time.Time, looks <-chan time.Time) {
	end := now().Round(0).Add(d)
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-looks:
			if !now().Round(0).Before(end) {
				return
			}
		}
	}
}
