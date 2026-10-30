package implement

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// A preflight waits for its turn as one whole run (itd-2610262127138979): `make
// preflight` runs its whole recipe through the gate's command form, so a queued
// preflight does no lint, eval, build, vet or test work, and the waiter is the
// parent of the whole run, its turn held until the run ends. These tests read
// the gate against process tables of that shape, and the slot count read again
// at every look.

// wholeRunCommand is the command the waiter of a whole run gates, as the
// Makefile's preflight recipe spells it.
var wholeRunCommand = []string{"make", "--no-print-directory", "preflight-stages"}

// wholeRun is one `make preflight` of the whole-run shape at pid under uid,
// started age before the clock's start: the make, the go run of the gate, and
// the built abcd waiter at pid+2. Running, the waiter's child is the inner make
// of the preflight's stages, with its unit stage's go test beneath it.
func wholeRun(pid int, uid uint32, age time.Duration, running bool) ([]machineload.Proc, map[int][]string) {
	gate := append([]string{"implement", "load", "--wait", "--timeout", "12h", "--"}, wholeRunCommand...)
	procs := []machineload.Proc{
		{PID: pid, PPID: 1, PGID: pid, UID: uid, Age: age + 2*time.Second, Name: "make"},
		{PID: pid + 1, PPID: pid, PGID: pid, UID: uid, Age: age + time.Second, Name: "go"},
		{PID: pid + 2, PPID: pid + 1, PGID: pid, UID: uid, Age: age, Name: "abcd"},
	}
	args := map[int][]string{
		pid:     {"preflight"},
		pid + 1: append([]string{"run", "./cmd/abcd"}, gate...),
		pid + 2: gate,
	}
	if running {
		procs = append(procs,
			machineload.Proc{PID: pid + 3, PPID: pid + 2, PGID: pid, UID: uid, Age: time.Minute, Name: "make"},
			machineload.Proc{PID: pid + 4, PPID: pid + 3, PGID: pid, UID: uid, Age: 30 * time.Second, Name: "go"},
		)
		args[pid+3] = wholeRunCommand[1:]
		args[pid+4] = []string{"test", "-p", "8", "-timeout", "40m", "./..."}
	}
	return procs, args
}

// table is a fake machine of whole runs: launchd and every run, aged with the
// clock, and their arguments exactly (no process is guessed to be a race run).
type runTable struct {
	clock *fakeClock
	// at returns the runs as they stand at now.
	at func(now time.Time) ([]machineload.Proc, map[int][]string)
	// last is the arguments of the last read.
	last map[int][]string
}

func (tb *runTable) read() (machineload.Snapshot, error) {
	now := tb.clock.Now()
	runs, args := tb.at(now)
	procs := append([]machineload.Proc{{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 50 * time.Hour, Name: "launchd"}}, runs...)
	aged := now.Sub(waitT0)
	for i := range procs {
		procs[i].Age += aged
	}
	tb.last = args
	return machineload.Snapshot{Load1: 3, Load5: 3, Load15: 3, HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}, nil
}

func (tb *runTable) readArgs(pids []int) (map[int][]string, error) {
	out := map[int][]string{}
	for _, p := range pids {
		if a, ok := tb.last[p]; ok {
			out[p] = a
		}
	}
	return out, nil
}

// join merges runs into one table.
func joinRuns(parts ...func() ([]machineload.Proc, map[int][]string)) ([]machineload.Proc, map[int][]string) {
	var procs []machineload.Proc
	args := map[int][]string{}
	for _, p := range parts {
		ps, as := p()
		procs = append(procs, ps...)
		for k, v := range as {
			args[k] = v
		}
	}
	return procs, args
}

func wholeRunOf(pid int, uid uint32, age time.Duration, running bool) func() ([]machineload.Proc, map[int][]string) {
	return func() ([]machineload.Proc, map[int][]string) { return wholeRun(pid, uid, age, running) }
}

// wholeRunReq is the waiter of the whole run at pid (its abcd at pid+2) under
// uid, against tb, in the command form.
func wholeRunReq(t *testing.T, home string, pid int, uid uint32, clock *fakeClock, tb *runTable) WaitRequest {
	t.Helper()
	req := waitReq(home, pid+2, clock)
	req.Read, req.ReadArgs = tb.read, tb.readArgs
	req.Self = machineload.Self{PID: pid + 2, UID: uid}
	req.Command = true
	return req
}

