//go:build unix

package loop

import (
	"context"
	"errors"
	"slices"
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

// TestAnIssueItsJudgeNeverAnswersIsHeldBack: after three unanswered judgements
// the drive holds that issue back and goes on to the next.
func TestAnIssueItsJudgeNeverAnswersIsHeldBack(t *testing.T) {
	w := newTerminalWorld(t, "none", "none", "none")
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the next issue's lane lands: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if !strings.Contains(strings.Join(w.lines, "\n"), "holding") {
		t.Fatalf("the run says it held the issue back: %q", w.lines)
	}
	st, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{eligibleIssue, secondIssue, thirdIssue} {
		if !slices.ContainsFunc(st.Skips, func(s DrainSkip) bool { return s.Target == id }) {
			continue
		}
		snap, err := claim.View(w.repo.Root(), w.clock.t)
		if err != nil {
			t.Fatal(err)
		}
		if r, ok := snap.Lookup(id); ok && r.State == claim.StateLive {
			t.Fatalf("the issue held back keeps its claim: %+v", r)
		}
	}
}
