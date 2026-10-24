package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// waitFixture points `implement load --wait` at a fake machine whose readings
// follow polls (the last repeats) and a fake clock that sleeps by advancing, in
// a committed repository under a temporary HOME. Every go process beside the
// caller's chain is a race run. It returns the clock's start.
func waitFixture(t *testing.T, polls ...machineload.Snapshot) time.Time {
	t.Helper()
	loadFixture(t, polls[0], nil)
	var mu sync.Mutex
	i := 0
	t0 := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	now := t0
	prevRead, prevArgs, prevNow, prevSleep, prevJitter := loadReader, loadArgsReader, loadNow, loadSleep, loadJitter
	loadReader = func() (machineload.Snapshot, error) {
		mu.Lock()
		defer mu.Unlock()
		s := polls[min(i, len(polls)-1)]
		i++
		return s, nil
	}
	loadArgsReader = func(pids []int) (map[int][]string, error) {
		out := map[int][]string{}
		for _, p := range pids {
			out[p] = []string{"test", "-race", "./..."}
		}
		return out, nil
	}
	loadNow = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	loadSleep = func(d time.Duration) { mu.Lock(); defer mu.Unlock(); now = now.Add(d) }
	loadJitter = func() time.Duration { return 20 * time.Second }
	t.Cleanup(func() {
		loadReader, loadArgsReader, loadNow, loadSleep, loadJitter = prevRead, prevArgs, prevNow, prevSleep, prevJitter
	})
	return t0
}

// liveRace is a live `go test -race` by another session.
var liveRace = machineload.Proc{PID: 70000, PPID: 1, PGID: 70000, UID: 501, Age: time.Minute, Name: "go"}

// TestImplementLoadWaitOpensOnceTheRaceRunExits: the waiter says once that it is
// waiting and on what, opens when the race run exits, and takes the hold.
func TestImplementLoadWaitOpensOnceTheRaceRunExits(t *testing.T) {
	waitFixture(t, machine(3, 3, 3, liveRace), machine(3, 3, 3, liveRace), machine(3, 3, 3))
	code, out, errOut := implementCLI(t, "implement", "load", "--wait")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if n := strings.Count(out, "waiting for capacity:"); n != 1 {
		t.Fatalf("%d waiting lines, want 1:\n%s", n, out)
	}
	for _, want := range []string{
		"waiting for capacity: a go test -race run is live (pid 70000)",
		"load gate open after 50s: load 3.0 is below 32 (2 x 16 online cores), and no other go test -race run, make preflight or waiter ahead is live;",
		"held for 2m0s, so one waiter starts at a time",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
	home, _ := os.UserHomeDir()
	if _, err := os.Stat(abcdhome.Path(home, implement.WaitHoldFileName)); err != nil {
		t.Fatalf("no hold written: %v", err)
	}
}

// TestImplementLoadWaitTimeoutExitsNonZero: a load that never falls ends the
// wait at --timeout with a non-zero exit naming what it still waited on.
func TestImplementLoadWaitTimeoutExitsNonZero(t *testing.T) {
	waitFixture(t, machine(40, 40, 40))
	code, out, _ := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 1 {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	want := "load gate still shut after 1m0s: load 40.0 is not below 32 (2 x 16 online cores)"
	if !strings.Contains(out, want) {
		t.Fatalf("output lacks %q:\n%s", want, out)
	}
}

// TestImplementLoadWaitJSON reports what it waited on and for how long, and
// prints no waiting line into the JSON.
func TestImplementLoadWaitJSON(t *testing.T) {
	waitFixture(t, machine(40, 40, 40), machine(4, 4, 4))
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	var res implement.WaitResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not one JSON document: %v\n%s", err, out)
	}
	if res.Status != implement.WaitOpen || res.WaitedS != 35 || len(res.WaitedOn) != 1 || res.WaitedOn[0] != implement.WaitOnLoad || res.Hold == nil {
		t.Fatalf("result = %+v", res)
	}
	// An open after a wait carries nothing still shut: the last shut look's
	// conditions are history, in waited_on, not the gate's state.
	var raw map[string]any
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatal(err)
	}
	if still, ok := raw["still"]; ok {
		t.Fatalf("an open gate reports still %v:\n%s", still, out)
	}
}

