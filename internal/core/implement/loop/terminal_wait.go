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
	if now.Sub(c.since) >= terminalOutageLimit {
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
func (c *terminalOutageClock) succeed() { *c = terminalOutageClock{} }

// sleepOr is the production sleep: d, or until the stop or ctx ends it.
func sleepOr(ctx context.Context, d time.Duration, stop <-chan struct{}) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-stop:
	case <-ctx.Done():
	}
}