// TestWaitThreeWholeRunsAdmitOne: three preflights started within a minute,
// each waiting as a whole run, three accounts. Only the earliest is admitted;
// the other two stay queued, doing nothing, and once the first runs they are
// shut on it as a running preflight and a waiter running its command.
func TestWaitThreeWholeRunsAdmitOne(t *testing.T) {
	type side struct {
		pid int
		uid uint32
		age time.Duration
	}
	runs := []side{{70000, 501, 50 * time.Second}, {80000, 502, 30 * time.Second}, {90000, 503, 10 * time.Second}}
	queued := func() ([]machineload.Proc, map[int][]string) {
		var parts []func() ([]machineload.Proc, map[int][]string)
		for _, r := range runs {
			parts = append(parts, wholeRunOf(r.pid, r.uid, r.age, false))
		}
		return joinRuns(parts...)
	}
	var opened []int
	for _, r := range runs {
		clock := &fakeClock{now: waitT0}
		tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) { return queued() }}
		req := wholeRunReq(t, t.TempDir(), r.pid, r.uid, clock, tb)
		req.Timeout = time.Minute
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		switch res.Status {
		case WaitOpen:
			opened = append(opened, r.pid)
		case WaitTimedOut:
			if !reflect.DeepEqual(res.Still, []string{WaitOnWaiter}) {
				t.Errorf("run %d: still on %v, want [waiter]: an earlier whole run is queued ahead", r.pid, res.Still)
			}
		}
	}
	if !reflect.DeepEqual(opened, []int{70000}) {
		t.Fatalf("admitted %v, want only the earliest whole run 70000", opened)
	}

	// The first runs its stages; the other two are shut on it, counted once.
	for _, r := range runs[1:] {
		clock := &fakeClock{now: waitT0}
		tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
			return joinRuns(wholeRunOf(70000, 501, 50*time.Second, true), wholeRunOf(80000, 502, 30*time.Second, false), wholeRunOf(90000, 503, 10*time.Second, false))
		}}
		req := wholeRunReq(t, t.TempDir(), r.pid, r.uid, clock, tb)
		req.Timeout = time.Minute
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitTimedOut || !slices.Contains(res.Still, WaitOnPreflight) || !reflect.DeepEqual(res.Preflights, []int{70000}) {
			t.Fatalf("run %d beside the running 70000: %+v, want shut on the running preflight 70000", r.pid, res)
		}
	}
}

// TestWaitRaceGateDoesNotCountItsOwnWholeRun: with two slots, a preflight's
// race gate runs beneath its own whole run's waiter, which is running its
// command. That waiter and that make preflight are its own ancestors, never
// another preflight or a waiter ahead, and a whole run queued at the first
// gate is not ahead of it either: it opens at its first look.
func TestWaitRaceGateDoesNotCountItsOwnWholeRun(t *testing.T) {
	const self = 70000
	race := []string{"implement", "load", "--race-gate", "--timeout", "2h"}
	clock := &fakeClock{now: waitT0}
	tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
		procs, args := joinRuns(wholeRunOf(self, 501, 20*time.Minute, true), wholeRunOf(80000, 502, 5*time.Minute, false))
		// The race gate's go run and abcd, beneath the inner make of the stages.
		procs = append(procs,
			machineload.Proc{PID: self + 10, PPID: self + 3, PGID: self, UID: 501, Age: 2 * time.Second, Name: "go"},
			machineload.Proc{PID: self + 11, PPID: self + 10, PGID: self, UID: 501, Age: time.Second, Name: "abcd"},
		)
		// The unit stage has finished.
		procs = slices.DeleteFunc(procs, func(p machineload.Proc) bool { return p.PID == self+4 })
		args[self+10] = append([]string{"run", "./cmd/abcd"}, race...)
		args[self+11] = race
		return procs, args
	}}
	req := waitReq(t.TempDir(), self+11, clock)
	req.Read, req.ReadArgs = tb.read, tb.readArgs
	req.Self = machineload.Self{PID: self + 11, UID: 501}
	req.Race = true
	req.Getenv = slotsEnv("2")
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || len(res.WaitedOn) != 0 || res.Gate != GateRace || res.Slots != 2 {
		t.Fatalf("race gate beneath its own whole run: %+v, want open at its first look with 2 slots", res)
	}
}

