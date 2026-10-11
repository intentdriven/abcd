package cli

import (
	"encoding/json"
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
	prevRead, prevArgs, prevNow, prevSleep := loadReader, loadArgsReader, loadNow, loadSleep
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
	t.Cleanup(func() { loadReader, loadArgsReader, loadNow, loadSleep = prevRead, prevArgs, prevNow, prevSleep })
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
		"load gate open after 30s: load 3.0 is below 16 (16 online cores), and no other go test -race run is live;",
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
	want := "load gate still shut after 1m0s: load 40.0 is not below 16 (16 online cores)"
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
	if res.Status != implement.WaitOpen || res.WaitedS != 15 || len(res.WaitedOn) != 1 || res.WaitedOn[0] != implement.WaitOnLoad || res.Hold == nil {
		t.Fatalf("result = %+v", res)
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
