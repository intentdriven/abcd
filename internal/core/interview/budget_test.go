package interview

import (
	"testing"
	"time"
)

// budget is how long a test of this package waits for work an idle machine
// does in a second or two (a stubbed turn's dispatch, a reading the test
// guards against a hang) and how long a stubbed turn sleeps to be stopped
// mid-run: three quarters of the time the test binary has left (t.Deadline),
// so a loaded runner, given a longer deadline, slows the test without failing
// it, and a wait that runs out still fails the test with its own message
// before the binary's timeout panic, with a quarter left to report and clean
// up (iss-2610102020447522). A fixed budget was overrun at load averages of
// 31 to 57 on 16 cores. With no deadline (-timeout 0) it is noDeadlineBudget.
func budget(t *testing.T) time.Duration {
	t.Helper()
	dl, ok := t.Deadline()
	if !ok {
		return noDeadlineBudget
	}
	return time.Until(dl) * 3 / 4
}

// noDeadlineBudget is budget for a run given no deadline: long enough for any
// loaded runner, and a bound on a hang all the same.
const noDeadlineBudget = 10 * time.Minute

// TestEveryWaitEndsBeforeTheTestDeadline: no wait a test of this package sets
// runs past the deadline the test binary was given, so a wait that does run
// out fails the test with its own message rather than the binary's timeout
// panic.
func TestEveryWaitEndsBeforeTheTestDeadline(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		t.Skip("run with no deadline (-timeout 0): there is nothing to end before")
	}
	for name, d := range waits(t) {
		if end := time.Now().Add(d); !end.Before(dl) {
			t.Errorf("%s is %s, ending %s past the test deadline", name, d, end.Sub(dl).Round(time.Millisecond))
		}
	}
}

// TestEveryWaitGrowsWithTheTestDeadline: every wait a test of this package
// sets is at least half the time the test binary has left, so a loaded
// runner, given the longer deadline a slow run is given, slows the tests
// without failing them (iss-2610102020447522).
func TestEveryWaitGrowsWithTheTestDeadline(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		t.Skip("run with no deadline (-timeout 0): there is nothing to grow with")
	}
	for name, d := range waits(t) {
		if left := time.Until(dl); d < left/2 {
			t.Errorf("%s is %s with %s left; want at least half the time left", name, d, left.Round(time.Second))
		}
	}
}

// waits is every wait the package's tests set, by what it bounds.
func waits(t *testing.T) map[string]time.Duration {
	t.Helper()
	script := stubOnPath(t)
	stubSleep(t, script, 1)
	return map[string]time.Duration{
		"a stubbed turn's dispatch timeout": newWrittenRun(t, "").w.Timeout,
		"a guarded reading's wait":          budget(t),
		"a stubbed turn's sleep":            stubSleepFor(script, "turn-1"),
	}
}
