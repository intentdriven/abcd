package implement

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// A command hold keeps its slot for as long as its command runs, and the
// waiters behind a long, slow one could not tell whether it was working
// (iss-2610241419324875). The waiters and the holder now say when its whole
// process tree has used little CPU, from what it uses between their looks, which the
// snapshot already carries (machineload.Proc.CPU), so no test here starts a
// process: each sets the tree's CPU against the fake clock.

// quietHolder is a command hold's holder (abcd 777, running make 778 and
// its shell 779) whose tree uses cpu(aged) of CPU time by aged after the
// clock's start, and its record, opened a minute before the clock's start in
// the checkout /work/repo.
func quietHolder(t *testing.T, home string) (machineload.Proc, []machineload.Proc, WaitHolder) {
	t.Helper()
	opened := waitT0.Add(-time.Minute)
	h := WaitHolder{PID: 777, StartUnix: opened.Add(-time.Second).Unix(), Opened: opened,
		Until: opened.Add(WaitCommandCap), Form: HoldCommand, Dir: "/work/repo"}
	writeHold(t, home, h)
	holder, tree := quietHolderProcs()
	return holder, tree, h
}

// quietHolderProcs is quietHolder's processes, without its record.
func quietHolderProcs() (machineload.Proc, []machineload.Proc) {
	holder := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Minute + time.Second, Name: "abcd"}
	tree := []machineload.Proc{
		{PID: 778, PPID: 777, PGID: 777, UID: 501, Age: time.Minute, Name: "make"},
		{PID: 779, PPID: 778, PGID: 777, UID: 501, Age: time.Minute, Name: "sh"},
	}
	return holder, tree
}

// withTreeCPU makes read report pid 779's CPU as cpu of the time since the
// clock's start, so the holder's tree uses that much between two looks.
func withTreeCPU(read func() (machineload.Snapshot, error), clock *fakeClock, cpu func(time.Duration) time.Duration) func() (machineload.Snapshot, error) {
	return func() (machineload.Snapshot, error) {
		s, err := read()
		for i := range s.Procs {
			if s.Procs[i].PID == 779 {
				s.Procs[i].CPU = time.Second + cpu(clock.Now().Sub(waitT0))
			}
		}
		return s, err
	}
}

// TestWaitSaysAHolderWhoseTreeIsQuiet: a command hold whose tree has used
// almost no CPU for QuietSpan is named to the waiter once, by pid, checkout
// and how long it has held the slot, and still holds the gate shut: the gate
// tells, it never takes.
func TestWaitSaysAHolderWhoseTreeIsQuiet(t *testing.T) {
	home := t.TempDir()
	holder, tree, _ := quietHolder(t, home)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: append([]machineload.Proc{holder}, tree...)})
	req.Read = withTreeCPU(req.Read, clock, func(d time.Duration) time.Duration { return d / 100 })
	req.Timeout = QuietSpan + 5*time.Minute
	var said []WaitResult
	req.OnQuiet = func(r WaitResult) { said = append(said, r) }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLock}) {
		t.Fatalf("result %+v; want the quiet holder's hold still honoured at the timeout", res)
	}
	if len(said) != 1 {
		t.Fatalf("said the quiet %d times, want once: %+v", len(said), said)
	}
	s := said[0].Quiet
	if s == nil || s.PID != 777 || s.Checkout != "/work/repo" || s.HeldS < (QuietSpan+time.Minute).Seconds() || s.QuietS < QuietSpan.Seconds() {
		t.Fatalf("quiet said = %+v, want pid 777 in /work/repo quiet at least %s", s, QuietSpan)
	}
	if res.Quiet == nil || res.Quiet.QuietS < 14*60 {
		t.Fatalf("final quiet = %+v, want the holder still quiet near the timeout", res.Quiet)
	}
}

// TestWaitNeverSaysABusyHolder: a holder whose tree keeps a core busy, or
// does a little over the quiet share, is never named, however long it holds.
func TestWaitNeverSaysABusyHolder(t *testing.T) {
	for name, cpu := range map[string]func(time.Duration) time.Duration{
		"a core flat out":     func(d time.Duration) time.Duration { return d },
		"just over the share": func(d time.Duration) time.Duration { return time.Duration(float64(d) * QuietShare * 1.2) },
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			holder, tree, _ := quietHolder(t, home)
			clock := &fakeClock{now: waitT0}
			req := waitReq(home, 60000, clock, poll{load: 3, procs: append([]machineload.Proc{holder}, tree...)})
			req.Read = withTreeCPU(req.Read, clock, cpu)
			req.Timeout = 2 * QuietSpan
			req.OnQuiet = func(r WaitResult) { t.Fatalf("a busy holder was said quiet: %+v", r.Quiet) }
			res, err := WaitLoad(req)
			if err != nil || res.Status != WaitTimedOut || res.Quiet != nil {
				t.Fatalf("result %+v, err %v; want a timeout with no quiet", res, err)
			}
		})
	}
}