// TestWaitJudgesBySlotCountReadAtEachLook: a waiter shut because the one slot
// is taken judges by a count raised while it waits at its next look, from the
// environment or from the machine's config.json, and its report names the
// value and where it came from.
func TestWaitJudgesBySlotCountReadAtEachLook(t *testing.T) {
	busy := func(time.Time) ([]machineload.Proc, map[int][]string) {
		return joinRuns(wholeRunOf(80000, 502, 10*time.Minute, true), wholeRunOf(90000, 501, time.Minute, false))
	}
	t.Run("environment", func(t *testing.T) {
		clock := &fakeClock{now: waitT0}
		tb := &runTable{clock: clock, at: busy}
		req := wholeRunReq(t, t.TempDir(), 90000, 501, clock, tb)
		slots := "1"
		req.Getenv = func(k string) string {
			if k == PreflightSlotsEnv {
				return slots
			}
			return ""
		}
		req.Sleep = func(d time.Duration) { clock.Sleep(d); slots = "2" }
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitOpen || res.Slots != 2 || res.SlotsSource != SlotsEnv || !slices.Contains(res.WaitedOn, WaitOnPreflight) {
			t.Fatalf("%+v; want shut on the running preflight with one slot, then open with 2 from %s", res, PreflightSlotsEnv)
		}
	})
	t.Run("machine config", func(t *testing.T) {
		home := t.TempDir()
		writeMachineConfig(t, home, `{"load": {"preflights": 1}}`)
		clock := &fakeClock{now: waitT0}
		tb := &runTable{clock: clock, at: busy}
		req := wholeRunReq(t, home, 90000, 501, clock, tb)
		req.Sleep = func(d time.Duration) {
			clock.Sleep(d)
			writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
		}
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitOpen || res.Slots != 2 || res.SlotsSource != SlotsMachine || !slices.Contains(res.WaitedOn, WaitOnPreflight) {
			t.Fatalf("%+v; want shut with one slot, then open with 2 from the machine's config.json", res)
		}
	})
}

