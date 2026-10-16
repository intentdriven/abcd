package implement

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Every wait below runs against a fake machine and a fake clock: no test starts
// a busy loop or a race run, and none sleeps.

// waitT0 is the fake clock's start.
var waitT0 = time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)

// fakeClock is one waiter's clock: Sleep advances it.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// waiterAt is a waiter's own chain: an agent's zsh, `go run`, and the built abcd
// at pid, started a second before the clock's start.
func waiterAt(pid int) []machineload.Proc {
	return []machineload.Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 50 * time.Hour, Name: "launchd"},
		{PID: pid - 2, PPID: 1, PGID: pid - 2, UID: 501, Age: time.Hour, Name: "zsh"},
		{PID: pid - 1, PPID: pid - 2, PGID: pid - 1, UID: 501, Age: time.Minute, Name: "go"},
		{PID: pid, PPID: pid - 1, PGID: pid - 1, UID: 501, Age: time.Second, Name: "abcd"},
	}
}

// waiterArgs is the waiter's go run, whose words carry "go test -race".
func waiterArgs(pid int) map[int][]string {
	return map[int][]string{pid - 1: {"run", "./cmd/abcd", "implement", "load", "--wait", "--", "go", "test", "-race", "./..."}}
}

// raceRun is a live `go test -race` by pid, child of parent.
func raceRun(pid, parent int) machineload.Proc {
	return machineload.Proc{PID: pid, PPID: parent, PGID: pid, UID: 502, Age: 3 * time.Minute, Name: "go"}
}

// poll is one reading of the fake machine: its load and the processes beside the
// waiter's own chain, their ages as at the clock's start.
type poll struct {
	load  float64
	procs []machineload.Proc
	// args are the arguments of procs, by pid; a go process absent from it is
	// a race run.
	args map[int][]string
}

// testJitter is the fixed settle-jitter every fixture's waiter waits once it
// sees the machine free, before it checks everything again.
const testJitter = 20 * time.Second

// waitReq builds a request for the waiter at pid against polls (the last one
// repeats), under home, on clock.
func waitReq(home string, pid int, clock *fakeClock, polls ...poll) WaitRequest {
	var mu sync.Mutex
	i := 0
	args := waiterArgs(pid)
	var last poll
	return WaitRequest{
		Getenv: noEnv,
		Read: func() (machineload.Snapshot, error) {
			mu.Lock()
			defer mu.Unlock()
			p := polls[min(i, len(polls)-1)]
			i++
			last = p
			// Every process ages with the clock, as a real one does, so a
			// holder's start time read at a later look is still its own.
			aged := clock.Now().Sub(waitT0)
			procs := append(waiterAt(pid), p.procs...)
			for j := range procs {
				procs[j].Age += aged
			}
			return machineload.Snapshot{Load1: p.load, Load5: p.load, Load15: p.load, HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}, nil
		},
		ReadArgs: func(pids []int) (map[int][]string, error) {
			mu.Lock()
			defer mu.Unlock()
			out := map[int][]string{}
			for _, p := range pids {
				if a, ok := args[p]; ok {
					out[p] = a
					continue
				}
				if a, ok := last.args[p]; ok {
					out[p] = a
					continue
				}
				// Every other go process in these fixtures is a race run.
				out[p] = []string{"test", "-race", "./internal/..."}
			}
			return out, nil
		},
		Home:    home,
		Self:    machineload.Self{PID: pid, UID: 501},
		Now:     clock.Now,
		Sleep:   clock.Sleep,
		Timeout: 5 * time.Minute,
		Poll:    15 * time.Second,
		Jitter:  func() time.Duration { return testJitter },
		// The command form's mark is a real child on a real machine; here it
		// starts nothing.
		Mark: func() (func(), error) { return func() {}, nil },
	}
}

// writeHold plants a hold record under home.
func writeHold(t *testing.T, home string, h WaitHolder) {
	t.Helper()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(h)
	if err := os.WriteFile(filepath.Join(dir, WaitHoldFileName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// readHold reads the hold record under home, or nil.
func readHold(t *testing.T, home string) *WaitHolder {
	t.Helper()
	raw, err := os.ReadFile(abcdhome.Path(home, WaitHoldFileName))
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

// TestWaitDoesNotCountItsOwnCommandLine is the self-match: the waiter's own chain
// carries "go test -race" in its command lines, the hand-written pgrep loop
// matched exactly that and never opened, and the waiter opens at once.
func TestWaitDoesNotCountItsOwnCommandLine(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(t.TempDir(), 60000, clock, poll{load: 3}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || res.Polls != 2 || len(res.WaitedOn) != 0 || len(res.RaceRuns) != 0 {
		t.Fatalf("the waiter counted itself: %+v", res)
	}
}

// TestWaitRaceRunHoldsTheGateUntilItExits: a live race run keeps the gate shut,
// and its exit opens it; the wait reports what it waited on and for how long.
func TestWaitRaceRunHoldsTheGateUntilItExits(t *testing.T) {
	home := t.TempDir()
	clock := &fakeClock{now: waitT0}
	race := poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}}
	res, err := WaitLoad(waitReq(home, 60000, clock, race, race, poll{load: 3}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || res.Polls != 4 || res.WaitedS != 50 {
		t.Fatalf("result = %+v, want free on the third poll after 30s, and open on the fourth after the 20s jitter", res)
	}
	if !reflect.DeepEqual(res.WaitedOn, []string{WaitOnRaceRun}) {
		t.Fatalf("waited on %v, want [%s]", res.WaitedOn, WaitOnRaceRun)
	}
	h := readHold(t, home)
	if h == nil || h.PID != 60000 || h.Form != HoldSettle || !h.Until.Equal(clock.Now().Add(WaitSettle)) {
		t.Fatalf("hold = %+v, want the waiter's settle hold until %s", h, clock.Now().Add(WaitSettle))
	}
	if res.Hold == nil || *res.Hold != *h {
		t.Fatalf("reported hold %+v, written %+v", res.Hold, h)
	}
}

// TestWaitAnnouncesOnce: the waiter says once that it is waiting, then nothing.
func TestWaitAnnouncesOnce(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 40}, poll{load: 40}, poll{load: 40}, poll{load: 2})
	var calls []WaitResult
	req.OnWait = func(r WaitResult) { calls = append(calls, r) }
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen {
		t.Fatalf("result %+v, err %v", res, err)
	}
	if len(calls) != 1 || !slices.Contains(calls[0].Still, WaitOnLoad) || calls[0].Load != 40 {
		t.Fatalf("announcements = %+v, want one naming the load", calls)
	}
}

// TestWaitTimesOutNamingTheCondition: a load that never falls below the limit
// ends the wait at its timeout, naming the load as what it still waited on.
func TestWaitTimesOutNamingTheCondition(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 40})
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLoad}) || res.WaitedS != 60 {
		t.Fatalf("result = %+v, want a timeout after 60s still on [load]", res)
	}
	if res.WaitLoad != 32 || res.LimitSource != LimitsDefault {
		t.Fatalf("limit %v from %q, want twice the core count by default", res.WaitLoad, res.LimitSource)
	}
	if res.Hold != nil {
		t.Fatalf("a timed-out wait holds the gate: %+v", res.Hold)
	}
}