// TestWaitQuietClearsWhenTheHolderPicksUp: a holder said quiet that picks up
// again is no longer named, and is said again if it goes quiet again.
func TestWaitQuietClearsWhenTheHolderPicksUp(t *testing.T) {
	home := t.TempDir()
	holder, tree, _ := quietHolder(t, home)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: append([]machineload.Proc{holder}, tree...)})
	// Quiet for the first 12 minutes, a core flat out for the next 3, then quiet.
	busyFrom, busyTo := 12*time.Minute, 15*time.Minute
	req.Read = withTreeCPU(req.Read, clock, func(d time.Duration) time.Duration {
		return min(max(d-busyFrom, 0), busyTo-busyFrom)
	})
	req.Timeout = busyTo + QuietSpan + 2*time.Minute
	var said []float64
	req.OnQuiet = func(r WaitResult) { said = append(said, r.Quiet.QuietS) }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(said) != 2 {
		t.Fatalf("said the quiet %d times (%v), want twice: before the holder picked up and after it went quiet again", len(said), said)
	}
	if res.Quiet == nil || res.Quiet.QuietS > (QuietSpan+3*time.Minute).Seconds() {
		t.Fatalf("final quiet = %+v, want one counted from the second quiet, not the first", res.Quiet)
	}
}

// TestWaitDoesNotSayAHolderWaitingAtAGate: a holder whose tree is itself
// queued at a gate (`--wait -- make preflight`, its preflight's own race
// waiter waiting) is waiting at that gate, not quiet, however little CPU it uses.
func TestWaitDoesNotSayAHolderWaitingAtAGate(t *testing.T) {
	home := t.TempDir()
	holder, tree, _ := quietHolder(t, home)
	inner := machineload.Proc{PID: 781, PPID: 779, PGID: 777, UID: 501, Age: time.Minute, Name: "abcd"}
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3,
		procs: append([]machineload.Proc{holder, inner}, tree...),
		args:  map[int][]string{777: {"implement", "load", "--wait", "--", "make", "preflight"}, 778: {"preflight"}, 781: {"implement", "load", "--race-gate"}}})
	req.Getenv = slotsEnv("2")
	req.Read = withTreeCPU(req.Read, clock, func(time.Duration) time.Duration { return 0 })
	req.Timeout = 2 * QuietSpan
	req.OnQuiet = func(r WaitResult) { t.Fatalf("a holder waiting at a gate was said quiet: %+v", r.Quiet) }
	if res, err := WaitLoad(req); err != nil || res.Quiet != nil {
		t.Fatalf("result %+v, err %v; want no quiet", res, err)
	}
}

// TestHoldWatchSaysItsOwnQuietWhileOthersWait: the holder's own watch names
// the waiters queued behind it once its tree has used little CPU for QuietSpan, and
// says nothing while none waits or while its tree works.
func TestHoldWatchSaysItsOwnQuietWhileOthersWait(t *testing.T) {
	holder, tree, _ := quietHolder(t, t.TempDir())
	queued := waiterAt(60000)
	for _, tc := range []struct {
		name    string
		waiters bool
		cpu     func(time.Duration) time.Duration
		want    []int
	}{
		{"quiet with a waiter", true, func(time.Duration) time.Duration { return 0 }, []int{60000}},
		{"quiet alone", false, func(time.Duration) time.Duration { return 0 }, nil},
		{"busy with a waiter", true, func(d time.Duration) time.Duration { return d }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clock := &fakeClock{now: waitT0}
			procs := append([]machineload.Proc{holder}, tree...)
			args := map[int][]string{}
			if tc.waiters {
				procs = append(procs, queued...)
				args = map[int][]string{60000: {"implement", "load", "--wait", "--", "make", "preflight"}}
			}
			read := func() (machineload.Snapshot, error) {
				ps := append([]machineload.Proc(nil), procs...)
				for i := range ps {
					ps[i].Age += clock.Now().Sub(waitT0)
				}
				return machineload.Snapshot{HasLoad: true, Load1: 3, Cores: 16, Procs: ps, HasProcs: true}, nil
			}
			w := HoldWatch{
				Read:     withTreeCPU(read, clock, tc.cpu),
				ReadArgs: func(pids []int) (map[int][]string, error) { return args, nil },
				Self:     machineload.Self{PID: 777, UID: 501},
				Now:      clock.Now,
				Gates:    []string{GateTests},
			}
			var quiet time.Duration
			var waiting []int
			for clock.Now().Sub(waitT0) <= QuietSpan+time.Minute {
				quiet, waiting = w.Look()
				clock.Sleep(time.Minute)
			}
			if tc.want == nil {
				if quiet > 0 && len(waiting) > 0 {
					t.Fatalf("said quiet %s with %v waiting; want nothing said", quiet, waiting)
				}
				return
			}
			if quiet < QuietSpan || !reflect.DeepEqual(waiting, tc.want) {
				t.Fatalf("quiet %s, waiting %v; want at least %s quiet with %v waiting", quiet, waiting, QuietSpan, tc.want)
			}
		})
	}
}

