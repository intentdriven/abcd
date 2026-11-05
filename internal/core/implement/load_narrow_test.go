package implement

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// narrowCmd is the words of a narrow waiter's command.
var narrowCmd = []string{"implement", "load", "--wait", "--narrow", "--timeout", "6h", "--", "go", "test", "-count=1", "./internal/core/spec"}

// narrowReq is a narrow waiter's request at pid: its own words read as a narrow
// run's, and its jitter the fixtures' own.
func narrowReq(home string, pid int, clock *fakeClock, polls ...poll) WaitRequest {
	req := waitReq(home, pid, clock, polls...)
	req.Command, req.Narrow = true, true
	return req
}

// narrowRun is another account's narrow waiter at pid; running gives it its go
// test child.
func narrowRun(p *poll, pid int, age time.Duration, running bool) {
	if p.args == nil {
		p.args = map[int][]string{}
	}
	p.procs = append(p.procs, machineload.Proc{PID: pid, PPID: 1, PGID: pid, UID: 502, Age: age, Name: "abcd"})
	p.args[pid] = narrowCmd
	if running {
		p.procs = append(p.procs, machineload.Proc{PID: pid + 1, PPID: pid, PGID: pid, UID: 502, Age: age, Name: "go"})
		p.args[pid+1] = []string{"test", "-count=1", "./internal/core/spec"}
	}
}

// ownNarrowRun is this account's narrow run at pid, running.
func ownNarrowRun(p *poll, pid int) {
	narrowRun(p, pid, time.Minute, true)
	for i := range p.procs {
		if p.procs[i].PID == pid || p.procs[i].PID == pid+1 {
			p.procs[i].UID = 501
		}
	}
}

// wholeWaiter is another account's whole-run waiter at pid, queued for age.
func wholeWaiter(p *poll, pid int, age time.Duration) {
	if p.args == nil {
		p.args = map[int][]string{}
	}
	p.procs = append(p.procs, machineload.Proc{PID: pid, PPID: 1, PGID: pid, UID: 502, Age: age, Name: "abcd"})
	p.args[pid] = []string{"implement", "load", "--wait", "--timeout", "12h", "--", "make", "preflight-stages"}
}

// runningPreflight is another account's make preflight at pid, in its tests.
func runningPreflight(p *poll, pid int) {
	if p.args == nil {
		p.args = map[int][]string{}
	}
	p.procs = append(p.procs,
		machineload.Proc{PID: pid, PPID: 1, PGID: pid, UID: 502, Age: 10 * time.Minute, Name: "make"},
		machineload.Proc{PID: pid + 1, PPID: pid, PGID: pid, UID: 502, Age: 9 * time.Minute, Name: "go"})
	p.args[pid] = []string{"preflight"}
	p.args[pid+1] = []string{"test", "./..."}
}

// TestANarrowRunOpensPastWholeRuns is the morning of 2026-11-04: every whole
// slot taken, whole runs queued ahead of it for an hour and a race run live, at a
// load well below the limit. A narrow run has no business with any of them.
func TestANarrowRunOpensPastWholeRuns(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
	clock := &fakeClock{now: waitT0}
	p := poll{load: 12}
	runningPreflight(&p, 80000)
	runningPreflight(&p, 81000)
	wholeWaiter(&p, 50001, time.Hour)
	wholeWaiter(&p, 50002, 50*time.Minute)
	p.procs = append(p.procs, raceRun(70000, 1))
	marked := false
	req := narrowReq(home, 60000, clock, p)
	req.Mark = func() (func(), error) { marked = true; return func() {}, nil }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || res.Gate != GateNarrow || len(res.WaitedOn) != 0 {
		t.Fatalf("result = %+v, want it open at the narrow gate, having waited on nothing", res)
	}
	if want := waitT0.Add(testJitter); !clock.Now().Equal(want) {
		t.Fatalf("opened at %v, want after its one jitter, %v", clock.Now(), want)
	}
	if marked {
		t.Error("a narrow run took the open look: only a whole run stands back from another")
	}
	if res.Hold == nil || res.Hold.Form != HoldSettle || res.Hold.Until.Sub(res.Hold.Opened) != WaitNarrowSettle {
		t.Fatalf("hold = %+v, want the short settle hold", res.Hold)
	}
	if _, err := os.Stat(abcdhome.Path(home, WaitNarrowHoldFileName)); err != nil {
		t.Errorf("no narrow hold: %v", err)
	}
	if _, err := os.Stat(abcdhome.Path(home, WaitHoldFileName)); !os.IsNotExist(err) {
		t.Errorf("a narrow run touched the first gate's hold: %v", err)
	}
}