// TestWaitOpensBelowTwiceTheCores: a busy multi-session machine sits near one
// core-count of load, so the default limit is twice the cores; a load of 24 on
// 16 cores opens the gate, and a load of 32 does not.
func TestWaitOpensBelowTwiceTheCores(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(t.TempDir(), 60000, clock, poll{load: 24}))
	if err != nil || res.Status != WaitOpen || res.WaitLoad != 32 {
		t.Fatalf("result %+v, err %v, want open at load 24 under the default limit of 32", res, err)
	}
	clock = &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 32})
	req.Timeout = 0
	res, err = WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLoad}) {
		t.Fatalf("result %+v, err %v, want shut at load 32: the limit is strict", res, err)
	}
}

// TestWaitLimitComesFromTheSettingsFile: wait-load in ~/.abcd.noindex/load-limits
// sets the limit.
func TestWaitLimitComesFromTheSettingsFile(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, machineload.LimitsFileName), []byte("wait-load 30\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 60000, clock, poll{load: 24}))
	if err != nil || res.Status != WaitOpen || res.WaitLoad != 30 || res.LimitSource != LimitsFile {
		t.Fatalf("result %+v, err %v, want open under a limit of 30 from the file", res, err)
	}
}

// TestWaitHerdOpensOneWaiter: two waiters released by one load drop race for the
// gate, and only one opens; the other keeps waiting on the hold until its
// timeout.
func TestWaitHerdOpensOneWaiter(t *testing.T) {
	home := t.TempDir()
	results := make([]WaitResult, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, pid := range []int{60000, 61000} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clock := &fakeClock{now: waitT0}
			req := waitReq(home, pid, clock, poll{load: 40}, poll{load: 4})
			req.Timeout = time.Minute
			results[i], errs[i] = WaitLoad(req)
		}()
	}
	wg.Wait()
	opened := 0
	for i, r := range results {
		if errs[i] != nil {
			t.Fatalf("waiter %d: %v", i, errs[i])
		}
		if r.Status == WaitOpen {
			opened++
			continue
		}
		if r.Status != WaitTimedOut || !reflect.DeepEqual(r.Still, []string{WaitOnLock}) || r.Holder == nil {
			t.Fatalf("the waiter that did not open: %+v, want a timeout still on [lock] naming the holder", r)
		}
	}
	if opened != 1 {
		t.Fatalf("%d waiters opened on one load drop, want 1: %+v", opened, results)
	}
}

// TestWaitReclaimsADeadHoldersLock: a command hold whose holder has exited is
// reclaimed at once, well before its cap.
func TestWaitReclaimsADeadHoldersLock(t *testing.T) {
	home := t.TempDir()
	writeHold(t, home, WaitHolder{PID: 777, StartUnix: waitT0.Add(-time.Hour).Unix(), Opened: waitT0.Add(-time.Minute),
		Until: waitT0.Add(WaitCommandCap), Form: HoldCommand})
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 60000, clock, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || res.Polls != 2 {
		t.Fatalf("result %+v, err %v, want the dead holder's lock reclaimed on the first poll and the gate open on the re-check", res, err)
	}
	if h := readHold(t, home); h == nil || h.PID != 60000 {
		t.Fatalf("hold after reclaiming = %+v, want the waiter's", h)
	}
}

