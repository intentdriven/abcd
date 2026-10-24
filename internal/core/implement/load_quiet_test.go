package implement

import (
	"reflect"
	"testing"
	"time"

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
	holder := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Minute + time.Second, Name: "abcd"}
	tree := []machineload.Proc{
		{PID: 778, PPID: 777, PGID: 777, UID: 501, Age: time.Minute, Name: "make"},
		{PID: 779, PPID: 778, PGID: 777, UID: 501, Age: time.Minute, Name: "sh"},
	}
	return holder, tree, h
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
