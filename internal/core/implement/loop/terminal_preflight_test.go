//go:build unix

package loop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// oneLane is a world held to one issue's lane.
func oneLane(t *testing.T, w *terminalWorld) TerminalOptions {
	t.Helper()
	o := w.options(t)
	o.Drain.Max = 1
	o.Drain.Skip, o.Drain.SkipReason = []string{secondIssue, thirdIssue}, "the one lane's alone"
	w.guardSleeps(t, &o)
	return o
}

// TestAFailingPreflightGoesToTheStuckCheck: a preflight that fails is the lane's
// refusal, not an outage: it reaches the stuck check within minutes, and the
// ledger goes on.
func TestAFailingPreflightGoesToTheStuckCheck(t *testing.T) {
	w := newTerminalWorld(t)
	w.noReceipt = true
	w.preflight = errors.New("exit status 2")
	o := oneLane(t, w)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if res.Ended == TerminalOutage {
		t.Fatalf("a failing preflight is no outage: %+v", res)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != HandBackStuck {
		t.Fatalf("the lane is handed back as stuck: %+v\n%s", st.HandBacks, strings.Join(w.lines, "\n"))
	}
	if got := slept(w.sleeps); got > 5*time.Minute {
		t.Fatalf("the ledger waits %v on one failing preflight", got)
	}
}

// TestAStopReachesAPreflight: one queued at the gate as much as one running ends
// when a stop is asked for, and the run ends on the stop.
func TestAStopReachesAPreflight(t *testing.T) {
	w := newTerminalWorld(t)
	w.needsPreflight = true
	o := oneLane(t, w)
	ended := make(chan bool, 1)
	o.Preflight = func(ctx context.Context, _ string) error {
		close(w.stop)
		select {
		case <-ctx.Done():
			ended <- true
			return ctx.Err()
		case <-time.After(10 * time.Second):
			ended <- false
			return errors.New("not stopped")
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if !<-ended {
		t.Fatal("the stop did not reach the preflight")
	}
	if res.Ended != TerminalStopped {
		t.Fatalf("the run ends on the stop: %+v", res)
	}
}

// TestAPreflightTheHookRefusesNeverRuns: a lane the drive's check refuses is
// handed back at once with the reason, and its preflight does not run.
func TestAPreflightTheHookRefusesNeverRuns(t *testing.T) {
	w := newTerminalWorld(t)
	w.needsPreflight = true
	o := oneLane(t, w)
	o.PreflightAllowed = func(string, string) error {
		return &PreflightRefused{Reason: "the lane changes what its gates run"}
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	if len(w.preflights) != 0 {
		t.Fatalf("the preflight ran: %q", w.preflights)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != HandBackStuck {
		t.Fatalf("the lane is handed back: %+v\n%s", st.HandBacks, strings.Join(w.lines, "\n"))
	}
}

// TestTheWallsAreCheckedBeforeAPreflight: a difference found as the preflight
// falls due stops the run, and the preflight does not run.
func TestTheWallsAreCheckedBeforeAPreflight(t *testing.T) {
	w := newTerminalWorld(t)
	w.needsPreflight = true
	o := oneLane(t, w)
	land := false
	for i := range w.stages {
		if w.stages[i].Name == StageLand {
			run := w.stages[i].Run
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				land = true
				return run(c, l)
			}
		}
	}
	checks := 0
	o.BeforeStage = func() error {
		if land {
			if checks++; checks > 0 {
				return errors.New("the walled directory's settings differ")
			}
		}
		return nil
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall || len(w.preflights) != 0 {
		t.Fatalf("the run stops before the preflight: %+v, %q", res, w.preflights)
	}
}

// TestARefusalNothingLogsHasADeadline: a step refused the same way for ever, as a
// wait by design is, ends the run at the deadline.
func TestARefusalNothingLogsHasADeadline(t *testing.T) {
	w := newTerminalWorld(t)
	for i := range w.stages {
		if w.stages[i].Name == StageBrief {
			w.stages[i].Run = func(_ Context, l *Lane) (Outcome, error) {
				r := refuse(string(StageBrief), "", l.ID, "origin could not be read", "wait")
				r.originUnread = true
				return Outcome{}, r
			}
		}
	}
	o := oneLane(t, w)
	// Each wait moves the clock on, so the deadline comes within a few moves.
	sleep := o.Sleep
	o.Sleep = func(ctx context.Context, d time.Duration) {
		sleep(ctx, d)
		w.clock.t = w.clock.t.Add(30 * time.Minute)
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWaited {
		t.Fatalf("the run ends at the deadline: %+v", res)
	}
}

// TestARunnerFailureIsNoUnansweredJudgement: a judge whose runner fails is an
// outage, so a run of them ends the run as one and holds nothing back.
func TestARunnerFailureIsNoUnansweredJudgement(t *testing.T) {
	w := newTerminalWorld(t, repeated("none", 80)...)
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalOutage || len(res.HeldBack) != 0 {
		t.Fatalf("an outage, nothing held back: %+v\n%s", res.HeldBack, strings.Join(w.lines, "\n"))
	}
}

// TestALimitedJudgeWaitsOutTheLimit: a judge that meets the account's limit is the
// limit's to wait for.
func TestALimitedJudgeWaitsOutTheLimit(t *testing.T) {
	w := newTerminalWorld(t, repeated("limited", 80)...)
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalLimited || len(res.HeldBack) != 0 {
		t.Fatalf("the limit ends the run, nothing held back: %+v %v\n%s", res.Ended, res.HeldBack, strings.Join(w.lines, "\n"))
	}
}

// TestAHeldBackIssueIsNamedAndKept: only a judge that ran and answered nothing
// holds an issue back; the result names it, a file outlasts the drain's skip list,
// and no claim on it stays.
func TestAHeldBackIssueIsNamedAndKept(t *testing.T) {
	w := newTerminalWorld(t, "noshape", "noshape", "noshape")
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if len(res.HeldBack) != 1 {
		t.Fatalf("one issue held back: %+v\n%s", res.HeldBack, strings.Join(w.lines, "\n"))
	}
	id := res.HeldBack[0]
	raw, err := os.ReadFile(filepath.Join(w.repo.Root(), filepath.FromSlash(HeldBackRel)))
	if err != nil || !strings.Contains(string(raw), id) {
		t.Fatalf("the fact is kept: %q %v", raw, err)
	}
	if r, ok := claimOf(t, w, id); ok && r.State == claim.StateLive {
		t.Fatalf("the issue held back keeps its claim: %+v", r)
	}
}

// claimOf reads an issue's reservation, if any.
func claimOf(t *testing.T, w *terminalWorld, id string) (claim.Reservation, bool) {
	t.Helper()
	snap, err := claim.View(w.repo.Root(), w.clock.t)
	if err != nil {
		t.Fatal(err)
	}
	return snap.Lookup(id)
}

// TestAStagesLineCannotRewriteTheRunState: what the driver wrote is compared after
// each stage, and a difference stops the run.
func TestAStagesLineCannotRewriteTheRunState(t *testing.T) {
	w := newTerminalWorld(t, "shape", "tamper")
	matches := func() string {
		st, _, err := readDrain(w.repo.Root())
		if err != nil || len(st.Lanes) == 0 {
			return ""
		}
		return filepath.Join(w.repo.Root(), filepath.FromSlash(StateRelPath(st.Lanes[0].RunID)))
	}
	o := oneLane(t, w)
	for i := range w.stages {
		if w.stages[i].Name == StageBrief {
			run := w.stages[i].Run
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				t.Setenv(terminalTamperEnv, matches())
				return run(c, l)
			}
		}
	}
	o.Options.RunStateWatch = true
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall {
		t.Fatalf("the run stops on the difference: %+v\n%s", res.Ended, strings.Join(w.lines, "\n"))
	}
}

// TestTheGateCheckRunsWhateverReceiptExists: a lane at its landing is checked even
// when no preflight is due.
func TestTheGateCheckRunsWhateverReceiptExists(t *testing.T) {
	w := newTerminalWorld(t)
	o := oneLane(t, w)
	o.PreflightAllowed = func(string, string) error {
		return &PreflightRefused{Reason: "the lane changes what its gates run"}
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != HandBackStuck || len(lanesDone(t, w.repo)) != 0 {
		t.Fatalf("the lane is handed back, not landed: %+v\n%s", st.HandBacks, strings.Join(w.lines, "\n"))
	}
}

// TestAnIssueTheDriveWouldRefuseIsPassedOver: one whose own path is a gate file is
// not opened, so no run is spent on it.
func TestAnIssueTheDriveWouldRefuseIsPassedOver(t *testing.T) {
	w := newTerminalWorld(t)
	o := w.options(t)
	o.Drain.Max = 1
	seen := []string{}
	o.Drain.PassOver = func(id, _ string) string {
		seen = append(seen, id)
		return "its path is a gate file; --trust-lane-gates opens it"
	}
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedEmpty || w.launches(t) != 0 || len(seen) == 0 {
		t.Fatalf("nothing opened: %+v, %d launches, %q", res.Ended, w.launches(t), seen)
	}
}

// TestABaseThatMovesBetweenStepsStopsTheRun: what a lane was cut from is held from
// the first read, so a rewrite of it does not change what judges the lane.
func TestABaseThatMovesBetweenStepsStopsTheRun(t *testing.T) {
	w := newTerminalWorld(t)
	o := oneLane(t, w)
	calls := 0
	o.BeforeStage = func() error {
		st, _, err := readDrain(w.repo.Root())
		if err != nil || len(st.Lanes) == 0 {
			return nil
		}
		if calls++; calls == 3 {
			path := filepath.Join(w.repo.Root(), filepath.FromSlash(StateRelPath(st.Lanes[0].RunID)))
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), driveBase) {
				t.Fatalf("the state holds no base to move:\n%s", raw)
			}
			moved := strings.ReplaceAll(string(raw), driveBase, strings.Repeat("e", len(driveBase)))
			if err := os.WriteFile(path, []byte(moved), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall {
		t.Fatalf("the run stops on the moved base: %+v\n%s", res.Ended, strings.Join(w.lines, "\n"))
	}
}