// TestWaitDistrustsAReusedPid: a command hold whose pid now belongs to another
// process (a different start time) is not trusted, and one whose holder is
// still the same process holds the gate.
func TestWaitDistrustsAReusedPid(t *testing.T) {
	started := waitT0.Add(-5 * time.Minute)
	hold := WaitHolder{PID: 777, StartUnix: started.Unix(), Opened: waitT0.Add(-time.Minute),
		Until: waitT0.Add(WaitCommandCap), Form: HoldCommand}

	// The pid is live, but the process at it started three hours ago.
	reused := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: 3 * time.Hour, Name: "zsh"}
	home := t.TempDir()
	writeHold(t, home, hold)
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 60000, clock, poll{load: 3, procs: []machineload.Proc{reused}}))
	if err != nil || res.Status != WaitOpen {
		t.Fatalf("result %+v, err %v, want a reused pid's hold distrusted", res, err)
	}

	// The same process still holds it: the gate stays shut.
	same := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: 5 * time.Minute, Name: "abcd"}
	home = t.TempDir()
	writeHold(t, home, hold)
	clock = &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: []machineload.Proc{same}})
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLock}) ||
		res.Holder == nil || res.Holder.PID != 777 {
		t.Fatalf("result %+v, err %v, want a live holder's lock honoured", res, err)
	}
}

// TestWaitCommandHoldOutlastsASlowPreflight: on a loaded machine a preflight
// takes longer than twenty minutes to reach its race step, so a command hold
// whose holder is alive 21 minutes after opening still holds the gate, and a
// second preflight is not admitted beside the first.
func TestWaitCommandHoldOutlastsASlowPreflight(t *testing.T) {
	home := t.TempDir()
	opened := waitT0.Add(-21 * time.Minute)
	started := opened.Add(-time.Second)
	writeHold(t, home, WaitHolder{PID: 777, StartUnix: started.Unix(), Opened: opened,
		Until: opened.Add(WaitCommandCap), Form: HoldCommand})
	holder := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: 21*time.Minute + time.Second, Name: "abcd"}
	makeProc := machineload.Proc{PID: 778, PPID: 777, PGID: 777, UID: 501, Age: 21 * time.Minute, Name: "make"}
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: []machineload.Proc{holder, makeProc}})
	req.Timeout = 0
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLock}) ||
		res.Holder == nil || res.Holder.PID != 777 {
		t.Fatalf("result %+v, err %v, want a live holder's hold honoured 21 minutes after it opened", res, err)
	}
}

// TestWaitCommandHoldEndsWhenItsRaceRunStarts: a command hold is retired once
// the work it gates has started its own race run, so when that run exits the
// next waiter opens without waiting out the cap.
func TestWaitCommandHoldEndsWhenItsRaceRunStarts(t *testing.T) {
	home := t.TempDir()
	started := waitT0.Add(-5 * time.Minute)
	writeHold(t, home, WaitHolder{PID: 777, StartUnix: started.Unix(), Opened: waitT0.Add(-4 * time.Minute),
		Until: waitT0.Add(WaitCommandCap), Form: HoldCommand})
	holder := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: 5 * time.Minute, Name: "abcd"}
	makeProc := machineload.Proc{PID: 778, PPID: 777, PGID: 777, UID: 501, Age: 4 * time.Minute, Name: "make"}
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 60000, clock,
		poll{load: 3, procs: []machineload.Proc{holder, makeProc, raceRun(779, 778)}},
		poll{load: 3, procs: []machineload.Proc{holder, makeProc}}))
	if err != nil || res.Status != WaitOpen || res.Polls != 3 {
		t.Fatalf("result %+v, err %v, want open once the holder's own race run has come and gone", res, err)
	}
}

// TestWaitSettleHoldOutlivesItsHolder: the plain form's holder exits right after
// opening, by design, so its hold lasts the settle interval whatever became of
// it.
func TestWaitSettleHoldOutlivesItsHolder(t *testing.T) {
	home := t.TempDir()
	writeHold(t, home, WaitHolder{PID: 777, StartUnix: waitT0.Add(-time.Minute).Unix(), Opened: waitT0.Add(-30 * time.Second),
		Until: waitT0.Add(90 * time.Second), Form: HoldSettle})
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 60000, clock, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || res.WaitedS != 110 || !reflect.DeepEqual(res.WaitedOn, []string{WaitOnLock}) {
		t.Fatalf("result %+v, err %v, want free once the settle hold lapses at 90s, and open after the 20s jitter", res, err)
	}
}

// TestReleaseWaitHoldRemovesOnlyItsOwn: the command form releases its hold when
// its command exits, and never a hold another waiter has since taken.
func TestReleaseWaitHoldRemovesOnlyItsOwn(t *testing.T) {
	home := t.TempDir()
	mine := WaitHolder{PID: 60000, StartUnix: waitT0.Unix(), Opened: waitT0, Until: waitT0.Add(WaitCommandCap), Form: HoldCommand}
	theirs := mine
	theirs.PID = 61000
	writeHold(t, home, theirs)
	if err := ReleaseWaitHold(home, mine); err != nil {
		t.Fatal(err)
	}
	if h := readHold(t, home); h == nil || h.PID != 61000 {
		t.Fatalf("released another waiter's hold: %+v", h)
	}
	writeHold(t, home, mine)
	if err := ReleaseWaitHold(home, mine); err != nil {
		t.Fatal(err)
	}
	if h := readHold(t, home); h != nil {
		t.Fatalf("own hold left after release: %+v", h)
	}
}

// TestWaitRefusesASymlinkedHome: a symlinked ~/.abcd.noindex is refused before
// the machine is read or anything is written, as every other home writer
// refuses it.
func TestWaitRefusesASymlinkedHome(t *testing.T) {
	home := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3})
	read := false
	inner := req.Read
	req.Read = func() (machineload.Snapshot, error) { read = true; return inner() }
	_, err := WaitLoad(req)
	if !errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		t.Fatalf("err = %v, want the symlinked-home refusal", err)
	}
	if read {
		t.Fatal("the machine was read before the refusal")
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Fatalf("wrote through the link: %v", entries)
	}
}