// TestWaitRefusesASlotCountMadeUnusableWhileItWaits: a count that becomes
// unusable while a waiter waits refuses at the next look, naming the file and
// the key, rather than going on with the count it read before.
func TestWaitRefusesASlotCountMadeUnusableWhileItWaits(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 1}}`)
	clock := &fakeClock{now: waitT0}
	tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
		return joinRuns(wholeRunOf(80000, 502, 10*time.Minute, true), wholeRunOf(90000, 501, time.Minute, false))
	}}
	req := wholeRunReq(t, home, 90000, 501, clock, tb)
	req.Sleep = func(d time.Duration) {
		clock.Sleep(d)
		writeMachineConfig(t, home, `{"load": {"preflights": 9}}`)
	}
	res, err := WaitLoad(req)
	if err == nil || !strings.Contains(err.Error(), "config.json") || !strings.Contains(err.Error(), PreflightSlotsKey) {
		t.Fatalf("result %+v, err %v; want a refusal naming config.json and %s at the next look", res, err, PreflightSlotsKey)
	}
	if res.Polls < 1 || strings.Contains(err.Error(), "nothing was waited for") {
		t.Fatalf("err %v after %d looks; want a refusal that says the wait stopped, not that nothing was waited for", err, res.Polls)
	}
}

// TestWaitRaceGateOpensOnceTheCountFallsToOne: the race gate's shortcut reads
// the count at the same look as the rest of the wait, so a race waiter shut
// with two slots opens at its next look once the count is one.
func TestWaitRaceGateOpensOnceTheCountFallsToOne(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}})
	req.Race = true
	slots := "2"
	req.Getenv = func(k string) string {
		if k == PreflightSlotsEnv {
			return slots
		}
		return ""
	}
	req.Sleep = func(d time.Duration) { clock.Sleep(d); slots = "1" }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || res.Slots != 1 || !reflect.DeepEqual(res.WaitedOn, []string{WaitOnRaceRun}) || res.Hold != nil {
		t.Fatalf("%+v; want shut on [race-run] with 2 slots, then open at once, without a hold, with 1", res)
	}
	// An open result names nothing that held it shut: that is WaitedOn's.
	if len(res.RaceRuns) != 0 || len(res.Waiters) != 0 || len(res.Preflights) != 0 || res.Holder != nil || len(res.Still) != 0 {
		t.Fatalf("%+v; want the open result to carry no race runs, waiters, preflights, holder or still", res)
	}
}

// TestWaitDiskFloorFollowsTheCountReadAtEachLook: the disk is judged against
// the floor the count read at that same look sizes (diskFloorFor), so a count
// raised while a waiter waits raises the floor its open look reads the disk
// at, and a race waiter whose count falls to one reads the disk before its
// shortcut opens, refusing there on a disk that filled while it waited.
func TestWaitDiskFloorFollowsTheCountReadAtEachLook(t *testing.T) {
	t.Run("first gate", func(t *testing.T) {
		var floors []uint64
		withReadDisk(t, func(_ string, floor uint64) DiskCheck {
			floors = append(floors, floor)
			return DiskCheck{Status: DiskOK, FloorBytes: floor, Volumes: []DiskVolume{}}
		})
		clock := &fakeClock{now: waitT0}
		tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
			return joinRuns(wholeRunOf(80000, 502, 10*time.Minute, true), wholeRunOf(90000, 501, time.Minute, false))
		}}
		req := wholeRunReq(t, t.TempDir(), 90000, 501, clock, tb)
		slots := "1"
		req.Getenv = func(k string) string {
			if k == PreflightSlotsEnv {
				return slots
			}
			return ""
		}
		req.Sleep = func(d time.Duration) { clock.Sleep(d); slots = "4" }
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitOpen || res.Slots != 4 {
			t.Fatalf("%+v; want shut with one slot, then open with 4", res)
		}
		if want := []uint64{diskFloorFor(1), diskFloorFor(4)}; !slices.Equal(floors, want) {
			t.Fatalf("the disk was judged against %v; want %v, the first look's floor and then the open look's", floors, want)
		}
	})
	t.Run("race shortcut", func(t *testing.T) {
		for _, low := range []bool{false, true} {
			home := t.TempDir()
			var floors []uint64
			withReadDisk(t, func(h string, floor uint64) DiskCheck {
				floors = append(floors, floor)
				free := 200 * gib
				if low && len(floors) > 1 {
					free = 3 * gib
				}
				p, _ := fakeDisk(h, 1, dataVolume(h, free))
				p.floor = floor
				return checkDisk(h, p)
			})
			clock := &fakeClock{now: waitT0}
			req := waitReq(home, 60000, clock, poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}})
			req.Race = true
			slots := "4"
			req.Getenv = func(k string) string {
				if k == PreflightSlotsEnv {
					return slots
				}
				return ""
			}
			req.Sleep = func(d time.Duration) { clock.Sleep(d); slots = "1" }
			res, err := WaitLoad(req)
			if want := []uint64{diskFloorFor(4), diskFloorFor(1)}; !slices.Equal(floors, want) {
				t.Fatalf("low=%v: the disk was judged against %v; want %v, the first look's floor and then the shortcut's", low, floors, want)
			}
			if !low {
				if err != nil || res.Status != WaitOpen || res.Slots != 1 {
					t.Fatalf("%+v, %v; want the race gate open once the count is one", res, err)
				}
				continue
			}
			if err == nil || !strings.Contains(err.Error(), "fell below the floor while this waited") || res.Status == WaitOpen {
				t.Fatalf("%+v, %v; want the shortcut refused on the disk that filled while it waited", res, err)
			}
		}
	})
}

// TestWaitWholeRunWithTwoSlotsHoldsOnlyToSettle: above one slot a whole run's
// waiter takes the settle hold, as the plain first gate inside a preflight
// did, not the command hold, which would keep the account's second slot shut
// until the first run reached its race stage. Once the hold has settled, a
// second whole run of the same account opens beside the first.
func TestWaitWholeRunWithTwoSlotsHoldsOnlyToSettle(t *testing.T) {
	home := t.TempDir()
	clock := &fakeClock{now: waitT0}
	tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
		return wholeRun(70000, 501, time.Minute, false)
	}}
	req := wholeRunReq(t, home, 70000, 501, clock, tb)
	req.Getenv = slotsEnv("2")
	first, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != WaitOpen || first.Hold == nil || first.Hold.Form != HoldSettle {
		t.Fatalf("first whole run: %+v (hold %+v); want open with the settle hold above one slot", first, first.Hold)
	}

	// The first opened after its jitter, so its hold lapses that much later.
	clock = &fakeClock{now: waitT0.Add(testJitter + WaitSettle + time.Second)}
	tb = &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
		return joinRuns(wholeRunOf(70000, 501, time.Minute, true), wholeRunOf(80000, 501, 0, false))
	}}
	req = wholeRunReq(t, home, 80000, 501, clock, tb)
	req.Getenv = slotsEnv("2")
	second, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != WaitOpen || len(second.WaitedOn) != 0 {
		t.Fatalf("second whole run of the account: %+v; want open beside the first once its hold has settled", second)
	}

	// With one slot the whole run keeps the command hold.
	clock = &fakeClock{now: waitT0}
	tb = &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
		return wholeRun(70000, 501, time.Minute, false)
	}}
	one, err := WaitLoad(wholeRunReq(t, t.TempDir(), 70000, 501, clock, tb))
	if err != nil || one.Status != WaitOpen || one.Hold == nil || one.Hold.Form != HoldCommand {
		t.Fatalf("one slot: %+v, %v; want open with the command hold", one, err)
	}
}

// TestWaitTwoWholeRunsAtOneMomentAcrossAccounts: two accounts' whole runs
// that start in the same second reach the gate together. Their starts tie,
// so the lower pid goes first: it opens after its jitter and open look, and
// the other, standing back if it opened in the same moment, opens once the
// first run ends. Neither waits on the other until its timeout.
func TestWaitTwoWholeRunsAtOneMomentAcrossAccounts(t *testing.T) {
	const (
		low, high = 70000, 80000
		age       = 10 * time.Second
		runFor    = 40 * time.Minute
		timeout   = 12 * time.Hour
	)
	openAt := waitT0.Add(testJitter)
	cmdAt := openAt.Add(WaitOpenPause)

	// The lower pid, beside the higher one queued behind it, then marking
	// itself in the same moment as this one.
	clock := &fakeClock{now: waitT0}
	tb := &runTable{clock: clock, at: func(now time.Time) ([]machineload.Proc, map[int][]string) {
		procs, args := joinRuns(wholeRunOf(low, 501, age, false), wholeRunOf(high, 502, age, false))
		// Its mark, started just after this one's re-check: ages here are as
		// at the clock's start, and the table ages every process with the
		// clock.
		if now.After(openAt) {
			procs = append(procs, machineload.Proc{PID: high + 5, PPID: high + 2, PGID: high, UID: 502, Age: -openAt.Sub(waitT0), Name: "sleep"})
		}
		return procs, args
	}}
	req := wholeRunReq(t, t.TempDir(), low, 501, clock, tb)
	req.Timeout = timeout
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || res.Backoffs != 0 || clock.Now().After(waitT0.Add(time.Minute+WaitOpenPause)) {
		t.Fatalf("lower pid: %+v at %s; want open within a minute, never standing back", res, clock.Now())
	}

	// The higher pid: the lower one queued ahead in the tie, then opened in
	// the same moment as this one's open look, running its whole run for
	// runFor, then gone.
	clock = &fakeClock{now: waitT0}
	lowGone := cmdAt.Add(runFor)
	tb = &runTable{clock: clock, at: func(now time.Time) ([]machineload.Proc, map[int][]string) {
		if !now.Before(lowGone) {
			return wholeRunOf(high, 502, age, false)()
		}
		running := !now.Before(cmdAt)
		procs, args := joinRuns(wholeRunOf(low, 501, age, running), wholeRunOf(high, 502, age, false))
		if now.After(openAt) && !running {
			procs = append(procs, machineload.Proc{PID: low + 5, PPID: low + 2, PGID: low, UID: 501, Age: -openAt.Sub(waitT0), Name: "sleep"})
		}
		return procs, args
	}}
	req = wholeRunReq(t, t.TempDir(), high, 502, clock, tb)
	req.Timeout = timeout
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || clock.Now().Before(lowGone) || !clock.Now().Before(waitT0.Add(timeout)) {
		t.Fatalf("higher pid: %+v at %s; want open once the lower pid's run ends at %s, long before its timeout", res, clock.Now(), lowGone)
	}
}

// TestWaitIgnoresARepositorysSlotCount: a waiter in a checkout whose own
// .abcd/config.json names a number of runs for the gate judges by the
// machine's count, one by default, never the repository's.
func TestWaitIgnoresARepositorysSlotCount(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(`{"load": {"preflights": 2}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	clock := &fakeClock{now: waitT0}
	tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
		return joinRuns(wholeRunOf(80000, 502, 10*time.Minute, true), wholeRunOf(90000, 501, time.Minute, false))
	}}
	req := wholeRunReq(t, t.TempDir(), 90000, 501, clock, tb)
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || res.Slots != 1 || res.SlotsSource != SlotsDefault || !slices.Contains(res.Still, WaitOnPreflight) {
		t.Fatalf("%+v; want shut on the running preflight with one slot from the default, the repository's 2 ignored", res)
	}
}

