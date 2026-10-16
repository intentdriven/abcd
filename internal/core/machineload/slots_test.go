package machineload

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The first gate admits a waiter while fewer than slots other units of work
// are in their test stages (iss-2610161639025702): one by default, so nothing
// changes for anyone who does not raise it. The race gate, before the race
// stage, admits one race run at a time whatever slots is.

// ownPreflightWaiter is waiterChain with the zsh replaced by this waiter's own
// make preflight, so the waiter is a preflight's first gate.
func ownPreflightWaiter() ([]Proc, map[int][]string) {
	procs, args := waiterChain()
	procs[1].Name = "make"
	args[59000] = []string{"preflight"}
	args[59990] = []string{"run", "./cmd/abcd", "implement", "load", "--wait", "--timeout", "2h"}
	args[60000] = []string{"implement", "load", "--wait", "--timeout", "2h"}
	return procs, args
}

// TestShutWithOneSlotIsTheOldGate: with one slot, a running preflight shuts
// the gate on preflight and any waiter ahead or running shuts it on waiter,
// exactly as the gate read before slots existed.
func TestShutWithOneSlotIsTheOldGate(t *testing.T) {
	for _, c := range []struct {
		a                 Ahead
		preflight, waiter bool
	}{
		{Ahead{}, false, false},
		{Ahead{Preflights: []int{1}, Busy: []int{1}}, true, false},
		{Ahead{Waiters: []int{2}, Queued: []int{2}}, false, true},
		{Ahead{Waiters: []int{2}, Busy: []int{2}}, false, true},
		{Ahead{Preflights: []int{1}, Waiters: []int{2}, Busy: []int{1, 2}}, true, true},
	} {
		for _, slots := range []int{0, 1} {
			p, w := c.a.Shut(slots)
			if p != c.preflight || w != c.waiter {
				t.Errorf("%+v.Shut(%d) = %v, %v, want %v, %v", c.a, slots, p, w, c.preflight, c.waiter)
			}
		}
	}
}

// TestQueueAdmitsASecondPreflightWithTwoSlots: with two slots, a preflight's
// gate opens while one other preflight is in its test stages, and stays shut
// while two are; with one slot the first other preflight already shuts it.
func TestQueueAdmitsASecondPreflightWithTwoSlots(t *testing.T) {
	procs, args := ownPreflightWaiter()
	first, firstArgs := otherPreflight(80000)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, first...), HasProcs: true}
	args = merge(args, firstArgs)

	got := QueueAhead(snap, waiter, args)
	if !reflect.DeepEqual(got.Busy, []int{80000}) {
		t.Fatalf("busy = %v, want the other preflight 80000", got.Busy)
	}
	if p, w := got.Shut(1); !p || w {
		t.Fatalf("one slot: Shut = %v, %v, want shut on preflight", p, w)
	}
	if p, w := got.Shut(2); p || w {
		t.Fatalf("two slots, one preflight in its tests: Shut = %v, %v, want open", p, w)
	}

	// A third: two preflights are in their test stages.
	second, secondArgs := otherPreflight(81000)
	snap.Procs = append(snap.Procs, second...)
	args = merge(args, secondArgs)
	got = QueueAhead(snap, waiter, args)
	if !reflect.DeepEqual(got.Busy, []int{80000, 81000}) {
		t.Fatalf("busy = %v, want both preflights", got.Busy)
	}
	if p, _ := got.Shut(2); !p {
		t.Fatalf("two slots, two preflights in their tests: the third is admitted (%+v)", got)
	}
	if p, _ := got.Shut(3); p {
		t.Fatalf("three slots, two preflights in their tests: the third is refused (%+v)", got)
	}
}