// TestANarrowRunRespectsTheLoadLimit: the load limit still shuts it, and
// says so.
func TestANarrowRunRespectsTheLoadLimit(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(narrowReq(t.TempDir(), 60000, clock, poll{load: 40}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnLoad}) {
		t.Fatalf("result = %+v, want timed out on load", res)
	}
}

// TestANarrowRunRespectsTheMemoryFloor: and so does memory.
func TestANarrowRunRespectsTheMemoryFloor(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	req := narrowReq(t.TempDir(), 60000, clock, poll{load: 3})
	req.ReadMemory = func() (machineload.Memory, error) {
		return machineload.Memory{Platform: machineload.MemoryDarwin, Level: machineload.MemoryWarn, Raw: "2"}, nil
	}
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnMemory}) {
		t.Fatalf("result = %+v, want timed out on memory", res)
	}
}

// TestNarrowRunsHaveTheirOwnLimit: running narrow runs, of any account, count
// against load.narrow_runs, and those queued ahead with them, so a free place
// goes to the earliest; one free place and nothing ahead opens.
func TestNarrowRunsHaveTheirOwnLimit(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 1, "narrow_runs": 2}}`)
	full, oneAndAhead, one := poll{load: 3}, poll{load: 3}, poll{load: 3}
	ownNarrowRun(&full, 51000)
	narrowRun(&full, 52000, time.Minute, true)
	ownNarrowRun(&oneAndAhead, 51000)
	narrowRun(&oneAndAhead, 53000, time.Hour, false)
	narrowRun(&one, 51000, time.Minute, true)

	for name, p := range map[string]poll{"two running": full, "one running and one queued ahead": oneAndAhead} {
		clock := &fakeClock{now: waitT0}
		req := narrowReq(home, 60000, clock, p)
		req.Timeout = time.Minute
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnWaiter}) || res.NarrowRuns != 2 {
			t.Errorf("%s: result = %+v, want shut on its own limit of 2", name, res)
		}
	}
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(narrowReq(home, 60000, clock, one))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen {
		t.Fatalf("one running of two: result = %+v, want open", res)
	}
}

// TestANarrowRunTakesTheShortHold: within an account, a narrow waiter's short
// hold keeps the next one back until the first shows in the table, and shuts
// nothing else.
func TestANarrowRunTakesTheShortHold(t *testing.T) {
	home := t.TempDir()
	clock := &fakeClock{now: waitT0}
	if res, err := WaitLoad(narrowReq(home, 60000, clock, poll{load: 3})); err != nil || res.Status != WaitOpen {
		t.Fatalf("first: %+v, %v", res, err)
	}
	second := narrowReq(home, 61000, clock, poll{load: 3})
	second.Timeout = 0
	if res, err := WaitLoad(second); err != nil || !slices.Contains(res.WaitedOn, WaitOnLock) {
		t.Fatalf("second: %+v, %v; want shut on the hold", res, err)
	}
	// The hold does not shut a whole run, and a whole run's does not shut it.
	whole := waitReq(home, 62000, clock, poll{load: 3})
	if res, err := WaitLoad(whole); err != nil || res.Status != WaitOpen || slices.Contains(res.WaitedOn, WaitOnLock) {
		t.Fatalf("whole beside the narrow hold: %+v, %v", res, err)
	}
	if WaitNarrowSettle != 30*time.Second {
		t.Errorf("a narrow run's hold is %s, too short for the load to show the one before", WaitNarrowSettle)
	}
}

// TestANarrowRunJudgesAWholeRunsSettle: what a whole run just admitted has not yet
// shown in the load or in memory, so an account's narrow runs wait out its settle,
// at either gate. A command hold is judged as that settle, so a whole run's length
// shuts no narrow run, and the record is left for the whole-run gate to judge.
func TestANarrowRunJudgesAWholeRunsSettle(t *testing.T) {
	for name, h := range map[string]struct {
		file string
		rec  WaitHolder
	}{
		"a command hold": {WaitHoldFileName, WaitHolder{PID: 70000, StartUnix: waitT0.Unix(), Opened: waitT0, Until: waitT0.Add(WaitCommandCap), Form: HoldCommand}},
		"a settle hold":  {WaitHoldFileName, WaitHolder{PID: 70000, Opened: waitT0, Until: waitT0.Add(WaitSettle), Form: HoldSettle}},
		"the race gate":  {WaitRaceHoldFileName, WaitHolder{PID: 70000, Opened: waitT0, Until: waitT0.Add(WaitSettle), Form: HoldSettle}},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			writeHold(t, home, h.rec)
			if h.file != WaitHoldFileName {
				if err := os.Rename(abcdhome.Path(home, WaitHoldFileName), abcdhome.Path(home, h.file)); err != nil {
					t.Fatal(err)
				}
			}
			p := poll{load: 3}
			p.procs = append(p.procs, machineload.Proc{PID: 70000, PPID: 1, PGID: 70000, UID: 501, Age: time.Second, Name: "abcd"})
			p.args = map[int][]string{70000: {"implement", "load", "--wait", "--", "make", "preflight"}}
			clock := &fakeClock{now: waitT0}
			req := narrowReq(home, 60000, clock, p)
			req.Timeout = time.Hour
			res, err := WaitLoad(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != WaitOpen || !slices.Contains(res.WaitedOn, WaitOnLock) || clock.Now().Before(waitT0.Add(WaitSettle)) {
				t.Fatalf("result = %+v at %s, want open once the settle has passed", res, clock.Now().Sub(waitT0))
			}
			if clock.Now().After(waitT0.Add(WaitSettle + time.Minute)) {
				t.Fatalf("still shut at %s: a whole run's length shut it", clock.Now().Sub(waitT0))
			}
			if _, err := os.Stat(abcdhome.Path(home, h.file)); err != nil {
				t.Errorf("the hold went: %v", err)
			}
		})
	}
}

// TestNarrowRoom: two classes share one budget. Narrow runs never pass half the
// cores in all, and on a machine the load shows busy they get only what the whole
// runs in their slots leave, and one place at least, so a test is never behind them.
func TestNarrowRoom(t *testing.T) {
	for _, c := range []struct {
		limit, cores int
		load         float64
		busy, slots  int
		want         int
	}{
		{4, 16, 20, 0, 4, 4},
		{4, 16, 20, 1, 4, 4},
		{4, 16, 20, 2, 4, 4},
		{4, 16, 20, 3, 4, 2},
		{4, 16, 20, 4, 4, 1},
		{4, 16, 20, 1, 2, 4},
		{4, 16, 20, 2, 2, 1},
		{4, 16, 20, 1, 1, 1},
		{4, 16, 14, 4, 4, 1},
		// The load shows the room the whole runs in their slots are not using.
		{4, 16, 13.9, 4, 4, 4},
		{4, 16, 12, 2, 2, 4},
		// A setting is the top of the range, not a floor under it.
		{8, 16, 20, 2, 4, 4},
		{2, 16, 3, 0, 1, 2},
		{1, 2, 9, 1, 1, 1},
	} {
		if got := narrowRoom(c.limit, c.cores, c.load, c.busy, c.slots); got != c.want {
			t.Errorf("narrowRoom(%d, %d cores, load %v, %d busy of %d) = %d, want %d", c.limit, c.cores, c.load, c.busy, c.slots, got, c.want)
		}
	}
}

// TestANarrowRunShrinksBesideWholeRuns: the budget reaches a look: with every whole
// slot taken and the load showing it, one narrow run running shuts the next.
func TestANarrowRunShrinksBesideWholeRuns(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
	p := poll{load: 20}
	runningPreflight(&p, 80000)
	runningPreflight(&p, 81000)
	ownNarrowRun(&p, 51000)
	clock := &fakeClock{now: waitT0}
	req := narrowReq(home, 60000, clock, p)
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnWaiter}) || res.NarrowRuns != 1 {
		t.Fatalf("result = %+v, want shut on a room of one", res)
	}
}

// TestAnotherAccountCannotTakeTheLastPlace: this account always keeps one place, so
// no number of another account's narrow runs shuts its gate by the count alone.
func TestAnotherAccountCannotTakeTheLastPlace(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"narrow_runs": 2}}`)
	p := poll{load: 3}
	for i := range 5 {
		narrowRun(&p, 51000+i*10, time.Minute, true)
	}
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(narrowReq(home, 60000, clock, p))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen {
		t.Fatalf("five of another account's runs shut it: %+v", res)
	}
	ownNarrowRun(&p, 52000)
	clock = &fakeClock{now: waitT0}
	req := narrowReq(t.TempDir(), 60000, clock, p)
	writeMachineConfig(t, req.Home, `{"load": {"narrow_runs": 2}}`)
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnWaiter}) {
		t.Fatalf("its own place taken: %+v, want shut", res)
	}
}

