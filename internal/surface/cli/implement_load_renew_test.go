package cli

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// renewFixture swaps the waiter's renewal for renew, run every millisecond,
// and returns a channel that receives once after each round has been
// reported.
func renewFixture(t *testing.T, renew func(root string) ([]claim.Renewal, error)) <-chan struct{} {
	t.Helper()
	rounds := make(chan struct{}, 64)
	prevRenew, prevEvery, prevRound := loadRenew, loadRenewEvery, loadRenewRound
	loadRenew, loadRenewEvery = renew, time.Millisecond
	loadRenewRound = func() { rounds <- struct{}{} }
	t.Cleanup(func() { loadRenew, loadRenewEvery, loadRenewRound = prevRenew, prevEvery, prevRound })
	return rounds
}

// gateReads makes the n-th read of the machine, from the second on, wait for
// the (n-1)-th renewal round, so the wait lasts as long as the rounds the test
// needs, and no longer.
func gateReads(t *testing.T, rounds <-chan struct{}, need int) {
	t.Helper()
	var mu sync.Mutex
	reads, seen := 0, 0
	prev := loadReader
	loadReader = func() (machineload.Snapshot, error) {
		mu.Lock()
		reads++
		for seen < min(reads-1, need) {
			select {
			case <-rounds:
				seen++
			case <-time.After(10 * time.Second):
				mu.Unlock()
				t.Error("no renewal round came while the waiter waited")
				return prev()
			}
		}
		mu.Unlock()
		return prev()
	}
	t.Cleanup(func() { loadReader = prev })
}

// TestImplementLoadWaitRenewsTheCallersHoldsWhileItWaits
// (iss-2610170926295770): while the waiter waits, its own reservations half
// through their lease are renewed and each renewal is said on standard error;
// a renewal that fails is said once, and the wait goes on and opens.
func TestImplementLoadWaitRenewsTheCallersHoldsWhileItWaits(t *testing.T) {
	waitFixture(t, machine(3, 3, 3, liveRace), machine(3, 3, 3, liveRace), machine(3, 3, 3))
	until := time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	var roots []string
	rounds := renewFixture(t, func(root string) ([]claim.Renewal, error) {
		mu.Lock()
		defer mu.Unlock()
		roots = append(roots, root)
		if len(roots) == 1 {
			return []claim.Renewal{{Record: "iss-5", Until: until}, {Record: "iss-6", Err: errors.New("the push was refused")}}, nil
		}
		return []claim.Renewal{{Record: "iss-6", Err: errors.New("the push was refused")}}, nil
	})
	gateReads(t, rounds, 2)
	code, out, errOut := implementCLI(t, "implement", "load", "--wait")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if want := "abcd implement load --wait: renewed your reservation on iss-5, half its lease having passed while this waits, until 2026-10-11T20:00:00Z\n"; !strings.Contains(errOut, want) {
		t.Errorf("standard error lacks %q:\n%s", want, errOut)
	}
	fail := "abcd implement load --wait: could not renew your reservation on iss-6 (the push was refused); the wait goes on\n"
	if n := strings.Count(errOut, fail); n != 1 {
		t.Errorf("the failed renewal is said %d times, want once until it succeeds:\n%s", n, errOut)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(roots) < 2 || roots[0] == "" {
		t.Fatalf("the renewal ran %d times, at %q; want at least two rounds in the checkout", len(roots), roots)
	}
}

// TestImplementLoadWaitNeverWaitsOnARenewal: a renewal that never returns,
// as a push stuck on the network would, does not delay the open, and after
// the open nothing more is said of it.
func TestImplementLoadWaitNeverWaitsOnARenewal(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	stuck := make(chan struct{})
	started := make(chan struct{}, 1)
	renewFixture(t, func(string) ([]claim.Renewal, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-stuck
		return []claim.Renewal{{Record: "iss-5", Until: time.Now()}}, nil
	})
	defer close(stuck)
	begin := time.Now()
	code, out, errOut := implementCLI(t, "implement", "load", "--wait")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if took := time.Since(begin); took > 5*time.Second {
		t.Fatalf("the gate opened after %s with a renewal stuck; want it not to wait on one", took)
	}
	if strings.Contains(errOut, "renew") {
		t.Errorf("a renewal is said on standard error:\n%s", errOut)
	}
}