// TestImplementLoadWaitRunsTheCommandAndReleases: the command form runs its
// command once the gate opens, exits with the command's code, and releases its
// hold when the command is done.
func TestImplementLoadWaitRunsTheCommandAndReleases(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", "echo gated-command-ran; exit 3")
	if code != 3 {
		t.Fatalf("exit %d, want the command's 3\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out, "gated-command-ran") {
		t.Fatalf("the command did not run:\n%s", out)
	}
	home, _ := os.UserHomeDir()
	if _, err := os.Stat(abcdhome.Path(home, implement.WaitHoldFileName)); !os.IsNotExist(err) {
		t.Fatalf("the hold outlived its command: %v", err)
	}
}

// TestImplementLoadWaitRefusals: --wait takes no --site, --timeout needs --wait,
// a command goes after --, and a negative timeout is refused, each at exit 2.
func TestImplementLoadWaitRefusals(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	for _, args := range [][]string{
		{"implement", "load", "--wait", "--site", "preflight"},
		{"implement", "load", "--site", "preflight", "--timeout", "1m"},
		{"implement", "load", "--wait", "make", "preflight"},
		{"implement", "load", "--site", "preflight", "--", "make"},
		{"implement", "load", "--wait", "--timeout", "-1m"},
	} {
		if code, out, errOut := implementCLI(t, args...); code != 2 {
			t.Errorf("%v: exit %d, want 2\n%s%s", args, code, out, errOut)
		}
	}
}

// TestImplementLoadWaitRefusesASymlinkedHome: a symlinked ~/.abcd.noindex is
// refused with the sentence every home writer uses, and nothing is waited for.
func TestImplementLoadWaitRefusesASymlinkedHome(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	home, _ := os.UserHomeDir()
	if err := os.RemoveAll(abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(home, abcdhome.Rel())); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := implementCLI(t, "implement", "load", "--wait")
	if code == 0 || !strings.Contains(out+errOut, "is a symlink, which abcd refuses rather than follows") {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
}

// TestImplementLoadWaitNamesAnotherAccountsPreflight: the waiter names another
// account's make preflight, read from the process table, as what it waits on;
// once the machine is free it says it checks again after the jitter, once, and
// then opens.
func TestImplementLoadWaitNamesAnotherAccountsPreflight(t *testing.T) {
	preflight := machineload.Proc{PID: 80000, PPID: 1, PGID: 80000, UID: 502, Age: 3 * time.Minute, Name: "make"}
	waitFixture(t, machine(3, 3, 3, preflight), machine(3, 3, 3))
	loadArgsReader = func(pids []int) (map[int][]string, error) {
		out := map[int][]string{}
		for _, p := range pids {
			if p == 80000 {
				out[p] = []string{"preflight"}
			}
		}
		return out, nil
	}
	code, out, errOut := implementCLI(t, "implement", "load", "--wait")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	for _, want := range []string{
		"waiting for capacity: another make preflight is running (pid 80000)",
		"load gate: the machine looks free; checking everything again in 20s before opening",
		"load gate open after 35s:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "checking everything again"); n != 1 {
		t.Fatalf("%d settle lines, want 1:\n%s", n, out)
	}
}

// TestImplementLoadWaitMarksUntilTheCommandStarts: the command form shows in
// the process table as running its command from the moment it opens, by its
// mark, and stops the mark once the command has started.
func TestImplementLoadWaitMarksUntilTheCommandStarts(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	marked, stopped := 0, 0
	prev := loadMark
	loadMark = func() (func(), error) {
		marked++
		return func() { stopped++ }, nil
	}
	t.Cleanup(func() { loadMark = prev })
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", "exit 0")
	if code != 0 || marked != 1 || stopped != 1 {
		t.Fatalf("exit %d, marked %d, stopped %d; want 0, one mark, stopped once\n%s%s", code, marked, stopped, out, errOut)
	}
}

// TestImplementLoadWaitSaysWhenItStandsBack: a waiter whose last look finds
// another waiter opened in the same moment, ahead of it by start, says so,
// names it, and opens later.
func TestImplementLoadWaitSaysWhenItStandsBack(t *testing.T) {
	opener := []machineload.Proc{
		{PID: 40001, PPID: 1, PGID: 40001, UID: 502, Age: time.Hour, Name: "abcd"},
		{PID: 40002, PPID: 40001, PGID: 40001, UID: 502, Name: "sleep"},
	}
	free := machine(3, 3, 3)
	waitFixture(t, free, free, machine(3, 3, 3, opener...), free)
	loadArgsReader = func(pids []int) (map[int][]string, error) {
		out := map[int][]string{}
		for _, p := range pids {
			if p == 40001 {
				out[p] = []string{"implement", "load", "--wait", "--", "make", "preflight"}
			}
		}
		return out, nil
	}
	prev := loadMark
	loadMark = func() (func(), error) { return func() {}, nil }
	t.Cleanup(func() { loadMark = prev })
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", "exit 0")
	if code != 0 || !strings.Contains(out, "standing back") || !strings.Contains(out, "pid 40001") {
		t.Fatalf("exit %d; want 0 and a line standing back from pid 40001\n%s%s", code, out, errOut)
	}
}

// TestImplementLoadWaitSaysAQuietHolder: a command hold whose holder's tree
// has used little CPU for the quiet span is named again in the waiting line,
// as a fact, with its pid, checkout and how long it has held the slot, and
// the wait still times out behind it: the gate says, it never stops the holder
// (iss-2610241419324875).
func TestImplementLoadWaitSaysAQuietHolder(t *testing.T) {
	t0 := waitFixture(t, machine(3, 3, 3))
	opened := t0.Add(-time.Minute)
	home, _ := os.UserHomeDir()
	raw, _ := json.Marshal(implement.WaitHolder{PID: 777, StartUnix: opened.Unix(), Opened: opened,
		Until: opened.Add(implement.WaitCommandCap), Form: implement.HoldCommand, Dir: "/work/repo"})
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, implement.WaitHoldFileName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	// The holder and its make age with the clock and use no CPU.
	loadReader = func() (machineload.Snapshot, error) {
		aged := loadNow().Sub(t0)
		return machine(3, 3, 3,
			machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Minute + aged, Name: "abcd"},
			machineload.Proc{PID: 778, PPID: 777, PGID: 777, UID: 501, Age: time.Minute + aged, CPU: time.Second, Name: "make"}), nil
	}
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "12m")
	if code != 1 {
		t.Fatalf("exit %d, want 1 (timed out)\n%s%s", code, out, errOut)
	}
	if n := strings.Count(out, "waiting for capacity:"); n != 2 {
		t.Fatalf("%d waiting lines, want 2 (the wait, then the quiet holder):\n%s", n, out)
	}
	for _, want := range []string{
		"another waiter holds the gate (pid 777, until",
		"; the holder (pid 777, /work/repo) has held the test slot for 11m0s using little CPU (under 10% of a core for the last 10m0s); if it is yours and stuck, stop it by its own pid",
		"load gate still shut after 12m0s",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

// lockedBuffer is a writer a goroutine writes while the test reads it.
type lockedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.b.String() }

// TestStartQuietWatchSaysItsOwnQuiet drives the holder's own watch at a short
// interval against a quiet tree on macOS's reader, a fresh ps child of the
// holder at every read: the reads are left out, and the holder says once,
// on its own output, that it has held the slot using little CPU while a
// waiter queues.
func TestStartQuietWatchSaysItsOwnQuiet(t *testing.T) {
	var mu sync.Mutex
	t0 := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	now, n := t0, 0
	prevRead, prevArgs, prevNow, prevEvery := loadReader, loadArgsReader, loadNow, loadQuietEvery
	t.Cleanup(func() { loadReader, loadArgsReader, loadNow, loadQuietEvery = prevRead, prevArgs, prevNow, prevEvery })
	loadQuietEvery = time.Millisecond
	loadNow = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	loadReader = func() (machineload.Snapshot, error) {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(time.Minute)
		n++
		aged := now.Sub(t0)
		ps := machineload.Proc{PID: 9000 + n, PPID: 777, PGID: 777, UID: 501, CPU: 20 * time.Millisecond, Name: "ps"}
		snap := machine(3, 3, 3,
			machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Minute + aged, Name: "abcd"},
			machineload.Proc{PID: 778, PPID: 777, PGID: 777, UID: 501, Age: time.Minute + aged, CPU: time.Second, Name: "make"},
			machineload.Proc{PID: 60000, PPID: 1, PGID: 60000, UID: 501, Age: aged, Name: "abcd"},
			ps)
		snap.ReaderPID = ps.PID
		return snap, nil
	}
	loadArgsReader = func([]int) (map[int][]string, error) {
		return map[int][]string{60000: {"implement", "load", "--wait", "--", "make", "preflight"}}, nil
	}
	var out lockedBuffer
	stop := startQuietWatch(&out, implement.WaitHolder{PID: 777, Opened: t0, Form: implement.HoldCommand, Gate: implement.GateTests, Dir: "/work/repo"}, nil)
	for deadline := time.Now().Add(10 * time.Second); !strings.Contains(out.String(), "using little CPU") && time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
	}
	stop()
	got := out.String()
	if strings.Count(got, "using little CPU") != 1 {
		t.Fatalf("want the quiet said once:\n%s", got)
	}
	for _, want := range []string{"this command (pid 777, /work/repo) has held the test slot for", "another waiter queues (pid 60000)", "stop it by its own pid"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output lacks %q:\n%s", want, got)
		}
	}
}