// TestWaitRaisedCountReachesARunQueuedBehindACommandHold: a whole run admitted
// at one slot takes the command hold, kept until its race run starts. A run of
// the same account queued behind it, once the count it reads is above one and
// its own command is not a race run, judges that hold as the settle hold
// (dcn-2610300618371321), so a raised count reaches it as iss-2610241414124940's
// resolution says. The running preflights are still counted against the
// slots, and a race command's waiter still waits on the hold.
func TestWaitRaisedCountReachesARunQueuedBehindACommandHold(t *testing.T) {
	// heldHome is a home where the first run, admitted at one slot, holds the
	// first gate with the command hold.
	heldHome := func(t *testing.T) string {
		home := t.TempDir()
		clock := &fakeClock{now: waitT0}
		tb := &runTable{clock: clock, at: func(time.Time) ([]machineload.Proc, map[int][]string) {
			return wholeRun(70000, 501, time.Minute, false)
		}}
		a, err := WaitLoad(wholeRunReq(t, home, 70000, 501, clock, tb))
		if err != nil || a.Status != WaitOpen || a.Hold == nil || a.Hold.Form != HoldCommand {
			t.Fatalf("first run at one slot: %+v, %v; want open with the command hold", a, err)
		}
		return home
	}
	// Past the settle time, long before the command hold's cap: the first run
	// is alive in its test stages, its race run not started.
	later := waitT0.Add(10 * time.Minute)
	queuedB := func(more ...func() ([]machineload.Proc, map[int][]string)) *runTable {
		c := &fakeClock{now: later}
		parts := append([]func() ([]machineload.Proc, map[int][]string){wholeRunOf(70000, 501, time.Minute, true), wholeRunOf(80000, 501, 0, false)}, more...)
		return &runTable{clock: c, at: func(time.Time) ([]machineload.Proc, map[int][]string) { return joinRuns(parts...) }}
	}

	t.Run("raised count reaches the queued run", func(t *testing.T) {
		tb := queuedB()
		req := wholeRunReq(t, heldHome(t), 80000, 501, tb.clock, tb)
		req.Timeout = 30 * time.Minute
		slots := "1"
		req.Getenv = func(k string) string {
			if k == PreflightSlotsEnv {
				return slots
			}
			return ""
		}
		req.Sleep = func(d time.Duration) { tb.clock.Sleep(d); slots = "2" }
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitOpen || res.Slots != 2 || !slices.Contains(res.WaitedOn, WaitOnPreflight) {
			t.Fatalf("%+v (holder %+v); want shut on the running preflight at one slot, then open at 2 despite its command hold", res, res.Holder)
		}
	})

	t.Run("still shut while two preflights run", func(t *testing.T) {
		tb := queuedB(wholeRunOf(75000, 502, 2*time.Minute, true))
		req := wholeRunReq(t, heldHome(t), 80000, 501, tb.clock, tb)
		req.Timeout = 5 * time.Minute
		req.Getenv = slotsEnv("2")
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitTimedOut || !slices.Contains(res.Still, WaitOnPreflight) || slices.Contains(res.Still, WaitOnLock) {
			t.Fatalf("%+v; want shut on the two running preflights at 2 slots, not on the settled hold", res)
		}
	})

	t.Run("a race command still waits on the hold", func(t *testing.T) {
		tb := queuedB()
		req := wholeRunReq(t, heldHome(t), 80000, 501, tb.clock, tb)
		req.Timeout = 5 * time.Minute
		req.RaceCommand = true
		req.Getenv = slotsEnv("2")
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitTimedOut || !slices.Contains(res.Still, WaitOnLock) {
			t.Fatalf("%+v; want a race command shut on the first run's command hold", res)
		}
	})
}
