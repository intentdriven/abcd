package implement

import (
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// The race gate's timeout counts only the time its queue is stalled
// (iss-2611232155470105): a preflight queued behind race stages that are all
// progressing does not give up at its timeout and throw away the stages it
// passed. Every wait here runs on the fake machine and clock.

// raceQueueReq is a race-gate wait above one slot, timeout long, behind the
// race run 70000, whose tree uses cores of CPU while it lives, and which exits
// after exitAfter (never when zero).
func raceQueueReq(t *testing.T, timeout time.Duration, cores float64, exitAfter time.Duration) (WaitRequest, *fakeClock) {
	t.Helper()
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}})
	read := req.Read
	req.Read = func() (machineload.Snapshot, error) {
		s, err := read()
		d := clock.Now().Sub(waitT0)
		kept := s.Procs[:0]
		for _, p := range s.Procs {
			if p.PID == 70000 {
				if exitAfter > 0 && d >= exitAfter {
					continue
				}
				p.CPU = time.Duration(cores * float64(d))
			}
			kept = append(kept, p)
		}
		s.Procs = kept
		return s, err
	}
	req.Getenv = slotsEnv("2")
	req.Race = true
	req.Timeout = timeout
	return req, clock
}

// TestWaitRaceGateWaitsPastItsTimeoutBehindABusyRaceRun: a race run ahead
// that keeps working (two and a half cores) holds the gate shut past the
// waiter's timeout, and the waiter opens when it ends instead of giving up.
func TestWaitRaceGateWaitsPastItsTimeoutBehindABusyRaceRun(t *testing.T) {
	req, _ := raceQueueReq(t, 30*time.Minute, 2.5, 50*time.Minute)
	var lines []WaitResult
	req.OnWait = func(r WaitResult) { lines = append(lines, r) }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen {
		t.Fatalf("result %+v; want the race gate open once the race run ahead ends, not a timeout", res)
	}
	if res.WaitedS < (50 * time.Minute).Seconds() {
		t.Fatalf("waited %.0fs, want past the race run's 50 minutes", res.WaitedS)
	}
	if len(lines) == 0 || !lines[len(lines)-1].Progress || lines[len(lines)-1].PausedS < (30*time.Minute).Seconds() {
		t.Fatalf("lines %+v; want the last waiting line to say the clock is paused, 30 minutes or more", lines)
	}
}

// TestWaitRaceGateTimesOutBehindAHungRaceRun: a race run ahead at no CPU is
// given one span to be measured, and then its time counts, so the waiter gives
// up a span after its timeout would have passed.
func TestWaitRaceGateTimesOutBehindAHungRaceRun(t *testing.T) {
	req, _ := raceQueueReq(t, 30*time.Minute, 0, 0)
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || res.Progress {
		t.Fatalf("result %+v; want a timeout on a stalled queue", res)
	}
	want := (30*time.Minute + QuietSpan).Seconds()
	if res.WaitedS < want-30 || res.WaitedS > want+30 {
		t.Fatalf("waited %.0fs, want about %.0fs: the timeout plus the one span the holder is not yet measured", res.WaitedS, want)
	}
	if res.PausedS < QuietSpan.Seconds()-30 || res.PausedS > QuietSpan.Seconds()+30 {
		t.Fatalf("paused %.0fs, want about one span", res.PausedS)
	}
}

// TestWaitRaceGateStopsAtItsCeiling: a race run ahead that never ends but
// keeps working pauses the clock only up to WaitRaceCeilingFactor times the
// timeout, so nothing waits for ever.
func TestWaitRaceGateStopsAtItsCeiling(t *testing.T) {
	req, _ := raceQueueReq(t, 10*time.Minute, 2.5, 0)
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	ceiling := (WaitRaceCeilingFactor * 10 * time.Minute).Seconds()
	if res.Status != WaitTimedOut || res.WaitedS < ceiling-1 || res.WaitedS > ceiling+1 {
		t.Fatalf("result %+v; want a timeout at the ceiling, %.0fs", res, ceiling)
	}
}

