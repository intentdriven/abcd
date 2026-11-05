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

// reservationOf reads the issue's reservation from the checkout as last
// fetched.
func reservationOf(t *testing.T, root, id string, now time.Time) claim.Reservation {
	t.Helper()
	snap, err := claim.View(root, now)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := snap.Lookup(id)
	if !ok {
		t.Fatalf("no reservation of %s", id)
	}
	return r
}

// stopOnLaunch closes the stop once n launches have started.
func (w *terminalWorld) stopOnLaunch(t *testing.T, n int) {
	t.Helper()
	path := filepath.Join(w.env.log, "claude.launches")
	stop := w.stop
	go func() {
		for range 2000 {
			raw, _ := os.ReadFile(path)
			if strings.Count(string(raw), "\n") >= n {
				close(stop)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
}

// stoppedLane is a run stopped while the first lane's implementer was in its
// call: the call finished, its receipt recorded.
func stoppedLane(t *testing.T) (*terminalWorld, TerminalResult) {
	t.Helper()
	w := newTerminalWorld(t, "shape", "slow")
	w.stopOnLaunch(t, 2)
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalStopped {
		t.Fatalf("the run ends on the stop: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	return w, res
}

// TestAStopLetsTheCallFinishAndReleasesTheLanesReservation: the agent in its
// call finishes, its receipt is recorded, the reservation is released on the
// remote, the lane stays on its branch, nothing is handed back, and the result
// names the run, the lane and the command that resumes it.
func TestAStopLetsTheCallFinishAndReleasesTheLanesReservation(t *testing.T) {
	w, res := stoppedLane(t)
	if res.Run == "" || res.Issue == "" || res.Lane == "" || !strings.Contains(res.Resume, "abcd drain --drive") {
		t.Fatalf("the result names the run, the lane and the resume: %+v", res)
	}
	run, err := ReadState(w.repo.Root(), res.Run)
	if err != nil {
		t.Fatal(err)
	}
	l := run.Lanes[0]
	if len(l.Receipts) != 1 {
		t.Fatalf("the call finished and its receipt is recorded: %+v", l.Receipts)
	}
	if l.HandBack != nil || l.Stage == StageDone {
		t.Fatalf("the lane stays on its branch, handed back to no one: %+v", l)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("the reservation is released: %+v", r)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 0 {
		t.Fatalf("nothing is handed back for a stop: %+v", st.HandBacks)
	}
}

// TestADriveOnTheSameCopyTakesTheLaneUpAndLandsIt: the resuming run reserves
// the issue again and lands the same run, opening no second one.
func TestADriveOnTheSameCopyTakesTheLaneUpAndLandsIt(t *testing.T) {
	w, res := stoppedLane(t)
	o := w.options(t)
	o.Stop = make(chan struct{})
	o.Drain.Max = 1
	next, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err == nil {
		t.Fatalf("the cap was set when the drain began: %+v", next)
	}
	o.Drain.Max = 0
	next, err = RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if next.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 3 {
		t.Fatalf("the run lands its lanes: %+v\n%s", next, strings.Join(w.lines, "\n"))
	}
	if n := runCount(t, w.repo); n != 3 {
		t.Fatalf("the stopped lane's run is the one that lands: %d runs", n)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateLive || r.Copy != claim.CopyThis {
		t.Fatalf("the issue was reserved again: %+v", r)
	}
}

// TestAPeersDriveTakesTheIssueUpAndTheStoppedCopyLetsItGo: the released issue
// is a peer's to land; the copy that stopped then finds it held, hands its own
// lane back as stuck, and goes on.
func TestAPeersDriveTakesTheIssueUpAndTheStoppedCopyLetsItGo(t *testing.T) {
	w, res := stoppedLane(t)
	q := otherCopy(t, w.bare, quinnEmail)
	o := w.options(t)
	o.Stop = make(chan struct{})
	o.Drain.Max = 1
	for _, id := range []string{eligibleIssue, secondIssue, thirdIssue} {
		if id != res.Issue {
			o.Drain.Skip = append(o.Drain.Skip, id)
		}
	}
	o.Drain.SkipReason = "the stopped lane's alone"
	var peer TerminalResult
	var err error
	asHomeOf(t, q, func() { peer, err = RunTerminal(context.Background(), q, o) })
	if err != nil {
		t.Fatal(err)
	}
	if peer.Ended != DrainStoppedCap {
		t.Fatalf("the peer lands the issue: %+v\n%s", peer, strings.Join(w.lines, "\n"))
	}
	st, _, err := readDrain(q)
	if err != nil || len(st.Lanes) != 1 || st.Lanes[0].Issue != res.Issue || st.Lanes[0].Outcome != DrainLaneDone {
		t.Fatalf("the peer's lane is the stopped issue's, landed: %+v %v", st.Lanes, err)
	}

	mine := w.options(t)
	mine.Stop = make(chan struct{})
	mine.Drain.Max = 0
	if _, err := RunTerminal(context.Background(), w.repo.Root(), mine); err != nil {
		t.Fatal(err)
	}
	back, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(w.lines, "\n"), "handed back") {
		t.Fatalf("the stopped copy lets the lane go: %+v\n%s", back.Lanes, strings.Join(w.lines, "\n"))
	}
	for _, l := range back.Lanes {
		if l.Issue == res.Issue && l.Outcome != DrainLaneHandedBack {
			t.Fatalf("its own lane is handed back: %+v", l)
		}
	}
}

// TestASecondSignalKillsTheProcessGroupAndNamesWhenTheReservationLapses: the
// context ending kills the agent and its child at once; nothing is released,
// so the result names when the reservation lapses.
func TestASecondSignalKillsTheProcessGroupAndNamesWhenTheReservationLapses(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type out struct {
		res TerminalResult
		err error
	}
	done := make(chan out, 1)
	o := w.options(t)
	o.Runners.Timeout = 40 * time.Second
	go func() {
		res, err := RunTerminal(ctx, w.repo.Root(), o)
		done <- out{res, err}
	}()
	child := waitPid(t, w.env, "claude.child.pid")
	leader := waitPid(t, w.env, "claude.pid")
	cancel()
	var got out
	select {
	case got = <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("the run did not return once its context ended")
	}
	if got.err != nil {
		t.Fatal(got.err)
	}
	if !gone(leader, 5*time.Second) || !gone(child, 5*time.Second) {
		t.Fatal("the agent's process group outlived the signal")
	}
	if got.res.Ended != TerminalKilled || got.res.Lapses == nil || got.res.Run == "" {
		t.Fatalf("the result names the run and when the reservation lapses: %+v", got.res)
	}
	if r := reservationOf(t, w.repo.Root(), got.res.Issue, w.clock.t); r.State != claim.StateLive {
		t.Fatalf("a kill releases nothing: %+v", r)
	}
}

// TestAStopThatBreaksAStepEndsAsAStop: the terminal's interrupt also reaches the
// git a step was running, which fails the step; the run still ends as the stop
// it is, releasing the lane, and not as a refusal.
func TestAStopThatBreaksAStepEndsAsAStop(t *testing.T) {
	w := newTerminalWorld(t)
	for i := range w.stages {
		if w.stages[i].Name == StageBrief {
			w.stages[i].Run = func(Context, *Lane) (Outcome, error) {
				close(w.stop)
				return Outcome{}, errors.New("git: signal: interrupt")
			}
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatalf("a stop is no refusal: %v", err)
	}
	if res.Ended != TerminalStopped || res.Run == "" {
		t.Fatalf("the run ends on the stop, naming the lane: %+v", res)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("the reservation is released: %+v", r)
	}
}

// TestALaneADrainLeftInProgressAsItEndedIsLetGoAndNamed: a drain that ended
// with a lane in progress, as a lost connection ends one, is no longer live; the
// lane's reservation is still released and the lane still named.
func TestALaneADrainLeftInProgressAsItEndedIsLetGoAndNamed(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o := w.options(t)
	o.Runners.Timeout = 40 * time.Second
	done := make(chan TerminalResult, 1)
	go func() {
		res, _ := RunTerminal(ctx, w.repo.Root(), o)
		done <- res
	}()
	waitPid(t, w.env, "claude.pid")
	cancel()
	killed := <-done
	if killed.Run == "" {
		t.Fatalf("the kill names the lane: %+v", killed)
	}
	st, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	ended := w.clock.t
	st.Stopped, st.EndedAt = DrainStoppedOutage, &ended
	if err := writeDrain(w.repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	if _, live, _ := readDrain(w.repo.Root()); live {
		t.Fatal("the drain has ended")
	}
	again := w.options(t)
	close(w.stop)
	again.Stop = w.stop
	res, err := RunTerminal(context.Background(), w.repo.Root(), again)
	if err != nil {
		t.Fatal(err)
	}
	if res.Run != killed.Run || res.Resume == "" {
		t.Fatalf("the lane is named: %+v", res)
	}
	if r := reservationOf(t, w.repo.Root(), killed.Issue, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("the reservation is released: %+v", r)
	}
}