// TestWaitSkipsOnCI: a CI runner holds no other session's work, so the waiter
// opens at once there and says why, writing nothing.
func TestWaitSkipsOnCI(t *testing.T) {
	home := t.TempDir()
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 40})
	req.Getenv = func(k string) string {
		if k == "GITHUB_ACTIONS" {
			return "true"
		}
		return ""
	}
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitSkipped || res.Reason == "" {
		t.Fatalf("result %+v, err %v, want skipped with a reason", res, err)
	}
	if _, err := os.Stat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the CI skip created %s", abcdhome.Display())
	}
}

// otherAccountsPreflight is another account's `make preflight` at pid, running
// its unit pass, with the arguments the process table shows for it.
func otherAccountsPreflight(pid int) ([]machineload.Proc, map[int][]string) {
	return []machineload.Proc{
			{PID: pid, PPID: 1, PGID: pid, UID: 502, Age: 4 * time.Minute, Name: "make"},
			{PID: pid + 1, PPID: pid, PGID: pid, UID: 502, Age: time.Minute, Name: "go"},
		}, map[int][]string{
			pid:     {"preflight"},
			pid + 1: {"test", "./..."},
		}
}

// TestWaitSeesAnotherAccountsPreflight is the cross-account gap
// (iss-2610140638074587): another account's make preflight writes no hold this
// account can read, and the process table shows it all the same, so the gate
// stays shut until it exits.
func TestWaitSeesAnotherAccountsPreflight(t *testing.T) {
	home := t.TempDir()
	procs, args := otherAccountsPreflight(80000)
	busy := poll{load: 3, procs: procs, args: args}
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, busy, busy, poll{load: 3})
	var announced []WaitResult
	req.OnWait = func(r WaitResult) { announced = append(announced, r) }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || !reflect.DeepEqual(res.WaitedOn, []string{WaitOnPreflight}) || res.WaitedS != 50 {
		t.Fatalf("result = %+v, want shut on [preflight] for 30s, then open after the jitter", res)
	}
	if len(announced) != 1 || !reflect.DeepEqual(announced[0].Preflights, []int{80000}) {
		t.Fatalf("announcements = %+v, want one naming the other account's preflight 80000", announced)
	}
}

// TestWaitRechecksAfterTheJitter: a waiter that sees the machine free waits
// the jitter and checks everything again before it opens, so a second starter
// that appears inside the window shuts it, and it takes no hold until a
// re-check finds the machine still free.
func TestWaitRechecksAfterTheJitter(t *testing.T) {
	home := t.TempDir()
	procs, args := otherAccountsPreflight(80000)
	starter := poll{load: 3, procs: procs, args: args}
	clock := &fakeClock{now: waitT0}
	// Free; the second starter inside the window; free; free.
	req := waitReq(home, 60000, clock, poll{load: 3}, starter, poll{load: 3})
	var jitters int
	req.Jitter = func() time.Duration { jitters++; return testJitter }
	var heldAtRecheck *WaitHolder
	inner := req.Read
	reads := 0
	req.Read = func() (machineload.Snapshot, error) {
		reads++
		// Before the first re-check, and before the look after it.
		if reads == 2 || reads == 3 {
			if h := readHold(t, home); h != nil {
				heldAtRecheck = h
			}
		}
		return inner()
	}
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if heldAtRecheck != nil {
		t.Fatalf("a hold was taken before the re-check found the machine still free: %+v", heldAtRecheck)
	}
	if res.Status != WaitOpen || res.Polls != 4 || jitters != 2 || !reflect.DeepEqual(res.WaitedOn, []string{WaitOnPreflight}) {
		t.Fatalf("result = %+v after %d jitters, want shut by the starter at the first re-check, then a second jitter and open on the fourth look", res, jitters)
	}
	// 0s free, 20s re-check shut, 35s free, 55s open.
	if res.WaitedS != 55 {
		t.Fatalf("waited %vs, want 55", res.WaitedS)
	}
	if h := readHold(t, home); h == nil || h.PID != 60000 {
		t.Fatalf("hold = %+v, want the waiter's own once it opened", h)
	}
}

// TestWaitJitterIsFifteenToSixtySeconds: the default jitter falls between 15
// and 60 seconds.
func TestWaitJitterIsFifteenToSixtySeconds(t *testing.T) {
	seen := map[bool]bool{}
	for range 2000 {
		j := waitJitter()
		if j < WaitJitterMin || j > WaitJitterMax {
			t.Fatalf("jitter %s is outside [%s, %s]", j, WaitJitterMin, WaitJitterMax)
		}
		seen[j < (WaitJitterMin+WaitJitterMax)/2] = true
	}
	if !seen[true] || !seen[false] {
		t.Fatal("the jitter does not spread over its range")
	}
}

// TestWaitQueuesBehindAnEarlierWaiter: another account's waiter that started
// earlier is ahead in the queue the process table orders, so this waiter
// stays shut while it is there.
func TestWaitQueuesBehindAnEarlierWaiter(t *testing.T) {
	early := []machineload.Proc{
		{PID: 50000, PPID: 1, PGID: 50000, UID: 502, Age: time.Minute, Name: "go"},
		{PID: 50001, PPID: 50000, PGID: 50000, UID: 502, Age: time.Minute, Name: "abcd"},
	}
	args := map[int][]string{
		50000: {"run", "./cmd/abcd", "implement", "load", "--wait"},
		50001: {"implement", "load", "--wait"},
	}
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: early, args: args})
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnWaiter}) || !reflect.DeepEqual(res.Waiters, []int{50001}) {
		t.Fatalf("result = %+v, want a timeout still on [waiter] naming 50001", res)
	}
}

