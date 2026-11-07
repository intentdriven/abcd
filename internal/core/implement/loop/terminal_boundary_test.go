package loop

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// boundaryWorld records every point the run reaches, in order: each boundary, and
// each stage of a lane the lane's stages run, so "never inside a lane" is read from
// the order itself.
func boundaryWorld(t *testing.T, script ...string) (*terminalWorld, TerminalOptions, *[]string) {
	t.Helper()
	w := newTerminalWorld(t, script...)
	var order []string
	for i := range w.stages {
		stage := w.stages[i]
		run := stage.Run
		stage.Run = func(c Context, l *Lane) (Outcome, error) {
			order = append(order, "stage:"+string(stage.Name))
			return run(c, l)
		}
		w.stages[i] = stage
	}
	o := w.options(t)
	o.Stages = w.stages
	o.AtBoundary = func(context.Context) bool {
		order = append(order, "boundary")
		return false
	}
	return w, o, &order
}

// TestTheBoundaryFallsBetweenLanesAndNeverInsideOne is the "boundary only" row: the
// step is called once a move has been made, between lanes, and at no point between a
// lane's first stage and its last, and not before the run's first move, whose start
// has just proved the binary it is running.
func TestTheBoundaryFallsBetweenLanesAndNeverInsideOne(t *testing.T) {
	w, o, order := boundaryWorld(t, "shape", "shape", "shape")
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	if res.Ended != DrainStoppedCap && res.Ended != DrainStoppedEmpty {
		t.Fatalf("ended %q\n%s", res.Ended, w.said())
	}
	got := *order
	if len(got) == 0 || got[0] == "boundary" {
		t.Fatalf("the run's first move comes before any boundary: %v", got)
	}
	if !slices.Contains(got, "boundary") {
		t.Fatalf("no boundary between the lanes: %v", got)
	}
	lanes := 0
	inLane := false
	for _, step := range got {
		switch step {
		case "stage:" + string(StageWorktree):
			inLane = true
			lanes++
		case "stage:" + string(StageLand):
			inLane = false
		case "boundary":
			if inLane {
				t.Fatalf("a boundary inside a lane: %v", got)
			}
		}
	}
	if lanes < 2 {
		t.Fatalf("the row needs lanes to fall between: %v\n%s", got, w.said())
	}
	between := 0
	for i := 1; i < len(got); i++ {
		if got[i] == "boundary" && got[i-1] == "stage:"+string(StageLand) {
			between++
		}
	}
	if between == 0 {
		t.Fatalf("a boundary follows a lane's end: %v", got)
	}
}

// TestAWallThatDoesNotHoldEndsTheRunBeforeTheBoundary is the loop's half of the
// "stage cannot decide" row: with a boundary due, a check that fails ends the run, and no
// release is looked for on the way out.
func TestAWallThatDoesNotHoldEndsTheRunBeforeTheBoundary(t *testing.T) {
	w, o, order := boundaryWorld(t, "shape")
	o.BeforeStage = func() error { return errors.New("the binary changed while the drive ran") }
	tm := newTerminal(w.repo.Root(), o)
	tm.boundaryDue = true
	res, err := tm.run(context.Background())
	if err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	if res.Ended != TerminalWall {
		t.Fatalf("ended %q\n%s", res.Ended, w.said())
	}
	if slices.Contains(*order, "boundary") {
		t.Fatalf("a boundary after the wall failed: %v", *order)
	}
}