// TestWaitRaceGateCountsTheLoadAlone: a race gate shut by the load alone, with
// no race queue ahead, keeps a wall clock: the load limit is not progress.
func TestWaitRaceGateCountsTheLoadAlone(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 40})
	req.Getenv = slotsEnv("2")
	req.Race = true
	req.Timeout = 30 * time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || res.WaitedS != (30*time.Minute).Seconds() || res.PausedS != 0 {
		t.Fatalf("result %+v; want a timeout at 30 minutes, nothing paused", res)
	}
}

// TestWaitFirstGateKeepsAWallClock: the first gate's timeout is not paused by
// a busy race run.
func TestWaitFirstGateKeepsAWallClock(t *testing.T) {
	req, _ := raceQueueReq(t, 30*time.Minute, 2.5, 0)
	req.Race = false
	req.Getenv = noEnv
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || res.WaitedS != (30*time.Minute).Seconds() || res.PausedS != 0 || res.Progress {
		t.Fatalf("result %+v; want the first gate to time out at 30 minutes", res)
	}
}

// TestRaceQueueProgress judges one look: a race queue shutting the gate
// progresses with a holder not quiet, a queue that moved, or a settle hold;
// a quiet holder, an unmoved queue of waiters, or a gate the queue does not
// shut does not.
func TestRaceQueueProgress(t *testing.T) {
	busy := []GateHolder{{PID: 1, Measured: true, CPU: 2}}
	fresh := []GateHolder{{PID: 1}}
	quiet := []GateHolder{{PID: 1, Measured: true, Quiet: true}}
	settle := &WaitHolder{PID: 9, Form: HoldSettle}
	command := &WaitHolder{PID: 9, Form: HoldCommand}
	for _, c := range []struct {
		name        string
		shut        []string
		holders     []GateHolder
		holder      *WaitHolder
		queue, prev []int
		seen        bool
		want        bool
	}{
		{"a busy race run", []string{WaitOnRaceRun}, busy, nil, []int{1}, []int{1}, true, true},
		{"a busy race run beside the load", []string{WaitOnLoad, WaitOnRaceRun}, busy, nil, []int{1}, []int{1}, true, true},
		{"a race run not yet measured", []string{WaitOnRaceRun}, fresh, nil, []int{1}, nil, false, true},
		{"a quiet race run", []string{WaitOnRaceRun}, quiet, nil, []int{1}, []int{1}, true, false},
		{"a quiet race run on the first look", []string{WaitOnRaceRun}, quiet, nil, []int{1}, nil, false, false},
		{"a waiter ahead left", []string{WaitOnWaiter}, nil, nil, []int{2}, []int{2, 3}, true, true},
		{"waiters ahead unmoved", []string{WaitOnWaiter}, nil, nil, []int{2}, []int{2}, true, false},
		{"a settle hold", []string{WaitOnLock}, nil, settle, nil, nil, true, true},
		{"a quiet command hold", []string{WaitOnLock}, quiet, command, nil, nil, true, false},
		{"the load alone", []string{WaitOnLoad}, busy, nil, []int{1}, []int{1}, true, false},
		{"memory alone, the queue moved", []string{WaitOnMemory}, nil, nil, nil, []int{1}, true, false},
	} {
		if got := raceQueueProgress(c.shut, c.holders, c.holder, c.queue, c.prev, c.seen); got != c.want {
			t.Errorf("%s: progress %v, want %v", c.name, got, c.want)
		}
	}
}

// TestWaitRaceGateSaysItsClockRunsAtTheCeiling: once the deadline has reached
// the ceiling, a look that finds the queue still progressing pauses nothing,
// so the result says the clock runs at the ceiling, not that it is paused.
func TestWaitRaceGateSaysItsClockRunsAtTheCeiling(t *testing.T) {
	req, _ := raceQueueReq(t, 10*time.Minute, 2.5, 0)
	var lines []WaitResult
	req.OnWait = func(r WaitResult) { lines = append(lines, r) }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || res.Progress || !res.AtCeiling {
		t.Fatalf("result %+v; want a timeout at the ceiling saying the clock ran there, not that it was paused", res)
	}
	// The line said once the deadline sits at the ceiling (50 minutes in,
	// the ceiling reached at 50 minutes counted 0 of the 10) says so too.
	last := lines[len(lines)-1]
	if last.WaitedS < (50*time.Minute).Seconds() || last.Progress || !last.AtCeiling {
		t.Fatalf("last waiting line %+v; want it past the ceiling's reach saying the clock runs", last)
	}
}