// TestWaitIgnoresItsAncestorsHold: `abcd implement load --wait -- make
// preflight` runs a preflight whose own gate is a waiter beneath the first; the
// first's command hold is its ancestor's, so the inner waiter is not shut by it
// and leaves it in place.
func TestWaitIgnoresItsAncestorsHold(t *testing.T) {
	home := t.TempDir()
	// The waiter's chain is zsh 59998, go 59999, abcd 60000; the outer waiter
	// is the go run's parent's parent here: re-parent the chain under it.
	outer := machineload.Proc{PID: 59000, PPID: 1, PGID: 59000, UID: 501, Age: 10 * time.Minute, Name: "abcd"}
	started := waitT0.Add(-10 * time.Minute)
	hold := WaitHolder{PID: 59000, StartUnix: started.Unix(), Opened: started.Add(time.Second),
		Until: started.Add(WaitCommandCap), Form: HoldCommand}
	writeHold(t, home, hold)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: []machineload.Proc{outer}})
	inner := req.Read
	req.Read = func() (machineload.Snapshot, error) {
		snap, err := inner()
		for i := range snap.Procs {
			if snap.Procs[i].PID == 59998 {
				snap.Procs[i].PPID = 59000
			}
		}
		return snap, err
	}
	req.Timeout = time.Minute
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("result = %+v, want open at once: the hold in force is the waiter's own ancestor's", res)
	}
	if h := readHold(t, home); h == nil || *h != hold {
		t.Fatalf("hold = %+v, want the ancestor's left in place", h)
	}
}

// TestPreflightParallelism: make preflight's go test -p is derived from the
// machine, and on a CI runner the machine is not read and go's default stands.
func TestPreflightParallelism(t *testing.T) {
	read := func() (machineload.Snapshot, error) {
		return machineload.Snapshot{Load1: 28, HasLoad: true, Cores: 16}, nil
	}
	if got := PreflightParallelism(noEnv, read); got != 4 {
		t.Fatalf("-p at load 28 on 16 cores = %d, want 4", got)
	}
	ci := func(k string) string {
		if k == "CI" {
			return "true"
		}
		return ""
	}
	called := false
	got := PreflightParallelism(ci, func() (machineload.Snapshot, error) {
		called = true
		return read()
	})
	if called || got != runtime.GOMAXPROCS(0) {
		t.Fatalf("on CI: -p %d (read %v), want go's default %d without reading the machine", got, called, runtime.GOMAXPROCS(0))
	}
	if got := PreflightParallelism(noEnv, func() (machineload.Snapshot, error) {
		return machineload.Snapshot{}, machineload.ErrUnsupported
	}); got != runtime.GOMAXPROCS(0) {
		t.Fatalf("on an unread machine: -p %d, want go's default %d", got, runtime.GOMAXPROCS(0))
	}
}

// pairWaiter is a waiter in the command form at pid under uid, its go run
// beneath launchd, started age before the clock's start; its command is not a
// race run, so only the queue and the hold can shut another waiter on it.
func pairWaiter(pid int, uid uint32, age time.Duration) ([]machineload.Proc, map[int][]string) {
	words := []string{"implement", "load", "--wait", "--", "sh", "-c", "work"}
	return []machineload.Proc{
			{PID: pid - 1, PPID: 1, PGID: pid - 1, UID: uid, Age: age + time.Second, Name: "go"},
			{PID: pid, PPID: pid - 1, PGID: pid - 1, UID: uid, Age: age, Name: "abcd"},
		}, map[int][]string{
			pid - 1: append([]string{"run", "./cmd/abcd"}, words...),
			pid:     words,
		}
}