// TestImplementLoadWaitReleasesItsHoldBeforeStoppingTheWatch: the holder's
// watch can be in the middle of a slow read of the table when the command
// ends, and the hold is released without waiting for that read.
func TestImplementLoadWaitReleasesItsHoldBeforeStoppingTheWatch(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	prevEvery, prevMark := loadQuietEvery, loadMark
	t.Cleanup(func() { loadQuietEvery, loadMark = prevEvery, prevMark })
	loadQuietEvery = 10 * time.Millisecond
	loadMark = func() (func(), error) { return func() {}, nil }
	home, _ := os.UserHomeDir()
	hold := abcdhome.Path(home, implement.WaitHoldFileName)
	started := filepath.Join(t.TempDir(), "started")
	read := loadReader
	var mu sync.Mutex
	released := map[bool]int{}
	loadReader = func() (machineload.Snapshot, error) {
		if _, err := os.Stat(started); err == nil {
			// The watch's read: slow until the hold is gone, as a ps run on a
			// loaded machine can be, or three seconds at most.
			gone := false
			for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
				if _, err := os.Stat(hold); errors.Is(err, os.ErrNotExist) {
					gone = true
					break
				}
			}
			mu.Lock()
			released[gone]++
			mu.Unlock()
		}
		return read()
	}
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", `touch "$1"; sleep 0.2`, "sh", started)
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if released[true] == 0 || released[false] != 0 {
		t.Fatalf("watch reads ending with the hold released / still held: %d / %d; want the hold released while a read was in flight", released[true], released[false])
	}
}

