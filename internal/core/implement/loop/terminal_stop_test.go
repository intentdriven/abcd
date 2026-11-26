//go:build unix

package loop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
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
	w.realWorktree()
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
	w.realWorktree()
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
	w.realWorktree()
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

// TestATakeUpAtTheCapWaitsAndHandsNothingBack: a stopped lane whose person now
// holds as many reservations as they may is waiting, as a drain at the cap
// waits, not refusing: the lane is not handed back as stuck, and once a
// reservation ends it is taken up and lands.
func TestATakeUpAtTheCapWaitsAndHandsNothingBack(t *testing.T) {
	w, res := stoppedLane(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"reservation": {"max_live": 1}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	root := w.repo.Root()
	o := w.options(t)
	o.Stop = make(chan struct{})
	o.Drain.Max = 0
	var others []string
	for _, id := range []string{eligibleIssue, secondIssue, thirdIssue} {
		if id != res.Issue {
			others = append(others, id)
		}
	}
	o.Drain.Skip, o.Drain.SkipReason = others, "the stopped lane's alone"
	other := "iss-2609230000000099"
	if _, err := claim.Reserve(root, claim.Request{Record: other, Footprint: []string{"docs/other.md"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	waits := 0
	w.onSleep = func(time.Duration) {
		if waits++; waits == 2 {
			if _, err := claim.Reserve(root, claim.Request{Record: other, Release: true}, claim.Options{}); err != nil {
				t.Fatal(err)
			}
		}
	}
	sleep := o.Sleep
	o.Sleep = func(ctx context.Context, d time.Duration) {
		if len(w.sleeps) > 50 {
			t.Fatalf("the wait did not end: %v", w.sleeps)
		}
		sleep(ctx, d)
	}
	if _, err := RunTerminal(context.Background(), root, o); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	back, _, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.HandBacks) != 0 || waits < 2 {
		t.Fatalf("the cap is waited out, nothing handed back: %d waits, %+v\n%s", waits, back.HandBacks, strings.Join(w.lines, "\n"))
	}
	for _, l := range back.Lanes {
		if l.Issue == res.Issue && l.Outcome != DrainLaneDone {
			t.Fatalf("the lane lands once the cap frees: %+v\n%s", l, strings.Join(w.lines, "\n"))
		}
	}
}

// TestAStopFromElsewhereEndsTheDriveAndLeavesItsClaimsToTheStop: a step refused
// at the stop's stage is no wait the drive sleeps through. It ends at once,
// naming the refusal and what lifts it, and releases nothing itself: what the run
// holds is the stop's to let go, and a save that failed holds it on purpose.
func TestAStopFromElsewhereEndsTheDriveAndLeavesItsClaimsToTheStop(t *testing.T) {
	w := newTerminalWorld(t)
	for i := range w.stages {
		if w.stages[i].Name == StageBrief {
			w.stages[i].Run = func(_ Context, l *Lane) (Outcome, error) {
				return Outcome{}, contend(StageStop, "", l.ID, "a stop set by another account holds", "only the person lifts it")
			}
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalHalted || res.Resume != "" || res.Run == "" {
		t.Fatalf("the run ends halted, naming the lane and no resume of its own: %+v", res)
	}
	said := strings.Join(w.lines, "\n")
	if !strings.Contains(said, "a stop set by another account holds") || !strings.Contains(said, "only the person lifts it") {
		t.Fatalf("the lines name the stop and what lifts it:\n%s", said)
	}
	if slices.Contains(w.sleeps, TerminalWaitFloor) {
		t.Fatalf("a stop is not slept through: %v", w.sleeps)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateLive {
		t.Fatalf("the drive releases nothing the stop holds: %+v", r)
	}
}

// TestALongWaitKeepsTheLanesReservation: a wait past the lease, the lane in
// progress, renews what the lane holds rather than leaving it to lapse.
func TestALongWaitKeepsTheLanesReservation(t *testing.T) {
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
	again := w.options(t)
	waiting := &terminal{repoRoot: w.repo.Root(), o: again}
	waiting.sleep(context.Background(), 5*time.Hour, "a long wait")
	if r := reservationOf(t, w.repo.Root(), killed.Issue, w.clock.t); r.State != claim.StateLive {
		t.Fatalf("the reservation stays live across the wait: %+v", r)
	}
	for _, d := range w.sleeps {
		if d > time.Hour {
			t.Fatalf("no stretch of a wait outlasts half a lease: %v", w.sleeps)
		}
	}
}

// TestAPushThatFailsAtTakeUpBacksOff: a push that fails while a stopped lane is
// taken up is passing trouble: the lane waits and goes on, and three quick
// refusals do not hand it back.
func TestAPushThatFailsAtTakeUpBacksOff(t *testing.T) {
	w, res := stoppedLane(t)
	hook := filepath.Join(w.bare, "hooks", "pre-receive")
	if err := os.MkdirAll(filepath.Dir(hook), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	waits := 0
	w.onSleep = func(time.Duration) {
		if waits++; waits == 2 {
			if err := os.Remove(hook); err != nil {
				t.Fatal(err)
			}
		}
	}
	o := w.options(t)
	o.Stop = make(chan struct{})
	o.Drain.Max = 0
	for _, id := range []string{eligibleIssue, secondIssue, thirdIssue} {
		if id != res.Issue {
			o.Drain.Skip = append(o.Drain.Skip, id)
		}
	}
	o.Drain.SkipReason = "the stopped lane's alone"
	w.guardSleeps(t, &o)
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	back, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(back.HandBacks) != 0 || waits < 2 {
		t.Fatalf("the failure is waited out: %d waits, %+v\n%s", waits, back.HandBacks, strings.Join(w.lines, "\n"))
	}
}

// An interrupt leaves every lane marked or reserved (spc-2611141617199772 step 2,
// itd-2611141610197753 criterion 10), against the terminal world and the stand-in process
// table of terminal_claim_test.go.

// stopLater is a drive typed in the first terminal after the drive before it has gone.
const stopLater = 7050

// stopLaterStart stands in the table a later start reads: the first terminal, with this
// drive in it, and every earlier drive gone.
func stopLaterStart(t *testing.T) {
	t.Helper()
	standIn(t, sessionname.Table{ReadAt: claimRead, Procs: []machineload.Proc{
		{PID: claimLogin, PPID: 1, Name: "login", Age: time.Hour},
		{PID: claimShell, PPID: claimLogin, Name: "-zsh", Age: 59 * time.Minute},
		{PID: stopLater, PPID: claimShell, Name: "abcd", Age: time.Minute},
	}}, claimShell, stopLater)
}

// stopKilled is a world whose first drive, started from the first terminal, was killed
// between two stages with its first lane in flight: the lane is in progress, no mark
// stands on its run and its reservation is live from this copy. It returns the run and
// the home the next drive reads session markers and a stop from.
func stopKilled(t *testing.T) (*terminalWorld, State, string) {
	t.Helper()
	w := newTerminalWorld(t)
	w.realWorktree()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	killed := false
	for i := range w.stages {
		if w.stages[i].Name == StageValidate {
			w.stages[i].Run = func(Context, *Lane) (Outcome, error) {
				if !killed {
					killed = true
					cancel()
				}
				return Outcome{Note: "validate done"}, nil
			}
		}
	}
	claimProcs(t, claimShell, claimFirst, claimLogin, claimShell, claimFirst)
	res, err := RunTerminal(ctx, w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalKilled || res.Run == "" {
		t.Fatalf("premise: the first drive is killed with a lane left: %+v\n%s", res, w.said())
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop != nil {
		t.Fatalf("premise: a kill leaves no mark: %+v", c.Stop)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateLive || r.Copy != claim.CopyThis {
		t.Fatalf("premise: a kill leaves the reservation live: %+v", r)
	}
	run, err := ReadState(w.repo.Root(), res.Run)
	if err != nil {
		t.Fatal(err)
	}
	home, _ := stopWorld(t, false)
	return w, run, home
}

// stopAtFirstWait runs the next start from the first terminal with a signal at its first
// wait, which is where a drive that cannot claim its lane's run waits.
func stopAtFirstWait(t *testing.T, w *terminalWorld, home string) TerminalResult {
	t.Helper()
	w.onSleep = func(time.Duration) {
		select {
		case <-w.stop:
		default:
			close(w.stop)
		}
	}
	defer func() { w.onSleep = nil }()
	res := claimStart(t, w, home)
	if res.Ended != TerminalStopped {
		t.Fatalf("the signal ends the start: %+v\n%s", res, w.said())
	}
	return res
}

// stopUnclaimable is a world whose lane's run the next drive cannot claim: its recorded
// driver, alive in a second terminal, took it over after the first drive was killed, and
// its file states version (an earlier abcd's, or this one's).
func stopUnclaimable(t *testing.T, version int) (*terminalWorld, State, string) {
	t.Helper()
	w, run, home := stopKilled(t)
	claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive, claimOtherLogin, claimOtherShell)
	claimDriverIs(t, w.repo.Root(), run.RunID, claimOtherLogin)
	if version != SchemaVersion {
		stateAtVersion(t, w.repo.Root(), run.RunID, version)
	}
	return w, run, home
}

// TestAnInterruptAppliesTheStopToAnEarlierVersionsRunItsTerminalDrove: the run's file
// states an earlier version and its recorded driver is the terminal the drive is typed in,
// so the drive's claim takes it and the drive is its driver; a signal then applies the
// stop: the run is marked and halted, what its lane left is saved, and its reservation
// goes with the stop.
func TestAnInterruptAppliesTheStopToAnEarlierVersionsRunItsTerminalDrove(t *testing.T) {
	w, run, home := stopKilled(t)
	claimNextStart(t)
	claimDriverIs(t, w.repo.Root(), run.RunID, claimLogin)
	stateAtVersion(t, w.repo.Root(), run.RunID, schemaVersionPrevious)
	claimHoldAtLand(w, func() { close(w.stop) })
	res := claimStart(t, w, home)
	if res.Ended != TerminalStopped {
		t.Fatalf("the signal ends the start: %+v\n%s", res, w.said())
	}
	c := controlOf(t, w.repo.Root(), run.RunID)
	if c.Stop == nil || c.Stop.FromFile || c.Stop.HaltedAt == nil || c.Driver == nil || c.Driver.PID == claimLogin {
		t.Fatalf("the stop is not applied as the run's driver: %+v %+v\n%s", c.Stop, c.Driver, w.said())
	}
	if !strings.Contains(w.said(), run.RunID+": halted, and what its lanes left is saved") || strings.Contains(w.said(), "could not be applied") {
		t.Fatalf("the drive does not say the lane's work is saved:\n%s", w.said())
	}
	if r := reservationOf(t, w.repo.Root(), run.Key, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("the stop did not let the reservation go: %+v", r)
	}
}

// TestAnInterruptKeepsTheHoldOfARunItCouldNotClaim is the lab's interrupt fixture at an
// earlier version (F-16): the drive holds the lane of a run it could not claim, so the
// stop cannot be saved on it. The drive writes no mark and releases nothing: it says when
// the reservation lapses, and the next start, the other terminal gone, reads the run as
// reserved and lands it without reserving it again.
func TestAnInterruptKeepsTheHoldOfARunItCouldNotClaim(t *testing.T) {
	w, run, home := stopUnclaimable(t, schemaVersionPrevious)
	root := w.repo.Root()
	res := stopAtFirstWait(t, w, home)
	if c := controlOf(t, root, run.RunID); c.Stop != nil {
		t.Fatalf("a mark was written on a run the drive could not claim: %+v", c.Stop)
	}
	r := reservationOf(t, root, run.Key, w.clock.t)
	if r.State != claim.StateLive || r.Copy != claim.CopyThis {
		t.Fatalf("the reservation was let go: %+v\n%s", r, w.said())
	}
	after, err := ReadState(root, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if letGoByDrive(after) || !after.holdsAny() {
		t.Fatalf("the run reads as let go: %+v", after.Record)
	}
	at := stamp(r.EffectiveExpiry)
	if !strings.Contains(w.said(), run.Key+": its reservation is kept") || !strings.Contains(w.said(), "lapses at "+at) {
		t.Fatalf("the drive does not say the reservation is kept and when it lapses (%s):\n%s", at, w.said())
	}
	if res.Run != run.RunID || res.Lapses == nil || !res.Lapses.Equal(r.EffectiveExpiry) {
		t.Fatalf("the end does not name when the lane's reservation lapses: %+v", res)
	}

	claimNextStart(t)
	next := claimStart(t, w, home)
	if next.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 3 {
		t.Fatalf("the next start lands the three lanes: %+v %v\n%s", next, lanesDone(t, w.repo), w.said())
	}
	if slices.ContainsFunc(w.lines, func(l string) bool { return strings.Contains(l, "took up") && strings.Contains(l, run.RunID) }) {
		t.Fatalf("the next start reserved again a run it should read as reserved:\n%s", w.said())
	}
}

// TestAnInterruptKeepsEveryMembersHold: a run with members whose stop cannot be saved keeps
// the hold of its lead and of every member, each named with when it lapses.
func TestAnInterruptKeepsEveryMembersHold(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive, claimOtherLogin, claimOtherShell)
	claimDriverIs(t, repo.Root(), runID, claimOtherLogin)
	stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)
	var lines []string
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}, Say: func(s string) { lines = append(lines, s) }})
	tm.haltOnSignal()
	tm.letGo()
	said := strings.Join(lines, "\n")
	for _, id := range bundleIssues {
		if tip := tipOf(t, bare, id); tip.released {
			t.Fatalf("%s was let go:\n%s", id, said)
		}
		if !strings.Contains(said, id+": its reservation is kept") {
			t.Fatalf("%s is not named as kept:\n%s", id, said)
		}
	}
	if st, err := ReadState(repo.Root(), runID); err != nil || len(linesAt(st, StageDriveReleased)) != 0 {
		t.Fatalf("the run records a let-go: %v %+v", err, st.Record)
	}
}

// TestAnInterruptAtThisVersionIsUnchanged: the same run at this version may be saved by
// any session, so the signal's stop is applied: the run is marked and halted, and its
// reservation goes with the stop.
func TestAnInterruptAtThisVersionIsUnchanged(t *testing.T) {
	w, run, home := stopUnclaimable(t, SchemaVersion)
	stopAtFirstWait(t, w, home)
	if c := controlOf(t, w.repo.Root(), run.RunID); c.Stop == nil || c.Stop.FromFile || c.Stop.HaltedAt == nil {
		t.Fatalf("the signal did not halt the run: %+v\n%s", c.Stop, w.said())
	}
	if r := reservationOf(t, w.repo.Root(), run.Key, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("the halt did not let the reservation go: %+v", r)
	}
	if strings.Contains(w.said(), "could not be applied") || strings.Contains(w.said(), "is kept") {
		t.Fatalf("the drive says a stop was not applied:\n%s", w.said())
	}
}

// TestALaterStartTakesUpARunNoDrainListsThatADrivesEndLetGo: the start steps a run no drain
// lists, and the drive's end lets it go, at its bound with no mark or on a signal with one.
// A later start, the drive before it gone, takes the run up again and drives it to its end;
// a person's stop that stands leaves it.
func TestALaterStartTakesUpARunNoDrainListsThatADrivesEndLetGo(t *testing.T) {
	for _, tc := range []struct {
		name  string
		end   func(*terminalWorld, *time.Time) func(*TerminalOptions)
		ended string
		took  string
	}{
		{"bound", func(w *terminalWorld, until *time.Time) func(*TerminalOptions) {
			claimHoldAtLand(w, func() { w.clock.t = until.Add(time.Minute) })
			return func(o *TerminalOptions) {
				*until = w.clock.t.Add(time.Hour)
				o.Until = *until
			}
		}, TerminalUntil, "took up"},
		{"signal", func(w *terminalWorld, _ *time.Time) func(*TerminalOptions) {
			claimHoldAtLand(w, func() { close(w.stop) })
			return nil
		}, TerminalStopped, "took over"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, run, home := claimReplaced(t)
			var until time.Time
			res, err := claimStartWith(t, w, home, tc.end(w, &until))
			if err != nil {
				t.Fatal(err)
			}
			if res.Ended != tc.ended {
				t.Fatalf("premise: the start ends at its %s: %+v\n%s", tc.name, res, w.said())
			}
			if r := reservationOf(t, w.repo.Root(), run.Key, w.clock.t); r.State != claim.StateReleased {
				t.Fatalf("premise: the drive's end let the run go: %+v", r)
			}
			stopLaterStart(t)
			claimStart(t, w, home)
			if !slices.ContainsFunc(w.lines, func(l string) bool { return strings.Contains(l, tc.took) && strings.Contains(l, run.RunID) }) {
				t.Fatalf("the later start does not take the run up:\n%s", w.said())
			}
			if after, err := ReadState(w.repo.Root(), run.RunID); err != nil || !after.Complete() {
				t.Fatalf("the run the later start took up is not driven to its end: %v\n%s", err, w.said())
			}
		})
	}
	t.Run("a person's stop", func(t *testing.T) {
		w, run, home := claimReplaced(t)
		var until time.Time
		claimHoldAtLand(w, func() { w.clock.t = until.Add(time.Minute) })
		if _, err := claimStartWith(t, w, home, func(o *TerminalOptions) {
			until = w.clock.t.Add(time.Hour)
			o.Until = until
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := runstop.Set(home, w.clock.t); err != nil {
			t.Fatal(err)
		}
		stopLaterStart(t)
		res := claimStart(t, w, home)
		i := slices.IndexFunc(res.PassedOver, func(p Excluded) bool { return p.ID == run.Key })
		if i < 0 || !strings.Contains(res.PassedOver[i].Reason, "lifts the stop") || strings.Contains(w.said(), "took up") {
			t.Fatalf("the later start does not leave the run to the person's stop: %+v\n%s", res.PassedOver, w.said())
		}
	})
}