// TestWaitTwoWaitersThatDisagreeOpenApart: a waiter that started earlier with
// the higher pid, as after the ids wrap, and one that started later with the
// lower pid. Neither is ahead of the other in the queue, so both find the
// machine free at once; what keeps them from opening in one moment is the
// settle wait and re-check, and within one account the hold. The waiter whose
// re-check comes first opens and runs its command; the other's re-check, at
// that same instant or later, finds the hold (one account) or the first
// running its command (any account), and it opens only once that command has
// exited. The fake table shows the first as running from the instant it
// opens, by its mark, and its command WaitOpenPause later: on a real machine
// the mark shows a moment after, and across accounts, which share no hold, a
// re-check falling inside that moment still opens; the open look then stands
// the higher pid back (TestWaitOpenersInOneMomentStandBackByPid).
func TestWaitTwoWaitersThatDisagreeOpenApart(t *testing.T) {
	const (
		early, late = 96274, 50460
		runFor      = 5 * time.Minute
	)
	ages := map[int]time.Duration{early: 86 * time.Minute, late: 83 * time.Minute}
	for _, c := range []struct {
		name         string
		first, then  int
		sameAccount  bool
		secondJitter time.Duration
	}{
		{"one account, the early waiter first, one instant", early, late, true, testJitter},
		{"one account, the late waiter first, later", late, early, true, testJitter + 15*time.Second},
		{"two accounts, the early waiter first, one instant", early, late, false, testJitter},
		{"two accounts, the late waiter first, later", late, early, false, testJitter + 15*time.Second},
	} {
		t.Run(c.name, func(t *testing.T) {
			homes := map[int]string{c.first: t.TempDir(), c.then: t.TempDir()}
			uids := map[int]uint32{c.first: 501, c.then: 502}
			if c.sameAccount {
				homes[c.then], uids[c.then] = homes[c.first], 501
			}
			// opened is when the first waiter opened; it shows as running from
			// then (its mark, then its command) for runFor. Zero until it has. The first runs to its open
			// before the second starts, so its hold, which the two share
			// within one account, is taken back off disk and planted again
			// once the second's clock reaches the instant it was taken.
			var opened time.Time
			var hold *WaitHolder
			read := func(self int, clock *fakeClock) func() (machineload.Snapshot, error) {
				return func() (machineload.Snapshot, error) {
					now := clock.Now()
					if self == c.then && hold != nil && !now.Before(opened) {
						writeHold(t, homes[self], *hold)
						hold = nil
					}
					aged := now.Sub(waitT0)
					procs := []machineload.Proc{{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 50 * time.Hour, Name: "launchd"}}
					for _, pid := range []int{c.first, c.then} {
						if pid != self && pid == c.first && !opened.IsZero() && !now.Before(opened.Add(runFor)) {
							continue // the first waiter's command has exited, and the waiter with it
						}
						w, _ := pairWaiter(pid, uids[pid], ages[pid])
						procs = append(procs, w...)
						if pid != self && pid == c.first && !opened.IsZero() && !now.Before(opened) {
							procs = append(procs, machineload.Proc{PID: pid + 1, PPID: pid, PGID: pid - 1, UID: uids[pid], Name: "sh"})
						}
					}
					for j := range procs {
						procs[j].Age += aged
					}
					return machineload.Snapshot{Load1: 3, HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}, nil
				}
			}
			args := map[int][]string{}
			for _, pid := range []int{c.first, c.then} {
				_, a := pairWaiter(pid, uids[pid], ages[pid])
				for k, v := range a {
					args[k] = v
				}
			}
			waiter := func(self int, jitter time.Duration) (WaitResult, time.Time, time.Time) {
				clock := &fakeClock{now: waitT0}
				req := waitReq(homes[self], self, clock)
				req.Read = read(self, clock)
				req.ReadArgs = func(pids []int) (map[int][]string, error) {
					out := map[int][]string{}
					for _, p := range pids {
						if a, ok := args[p]; ok {
							out[p] = a
						}
					}
					return out, nil
				}
				req.Self = machineload.Self{PID: self, UID: uids[self]}
				req.Command = true
				req.Timeout = 30 * time.Minute
				req.Jitter = func() time.Duration { return jitter }
				var settled time.Time
				req.OnSettle = func(time.Duration) {
					if settled.IsZero() {
						settled = clock.Now()
					}
				}
				res, err := WaitLoad(req)
				if err != nil {
					t.Fatal(err)
				}
				return res, settled, clock.Now()
			}

			// The first waiter's re-check comes first, so nothing the second
			// does can touch it before it opens.
			res, settled, at := waiter(c.first, testJitter)
			if res.Status != WaitOpen || !settled.Equal(waitT0) || !at.Equal(waitT0.Add(testJitter+WaitOpenPause)) {
				t.Fatalf("first waiter %d: %+v, settled at %s, done at %s; want it free at its first look, open after the jitter and done after the open look", c.first, res, settled, at)
			}
			opened = at.Add(-WaitOpenPause)
			if c.sameAccount {
				if hold = readHold(t, homes[c.first]); hold == nil || hold.Form != HoldCommand {
					t.Fatalf("first waiter %d took the hold %+v, want its command hold", c.first, hold)
				}
				if err := os.Remove(abcdhome.Path(homes[c.first], WaitHoldFileName)); err != nil {
					t.Fatal(err)
				}
			}

			res, settled, at = waiter(c.then, c.secondJitter)
			if !settled.Equal(waitT0) {
				t.Fatalf("second waiter %d first settled at %s, want at its first look: the first waiter is not ahead of it", c.then, settled)
			}
			if res.Status != WaitOpen {
				t.Fatalf("second waiter %d: %+v, want open once the first's command exits", c.then, res)
			}
			if at.Before(opened.Add(runFor)) {
				t.Fatalf("second waiter %d opened at %s, while the first's command ran from %s to %s", c.then, at, opened, opened.Add(runFor))
			}
			want := WaitOnWaiter
			if c.sameAccount {
				want = WaitOnLock
			}
			if !slices.Contains(res.WaitedOn, want) {
				t.Fatalf("second waiter %d waited on %v, want %s among them", c.then, res.WaitedOn, want)
			}
		})
	}
}

// opener is what one waiter of the command form does at each moment, as
// another waiter's fake table shows it: child names the child it has at now
// and when that child started (none while it waits), and gone reports that it
// has exited.
type opener struct {
	child func(now time.Time) (name string, since time.Time, ok bool)
	gone  func(now time.Time) bool
}

// openerRead is the fake machine as the waiter self sees it beside other:
// launchd, both waiters' chains, and the other's child as o says.
func openerRead(clock *fakeClock, self, other int, uids map[int]uint32, ages map[int]time.Duration, o opener) func() (machineload.Snapshot, error) {
	return func() (machineload.Snapshot, error) {
		now := clock.Now()
		aged := now.Sub(waitT0)
		procs := []machineload.Proc{{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 50 * time.Hour, Name: "launchd"}}
		for _, pid := range []int{self, other} {
			if pid == other && o.gone(now) {
				continue
			}
			w, _ := pairWaiter(pid, uids[pid], ages[pid])
			procs = append(procs, w...)
		}
		for j := range procs {
			procs[j].Age += aged
		}
		if !o.gone(now) {
			if name, since, ok := o.child(now); ok {
				procs = append(procs, machineload.Proc{PID: other + 1, PPID: other, PGID: other - 1, UID: uids[other], Age: now.Sub(since), Name: name})
			}
		}
		return machineload.Snapshot{Load1: 3, HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}, nil
	}
}