// TestQueueKeepsItsOrderWithTwoSlots: a free slot does not let a waiter jump
// the queue: one queued ahead of it still shuts it, whatever slots is.
func TestQueueKeepsItsOrderWithTwoSlots(t *testing.T) {
	procs, args := ownPreflightWaiter()
	early, earlyArgs := peerWaiter(50001, 1, otherUID)
	early[0].Age, early[1].Age = 6*time.Minute, 5*time.Minute
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, early...), HasProcs: true}
	args = merge(args, earlyArgs)
	got := QueueAhead(snap, waiter, args)
	if !reflect.DeepEqual(got.Queued, []int{50001}) || len(got.Busy) != 0 {
		t.Fatalf("ahead = %+v, want 50001 queued and nothing busy", got)
	}
	if _, w := got.Shut(2); !w {
		t.Fatalf("two slots: a waiter queued ahead does not shut the gate (%+v)", got)
	}
}

// TestQueueCountsACommandAndItsPreflightOnce: `--wait -- make preflight` is
// one unit of work, a waiter running its command with the preflight beneath
// it, and takes one slot, not two.
func TestQueueCountsACommandAndItsPreflightOnce(t *testing.T) {
	procs, args := ownPreflightWaiter()
	outer := []Proc{
		{PID: 70001, PPID: 1, PGID: 70001, UID: otherUID, Age: 9 * time.Minute, Name: "abcd"},
		{PID: 70002, PPID: 70001, PGID: 70001, UID: otherUID, Age: 9 * time.Minute, Name: "make"},
		{PID: 70003, PPID: 70002, PGID: 70001, UID: otherUID, Age: time.Minute, Name: "go"},
	}
	args = merge(args, map[int][]string{
		70001: {"implement", "load", "--wait", "--", "make", "preflight"},
		70002: {"preflight"},
		70003: {"test", "./..."},
	})
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, outer...), HasProcs: true}
	got := QueueAhead(snap, waiter, args)
	if !reflect.DeepEqual(got.Busy, []int{70001}) {
		t.Fatalf("busy = %v, want the one unit at 70001", got.Busy)
	}
	if p, w := got.Shut(2); p || w {
		t.Fatalf("two slots, one unit in its tests: Shut = %v, %v, want open", p, w)
	}
	if p, w := got.Shut(1); !p || !w {
		t.Fatalf("one slot: Shut = %v, %v, want shut on preflight and waiter, as before", p, w)
	}
}

// raceStagePreflight is another account's make preflight at pid whose race
// gate's waiter is queued: its sh runs go run, whose abcd waits with
// --race-gate.
func raceStagePreflight(pid int, age time.Duration) ([]Proc, map[int][]string) {
	procs := []Proc{
		{PID: pid, PPID: 1, PGID: pid, UID: otherUID, Age: 9 * time.Minute, Name: "make"},
		{PID: pid + 1, PPID: pid, PGID: pid, UID: otherUID, Age: age, Name: "sh"},
		{PID: pid + 2, PPID: pid + 1, PGID: pid, UID: otherUID, Age: age, Name: "go"},
		{PID: pid + 3, PPID: pid + 2, PGID: pid, UID: otherUID, Age: age, Name: "abcd"},
	}
	args := map[int][]string{
		pid:     {"preflight"},
		pid + 2: {"run", "./cmd/abcd", "implement", "load", "--race-gate", "--timeout", "2h"},
		pid + 3: {"implement", "load", "--race-gate", "--timeout", "2h"},
	}
	return procs, args
}

// TestQueueReadsAPreflightAtItsRaceGateAsInItsTests: a preflight whose waiter
// is the race gate's has passed its first gate and is in its test stages, so
// it is busy, and its race waiter is not in the first gate's queue: at one
// slot the gate stays shut on it, as it did before the race gate existed.
func TestQueueReadsAPreflightAtItsRaceGateAsInItsTests(t *testing.T) {
	procs, args := ownPreflightWaiter()
	// This waiter started after the race waiter.
	other, otherArgs := raceStagePreflight(80000, 2*time.Minute)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, other...), HasProcs: true}
	args = merge(args, otherArgs)
	got := QueueAhead(snap, waiter, args)
	if !reflect.DeepEqual(got.Preflights, []int{80000}) || len(got.Waiters) != 0 || !reflect.DeepEqual(got.Busy, []int{80000}) {
		t.Fatalf("ahead = %+v, want the preflight 80000 busy and no first-gate waiter", got)
	}
	if p, _ := got.Shut(1); !p {
		t.Fatalf("one slot: a preflight at its race gate does not shut the first gate (%+v)", got)
	}
}

