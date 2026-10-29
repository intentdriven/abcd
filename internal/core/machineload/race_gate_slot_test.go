package machineload

import (
	"reflect"
	"testing"
	"time"
)

// A preflight queued at its race gate runs nothing: its tests are done and its
// race stage has not started (iss-2610282016250805). Above one slot it takes
// no first-gate slot, so it no longer leaves the machine idle while it waits
// for another preflight's race run; once its race run starts it takes one
// again. With one slot the race gate opens at once, so the gate reads as it
// always did.

// raceRunPreflight is another account's make preflight at pid running its
// race stage: its race waiter has exited and its sh runs go test -race.
func raceRunPreflight(pid int) ([]Proc, map[int][]string) {
	procs, args := otherPreflight(pid)
	args[pid+2] = []string{"test", "-race", "-p", "4", "./internal/..."}
	return procs, args
}

// TestARaceGateWaiterTakesNoSlot is the queue of 2026-10-28 20:15Z: with two
// slots, preflight A queued at its race gate and preflight B in its race run
// left a third preflight's first gate shut on preflight, though only one
// stage ran. A takes no slot, so the third is admitted beside B; a second
// stage in its tests fills both slots again.
func TestARaceGateWaiterTakesNoSlot(t *testing.T) {
	procs, args := ownPreflightWaiter()
	a, aArgs := raceStagePreflight(80000, 2*time.Minute)
	b, bArgs := raceRunPreflight(81000)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append(procs, a...), b...), HasProcs: true}
	args = merge(merge(args, aArgs), bArgs)

	got := QueueAhead(snap, waiter, args)
	if !reflect.DeepEqual(got.AtRaceGate, []int{80000}) {
		t.Fatalf("at the race gate = %v, want the idle preflight 80000 (%+v)", got.AtRaceGate, got)
	}
	if p, w := got.Shut(2); p || w {
		t.Fatalf("two slots, one at its race gate and one in its race run: Shut = %v, %v, want open (%+v)", p, w, got)
	}
	if view := got.TakingSlots(2); !reflect.DeepEqual(view.Busy, []int{81000}) || !reflect.DeepEqual(view.Preflights, []int{81000}) {
		t.Fatalf("two slots: taking slots = %+v, want the race run 81000 alone", view)
	}
	if p, _ := got.Shut(1); !p {
		t.Fatalf("one slot: a preflight at its race gate no longer shuts the first gate (%+v)", got)
	}

	c, cArgs := otherPreflight(82000)
	snap.Procs = append(snap.Procs, c...)
	args = merge(args, cArgs)
	got = QueueAhead(snap, waiter, args)
	if p, _ := got.Shut(2); !p {
		t.Fatalf("two slots, a race run and a unit stage: the first gate is open (%+v)", got)
	}
	if p, w := got.Shut(3); p || w {
		t.Fatalf("three slots, a race run and a unit stage: Shut = %v, %v, want open", p, w)
	}
}

// TestARunningRaceRunStillTakesASlot: once its race run starts a preflight is
// busy again, whether its race waiter has exited and the race stage runs
// beneath make, or the race waiter runs the race run as its command.
func TestARunningRaceRunStillTakesASlot(t *testing.T) {
	procs, args := ownPreflightWaiter()
	b, bArgs := raceRunPreflight(81000)
	c, cArgs := otherPreflight(82000)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append(procs, b...), c...), HasProcs: true}
	args = merge(merge(args, bArgs), cArgs)
	got := QueueAhead(snap, waiter, args)
	if len(got.AtRaceGate) != 0 || !reflect.DeepEqual(got.Busy, []int{81000, 82000}) {
		t.Fatalf("ahead = %+v, want the race run 81000 and the unit stage 82000 busy", got)
	}
	if p, _ := got.Shut(2); !p {
		t.Fatalf("two slots: a race run beneath make takes no slot (%+v)", got)
	}

	// A race waiter running its command: its go test -race is its child.
	procs, args = ownPreflightWaiter()
	a, aArgs := raceStagePreflight(80000, 2*time.Minute)
	a = append(a, Proc{PID: 80009, PPID: 80003, PGID: 80000, UID: otherUID, Age: time.Minute, Name: "go"})
	aArgs[80002] = []string{"run", "./cmd/abcd", "implement", "load", "--race-gate", "--", "go", "test", "-race", "./..."}
	aArgs[80003] = []string{"implement", "load", "--race-gate", "--", "go", "test", "-race", "./..."}
	aArgs[80009] = []string{"test", "-race", "./..."}
	snap = Snapshot{HasLoad: true, Cores: 16, Procs: append(append(procs, a...), c...), HasProcs: true}
	args = merge(args, aArgs)
	args = merge(args, cArgs)
	got = QueueAhead(snap, waiter, args)
	if len(got.AtRaceGate) != 0 || !reflect.DeepEqual(got.Busy, []int{80000, 82000}) {
		t.Fatalf("ahead = %+v, want the race command's preflight 80000 and 82000 busy", got)
	}
	if p, _ := got.Shut(2); !p {
		t.Fatalf("two slots: a race waiter running its race run takes no slot (%+v)", got)
	}
}

// TestACommandWhosePreflightIsAtItsRaceGateTakesNoSlot: `--wait -- make
// preflight` is one unit, and while its preflight is queued at its race gate
// the whole unit is idle, the waiter running make included.
func TestACommandWhosePreflightIsAtItsRaceGateTakesNoSlot(t *testing.T) {
	procs, args := ownPreflightWaiter()
	outer := []Proc{
		{PID: 70001, PPID: 1, PGID: 70001, UID: otherUID, Age: 9 * time.Minute, Name: "abcd"},
		{PID: 70002, PPID: 70001, PGID: 70001, UID: otherUID, Age: 9 * time.Minute, Name: "make"},
		{PID: 70003, PPID: 70002, PGID: 70001, UID: otherUID, Age: time.Minute, Name: "go"},
		{PID: 70004, PPID: 70003, PGID: 70001, UID: otherUID, Age: time.Minute, Name: "abcd"},
	}
	args = merge(args, map[int][]string{
		70001: {"implement", "load", "--wait", "--", "make", "preflight"},
		70002: {"preflight"},
		70003: {"run", "./cmd/abcd", "implement", "load", "--race-gate", "--timeout", "2h"},
		70004: {"implement", "load", "--race-gate", "--timeout", "2h"},
	})
	b, bArgs := raceRunPreflight(81000)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append(procs, outer...), b...), HasProcs: true}
	args = merge(args, bArgs)
	got := QueueAhead(snap, waiter, args)
	if p, w := got.Shut(2); p || w {
		t.Fatalf("two slots, the command's preflight at its race gate: Shut = %v, %v, want open (%+v)", p, w, got)
	}
	if p, w := got.Shut(1); !p || !w {
		t.Fatalf("one slot: Shut = %v, %v, want shut on preflight and waiter, as before", p, w)
	}
}
