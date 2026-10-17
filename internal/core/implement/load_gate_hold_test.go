package implement

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// Each gate judges only its own gate's hold. With two slots on one account, a
// first gate's command hold (`--wait -- make preflight`, held until its
// preflight starts its race run) shut every race waiter of the account but
// its own preflight's; that race waiter queued behind an earlier race waiter
// the hold had shut, and neither moved until its timeout.

// heldPreflights is one account's table at the moment of the stall, ages as at
// the clock's start:
//
//   - preflight A (make 30000), started by hand, at its race gate (abcd 30101);
//   - `--wait -- make preflight` (abcd 40000), holding the first gate's command
//     hold, whose preflight (make 40001) has just reached its own race gate
//     (abcd 40101), a minute after A's.
func heldPreflights() ([]machineload.Proc, map[int][]string) {
	m := time.Minute
	procs := []machineload.Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 50 * time.Hour, Name: "launchd"},
		{PID: 30000, PPID: 1, PGID: 30000, UID: 501, Age: 40 * m, Name: "make"},
		{PID: 30050, PPID: 30000, PGID: 30000, UID: 501, Age: 2 * m, Name: "sh"},
		{PID: 30100, PPID: 30050, PGID: 30000, UID: 501, Age: 2 * m, Name: "go"},
		{PID: 30101, PPID: 30100, PGID: 30000, UID: 501, Age: m, Name: "abcd"},
		{PID: 39000, PPID: 1, PGID: 39000, UID: 501, Age: time.Hour, Name: "zsh"},
		{PID: 40000, PPID: 39000, PGID: 40000, UID: 501, Age: 30 * m, Name: "abcd"},
		{PID: 40001, PPID: 40000, PGID: 40000, UID: 501, Age: 30 * m, Name: "make"},
		{PID: 40050, PPID: 40001, PGID: 40000, UID: 501, Age: 10 * time.Second, Name: "sh"},
		{PID: 40100, PPID: 40050, PGID: 40000, UID: 501, Age: 10 * time.Second, Name: "go"},
		{PID: 40101, PPID: 40100, PGID: 40000, UID: 501, Age: 5 * time.Second, Name: "abcd"},
	}
	gate := []string{"implement", "load", "--race-gate", "--timeout", "2h"}
	args := map[int][]string{
		30000: {"preflight"},
		30100: append([]string{"run", "./cmd/abcd"}, gate...),
		30101: gate,
		40000: {"implement", "load", "--wait", "--", "make", "preflight"},
		40001: {"preflight"},
		40100: append([]string{"run", "./cmd/abcd"}, gate...),
		40101: gate,
	}
	return procs, args
}

// heldReq is the race waiter at self in heldPreflights, with two slots.
func heldReq(home string, self int, clock *fakeClock) WaitRequest {
	procs, args := heldPreflights()
	return WaitRequest{
		Getenv: slotsEnv("2"),
		Read: func() (machineload.Snapshot, error) {
			aged := clock.Now().Sub(waitT0)
			ps := append([]machineload.Proc(nil), procs...)
			for j := range ps {
				ps[j].Age += aged
			}
			return machineload.Snapshot{Load1: 2, Load5: 2, Load15: 2, HasLoad: true, Cores: 16, Procs: ps, HasProcs: true}, nil
		},
		ReadArgs: func(pids []int) (map[int][]string, error) {
			out := map[int][]string{}
			for _, p := range pids {
				if a, ok := args[p]; ok {
					out[p] = a
				}
			}
			return out, nil
		},
		Home: home, Self: machineload.Self{PID: self, UID: 501},
		Now: clock.Now, Sleep: clock.Sleep, Timeout: 10 * time.Minute, Poll: 15 * time.Second,
		Jitter: func() time.Duration { return testJitter },
		Mark:   func() (func(), error) { return func() {}, nil },
		Race:   true,
	}
}

// TestWaitRaceGateIgnoresAFirstGatesHold: A's race gate is not shut by the
// first gate's command hold, so it opens, taking the race gate's own hold and
// leaving the first gate's hold as it was; B's race gate, queued behind A,
// waits for A rather than for a hold, and so moves once A's race stage does.
func TestWaitRaceGateIgnoresAFirstGatesHold(t *testing.T) {
	home := t.TempDir()
	opened := waitT0.Add(-30 * time.Minute)
	held := WaitHolder{PID: 40000, Opened: opened, Until: opened.Add(WaitCommandCap), Form: HoldCommand, StartUnix: opened.Unix()}
	writeHold(t, home, held)
	before, err := os.ReadFile(abcdhome.Path(home, WaitHoldFileName))
	if err != nil {
		t.Fatal(err)
	}

	a, err := WaitLoad(heldReq(home, 30101, &fakeClock{now: waitT0}))
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != WaitOpen || len(a.WaitedOn) != 0 || a.Hold == nil || a.Hold.Gate != GateRace || a.Hold.Form != HoldSettle {
		t.Fatalf("A = %+v (hold %+v); want A's race gate open at once beside the first gate's hold, taking a race gate's settle hold", a, a.Hold)
	}
	after, err := os.ReadFile(abcdhome.Path(home, WaitHoldFileName))
	if err != nil || string(after) != string(before) {
		t.Fatalf("the first gate's hold changed under a race gate: %v\nbefore %s\nafter  %s", err, before, after)
	}
	if h := readGateHold(t, home, GateRace); h == nil || h.PID != 30101 || h.Gate != GateRace {
		t.Fatalf("race gate's hold = %+v, want A's (pid 30101)", h)
	}

	// B, beneath the holder, waits on A alone, never on a hold.
	home = t.TempDir()
	writeHold(t, home, held)
	req := heldReq(home, 40101, &fakeClock{now: waitT0})
	req.Timeout = time.Minute
	b, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != WaitTimedOut || !reflect.DeepEqual(b.Still, []string{WaitOnWaiter}) || !reflect.DeepEqual(b.Waiters, []int{30101}) {
		t.Fatalf("B = %+v; want B still on [waiter], queued behind A (30101)", b)
	}
}

// TestWaitFirstGateIgnoresARaceGatesHold: the first gate is not shut by a
// race gate's settle hold; its own gate's hold still shuts it.
func TestWaitFirstGateIgnoresARaceGatesHold(t *testing.T) {
	home := t.TempDir()
	writeGateHold(t, home, GateRace, WaitHolder{PID: 30101, Opened: waitT0, Until: waitT0.Add(WaitSettle), Form: HoldSettle, Gate: GateRace})
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.Getenv = slotsEnv("2")
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || len(res.WaitedOn) != 0 || res.Hold == nil || res.Hold.Gate != GateTests {
		t.Fatalf("result = %+v; want the first gate open beside a race gate's hold, taking its own", res)
	}
	if h := readGateHold(t, home, GateRace); h == nil || h.PID != 30101 {
		t.Fatalf("race gate's hold = %+v, want it left in place (pid 30101)", h)
	}
}

// writeGateHold plants gate's hold record under home.
func writeGateHold(t *testing.T, home, gate string, h WaitHolder) {
	t.Helper()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(h)
	if err := os.WriteFile(filepath.Join(dir, holdFileName(gate)), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// readGateHold reads gate's hold record under home, or nil.
func readGateHold(t *testing.T, home, gate string) *WaitHolder {
	t.Helper()
	raw, err := os.ReadFile(abcdhome.Path(home, holdFileName(gate)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var h WaitHolder
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatal(err)
	}
	return &h
}
