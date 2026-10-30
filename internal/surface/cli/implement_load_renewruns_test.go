//go:build unix

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// runsFixture swaps the waiter's renewal of the runs for renew, run every
// millisecond with the caller's own reservations renewing nothing.
func runsFixture(t *testing.T, renew func(root string) []loop.RunRenewal) {
	t.Helper()
	renewFixture(t, func(string) ([]claim.Renewal, error) { return nil, nil })
	prev := loadRenewRuns
	loadRenewRuns = renew
	t.Cleanup(func() { loadRenewRuns = prev })
}

// TestImplementLoadWaitRenewsTheRunsWhileItsCommandRuns
// (iss-2610261837062463, dcn-2610291352512058): the waiter goes on renewing
// while its gated command runs, not only while it waits for the gate, the
// reservations of the runs it serves as well as the caller's own. Each run's
// renewal is said on standard error, and one that fails is said once until
// what is said of it changes. The command waits for a renewal round made
// after it started, and fails when none comes.
func TestImplementLoadWaitRenewsTheRunsWhileItsCommandRuns(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	dir := t.TempDir()
	started, seen := filepath.Join(dir, "started"), filepath.Join(dir, "seen")
	var mu sync.Mutex
	var roots []string
	fail := []loop.RunRenewal{{RunID: "run-1", Key: "itd-82", Note: "could not renew the reservation on itd-82: origin is unreachable"}}
	runsFixture(t, func(root string) []loop.RunRenewal {
		if _, err := os.Stat(started); err != nil {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		roots = append(roots, root)
		switch n := len(roots); {
		case n == 1:
			return []loop.RunRenewal{{RunID: "run-1", Key: "itd-82", Renewed: true,
				Note: "renewed the reservation on itd-82, half its lease having passed, until 2026-10-11T20:00:00Z"}}
		case n >= 4:
			if err := os.WriteFile(seen, nil, 0o600); err != nil {
				t.Error(err)
			}
		}
		return fail
	})
	script := `: > "$1"; i=0; while [ ! -e "$2" ] && [ $i -lt 1000 ]; do sleep 0.01; i=$((i+1)); done; [ -e "$2" ]`
	begin := time.Now()
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", script, "sh", started, seen)
	if code != 0 {
		t.Fatalf("the command saw no renewal round while it ran (exit %d after %s):\n%s%s", code, time.Since(begin), out, errOut)
	}
	if want := "abcd implement load --wait: run run-1: renewed the reservation on itd-82, half its lease having passed, until 2026-10-11T20:00:00Z\n"; !strings.Contains(errOut, want) {
		t.Errorf("standard error lacks %q:\n%s", want, errOut)
	}
	failed := "abcd implement load --wait: run run-1: could not renew the reservation on itd-82: origin is unreachable\n"
	if n := strings.Count(errOut, failed); n != 1 {
		t.Errorf("the failed renewal is said %d times, want once until it changes:\n%s", n, errOut)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(roots) == 0 || roots[0] == "" {
		t.Fatalf("the runs were renewed from %q; want the waiter's checkout", roots)
	}
}

// TestImplementLoadWaitLetsARenewalInFlightFinish (dcn-2610292238191496): a
// renewal round still in flight when the gated command exits, its push made
// and its state not yet written, is waited for before the waiter exits, so
// the renewal it made is said on standard error, not dropped unsaid.
func TestImplementLoadWaitLetsARenewalInFlightFinish(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	dir := t.TempDir()
	started, inFlight, exiting := filepath.Join(dir, "started"), filepath.Join(dir, "in-flight"), filepath.Join(dir, "exiting")
	runsFixture(t, func(string) []loop.RunRenewal {
		if _, err := os.Stat(started); err != nil {
			return nil
		}
		if _, err := os.Stat(inFlight); err == nil {
			return nil
		}
		if err := os.WriteFile(inFlight, nil, 0o600); err != nil {
			t.Error(err)
		}
		// The round is still in flight when the command exits, and goes on
		// a while after it.
		for range 1000 {
			if _, err := os.Stat(exiting); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		time.Sleep(500 * time.Millisecond)
		return []loop.RunRenewal{{RunID: "run-1", Key: "itd-82", Renewed: true,
			Note: "renewed the reservation on itd-82, its lease having lapsed, until 2026-10-11T20:00:00Z"}}
	})
	script := `: > "$1"; i=0; while [ ! -e "$2" ] && [ $i -lt 1000 ]; do sleep 0.01; i=$((i+1)); done; [ -e "$2" ] && : > "$3"`
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", script, "sh", started, inFlight, exiting)
	if code != 0 {
		t.Fatalf("no renewal round was in flight while the command ran (exit %d):\n%s%s", code, out, errOut)
	}
	if want := "abcd implement load --wait: run run-1: renewed the reservation on itd-82, its lease having lapsed, until 2026-10-11T20:00:00Z\n"; !strings.Contains(errOut, want) {
		t.Errorf("the renewal in flight when the command exited went unsaid; standard error lacks %q:\n%s", want, errOut)
	}
}

// TestImplementStatusRenewsTheRunsItShows (iss-2610261837062463,
// dcn-2610291352512058): before it renders, implement status renews a due
// reservation of each run it shows, those of the checkout it runs in or the
// one --run names, and its help no longer says it writes nothing.
func TestImplementStatusRenewsTheRunsItShows(t *testing.T) {
	repo := buildRepo(t)
	var res struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &res); err != nil || res.RunID == "" {
		t.Fatalf("build: %v %+v", err, res)
	}
	type call struct{ root, runID string }
	var calls []call
	prev := statusRenewRuns
	statusRenewRuns = func(root, runID string) []loop.RunRenewal {
		calls = append(calls, call{root, runID})
		return nil
	}
	t.Cleanup(func() { statusRenewRuns = prev })
	mustImplement(t, "implement", "status")
	mustImplement(t, "implement", "status", "--run", res.RunID, "--json")
	root, err := filepath.EvalSymlinks(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].runID != "" || calls[1].runID != res.RunID {
		t.Fatalf("status renewed %+v; want the checkout's runs, then the run --run names", calls)
	}
	for _, c := range calls {
		if got, _ := filepath.EvalSymlinks(c.root); got != root {
			t.Fatalf("status renewed the runs of %q; want the checkout %q", c.root, root)
		}
	}
	help := mustImplement(t, "implement", "status", "--help")
	if strings.Contains(help, "Read-only") || strings.Contains(help, "Writes nothing;") || !strings.Contains(help, "renews the") {
		t.Fatalf("the help says what status writes:\n%s", help)
	}
}