// TestIsRaceWaiter: a race waiter is an `implement load --race-gate`, the
// flag before any `--`; a first gate's waiter (--wait) is never one.
func TestIsRaceWaiter(t *testing.T) {
	for _, c := range []struct {
		args       []string
		race, wait bool
	}{
		{[]string{"implement", "load", "--race-gate"}, true, true},
		{[]string{"implement", "load", "--race-gate", "--timeout", "2h"}, true, true},
		{[]string{"implement", "load", "--race-gate=true"}, true, true},
		{[]string{"implement", "load", "--race-gate=false"}, false, false},
		{[]string{"implement", "load", "--wait"}, false, true},
		{[]string{"implement", "load", "--race"}, false, false},
		{[]string{"implement", "load", "--wait", "--", "go", "test", "--race-gate"}, false, true},
		{[]string{"implement", "load", "--", "abcd", "implement", "load", "--race-gate"}, false, false},
	} {
		if got := isRaceWaiter(c.args); got != c.race {
			t.Errorf("isRaceWaiter(%q) = %v, want %v", c.args, got, c.race)
		}
		if got := isLoadWaiter(c.args); got != c.wait {
			t.Errorf("isLoadWaiter(%q) = %v, want %v", c.args, got, c.wait)
		}
	}
}

// baseIsLoadWaiter is the waiter-recognition rule of the base commit,
// ea4865caa, copied verbatim: the rule every abcd released before the race
// gate applies to the process table. It reads any `implement load` with
// --wait as a waiter at its first gate.
func baseIsLoadWaiter(args []string) bool {
	var verb []string
	wait := false
	for _, a := range args {
		if a == "--" {
			break
		}
		if v, ok := strings.CutPrefix(a, "--wait="); ok {
			b, err := strconv.ParseBool(v)
			wait = err == nil && b
			continue
		}
		if a == "--wait" {
			wait = true
			continue
		}
		if !strings.HasPrefix(a, "-") && len(verb) < 2 {
			verb = append(verb, a)
		}
	}
	return wait && slices.Equal(verb, []string{"implement", "load"})
}

// preflightGateArgs returns the arguments abcd gets from each `go run
// ./cmd/abcd implement load` line of the Makefile's preflight recipe: the
// first gate's, before the unit stage, and the race gate's, before the race
// stage.
func preflightGateArgs(t *testing.T) (first, race []string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	const call = "go run ./cmd/abcd "
	var gates [][]string
	in := false
	for _, l := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(l, "preflight:") {
			in = true
			continue
		}
		if in && !strings.HasPrefix(l, "\t") {
			break
		}
		if l = strings.TrimSpace(l); in && strings.HasPrefix(l, call+"implement load ") {
			gates = append(gates, strings.Fields(strings.TrimPrefix(l, call)))
		}
	}
	if len(gates) != 2 {
		t.Fatalf("the preflight recipe runs %d load gates, want 2 (the first gate and the race gate): %q", len(gates), gates)
	}
	return gates[0], gates[1]
}

// TestARaceGateIsNotAFirstGateWaiterToAnOlderAbcd: an abcd released before
// the race gate reads any `implement load --wait` as a waiter at its first
// gate. Spelt with --wait, a preflight at its race gate would read to it as
// queued at its first gate, not in its test stages, so an older waiter would
// open beside it and its ungated race stage overlap this one's, even with one
// slot. The race gate is spelt without --wait, so the base commit's rule does
// not match it, and the first gate still does.
func TestARaceGateIsNotAFirstGateWaiterToAnOlderAbcd(t *testing.T) {
	first, race := preflightGateArgs(t)
	if !baseIsLoadWaiter(first) || isRaceWaiter(first) {
		t.Fatalf("first gate %q: base rule %v, race %v; want a first gate's waiter to both", first, baseIsLoadWaiter(first), isRaceWaiter(first))
	}
	if baseIsLoadWaiter(race) {
		t.Fatalf("race gate %q: the base commit's rule reads it as a first gate's waiter, so an older abcd counts a preflight at its race gate as queued", race)
	}
	if !isRaceWaiter(race) {
		t.Fatalf("race gate %q: not a race waiter to this abcd", race)
	}
	for _, a := range race {
		if a == "--" {
			break
		}
		if a == "--wait" || strings.HasPrefix(a, "--wait=") {
			t.Fatalf("race gate %q carries a --wait token", race)
		}
	}
}

