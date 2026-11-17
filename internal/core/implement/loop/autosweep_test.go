package loop

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
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

// TestADrivesEndSweepsUnlessKilledHaltedOrStopped: a terminal drive that ends
// sweeps once and carries the line for its end to print, saying nothing of it
// on its own, so the line is printed once; one killed, halted on a check or
// stopped by a person sweeps nothing.
func TestADrivesEndSweepsUnlessKilledHaltedOrStopped(t *testing.T) {
	repo := drainRepo(t)
	calls := sweepCalls(t, func(string) string { return "abcd swept on its own: 1 removed or deleted" })
	var said []string
	term := newTerminal(repo.Root(), TerminalOptions{Say: func(s string) { said = append(said, s) }})
	term.ctx = context.Background()
	res := term.ended(TerminalPassedLanes)
	if len(*calls) != 1 || res.Swept != "abcd swept on its own: 1 removed or deleted" || strings.Contains(strings.Join(said, "\n"), "abcd swept on its own") {
		t.Fatalf("calls %d, swept %q, said %q; want one sweep, its line carried and not said", len(*calls), res.Swept, said)
	}
	for _, why := range []string{TerminalKilled, TerminalHalted, TerminalStopped} {
		*calls = nil
		res = term.ended(why)
		if len(*calls) != 0 || res.Swept != "" {
			t.Fatalf("a drive ended %s swept %d times (%q)", why, len(*calls), res.Swept)
		}
	}
}

// TestTheAutomaticSweepIsBoundWellBelowALanesCommand: the sweep runs under a
// deadline of at most two minutes, the caller's context's values reach it,
// and a caller's context or a drive's stop already ended sweeps nothing.
func TestTheAutomaticSweepIsBoundWellBelowALanesCommand(t *testing.T) {
	type key struct{}
	var (
		got   context.Context
		calls int
	)
	t.Cleanup(SetAutoSweeper(func(ctx context.Context, _ string) string {
		calls++
		got = ctx
		return ""
	}))
	autoSweep(context.WithValue(context.Background(), key{}, "the step's"), nil, "repo")
	if calls != 1 || got.Value(key{}) != "the step's" {
		t.Fatalf("calls %d; the caller's context did not reach the sweep", calls)
	}
	if dl, ok := got.Deadline(); !ok || time.Until(dl) > 2*time.Minute {
		t.Fatalf("the sweep's deadline is %v away (set %v); want two minutes at most", time.Until(dl), ok)
	}
	stop := make(chan struct{})
	close(stop)
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	autoSweep(context.Background(), stop, "repo")
	autoSweep(ended, nil, "repo")
	if calls != 0 {
		t.Fatalf("a stopped drive or an ended context swept %d times", calls)
	}
}

// TestAStopWhileTheSweepRunsEndsItsContext: a person's stop of the drive that
// comes while the sweep runs ends the sweep's context, so it stops between
// removals rather than at its bound.
func TestAStopWhileTheSweepRunsEndsItsContext(t *testing.T) {
	stop := make(chan struct{})
	ended := false
	t.Cleanup(SetAutoSweeper(func(ctx context.Context, _ string) string {
		close(stop)
		select {
		case <-ctx.Done():
			ended = true
		case <-time.After(30 * time.Second):
		}
		return ""
	}))
	autoSweep(context.Background(), stop, "repo")
	if !ended {
		t.Fatal("a stop that came while the sweep ran did not end its context")
	}
}

// TestALandingsSweepTakesTheStepsContextAndADrivesStop: a landing driven
// through Drive sweeps under Drive's context; inside a drive whose person has
// stopped it, the landing sweeps nothing.
func TestALandingsSweepTakesTheStepsContextAndADrivesStop(t *testing.T) {
	type key struct{}
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	var got context.Context
	t.Cleanup(SetAutoSweeper(func(ctx context.Context, _ string) string { got = ctx; return "" }))
	f.merged(t, l.HeadSHA)
	ctx := context.WithValue(context.Background(), key{}, "the step's")
	res, err := Drive(ctx, f.repo.Root(), f.runID, f.stages, Options{}, Runners{})
	if err != nil || res.PerformedStage != StageLand {
		t.Fatalf("the landing: %+v, %v", res, err)
	}
	if got == nil || got.Value(key{}) != "the step's" {
		t.Fatal("the landing's sweep did not run under the step's context")
	}

	f = newLandFixture(t, queueRuleset("MERGE"))
	l = f.landedToArmedOnce(t)
	got = nil
	f.merged(t, l.HeadSHA)
	stop := make(chan struct{})
	close(stop)
	term := newTerminal(f.repo.Root(), TerminalOptions{Stop: stop})
	if _, err := advance(f.repo.Root(), f.runID, f.stages, term.o.Options); err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("a landing inside a stopped drive swept")
	}
}

// TestABuildThatClosesAMergedLaneSweeps: a build's resume that closes a lane
// whose pull request merged sweeps on its own once, after the closing, the
// sweep's line among the start's diagnostics.
func TestABuildThatClosesAMergedLaneSweeps(t *testing.T) {
	f, l := mergedLaneFixture(t)
	f.mergeCommit(t, l.Landing.Pushed)
	calls := sweepCalls(t, func(repoRoot string) string {
		if _, err := os.Stat(l.Worktree); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the sweep runs after the closed lane's worktree is removed: %v", err)
		}
		return "abcd swept on its own: 1 removed or deleted"
	})
	res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil || len(res.Closed) != 1 {
		t.Fatalf("the build closes the merged lane: %+v, %v", res.Closed, err)
	}
	if len(*calls) != 1 || !slices.Contains(res.Diagnostics, "abcd swept on its own: 1 removed or deleted") {
		t.Fatalf("calls %d, diagnostics %q; want one sweep, reported", len(*calls), res.Diagnostics)
	}
}

// TestASweepsPanicIsOneBoundedLine: a panic's text is cut to its bound and
// cleaned for one line, a line separator included.
func TestASweepsPanicIsOneBoundedLine(t *testing.T) {
	t.Cleanup(SetAutoSweeper(func(context.Context, string) string {
		panic(strings.Repeat("origin went away ", 400))
	}))
	line := autoSweep(context.Background(), nil, "repo")
	if !strings.HasPrefix(line, "abcd's own sweep stopped on a fault (") || strings.ContainsRune(line, ' ') || len([]rune(line)) > 2*autoSweepFaultCap {
		t.Fatalf("the panic's line runs %d runes: %q", len([]rune(line)), line)
	}
}
