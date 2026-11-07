//go:build unix

package loop

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/runstop"
)

// stopOnFirstWait sets the person's stop in a fixture home as the drive first sleeps, and
// has the options read it.
func stopOnFirstWait(t *testing.T, w *terminalWorld, o *TerminalOptions) string {
	t.Helper()
	home, _ := stopWorld(t, false)
	o.Options.stopHome = home
	set := false
	w.onSleep = func(time.Duration) {
		if !set {
			set = true
			if _, err := runstop.Set(home, w.clock.t); err != nil {
				t.Error(err)
			}
		}
	}
	return home
}

// TestABoundedWaitOnALimitEndsAtAStop: a wait two hours long is no reason to go on past the
// person's stop: it ends at the next look, with no further launch, and the lane settles.
func TestABoundedWaitOnALimitEndsAtAStop(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.resetIn(t, 2*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	stopOnFirstWait(t, w, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalHalted || !strings.Contains(res.Why, "a stop set by") {
		t.Fatalf("the drive ends on the stop: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if len(w.sleeps) != 1 {
		t.Fatalf("the wait ended at the stop: %v", w.sleeps)
	}
	if n := w.launches(t); n != 2 {
		t.Fatalf("no launch follows the stop: %d", n)
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop == nil {
		t.Fatalf("the lane took the stop: %+v", c)
	}
}

// TestABoundedWaitOnAnOutageEndsAtAStop: the outage's schedule is a wait like any other. An
// hour's wait on it, through the production sleep, ends at the look after the stop is set.
func TestABoundedWaitOnAnOutageEndsAtAStop(t *testing.T) {
	old := wallLookEvery
	wallLookEvery = 10 * time.Millisecond
	t.Cleanup(func() { wallLookEvery = old })
	w := newTerminalWorld(t, "shape")
	o := w.bounded(t, 30*24*time.Hour)
	o.Sleep = nil
	home, _ := stopWorld(t, false)
	o.Options.stopHome = home
	tm := newTerminal(w.repo.Root(), o)
	tm.ctx = context.Background()
	tm.outage = terminalOutageClock{failures: terminalFailuresToOutage, waits: len(implement.DefaultSchedule.Waits), since: w.clock.t, noLimit: true}
	done := make(chan struct{})
	go func() {
		tm.failed(context.Background(), false, "the model service is out")
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("the wait ended before any stop")
	default:
	}
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the outage's wait went on past the stop")
	}
}

// TestTheRealSleepLooksForAStop: with no test's sleep in the way, a wait of two hours ends
// at the look after the stop is set, and well short of its two hours.
func TestTheRealSleepLooksForAStop(t *testing.T) {
	old := wallLookEvery
	wallLookEvery = 10 * time.Millisecond
	t.Cleanup(func() { wallLookEvery = old })
	w := newTerminalWorld(t, "shape", "limited")
	w.resetIn(t, 2*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	o.Options.Now = nil
	o.Until = time.Now().Add(30 * 24 * time.Hour)
	o.Sleep = nil
	home, _ := stopWorld(t, false)
	o.Options.stopHome = home
	go func() {
		for range 4000 {
			if w.launches(t) >= 2 {
				time.Sleep(100 * time.Millisecond)
				_, _ = runstop.Set(home, time.Now())
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	done := make(chan TerminalResult, 1)
	go func() {
		res, err := RunTerminal(context.Background(), w.repo.Root(), o)
		if err != nil {
			t.Error(err)
		}
		done <- res
	}()
	select {
	case res := <-done:
		if res.Ended != TerminalHalted {
			t.Fatalf("the drive ends on the stop: %+v", res)
		}
	case <-time.After(2 * time.Minute):
		t.Fatal("the wait outlasted the stop")
	}
}

// TestWallSleepEndsAtTheLookAfterAStop: the production sleep asks at each look.
func TestWallSleepEndsAtTheLookAfterAStop(t *testing.T) {
	looks := make(chan time.Time)
	asked := 0
	done := make(chan struct{})
	go func() {
		wallSleep(context.Background(), 2*time.Hour, nil, func() bool {
			asked++
			return asked == 2
		}, time.Now, looks)
		close(done)
	}()
	looks <- time.Now()
	looks <- time.Now()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the sleep went on past the stop")
	}
}

// TestAStopAtTheDrivesStartStartsNothing: the drive reads it before its first move.
func TestAStopAtTheDrivesStartStartsNothing(t *testing.T) {
	w := newTerminalWorld(t, "shape")
	o := w.options(t)
	home, _ := stopWorld(t, false)
	o.Options.stopHome = home
	if _, err := runstop.Set(home, w.clock.t); err != nil {
		t.Fatal(err)
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalHalted || w.launches(t) != 0 || !strings.Contains(res.Why, "lifts") {
		t.Fatalf("nothing starts: %+v, %d launches", res, w.launches(t))
	}
}

// TestASignalHaltsTheLaneAndTheNextDriveTakesItOver: the first press settles the lane as a
// stop does, the drive says how to go on, and that command does take it over, reserving
// it again and clearing the signal's mark in the same act.
func TestASignalHaltsTheLaneAndTheNextDriveTakesItOver(t *testing.T) {
	w, res := stoppedLane(t)
	c := controlOf(t, w.repo.Root(), res.Run)
	if c.Stop == nil || c.Stop.FromFile || c.Stop.HaltedAt == nil || c.Stop.SetBy != "a signal to the drive" {
		t.Fatalf("the signal's stop: %+v", c.Stop)
	}
	if !strings.Contains(res.Resume, TerminalResumeCommand) {
		t.Fatalf("the resume: %+v", res)
	}
	o := w.options(t)
	o.Stop = make(chan struct{})
	o.Drain.Max = 0
	next, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if next.Ended == TerminalHalted {
		t.Fatalf("the command it named did not take the run over: %+v\n%s", next, strings.Join(w.lines, "\n"))
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop != nil {
		t.Fatalf("the mark stands: %+v", c.Stop)
	}
}

// personHaltedLane is a run a person's stop halted while its implementer was in its call,
// the stop lifted since.
func personHaltedLane(t *testing.T) (*terminalWorld, TerminalResult, string) {
	t.Helper()
	w := newTerminalWorld(t, "shape", "slow")
	w.realWorktree()
	home, _ := stopWorld(t, false)
	o := w.options(t)
	o.Options.stopHome = home
	go func() {
		for range 4000 {
			if w.launches(t) >= 2 {
				_, _ = runstop.Set(home, time.Now())
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalHalted || res.Run == "" {
		t.Fatalf("the drive ends on the stop: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop == nil || !c.Stop.FromFile || c.Stop.HaltedAt == nil {
		t.Fatalf("the run took the person's stop: %+v", c.Stop)
	}
	if _, err := runstop.Lift(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	return w, res, home
}

// TestAPersonsHaltWaitsForItsTakeOver: nothing resumes by itself when a person's stop lifts,
// so a new drive names the run and passes over it, reserving nothing for it and taking up
// other work. The explicit take-over is what it waits for, and it still works.
func TestAPersonsHaltWaitsForItsTakeOver(t *testing.T) {
	w, res, home := personHaltedLane(t)
	o := w.options(t)
	o.Stop = make(chan struct{})
	o.Drain.Max = 0
	o.Options.stopHome = home
	next, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("a new drive took the run over: %+v\n%s", c.Stop, strings.Join(w.lines, "\n"))
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("a new drive reserved the halted run's issue: %+v", r)
	}
	named := false
	for _, p := range next.PassedOver {
		named = named || (p.ID == res.Issue && strings.Contains(p.Reason, "implement resume --take "+res.Run))
	}
	if !named {
		t.Fatalf("the drive does not name the run and its take-over: %+v", next.PassedOver)
	}
	if next.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 2 {
		t.Fatalf("the drive goes on to other work: %+v\n%s", next, strings.Join(w.lines, "\n"))
	}
	again, err := Drain(w.repo.Root(), o.Options, o.Drain)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(again.Passed, func(p Excluded) bool { return p.ID == res.Issue })
	if i < 0 || again.Passed[i].Check != CheckNeedsPerson || !strings.Contains(again.Passed[i].Reason, "implement resume --take "+res.Run) {
		t.Fatalf("a later drain does not name the run and its take-over: %+v", again.Passed)
	}
	if _, err := TakeRun(w.repo.Root(), res.Run, Options{stopHome: home, Now: w.clock.now}); err != nil {
		t.Fatalf("the take-over it waits for: %v", err)
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop != nil {
		t.Fatalf("the take-over left the mark: %+v", c.Stop)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateLive {
		t.Fatalf("the take-over reserves again: %+v", r)
	}
}

// TestNothingIsReservedWhileAStopHoldsAMarkedRun: a drive that let the run's reservation go
// would take it again on its way back to the lane. A person's mark on the run stops that.
func TestNothingIsReservedWhileAStopHoldsAMarkedRun(t *testing.T) {
	w, res, home := personHaltedLane(t)
	if err := mutate(w.repo.Root(), res.Run, func(_ *os.Root, st *State) (bool, error) {
		st.Record = append(st.Record, Entry{At: w.clock.t, Stage: StageDriveReleased, Note: "let go"})
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	o := w.options(t)
	o.Options.stopHome = home
	done, ended := newTerminal(w.repo.Root(), o).takeUp(context.Background(), res.Run)
	if !done || ended != "" {
		t.Fatalf("the take-up: %v, %q", done, ended)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("something reserved under the mark: %+v", r)
	}
}

// TestALanePassedOverSaysWhenItWasTakenOver: the drain's lines say what became of each lane,
// so a lane passed over reads halted only while the run is, and reads as a person's once they
// have taken it over.
func TestALanePassedOverSaysWhenItWasTakenOver(t *testing.T) {
	w, res, home := personHaltedLane(t)
	o := w.options(t)
	o.Options.stopHome = home
	if done, ended := newTerminal(w.repo.Root(), o).takeUp(context.Background(), res.Run); !done || ended != "" {
		t.Fatalf("the take-up: %v, %q", done, ended)
	}
	outcome := func() string {
		t.Helper()
		move, err := Drain(w.repo.Root(), o.Options, o.Drain)
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(move.Lanes, func(l DrainLane) bool { return l.RunID == res.Run })
		if i < 0 {
			t.Fatalf("the drain has no lane for the run: %+v", move.Lanes)
		}
		return move.Lanes[i].Outcome
	}
	if got := outcome(); got != DrainLaneHalted {
		t.Fatalf("before the take-over the lane reads %q", got)
	}
	if _, err := TakeRun(w.repo.Root(), res.Run, Options{stopHome: home, Now: w.clock.now}); err != nil {
		t.Fatal(err)
	}
	if got := outcome(); got != "taken-over" {
		t.Fatalf("after the take-over the lane reads %q", got)
	}
}

// TestASignalsHaltComposesNoPage: the day's page says the person's stop holds and how to lift
// it, which a drive's own signal is not.
func TestASignalsHaltComposesNoPage(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, folder := stopWorld(t, true)
	later := time.Now().Add(3 * time.Hour)
	rec, err := haltOnce(repo.Root(), runID, runstop.Stop{SetBy: "a signal to the drive", SetAt: later}, Options{stopHome: home, Now: func() time.Time { return later }})
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || rec.HaltedAt == nil {
		t.Fatalf("the signal's stop: %+v", rec)
	}
	if entries, _ := os.ReadDir(filepath.Join(folder, accountDigestDir())); len(entries) != 0 {
		t.Fatalf("a signal wrote a page: %v", entries)
	}
}

// TestAStepNamesBothWhereBothStand: the person has to act on each, so a step says so; with
// the stop lifted it names only what is left.
func TestAStepNamesBothWhereBothStand(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	if err := recordDriveStop(runID); err != nil {
		t.Fatal(err)
	}
	p, err := driveStopPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(p) })
	o, home, _ := stopped(t)
	_, stepped := advance(repo.Root(), runID, DefaultStages(), o)
	_, driven := Drive(context.Background(), repo.Root(), runID, DefaultStages(), o, Runners{})
	_, restarted := restartLane(repo.Root(), runID, "lane-1", "", o)
	for _, err := range []error{stepped, driven, restarted} {
		r := wantStopRefusal(t, err)
		if !strings.Contains(r.Reason, "a stop set by") || !strings.Contains(r.Reason, "a drive stopped on") ||
			!strings.Contains(r.Remedy, "--lift") || !strings.Contains(r.Remedy, driveStopsDir) {
			t.Fatalf("the refusal names one of them: %+v", r)
		}
	}
	if _, err := runstop.Lift(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, stepped = advance(repo.Root(), runID, DefaultStages(), o)
	r, ok := AsRefusal(stepped)
	if !ok || strings.Contains(r.Reason, "a stop set by") || !strings.Contains(r.Reason, "a drive stopped on") {
		t.Fatalf("after the lift: %v", stepped)
	}
}

// TestAStopAndADriveStopAreTwoThings: while both stand nothing is settled from a state not
// known to be the driver's. Lifting the stop leaves the drive's stop; removing that leaves
// the stop, which the run then takes in its order. Neither answers for the other.
func TestAStopAndADriveStopAreTwoThings(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	if err := recordDriveStop(runID); err != nil {
		t.Fatal(err)
	}
	p, err := driveStopPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(p) })
	o, home, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }

	r := wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if !strings.Contains(r.Reason, "a drive stopped on") || !strings.Contains(r.Remedy, "--lift") {
		t.Fatalf("the refusal names both: %+v", r)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop != nil {
		t.Fatalf("the stop was settled over the drive's stop: %+v", c.Stop)
	}
	l, err := ListRuns(repo.Root(), runID, o)
	if err != nil {
		t.Fatal(err)
	}
	if next := theRun(t, l, runID).Next; !strings.Contains(next, "a stop holds") || !strings.Contains(next, "a drive stopped on") {
		t.Fatalf("the listing says %q", next)
	}
	if _, err := TakeRun(repo.Root(), runID, o); err == nil {
		t.Fatal("a take-over went ahead under both")
	}

	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	if _, err := TakeRun(repo.Root(), runID, o); err == nil || !strings.Contains(err.Error(), "drive stopped on") {
		t.Fatalf("the lift answered for the drive's stop: %v", err)
	}
	l, err = ListRuns(repo.Root(), runID, o)
	if err != nil {
		t.Fatal(err)
	}
	if next := theRun(t, l, runID).Next; !strings.Contains(next, "a drive stopped on") {
		t.Fatalf("the listing after the lift says %q", next)
	}

	if _, err := runstop.Set(home, later); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("removing the drive's stop answered for the stop: %+v", c.Stop)
	}
	asideOf(t, dir)
}

// TestADriveWithBothStandingEndsOnTheStop: the same at a drive's head, which names both
// and no way on that would be refused.
func TestADriveWithBothStandingEndsOnTheStop(t *testing.T) {
	w, res := stoppedLane(t)
	if err := recordDriveStop(res.Run); err != nil {
		t.Fatal(err)
	}
	p, err := driveStopPath(res.Run)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(p) })
	o := w.options(t)
	o.Stop = make(chan struct{})
	o.Drain.Max = 0
	home, _ := stopWorld(t, false)
	o.Options.stopHome = home
	if _, err := runstop.Set(home, w.clock.t); err != nil {
		t.Fatal(err)
	}
	next, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if next.Ended != TerminalHalted || !strings.Contains(next.Why, "a drive stopped on") {
		t.Fatalf("the drive names both: %+v", next)
	}
	if next.Resume != "" {
		t.Fatalf("it names a way on that would be refused: %+v", next)
	}
}

// TestAHaltedEndSaysWhatIsTrue: what ended the drive is the refusal's own account, in the
// end it gives and under --json, whichever kind of stop it was.
func TestAHaltedEndSaysWhatIsTrue(t *testing.T) {
	w := newTerminalWorld(t, "shape")
	for _, err := range []error{
		contend(StageStop, "", "", "run-1 was halted by a stop; the stop has lifted", "run the take-over"),
		contend(StageStop, "", "", "run-1 was halted by a stop; it came from a signal", "run the take-over"),
		heldRefusal("the shared folder this run started under has gone"),
	} {
		tm := newTerminal(w.repo.Root(), w.options(t))
		if !tm.haltedBy(err) {
			t.Fatalf("not read as a halt: %v", err)
		}
		r, _ := AsRefusal(err)
		res := tm.ended(TerminalHalted)
		if res.Ended != TerminalHalted || !strings.Contains(res.Why, r.Reason) || !strings.Contains(res.Why, r.Remedy) {
			t.Fatalf("the end says %+v of %+v", res, r)
		}
	}
}

// TestADriveClaimsWithItsOwnProcess: the process that holds the run, not a terminal that
// outlives it or the first process under nohup.
func TestADriveClaimsWithItsOwnProcess(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	procsOf(t, hostOne, hostOne, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home, driverIsSelf: true}); err != nil {
		t.Fatal(err)
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Driver == nil || c.Driver.PID != hostOne+shellOf {
		t.Fatalf("the driver: %+v", c.Driver)
	}
	stepped := *c.Driver
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err == nil {
		t.Fatalf("a step above took a run its own drive holds: %+v", stepped)
	}
}

// TestTheStopHomeIsTheMachines: under a walled stage the person's own home, not the stage's.
func TestTheStopHomeIsTheMachines(t *testing.T) {
	stage, person := t.TempDir(), t.TempDir()
	t.Setenv("HOME", stage)
	t.Setenv(abcdhome.StageHomeEnv, stage)
	t.Setenv(abcdhome.PersonHomeEnv, person)
	if got := runstop.MachineStopHome(); got != person {
		t.Fatalf("the machine's stop home = %q, want the person's %q", got, person)
	}
}

// TestAStopInTheAccountsHomeReachesAProcessToldOtherwise: a process whose home variable
// moved still reads the stop the verb wrote in the account's home.
func TestAStopInTheAccountsHomeReachesAProcessToldOtherwise(t *testing.T) {
	account, _ := stopWorld(t, false)
	moved, _ := stopWorld(t, false)
	if _, err := runstop.Set(account, time.Now()); err != nil {
		t.Fatal(err)
	}
	oldAccount, oldHome, oldOff := accountStopHome, defaultStopHome, stopReadingOff
	accountStopHome, defaultStopHome, stopReadingOff = func() string { return account }, func() string { return moved }, false
	t.Cleanup(func() { accountStopHome, defaultStopHome, stopReadingOff = oldAccount, oldHome, oldOff })
	if r := (Options{}).stopRead(nil); r.Stop == nil {
		t.Fatalf("the account's stop was not read: %+v", r)
	}
}

// TestASaveThatFailsHoldsTheClaimsThroughASignal: work saved nowhere is no reason to let
// the reservation go, so the signal's stop leaves the claim, and says so.
func TestASaveThatFailsHoldsTheClaimsThroughASignal(t *testing.T) {
	w := newTerminalWorld(t, "shape", "slow")
	w.stopOnLaunch(t, 2)
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalStopped {
		t.Fatalf("the end: %+v", res)
	}
	c := controlOf(t, w.repo.Root(), res.Run)
	if c.Stop == nil || c.Stop.HaltedAt != nil {
		t.Fatalf("a fixture's worktree git cannot read does not settle: %+v", c.Stop)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateLive {
		t.Fatalf("the reservation went with the work unsaved: %+v", r)
	}
}