// openerArgs reads the pair's arguments.
func openerArgs(uids map[int]uint32, ages map[int]time.Duration, pids ...int) func([]int) (map[int][]string, error) {
	args := map[int][]string{}
	for _, pid := range pids {
		_, a := pairWaiter(pid, uids[pid], ages[pid])
		for k, v := range a {
			args[k] = v
		}
	}
	return func(want []int) (map[int][]string, error) {
		out := map[int][]string{}
		for _, p := range want {
			if a, ok := args[p]; ok {
				out[p] = a
			}
		}
		return out, nil
	}
}

// marks records a waiter's marks: when each started and when each stopped.
type marks struct {
	clock         *fakeClock
	starts, stops []time.Time
}

func (m *marks) mark() (func(), error) {
	m.starts = append(m.starts, m.clock.Now())
	stopped := false
	return func() {
		if !stopped {
			stopped = true
			m.stops = append(m.stops, m.clock.Now())
		}
	}, nil
}

// TestWaitOpenersInOneMomentStandBackByPid is the stall the review of
// 2026-10-16 found. Two waiters whose start and pid disagree, as after the ids
// wrap, wait on neither, and across accounts no hold keeps them apart: their
// re-checks fall in one moment, before either shows in the table as running
// its command, and both open. Under `--wait -- make preflight` each
// preflight's own gate then waits on the other's running waiter, and both
// stand until their two-hour timeouts. So the command form shows itself as
// running from the moment it opens, waits WaitOpenPause and looks once more,
// and the waiter with the higher pid stands back from one with the lower that
// opened within that moment: it does not start its command, it gives up its
// hold, re-draws its jitter and waits again, and opens once the other's
// command has exited. The lower pid keeps going. Within one account the hold
// already admits only one, and nothing changes.
func TestWaitOpenersInOneMomentStandBackByPid(t *testing.T) {
	const (
		early, late = 96274, 50460 // early: higher pid; late: lower pid
		runFor      = 5 * time.Minute
	)
	ages := map[int]time.Duration{early: 86 * time.Minute, late: 83 * time.Minute}
	// Both re-checks fall at this instant.
	at := waitT0.Add(testJitter)
	cmdAt := at.Add(WaitOpenPause)
	never := func(time.Time) bool { return false }

	run := func(t *testing.T, self, other int, home string, uids map[int]uint32, o opener) (WaitResult, *marks, []time.Time, time.Time) {
		t.Helper()
		clock := &fakeClock{now: waitT0}
		m := &marks{clock: clock}
		req := waitReq(home, self, clock)
		req.Read = openerRead(clock, self, other, uids, ages, o)
		req.ReadArgs = openerArgs(uids, ages, self, other)
		req.Self = machineload.Self{PID: self, UID: uids[self]}
		req.Command = true
		req.Timeout = 30 * time.Minute
		req.Mark = m.mark
		var backoffs []time.Time
		req.OnBackoff = func(lower []int, j time.Duration) {
			if !reflect.DeepEqual(lower, []int{other}) || j != testJitter {
				t.Errorf("waiter %d stood back from %v, re-drawing %s; want from %d, re-drawing %s", self, lower, j, other, testJitter)
			}
			if h := readHold(t, home); h != nil && h.PID == self {
				t.Errorf("waiter %d stood back but kept its hold %+v", self, h)
			}
			backoffs = append(backoffs, clock.Now())
		}
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		return res, m, backoffs, clock.Now()
	}

	t.Run("two accounts", func(t *testing.T) {
		uids := map[int]uint32{early: 502, late: 501}
		homes := map[int]string{early: t.TempDir(), late: t.TempDir()}

		// The lower pid, as the other's table shows the higher: its mark from
		// the moment it opened, until it stands back.
		res, m, backoffs, done := run(t, late, early, homes[late], uids, opener{
			child: func(now time.Time) (string, time.Time, bool) {
				return "sleep", at, now.After(at) && !now.After(cmdAt)
			},
			gone: never,
		})
		if res.Status != WaitOpen || res.Backoffs != 0 || len(backoffs) != 0 || !done.Equal(cmdAt) {
			t.Fatalf("lower pid %d: %+v, stood back at %v, done at %s; want open at %s without standing back: the other opener has the higher pid", late, res, backoffs, done, cmdAt)
		}
		if !reflect.DeepEqual(m.starts, []time.Time{at}) || len(m.stops) != 0 || res.Unmark == nil {
			t.Fatalf("lower pid %d marked at %v and stopped at %v; want marked once it opened, at %s, and left to its caller to stop", late, m.starts, m.stops, at)
		}
		res.Unmark()
		if len(m.stops) != 1 {
			t.Fatalf("the result's Unmark stopped %d marks, want 1", len(m.stops))
		}

		// The higher pid, as its table shows the lower: its mark from the
		// moment it opened, then its command for runFor.
		res, m, backoffs, done = run(t, early, late, homes[early], uids, opener{
			child: func(now time.Time) (string, time.Time, bool) {
				if now.Before(cmdAt) {
					return "sleep", at, now.After(at)
				}
				return "sh", at, true
			},
			gone: func(now time.Time) bool { return !now.Before(cmdAt.Add(runFor)) },
		})
		if !reflect.DeepEqual(backoffs, []time.Time{cmdAt}) || res.Backoffs != 1 {
			t.Fatalf("higher pid %d stood back at %v (%d), want once, at %s, when its look found the lower pid opened beside it", early, backoffs, res.Backoffs, cmdAt)
		}
		if len(m.stops) == 0 || !m.stops[0].Equal(cmdAt) {
			t.Fatalf("higher pid %d stopped its marks at %v, want its first stopped as it stood back, at %s", early, m.stops, cmdAt)
		}
		if res.Status != WaitOpen || done.Before(cmdAt.Add(runFor)) || !slices.Contains(res.WaitedOn, WaitOnWaiter) {
			t.Fatalf("higher pid %d: %+v, done at %s; want open once the lower pid's command exited at %s, having waited on it", early, res, done, cmdAt.Add(runFor))
		}
	})

	t.Run("one account", func(t *testing.T) {
		uids := map[int]uint32{early: 501, late: 501}
		home := t.TempDir()
		// The lower pid opens; the higher, shut by its hold, never marks.
		res, _, backoffs, done := run(t, late, early, home, uids, opener{
			child: func(time.Time) (string, time.Time, bool) { return "", time.Time{}, false },
			gone:  never,
		})
		if res.Status != WaitOpen || res.Backoffs != 0 || len(backoffs) != 0 || !done.Equal(cmdAt) {
			t.Fatalf("lower pid %d: %+v, stood back at %v, done at %s; want open at %s", late, res, backoffs, done, cmdAt)
		}
		hold := readHold(t, home)
		if hold == nil || hold.PID != late {
			t.Fatalf("hold %+v, want the lower pid's", hold)
		}
		// The higher pid's re-check comes in the same moment and finds the
		// hold, as the one account's two waiters share it.
		res, m, backoffs, done := run(t, early, late, home, uids, opener{
			child: func(now time.Time) (string, time.Time, bool) {
				if now.Before(cmdAt) {
					return "sleep", at, now.After(at)
				}
				return "sh", at, true
			},
			gone: func(now time.Time) bool { return !now.Before(cmdAt.Add(runFor)) },
		})
		if res.Status != WaitOpen || res.Backoffs != 0 || len(backoffs) != 0 || len(m.starts) != 1 {
			t.Fatalf("higher pid %d: %+v, stood back at %v, marked at %v; want it shut by the hold, never standing back, and opening once", early, res, backoffs, m.starts)
		}
		if done.Before(cmdAt.Add(runFor)) || !slices.Contains(res.WaitedOn, WaitOnLock) {
			t.Fatalf("higher pid %d: done at %s, waited on %v; want open after the lower pid's command, having waited on the hold", early, done, res.WaitedOn)
		}
	})
}