// TestANarrowRunStopsCountingAStuckQueue: a narrow waiter queued ahead can be shut by
// its own account's stricter settings, which no other account can see. So waiters
// queued ahead that alone hold a narrow run shut
// stop counting after WaitNarrowQueueCap, flat whatever their number: a stuck queue
// costs minutes, never the wait.
func TestANarrowRunStopsCountingAStuckQueue(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"narrow_runs": 2}}`)
	p := poll{load: 3}
	for i := range 5 {
		narrowRun(&p, 51000+i*10, time.Hour, false)
	}
	clock := &fakeClock{now: waitT0}
	req := narrowReq(home, 60000, clock, p)
	req.Timeout = time.Hour
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || !slices.Equal(res.WaitedOn, []string{WaitOnWaiter}) {
		t.Fatalf("result = %+v, want open past the stuck queue", res)
	}
	if got := clock.Now().Sub(waitT0); got < WaitNarrowQueueCap || got > WaitNarrowQueueCap+2*testJitter {
		t.Fatalf("opened after %s, want the flat cap of %s", got, WaitNarrowQueueCap)
	}
}

// TestANarrowRunStandsAsideForAnOverdueWholeRun is the honesty of a whole run's
// place: a whole waiter queued ahead for WaitNarrowYieldAfter with a slot free
// is held back by something it cannot get past, so narrow runs stop being
// admitted until it opens, and for at most WaitNarrowYieldCap: a whole waiter
// shut for a reason of its own account's costs each narrow run minutes at most.
func TestANarrowRunStandsAsideForAnOverdueWholeRun(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
	p := poll{load: 30}
	wholeWaiter(&p, 50001, WaitNarrowYieldAfter+time.Minute)
	runningPreflight(&p, 80000)
	clock := &fakeClock{now: waitT0}
	req := narrowReq(home, 60000, clock, p)
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnWholeRun}) || !slices.Equal(res.Overdue, []int{50001}) {
		t.Fatalf("result = %+v, want shut on the overdue whole run", res)
	}

	// Once it opens, narrow runs go on.
	opened := poll{load: 30}
	runningPreflight(&opened, 80000)
	runningPreflight(&opened, 81000)
	clock = &fakeClock{now: waitT0}
	req = narrowReq(home, 60000, clock, p, p, opened)
	req.Timeout = time.Hour
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || !slices.Equal(res.WaitedOn, []string{WaitOnWholeRun}) || clock.Now().Sub(waitT0) > 2*time.Minute {
		t.Fatalf("result = %+v at %s, want open soon after the whole run took its slot", res, clock.Now().Sub(waitT0))
	}

	// One that never opens holds a narrow run back for the cap and no longer.
	clock = &fakeClock{now: waitT0}
	req = narrowReq(home, 60000, clock, p)
	req.Timeout = time.Hour
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := clock.Now().Sub(waitT0); res.Status != WaitOpen || got < WaitNarrowYieldCap || got > WaitNarrowYieldCap+2*testJitter {
		t.Fatalf("result = %+v after %s, want open past the cap of %s", res, got, WaitNarrowYieldCap)
	}
}

// TestANarrowRunDoesNotStandAside: not for a whole waiter whose slots are all
// taken (the whole point), not for one still young, and never for one that
// started after it: an older abcd counts a narrow waiter started before it as
// ahead, so standing aside for a later start would have each wait on the other
// until a timeout.
func TestANarrowRunDoesNotStandAside(t *testing.T) {
	full := poll{load: 3}
	wholeWaiter(&full, 50001, time.Hour)
	runningPreflight(&full, 80000)
	runningPreflight(&full, 81000)
	young := poll{load: 3}
	wholeWaiter(&young, 50001, WaitNarrowYieldAfter-time.Minute)
	later := poll{load: 3}
	wholeWaiter(&later, 61000, 0)
	for name, c := range map[string]struct {
		p   poll
		age time.Duration
	}{
		"all whole slots taken": {full, time.Second},
		"a young whole waiter":  {young, time.Second},
		"a later start":         {later, time.Hour},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
			clock := &fakeClock{now: waitT0}
			req := narrowReq(home, 60000, clock, c.p)
			read := req.Read
			req.Read = func() (machineload.Snapshot, error) {
				s, err := read()
				for i := range s.Procs {
					if s.Procs[i].PID == 60000 {
						s.Procs[i].Age = c.age
					}
				}
				return s, err
			}
			res, err := WaitLoad(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != WaitOpen || len(res.WaitedOn) != 0 {
				t.Fatalf("result = %+v, want open having waited on nothing", res)
			}
		})
	}
}

// TestAWholeRunCountsNarrowRunsThroughTheLoadOnly: neither slot is taken by a
// narrow run, and none queued ahead shuts it; the load they add does.
func TestAWholeRunCountsNarrowRunsThroughTheLoadOnly(t *testing.T) {
	home := t.TempDir()
	p := poll{load: 3}
	narrowRun(&p, 51000, time.Minute, true)
	narrowRun(&p, 52000, time.Minute, true)
	narrowRun(&p, 53000, time.Hour, false)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, p)
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 || res.Gate != GateTests {
		t.Fatalf("result = %+v, %v; want open past the narrow runs", res, err)
	}
	loaded := p
	loaded.load = 40
	clock = &fakeClock{now: waitT0}
	res, err = WaitLoad(waitReq(t.TempDir(), 60000, clock, loaded))
	if err != nil || res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLoad}) {
		t.Fatalf("result = %+v, %v; want shut on the load they add", res, err)
	}
}

// TestDefaultNarrowRuns: half the online cores, divided by what one takes, and
// one at the least.
func TestDefaultNarrowRuns(t *testing.T) {
	for cores, want := range map[int]int{16: 16 / 2 / NarrowRunCores, 8: 8 / 2 / NarrowRunCores, 2: 1, 1: 1} {
		if got := DefaultNarrowRuns(cores); got != max(want, 1) {
			t.Errorf("DefaultNarrowRuns(%d) = %d, want %d", cores, got, max(want, 1))
		}
	}
	if DefaultNarrowRuns(16) < 2 {
		t.Error("a 16-core machine admits fewer than two narrow runs")
	}
}

// TestNarrowRunsSetting: load.narrow_runs in the person's own config.json sets
// the limit; a value outside its range refuses, naming the key; no repository
// file and no environment variable is read for it; a config.json it cannot use
// is read past, as for load.preflights.
func TestNarrowRunsSetting(t *testing.T) {
	home := t.TempDir()
	if n, src, err := NarrowRuns(home); err != nil || n != 0 || src != SlotsDefault {
		t.Fatalf("unset: %d, %q, %v", n, src, err)
	}
	writeMachineConfig(t, home, `{"load": {"narrow_runs": 7}}`)
	if n, src, err := NarrowRuns(home); err != nil || n != 7 || src != SlotsMachine {
		t.Fatalf("set: %d, %q, %v", n, src, err)
	}
	for _, body := range []string{`{"load": {"narrow_runs": 0}}`, `{"load": {"narrow_runs": 33}}`, `{"load": {"narrow_runs": "4"}}`, `{"load": {"narrow_runs": 1.5}}`} {
		h := t.TempDir()
		writeMachineConfig(t, h, body)
		if _, _, err := NarrowRuns(h); err == nil || !strings.Contains(err.Error(), NarrowRunsKey) {
			t.Errorf("%s: err = %v, want a refusal naming the key", body, err)
		}
	}
	h := t.TempDir()
	writeMachineConfig(t, h, `{"load": `)
	if n, _, err := NarrowRuns(h); err != nil || n != 0 {
		t.Errorf("a config it cannot read: %d, %v; want the default", n, err)
	}

	// A repository's key is read by nothing: the wait reads its own home alone.
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(`{"load": {"narrow_runs": 1}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p := poll{load: 3}
	ownNarrowRun(&p, 51000)
	clock := &fakeClock{now: waitT0}
	req := narrowReq(t.TempDir(), 60000, clock, p)
	req.Dir = repo
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.NarrowRuns != DefaultNarrowRuns(16) {
		t.Fatalf("a repository's limit of one was heeded: %+v, %v", res, err)
	}
}

// TestPreflightSlotsClaimsNarrowRuns: the key is known to the reader that names
// unknown keys under load, so setting it adds no note.
func TestPreflightSlotsClaimsNarrowRuns(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2, "narrow_runs": 4}}`)
	n, src, note, err := PreflightSlots(noEnv, home)
	if err != nil || n != 2 || src != SlotsMachine || note != "" {
		t.Fatalf("PreflightSlots = %d, %q, note %q, %v", n, src, note, err)
	}
}