// TestImplementLoadWaitLinesShowEachHolder: the first waiting line names the
// holder's age and says it is not yet measured; the line a span later gives
// its tree's CPU over that span (iss-2610241457170255).
func TestImplementLoadWaitLinesShowEachHolder(t *testing.T) {
	t0 := waitFixture(t, machine(3, 3, 3))
	opened := t0.Add(-time.Minute)
	home, _ := os.UserHomeDir()
	raw, _ := json.Marshal(implement.WaitHolder{PID: 777, StartUnix: opened.Unix(), Opened: opened,
		Until: opened.Add(implement.WaitCommandCap), Form: implement.HoldCommand, Dir: "/work/repo"})
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, implement.WaitHoldFileName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	// The holder's make keeps a whole core busy.
	loadReader = func() (machineload.Snapshot, error) {
		aged := loadNow().Sub(t0)
		return machine(3, 3, 3,
			machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Minute + aged, Name: "abcd"},
			machineload.Proc{PID: 778, PPID: 777, PGID: 777, UID: 501, Age: time.Minute + aged, CPU: aged, Name: "make"}), nil
	}
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "11m")
	if code != 1 {
		t.Fatalf("exit %d, want 1 (timed out)\n%s%s", code, out, errOut)
	}
	if n := strings.Count(out, "waiting for capacity:"); n != 2 {
		t.Fatalf("%d waiting lines, want 2:\n%s", n, out)
	}
	for _, want := range []string{
		"; holding: pid 777 for 1m0s, not yet measured\n",
		"; holding: pid 777 for 11m0s, using 100% of a core over the last 10m0s\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}