// TestRaceAheadOrdersRaceWaitersOnly: at the race gate the queue is the race
// waiters alone, ordered by start as the first gate's is; a waiter at a first
// gate, or a preflight in its unit stage, is not ahead of it.
func TestRaceAheadOrdersRaceWaitersOnly(t *testing.T) {
	procs, args := waiterChain()
	args[60000] = []string{"implement", "load", "--race-gate", "--timeout", "2h"}
	procs[3].Age = time.Minute
	unit, unitArgs := otherPreflight(80000)
	first, firstArgs := peerWaiter(50001, 1, otherUID)
	first[0].Age, first[1].Age = 6*time.Minute, 5*time.Minute
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append(procs, unit...), first...), HasProcs: true}
	args = merge(merge(args, unitArgs), firstArgs)
	if got := RaceAhead(snap, waiter, args); len(got) != 0 {
		t.Fatalf("race waiters ahead = %v, want none: a first gate's waiter and a unit stage are not", got)
	}

	// A race waiter that started earlier is ahead; one that started later is not.
	early, earlyArgs := raceStagePreflight(81000, 3*time.Minute)
	late, lateArgs := raceStagePreflight(82000, 10*time.Second)
	snap.Procs = append(append(snap.Procs, early...), late...)
	args = merge(merge(args, earlyArgs), lateArgs)
	if got := RaceAhead(snap, waiter, args); !reflect.DeepEqual(got, []int{81003}) {
		t.Fatalf("race waiters ahead = %v, want the earlier one, 81003", got)
	}
	// From the earlier one's side, nothing is ahead.
	if got := RaceAhead(snap, Self{PID: 81003, UID: otherUID}, args); len(got) != 0 {
		t.Fatalf("the earliest race waiter sees %v ahead, want none", got)
	}
	// A race waiter running its command is ahead whenever it started.
	snap.Procs = append(snap.Procs, Proc{PID: 82009, PPID: 82003, PGID: 82000, UID: otherUID, Name: "go"})
	args[82009] = []string{"test", "-race", "./..."}
	if got := RaceAhead(snap, waiter, args); !reflect.DeepEqual(got, []int{81003, 82003}) {
		t.Fatalf("race waiters ahead = %v, want 81003 and the one running its command, 82003", got)
	}
}

// TestGoTestParallelismSharesTheMachineBetweenSlots: with two preflights
// admitted at once, each test stage's -p is at most half the cores, so two
// concurrent stages do not each take the whole machine; with one slot it is
// what it always was.
func TestGoTestParallelismSharesTheMachineBetweenSlots(t *testing.T) {
	for _, c := range []struct {
		load    float64
		slots   int
		want    int
		comment string
	}{
		{2, 1, 16, "one slot, an idle machine: every core"},
		{2, 2, 8, "two slots, an idle machine: half the cores each"},
		{16, 2, 8, "two slots while the other stage loads the machine: still half"},
		{28, 2, 4, "two slots on a loaded machine: the floor"},
		{2, 0, 16, "an unset slot count reads as one"},
	} {
		snap := Snapshot{Load1: c.load, HasLoad: true, Cores: 16}
		if got := GoTestParallelism(snap, c.slots); got != c.want {
			t.Errorf("%s: GoTestParallelism(load %v on 16, %d slots) = %d, want %d", c.comment, c.load, c.slots, got, c.want)
		}
	}
}
