package loop

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// sweepCalls installs an automatic sweep that records each repository it is
// asked to sweep and answers with do, restored when the test ends.
func sweepCalls(t *testing.T, do func(repoRoot string) string) *[]string {
	t.Helper()
	var calls []string
	t.Cleanup(SetAutoSweeper(func(_ context.Context, repoRoot string) string {
		calls = append(calls, repoRoot)
		return do(repoRoot)
	}))
	return &calls
}

// TestALandingThatSeesItsMergeSweepsOnce: the landing's last step, once its
// pull request has merged, sweeps on its own exactly once, after the step's
// save has written the lane done and its worktree and branch are removed, and
// the step's answer carries the sweep's line. A later step of the finished
// run sweeps nothing.
func TestALandingThatSeesItsMergeSweepsOnce(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	var seen State
	calls := sweepCalls(t, func(repoRoot string) string {
		st, err := ReadState(repoRoot, f.runID)
		if err != nil {
			t.Errorf("the sweep reads the run's state: %v", err)
		}
		seen = st
		if _, err := os.Stat(l.Worktree); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the sweep runs after the lane's own worktree is removed: %v", err)
		}
		return "abcd swept on its own: 2 removed or deleted, 0 saved as a patch and kept, 1 left as they were [removed  worktree  old]"
	})

	// Not merged yet: no sweep.
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err == nil {
		t.Fatal("an unmerged lane waits for its merge")
	}
	if len(*calls) != 0 {
		t.Fatalf("a landing still waiting on its merge swept %d times", len(*calls))
	}

	f.merged(t, l.HeadSHA)
	res := f.step(t)
	if res.PerformedStage != StageLand || !res.Complete {
		t.Fatalf("once merged, the landing completes: %+v", res)
	}
	if len(*calls) != 1 || (*calls)[0] != f.repo.Root() {
		t.Fatalf("the merged landing swept %q; want the checkout once", *calls)
	}
	if len(seen.Lanes) == 0 || seen.Lanes[0].Stage != StageDone {
		t.Fatalf("the sweep ran before the step's save wrote the lane done: %+v", seen.Lanes)
	}
	if !strings.Contains(res.Next, "abcd swept on its own: 2 removed or deleted") || !strings.Contains(res.Next, "[removed  worktree  old]") {
		t.Fatalf("the step's answer does not carry the sweep's report: %q", res.Next)
	}

	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 {
		t.Fatalf("a step of the finished run swept again: %d calls", len(*calls))
	}
}

// TestASweepFaultLeavesTheLandingLanded: a sweep that panics is reported in
// the step's answer, and the landing still reads as landed: the lane done,
// the run complete, no error.
func TestASweepFaultLeavesTheLandingLanded(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	sweepCalls(t, func(string) string { panic("origin went away mid-sweep") })

	f.merged(t, l.HeadSHA)
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil {
		t.Fatalf("a sweep's fault failed the landing: %v", err)
	}
	if res.PerformedStage != StageLand || !res.Complete {
		t.Fatalf("the landing reads as landed: %+v", res)
	}
	if !strings.Contains(res.Next, "abcd's own sweep stopped on a fault (origin went away mid-sweep); the act it followed stands") {
		t.Fatalf("the step's answer does not report the sweep's fault: %q", res.Next)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil || len(st.Lanes) == 0 || st.Lanes[0].Stage != StageDone || st.Lanes[0].Landing == nil || st.Lanes[0].Landing.Merged == "" {
		t.Fatalf("the lane is recorded landed: %+v, %v", st.Lanes, err)
	}
}

// TestTheDrainsEndSweepsOnce: a move that ends the drain sweeps once, its line
// joining the move's answer; a drive's move that ends the drain leaves the
// sweep to the drive's own end.
func TestTheDrainsEndSweepsOnce(t *testing.T) {
	repo := drainRepo(t)
	calls := sweepCalls(t, func(string) string { return "abcd swept on its own: 0 removed or deleted" })
	res, err := Drain(repo.Root(), Options{}, DrainOptions{Skip: []string{secondIssue, eligibleIssue}, SkipReason: "held for the cut"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Complete {
		t.Fatalf("the drain did not end: %+v", res)
	}
	if len(*calls) != 1 || !strings.Contains(res.Next, "abcd swept on its own") {
		t.Fatalf("the drain's end swept %d times, answer %q; want once, reported", len(*calls), res.Next)
	}

	repo = drainRepo(t)
	*calls = nil
	res, err = Drain(repo.Root(), Options{}, DrainOptions{Skip: []string{secondIssue, eligibleIssue}, SkipReason: "held", driveEnds: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Complete || len(*calls) != 0 {
		t.Fatalf("a drive's ending move swept %d times; the drive sweeps at its own end", len(*calls))
	}
}

// TestADrivesEndSweepsUnlessKilled: a terminal drive that ends sweeps once
// and carries the line; one killed sweeps nothing.
func TestADrivesEndSweepsUnlessKilled(t *testing.T) {
	repo := drainRepo(t)
	calls := sweepCalls(t, func(string) string { return "abcd swept on its own: 1 removed or deleted" })
	var said []string
	term := newTerminal(repo.Root(), TerminalOptions{Say: func(s string) { said = append(said, s) }})
	term.ctx = context.Background()
	res := term.ended(TerminalPassedLanes)
	if len(*calls) != 1 || res.Swept != "abcd swept on its own: 1 removed or deleted" || !strings.Contains(strings.Join(said, "\n"), "abcd swept on its own") {
		t.Fatalf("calls %d, swept %q, said %q", len(*calls), res.Swept, said)
	}
	*calls = nil
	res = term.ended(TerminalKilled)
	if len(*calls) != 0 || res.Swept != "" {
		t.Fatalf("a killed drive swept %d times (%q)", len(*calls), res.Swept)
	}
}