// TestWaitCountsShortLivedChildrenAsWork: a slow run whose own processes use
// little CPU while it starts short git children, each gone by the next look,
// is working, and is never said quiet: the tree is measured whole, and a
// child that comes and goes counts.
func TestWaitCountsShortLivedChildrenAsWork(t *testing.T) {
	home := t.TempDir()
	holder, tree, _ := quietHolder(t, home)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: append([]machineload.Proc{holder}, tree...)})
	read := req.Read
	n := 0
	req.Read = func() (machineload.Snapshot, error) {
		s, err := read()
		n++
		s.Procs = append(s.Procs, machineload.Proc{PID: 5000 + n, PPID: 779, PGID: 777, UID: 501, Age: time.Second, CPU: 10 * time.Millisecond, Name: "git"})
		return s, err
	}
	req.Timeout = 2 * QuietSpan
	req.OnQuiet = func(r WaitResult) { t.Fatalf("a run starting git children was said quiet: %+v", r.Quiet) }
	if res, err := WaitLoad(req); err != nil || res.Quiet != nil {
		t.Fatalf("result %+v, err %v; want no quiet", res, err)
	}
}

// holdWatchLooks runs w's looks a minute apart on clock until QuietSpan and a
// minute more have passed, and returns the last look's answer.
func holdWatchLooks(w *HoldWatch, clock *fakeClock) (time.Duration, []int) {
	var quiet time.Duration
	var waiting []int
	for clock.Now().Sub(waitT0) <= QuietSpan+time.Minute {
		quiet, waiting = w.Look()
		clock.Sleep(time.Minute)
	}
	return quiet, waiting
}

// quietTable is a quiet holder's table on clock, with waiter 60000 queued
// beside it, and a fresh ps child of psParent at each read (none
// when psParent is zero), named as the snapshot's reader when reader is set.
func quietTable(clock *fakeClock, psParent int, reader bool) func() (machineload.Snapshot, error) {
	holder, tree := quietHolderProcs()
	procs := append(append([]machineload.Proc{holder}, tree...), waiterAt(60000)...)
	n := 0
	return func() (machineload.Snapshot, error) {
		ps := append([]machineload.Proc(nil), procs...)
		for i := range ps {
			ps[i].Age += clock.Now().Sub(waitT0)
		}
		snap := machineload.Snapshot{HasLoad: true, Load1: 3, Cores: 16, HasProcs: true}
		if psParent != 0 {
			n++
			p := machineload.Proc{PID: 9000 + n, PPID: psParent, PGID: 777, UID: 501, CPU: 20 * time.Millisecond, Name: "ps"}
			ps = append(ps, p)
			if reader {
				snap.ReaderPID = p.PID
			}
		}
		snap.Procs = ps
		return snap, nil
	}
}

// TestHoldWatchLeavesOutItsOwnPs: on macOS the holder's own read of the table
// is a ps child of the holder that lists itself and is gone by the next look;
// counted as part of the tree, its going read as work at every look and the
// holder's notice never came. The reader is left out, and the notice comes.
func TestHoldWatchLeavesOutItsOwnPs(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	args := map[int][]string{60000: {"implement", "load", "--wait", "--", "make", "preflight"}}
	w := &HoldWatch{
		Read:     quietTable(clock, 777, true),
		ReadArgs: func([]int) (map[int][]string, error) { return args, nil },
		Self:     machineload.Self{PID: 777, UID: 501},
		Now:      clock.Now,
		Gates:    []string{GateTests},
	}
	if quiet, waiting := holdWatchLooks(w, clock); quiet < QuietSpan || !reflect.DeepEqual(waiting, []int{60000}) {
		t.Fatalf("quiet %s, waiting %v; want at least %s quiet with [60000] waiting despite the reader's own ps", quiet, waiting, QuietSpan)
	}
}

// TestWaitLeavesOutTheHoldersOwnPs: a waiter's look can catch the holder's
// own read of the table, a ps child of the holder that is gone by the next
// look; it is not counted as the holder's work.
func TestWaitLeavesOutTheHoldersOwnPs(t *testing.T) {
	home := t.TempDir()
	quietHolder(t, home)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3})
	// The table's waiter 60000 is this one.
	req.Read = quietTable(clock, 777, false)
	req.Timeout = QuietSpan + 2*time.Minute
	var said int
	req.OnQuiet = func(WaitResult) { said++ }
	if res, err := WaitLoad(req); err != nil || said != 1 || res.Quiet == nil {
		t.Fatalf("result %+v, err %v, said %d times; want the quiet holder said once despite its own ps", res, err, said)
	}
}