// TestAStepThatTookTimeSendsTheRunRoundTheChecksAgain: the step can hold the run for a
// fetch and a live launch, so a true sends it back to the stop, the bound and the walls
// before anything starts, and the same boundary is not taken twice.
func TestAStepThatTookTimeSendsTheRunRoundTheChecksAgain(t *testing.T) {
	w, o, order := boundaryWorld(t, "shape", "shape", "shape")
	checks := 0
	before := o.BeforeStage
	o.BeforeStage = func() error {
		checks++
		*order = append(*order, "check")
		if before != nil {
			return before()
		}
		return nil
	}
	o.AtBoundary = func(context.Context) bool {
		*order = append(*order, "boundary")
		return true
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	got := *order
	found := false
	for i, step := range got {
		if step != "boundary" {
			continue
		}
		found = true
		if i+1 >= len(got) || got[i+1] != "check" {
			t.Fatalf("the step's return is not followed by a check: %v", got)
		}
		if i+2 < len(got) && got[i+2] == "boundary" {
			t.Fatalf("one boundary taken twice: %v", got)
		}
	}
	if !found {
		t.Fatalf("no boundary reached: %v", got)
	}
}

// TestSkipsReappliedAtAResumeHoldOneEntry is the skip half of the "resume" row: the same
// arguments given again change what stands to the same thing.
func TestSkipsReappliedAtAResumeHoldOneEntry(t *testing.T) {
	var st DrainState
	d := DrainOptions{Skip: []string{"iss-1", "internal/core"}, SkipReason: "held"}
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for range 2 {
		if _, err := applySkips(&st, d, at); err != nil {
			t.Fatal(err)
		}
	}
	if len(st.Skips) != 2 || st.Skips[0].Target != "iss-1" || st.Skips[0].Reason != "held" {
		t.Fatalf("the skip list: %+v", st.Skips)
	}
}

// TestMovedFallsAfterEachMoveAndBeforeItsBoundary: the surface reads this as the run having
// moved on the binary it runs.
func TestMovedFallsAfterEachMoveAndBeforeItsBoundary(t *testing.T) {
	w, o, order := boundaryWorld(t, "shape", "shape")
	o.Moved = func() { *order = append(*order, "moved") }
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	got := *order
	if len(got) == 0 || got[0] != "moved" {
		t.Fatalf("the first thing the run reaches is its first move: %v", got)
	}
	first := slices.Index(got, "boundary")
	if first < 0 || !slices.Contains(got[:first], "moved") {
		t.Fatalf("a boundary with no move before it: %v", got)
	}
}

// TestARefusedFirstMoveIsNoMove: a drain that refuses has not moved.
func TestARefusedFirstMoveIsNoMove(t *testing.T) {
	w, o, order := boundaryWorld(t, "shape")
	o.Moved = func() { *order = append(*order, "moved") }
	o.Drain.Max = -1
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err == nil {
		t.Fatalf("the drain should refuse: %+v", res)
	}
	if slices.Contains(*order, "moved") {
		t.Fatalf("a refusal counted as a move: %v", *order)
	}
}

// lanesInProgress names the lanes the drain's state holds in progress.
func lanesInProgress(t *testing.T, root string) []string {
	t.Helper()
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	var open []string
	for _, l := range st.Lanes {
		run, err := ReadState(root, l.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if outcome, _, _ := laneOutcome(run); outcome == DrainLaneInProgress {
			open = append(open, l.Issue)
		}
	}
	return open
}

// TestNoBoundaryWhileARefusedStepLeavesItsLaneInProgress: a refused step comes back to the
// run with the lane still reserved, and the fetch and the proof would hold that reservation
// with nothing renewing it. The boundary stays due until the lane has ended.
func TestNoBoundaryWhileARefusedStepLeavesItsLaneInProgress(t *testing.T) {
	w, o, _ := boundaryWorld(t)
	w.noReceipt = true
	o.Drain.Max = 1
	o.Drain.Skip, o.Drain.SkipReason = []string{secondIssue, thirdIssue}, "the one lane's alone"
	w.guardSleeps(t, &o)
	boundaries := 0
	o.AtBoundary = func(context.Context) bool {
		boundaries++
		if open := lanesInProgress(t, w.repo.Root()); len(open) != 0 {
			t.Errorf("a boundary with %v in progress\n%s", open, w.said())
		}
		return false
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 1 {
		t.Fatalf("the row needs the lane handed back in the end: %+v\n%s", st.HandBacks, w.said())
	}
}

// TestABoundaryThatFallsDueInALaneStillComes: a due boundary waits for the lane, and is not
// lost to it.
func TestABoundaryThatFallsDueInALaneStillComes(t *testing.T) {
	w, o, order := boundaryWorld(t, "shape", "shape", "shape")
	tm := newTerminal(w.repo.Root(), o)
	tm.inHand, tm.boundaryDue = true, true
	if _, err := tm.run(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	if got := *order; len(got) == 0 || got[0] == "boundary" || !slices.Contains(got, "boundary") {
		t.Fatalf("the order: %v", got)
	}
}

// TestABoundaryComesAfterAHeldSkipHasBeenUsed: the skip a run holds an issue back with is
// taken by the next move. A process that moved before that move would drop it, and the
// judgement's hold is still in hand until then.
func TestABoundaryComesAfterAHeldSkipHasBeenUsed(t *testing.T) {
	w, o, _ := boundaryWorld(t, repeated("noshape", 3*terminalJudgeAsks)...)
	w.guardSleeps(t, &o)
	boundaries := 0
	o.AtBoundary = func(context.Context) bool {
		boundaries++
		st, _, err := readDrain(w.repo.Root())
		if err != nil {
			t.Fatal(err)
		}
		if st.Judging != nil {
			t.Errorf("a boundary with %s's judgement in hand\n%s", st.Judging.Issue, w.said())
		}
		for _, line := range w.lines {
			var held string
			if _, err := fmt.Sscanf(line, "holding %s back", &held); err != nil {
				continue
			}
			if !slices.ContainsFunc(st.Skips, func(s DrainSkip) bool { return s.Target == held }) {
				t.Errorf("a boundary before the skip of %s was used\n%s", held, w.said())
			}
		}
		return false
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	if len(res.HeldBack) == 0 || boundaries == 0 {
		t.Fatalf("held %v over %d boundaries\n%s", res.HeldBack, boundaries, w.said())
	}
}

// TestWhatARunHeldBackOutlivesItsProcess: the skip stands in the drain's state, and so what
// a run held back is reported by the process that goes on, however it came to go on. A
// process started afresh reports only its own.
func TestWhatARunHeldBackOutlivesItsProcess(t *testing.T) {
	first := func(t *testing.T) (*terminalWorld, TerminalOptions, []string) {
		t.Helper()
		w, o, _ := boundaryWorld(t, append(repeated("noshape", terminalJudgeAsks), "shape", "shape", "shape")...)
		w.guardSleeps(t, &o)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// The move to a new release ends this process at its first boundary.
		o.AtBoundary = func(context.Context) bool {
			cancel()
			return true
		}
		res, err := RunTerminal(ctx, w.repo.Root(), o)
		if err != nil {
			t.Fatalf("%v\n%s", err, w.said())
		}
		if len(res.HeldBack) != 1 {
			t.Fatalf("held %v\n%s", res.HeldBack, w.said())
		}
		o.AtBoundary = nil
		return w, o, res.HeldBack
	}
	t.Run("resumed", func(t *testing.T) {
		w, o, held := first(t)
		o.Resumed = true
		res, err := RunTerminal(context.Background(), w.repo.Root(), o)
		if err != nil {
			t.Fatalf("%v\n%s", err, w.said())
		}
		if !slices.Equal(res.HeldBack, held) {
			t.Fatalf("held %v, want %v\n%s", res.HeldBack, held, w.said())
		}
	})
	t.Run("started afresh", func(t *testing.T) {
		w, o, _ := first(t)
		res, err := RunTerminal(context.Background(), w.repo.Root(), o)
		if err != nil {
			t.Fatalf("%v\n%s", err, w.said())
		}
		if len(res.HeldBack) != 0 {
			t.Fatalf("held %v\n%s", res.HeldBack, w.said())
		}
	})
}

// TestAStuckLaneHoldsTheReleaseCheckForOneIntervalAtMost: a lane that keeps coming back
// refused under a bound would keep a release out for the whole run, and nothing would say
// so. The hold has a ceiling, and the wait says once that a release may be waiting.
func TestAStuckLaneHoldsTheReleaseCheckForOneIntervalAtMost(t *testing.T) {
	const ceiling = TerminalWaitFloor
	w, _, _ := boundaryWorld(t)
	w.noReceipt = true
	o := w.bounded(t, 30*24*time.Hour)
	o.Stages = w.stages
	o.BoundaryCeiling = ceiling
	began := w.clock.t
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var open []string
	var at time.Time
	o.AtBoundary = func(context.Context) bool {
		open, at = lanesInProgress(t, w.repo.Root()), w.clock.t
		cancel()
		return true
	}
	if _, err := RunTerminal(ctx, w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	if len(open) == 0 || at.Sub(began) < ceiling {
		t.Fatalf("a boundary at %v with %v in progress\n%s", at.Sub(began), open, w.said())
	}
	if got := strings.Count(w.said(), "a new release"); got != 1 {
		t.Fatalf("%d lines say a release may be waiting\n%s", got, w.said())
	}
}

// TestASkipTheRunGaveHoldsTheCheckWhateverTheCeiling: a process that moved before the next
// move took the skip would drop it.
func TestASkipTheRunGaveHoldsTheCheckWhateverTheCeiling(t *testing.T) {
	w, o, _ := boundaryWorld(t)
	o.BoundaryCeiling = time.Minute
	tm := newTerminal(w.repo.Root(), o)
	tm.inHand, tm.heldSince = true, w.clock.t.Add(-time.Hour)
	if tm.boundaryHeld() {
		t.Fatal("past its ceiling the hold ends")
	}
	tm.o.Drain.Skip = []string{eligibleIssue}
	if !tm.boundaryHeld() {
		t.Fatal("a skip not yet used holds it whatever the ceiling")
	}
	tm.inHand = false
	if !tm.boundaryHeld() {
		t.Fatal("a skip not yet used holds it with nothing in hand")
	}
}

// TestACheckPastItsCeilingRenewsWhatTheRunHolds: the fetch and the proof must not outlast
// the reservation of the lane they run beside.
func TestACheckPastItsCeilingRenewsWhatTheRunHolds(t *testing.T) {
	was := terminalRenewTick
	terminalRenewTick = 20 * time.Millisecond
	t.Cleanup(func() { terminalRenewTick = was })
	w, _, _ := boundaryWorld(t)
	w.noReceipt = true
	o := w.bounded(t, 30*24*time.Hour)
	o.Stages = w.stages
	o.BoundaryCeiling = TerminalWaitFloor
	var clock atomic.Int64
	clock.Store(w.clock.t.UnixNano())
	o.Options.Now = func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	o.Sleep = func(_ context.Context, d time.Duration) { clock.Add(int64(d)) }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	renewed := false
	o.AtBoundary = func(context.Context) bool {
		defer cancel()
		st, _, err := readDrain(w.repo.Root())
		if err != nil || len(st.Lanes) == 0 {
			t.Fatalf("no lane: %v", err)
		}
		runID := st.Lanes[len(st.Lanes)-1].RunID
		clock.Add(int64(90 * time.Minute))
		count := func() int {
			run, err := ReadState(w.repo.Root(), runID)
			if err != nil {
				return 0
			}
			n := 0
			for _, e := range run.Record {
				if e.Stage == StageReservationRenewed {
					n++
				}
			}
			return n
		}
		before := count()
		for range 400 {
			if count() > before {
				renewed = true
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
		return true
	}
	if _, err := RunTerminal(ctx, w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	if !renewed {
		t.Fatalf("the reservation was not renewed during the check\n%s", w.said())
	}
}
