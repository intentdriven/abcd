package implement

import (
	"reflect"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// A lone race run gated with `--wait -- go test -race ./...` waits at the
// first gate, where with two slots a live race run does not shut the gate by
// itself. Its command is a race run, though, so it counts as one for both
// gates: it is shut by a live race run, others count it as a race run from the
// moment it opens, and within one account it judges and takes both gates'
// holds. Without that, a second such run saw one unit of two busy and opened,
// two race runs at their full -p on the machine at once.

// raceCommandWaiter is another account's `--wait -- go test -race` waiter at
// pid, running its command and shown so by its mark alone, its go test not yet
// in the table.
func raceCommandWaiter(pid int) poll {
	return poll{load: 3, procs: []machineload.Proc{
		{PID: pid, PPID: 1, PGID: pid, UID: 502, Age: 2 * time.Minute, Name: "abcd"},
		{PID: pid + 1, PPID: pid, PGID: pid, UID: 502, Age: time.Second, Name: "sleep"},
	}, args: map[int][]string{
		pid: {"implement", "load", "--wait", "--", "go", "test", "-race", "./..."},
	}}
}

// TestWaitRaceCommandIsShutByARaceRun: with two slots a race command's waiter
// stays shut while a race run is live, where a plain waiter opens.
func TestWaitRaceCommandIsShutByARaceRun(t *testing.T) {
	race := poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}}
	req := waitReq(t.TempDir(), 60000, &fakeClock{now: waitT0}, race)
	req.Getenv = slotsEnv("2")
	req.Command, req.RaceCommand = true, true
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnRaceRun}) || !reflect.DeepEqual(res.RaceRuns, []int{70000}) {
		t.Fatalf("result = %+v; want a race command's waiter still on [race-run] (70000) with two slots", res)
	}
}

// TestWaitCountsARaceCommandAsARaceRun: another account's race command, shown
// running by its mark before its go test starts, shuts both a race command's
// waiter at the first gate and a race gate, with two slots; a plain first
// gate's waiter counts it as one unit in its test stages and opens beside it.
func TestWaitCountsARaceCommandAsARaceRun(t *testing.T) {
	for _, c := range []struct {
		name              string
		race, raceCommand bool
	}{
		{"a race command's waiter", false, true},
		{"a race gate", true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			req := waitReq(t.TempDir(), 60000, &fakeClock{now: waitT0}, raceCommandWaiter(70001))
			req.Getenv = slotsEnv("2")
			req.Race, req.RaceCommand, req.Command = c.race, c.raceCommand, c.raceCommand
			req.Timeout = time.Minute
			res, err := WaitLoad(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnRaceRun}) || !reflect.DeepEqual(res.RaceRuns, []int{70001}) {
				t.Fatalf("result = %+v; want still on [race-run] naming the race command's waiter 70001", res)
			}
		})
	}
	req := waitReq(t.TempDir(), 60000, &fakeClock{now: waitT0}, raceCommandWaiter(70001))
	req.Getenv = slotsEnv("2")
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen {
		t.Fatalf("plain first gate = %+v, %v; want open beside one unit with two slots", res, err)
	}
}

// TestWaitRaceCommandHoldsBothGates: within one account a race command's
// waiter is shut by a race gate's hold, and the hold it takes is written to
// both gates' records, so a race gate of the account then waits on it.
func TestWaitRaceCommandHoldsBothGates(t *testing.T) {
	home := t.TempDir()
	writeGateHold(t, home, GateRace, WaitHolder{PID: 30101, Opened: waitT0, Until: waitT0.Add(WaitSettle), Form: HoldSettle, Gate: GateRace})
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.Getenv = slotsEnv("2")
	req.Command, req.RaceCommand = true, true
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLock}) || res.Holder == nil || res.Holder.PID != 30101 {
		t.Fatalf("result = %+v; want still on [lock], held by the race gate's 30101", res)
	}

	home = t.TempDir()
	req = waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.Getenv = slotsEnv("2")
	req.Command, req.RaceCommand = true, true
	res, err = WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.Hold == nil || res.Hold.Form != HoldCommand {
		t.Fatalf("result = %+v, %v; want open with a command hold", res, err)
	}
	for _, gate := range []string{GateTests, GateRace} {
		if h := readGateHold(t, home, gate); h == nil || h.PID != 60000 || h.Gate != gate {
			t.Fatalf("%s gate's hold = %+v, want the race command's (pid 60000)", gate, h)
		}
	}
	holder := poll{load: 3, procs: waiterAt(60000)[1:], args: map[int][]string{
		59999: {"run", "./cmd/abcd", "implement", "load", "--wait", "--", "go", "test", "-race", "./..."},
		60000: {"implement", "load", "--wait", "--", "go", "test", "-race", "./..."},
	}}
	race := waitReq(home, 61000, &fakeClock{now: waitT0.Add(time.Minute)}, holder)
	race.Getenv = slotsEnv("2")
	race.Race = true
	race.Timeout = time.Minute
	got, err := WaitLoad(race)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != WaitTimedOut || !reflect.DeepEqual(got.Still, []string{WaitOnLock}) {
		t.Fatalf("race gate = %+v; want it shut on lock by the race command's hold", got)
	}
	if err := ReleaseWaitHold(home, *res.Hold); err != nil {
		t.Fatal(err)
	}
	for _, gate := range []string{GateTests, GateRace} {
		if h := readGateHold(t, home, gate); h != nil {
			t.Fatalf("%s gate's hold = %+v after release, want none", gate, h)
		}
	}
}
