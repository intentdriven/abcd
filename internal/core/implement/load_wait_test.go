package implement

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
}

// waitReq builds a request for the waiter at pid against polls (the last one
// repeats), under home, on clock.
func waitReq(home string, pid int, clock *fakeClock, polls ...poll) WaitRequest {
	var mu sync.Mutex
	i := 0
	args := waiterArgs(pid)
	return WaitRequest{
		Getenv: noEnv,
		Read: func() (machineload.Snapshot, error) {
			mu.Lock()
			defer mu.Unlock()
			p := polls[min(i, len(polls)-1)]
			i++
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
			out := map[int][]string{}
			for _, p := range pids {
				if a, ok := args[p]; ok {
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
	if res.Status != WaitOpen || res.Polls != 1 || len(res.WaitedOn) != 0 || len(res.RaceRuns) != 0 {
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
	if res.Status != WaitOpen || res.Polls != 3 || res.WaitedS != 30 {
		t.Fatalf("result = %+v, want open on the third poll after 30s", res)
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
	if res.WaitLoad != 16 || res.LimitSource != LimitsDefault {
		t.Fatalf("limit %v from %q, want the core count by default", res.WaitLoad, res.LimitSource)
	}
	if res.Hold != nil {
		t.Fatalf("a timed-out wait holds the gate: %+v", res.Hold)
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
	if err != nil || res.Status != WaitOpen || res.Polls != 1 {
		t.Fatalf("result %+v, err %v, want the dead holder's lock reclaimed on the first poll", res, err)
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
	if err != nil || res.Status != WaitOpen || res.Polls != 2 {
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
	if err != nil || res.Status != WaitOpen || res.WaitedS != 90 || !reflect.DeepEqual(res.WaitedOn, []string{WaitOnLock}) {
		t.Fatalf("result %+v, err %v, want open once the settle hold lapses at 90s", res, err)
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