// TestHoldWatchCountsWaitersAtEveryGateItHolds: a race command above one slot
// holds both gates, so a race waiter queued behind it is waiting on it too.
func TestHoldWatchCountsWaitersAtEveryGateItHolds(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	args := map[int][]string{60000: {"implement", "load", "--race-gate"}}
	w := &HoldWatch{
		Read:     quietTable(clock, 0, false),
		ReadArgs: func([]int) (map[int][]string, error) { return args, nil },
		Self:     machineload.Self{PID: 777, UID: 501},
		Now:      clock.Now,
		Gates:    []string{GateTests, GateRace},
	}
	if quiet, waiting := holdWatchLooks(w, clock); quiet < QuietSpan || !reflect.DeepEqual(waiting, []int{60000}) {
		t.Fatalf("quiet %s, waiting %v; want the race waiter counted for a hold on both gates", quiet, waiting)
	}
}

// TestWaitHoldWithALongCheckoutStillReads: a checkout path long enough, or
// JSON-escaped enough, to take the hold record past maxHoldBytes would make
// the record read as no hold at all, and admit a second holder; the checkout
// is left out of such a record instead, and the hold still shuts the gate.
func TestWaitHoldWithALongCheckoutStillReads(t *testing.T) {
	for name, dir := range map[string]string{
		"long":    "/" + strings.Repeat("d", 4096),
		"escaped": "/" + strings.Repeat("<&>", 400),
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
			req.Dir = dir
			first, err := WaitLoad(req)
			if err != nil || first.Status != WaitOpen {
				t.Fatalf("first = %+v, err %v; want open", first, err)
			}
			raw, err := os.ReadFile(abcdhome.Path(home, WaitHoldFileName))
			if err != nil || len(raw) > maxHoldBytes {
				t.Fatalf("hold record is %d bytes (err %v), want at most %d", len(raw), err, maxHoldBytes)
			}
			second := waitReq(home, 61000, &fakeClock{now: waitT0}, poll{load: 3})
			second.Timeout = 0
			res, err := WaitLoad(second)
			if err != nil || res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLock}) {
				t.Fatalf("second = %+v, err %v; want it shut by the first's hold", res, err)
			}
		})
	}
}

// TestWaitLinesShowEachHoldersAgeAndCPU: every waiting line carries each
// holder ahead, the hold record's and a waiter of another account running
// its command alike, with its age, and its tree's CPU over the last
// QuietSpan once this waiter has watched it that long; before then it is not
// yet measured (iss-2610241457170255).
func TestWaitLinesShowEachHoldersAgeAndCPU(t *testing.T) {
	home := t.TempDir()
	holder, tree, _ := quietHolder(t, home)
	other := []machineload.Proc{
		{PID: 80000, PPID: 1, PGID: 80000, UID: 502, Age: 3 * time.Minute, Name: "abcd"},
		{PID: 80001, PPID: 80000, PGID: 80000, UID: 502, Age: 3 * time.Minute, Name: "make"},
	}
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3,
		procs: append(append([]machineload.Proc{holder}, tree...), other...),
		args:  map[int][]string{80000: {"implement", "load", "--wait", "--", "make", "preflight"}, 80001: {"preflight"}}})
	// The hold's tree keeps half a core busy; the other's uses none.
	req.Read = withTreeCPU(req.Read, clock, func(d time.Duration) time.Duration { return d / 2 })
	req.Timeout = QuietSpan + 2*time.Minute
	var lines []WaitResult
	req.OnWait = func(r WaitResult) { lines = append(lines, r) }
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut {
		t.Fatalf("result %+v, err %v; want a timeout", res, err)
	}
	if len(lines) != 2 {
		t.Fatalf("%d waiting lines, want 2: the first look's and one a span later", len(lines))
	}
	want := []GateHolder{{PID: 777, Hold: true, AgeS: 60}, {PID: 80000, AgeS: 180}}
	if !reflect.DeepEqual(lines[0].Holders, want) {
		t.Fatalf("first line's holders = %+v, want %+v, neither yet measured", lines[0].Holders, want)
	}
	later := lines[1].Holders
	if len(later) != 2 || !later[0].Measured || !later[1].Measured ||
		later[0].CPU < 0.45 || later[0].CPU > 0.55 || later[1].CPU != 0 || later[0].AgeS < 11*60 {
		t.Fatalf("a span later the holders = %+v, want both measured, 777 at half a core and 80000 at none", later)
	}
}