// TestWaitBackoffsAreBounded: a waiter that finds another opening beside it
// with the lower pid at every look stands back WaitBackoffCap times, then
// opens without that look, so it never goes round for ever.
func TestWaitBackoffsAreBounded(t *testing.T) {
	const self, other = 96274, 50460
	uids := map[int]uint32{self: 502, other: 501}
	ages := map[int]time.Duration{self: 86 * time.Minute, other: 83 * time.Minute}
	clock := &fakeClock{now: waitT0}
	m := &marks{clock: clock}
	req := waitReq(t.TempDir(), self, clock)
	// The other opens whenever this one is marked, and stands back as soon.
	req.Read = openerRead(clock, self, other, uids, ages, opener{
		child: func(time.Time) (string, time.Time, bool) {
			if len(m.starts) == len(m.stops) {
				return "", time.Time{}, false
			}
			return "sleep", m.starts[len(m.starts)-1], true
		},
		gone: func(time.Time) bool { return false },
	})
	req.ReadArgs = openerArgs(uids, ages, self, other)
	req.Self = machineload.Self{PID: self, UID: uids[self]}
	req.Command = true
	req.Timeout = 30 * time.Minute
	req.Mark = m.mark
	n := 0
	req.OnBackoff = func([]int, time.Duration) { n++ }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || res.Backoffs != WaitBackoffCap || n != WaitBackoffCap || len(m.starts) != WaitBackoffCap {
		t.Fatalf("%+v after %d back-offs and %d marks; want open after standing back %d times, the last open unmarked", res, n, len(m.starts), WaitBackoffCap)
	}
}

// TestWaitPlainFormNeitherMarksNorLooksAgain: the plain form starts no command
// of its own, so it opens without the mark or the last look.
func TestWaitPlainFormNeitherMarksNorLooksAgain(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3})
	req.Mark = func() (func(), error) {
		t.Fatal("the plain form marked itself")
		return nil, nil
	}
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.Unmark != nil || !clock.Now().Equal(waitT0.Add(testJitter)) {
		t.Fatalf("%+v, err %v, done at %s; want open after the jitter alone, at %s", res, err, clock.Now(), waitT0.Add(testJitter))
	}
}

// TestSleepMarkShowsAsAChild reads the real process table: the default mark is
// a child of this process other than ps, which is what the table reads as a
// waiter running its command, and stopping it leaves no such child.
func TestSleepMarkShowsAsAChild(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the process table is read on macOS and Linux only")
	}
	children := func() []string {
		snap, err := machineload.Read()
		if err != nil || !snap.HasProcs {
			t.Fatalf("reading the machine: %v", err)
		}
		var out []string
		for _, p := range snap.Procs {
			if p.PPID == os.Getpid() && p.Name != "ps" {
				out = append(out, p.Name)
			}
		}
		return out
	}
	stop, err := sleepMark()
	if err != nil {
		t.Fatal(err)
	}
	if got := children(); !slices.Contains(got, "sleep") {
		stop()
		t.Fatalf("children %v while marked, want a sleep", got)
	}
	stop()
	if got := children(); slices.Contains(got, "sleep") {
		t.Fatalf("children %v once stopped, want no sleep", got)
	}
}
