package loop

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestAMovesWaitIsTheEarliestItNamesNeverBelowTheFloor: the pace window, the
// cap wait and the wait on origin each name a time; the earliest stands, and
// no wait is shorter than the floor.
func TestAMovesWaitIsTheEarliestItNamesNeverBelowTheFloor(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time { v := now.Add(d); return &v }
	for _, c := range []struct {
		name string
		res  DrainResult
		want time.Duration
		ok   bool
	}{
		{"none", DrainResult{}, 0, false},
		{"pace", DrainResult{NextEligibleAt: at(20 * time.Minute)}, 20 * time.Minute, true},
		{"cap", DrainResult{WaitingUntil: at(15 * time.Minute)}, 15 * time.Minute, true},
		{"origin", DrainResult{OriginWait: &DrainOriginWait{NextProbeAt: now.Add(5 * time.Minute)}}, 5 * time.Minute, true},
		{"earliest", DrainResult{NextEligibleAt: at(20 * time.Minute), WaitingUntil: at(15 * time.Minute)}, 15 * time.Minute, true},
		{"floor", DrainResult{WaitingUntil: at(5 * time.Second)}, TerminalWaitFloor, true},
		{"past", DrainResult{NextEligibleAt: at(-time.Hour)}, TerminalWaitFloor, true},
	} {
		got, ok := moveWait(c.res, now)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: moveWait = %v %v, want %v %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

// TestTheLoopSleepsOutAMovesWaitWithTheLockFree: every eligible issue held by
// another makes the move wait; while it sleeps, the drain's lock is free, and
// once the hold is gone the run goes on.
func TestTheLoopSleepsOutAMovesWaitWithTheLockFree(t *testing.T) {
	w := newTerminalWorld(t)
	q := otherCopy(t, w.bare, quinnEmail)
	foundAt := map[string]string{eligibleIssue: "AGENTS.md", secondIssue: "commands/drain.md", thirdIssue: "commands/other.md"}
	for id, path := range foundAt {
		if _, err := claim.Reserve(q, claim.Request{Record: id, Footprint: []string{path}}, as(t, q, claim.Options{})); err != nil {
			t.Fatal(err)
		}
	}
	lockFree := false
	w.onSleep = func(time.Duration) {
		if lockFree {
			return
		}
		lock := filepath.Join(w.repo.Root(), filepath.FromSlash(drainLockRel))
		lockFree = fsutil.WithFileLock(lock, time.Second, func() error { return nil }) == nil
		for id := range foundAt {
			if _, err := claim.Reserve(q, claim.Request{Record: id, Release: true}, as(t, q, claim.Options{})); err != nil {
				t.Fatal(err)
			}
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if !lockFree {
		t.Fatal("the drain's lock is held only during a move")
	}
	if len(w.sleeps) == 0 || w.sleeps[0] != drainWaitMinutes*time.Minute {
		t.Fatalf("the loop sleeps until the time the move named: %v", w.sleeps)
	}
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 3 {
		t.Fatalf("the run goes on once the hold is gone: %+v", res)
	}
}
