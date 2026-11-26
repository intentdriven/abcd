package loop

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// A drive's start takes every interrupted run it may (spc-2611141617199772 step 1;
// itd-2611141610197753 decisions 1 and 2), against the terminal world and a stand-in
// process table: one terminal (its login process and shell) the first drive was started
// from and the next start is typed in, and a second terminal.
const (
	claimLogin      = 7000
	claimShell      = 7001
	claimFirst      = 7010
	claimDrive      = 7030
	claimOtherLogin = 8000
	claimOtherShell = 8001
)

// claimRead is when every stand-in table was read.
var claimRead = time.Now()

// claimProcs stands in a process table holding alive, with from the process a host walk
// starts at and self this process.
func claimProcs(t *testing.T, from, self int, alive ...int) {
	t.Helper()
	all := map[int]machineload.Proc{
		claimLogin:      {PID: claimLogin, PPID: 1, Name: "login", Age: time.Hour},
		claimShell:      {PID: claimShell, PPID: claimLogin, Name: "-zsh", Age: 59 * time.Minute},
		claimFirst:      {PID: claimFirst, PPID: claimShell, Name: "abcd", Age: 50 * time.Minute},
		claimDrive:      {PID: claimDrive, PPID: claimShell, Name: "abcd", Age: 10 * time.Minute},
		claimOtherLogin: {PID: claimOtherLogin, PPID: 1, Name: "login", Age: 30 * time.Minute},
		claimOtherShell: {PID: claimOtherShell, PPID: claimOtherLogin, Name: "-zsh", Age: 29 * time.Minute},
	}
	tab := sessionname.Table{ReadAt: claimRead}
	for _, pid := range alive {
		tab.Procs = append(tab.Procs, all[pid])
	}
	standIn(t, tab, from, self)
}

// claimNextStart is the table a start typed in the first terminal reads: the first drive
// is gone, and so is anything it ran.
func claimNextStart(t *testing.T) {
	t.Helper()
	claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive)
}

// claimInterrupted is a world whose first drive, started from the first terminal, took a
// signal between two stages with its first lane in flight. The lane's run is halted by the
// signal and settled, its driver that first drive. It returns the run and the home the
// next drive reads session markers and a stop from.
func claimInterrupted(t *testing.T) (*terminalWorld, State, string) {
	t.Helper()
	w := newTerminalWorld(t)
	w.realWorktree()
	asked := false
	for i := range w.stages {
		if w.stages[i].Name == StageValidate {
			w.stages[i].Run = func(Context, *Lane) (Outcome, error) {
				if !asked {
					asked = true
					close(w.stop)
				}
				return Outcome{Note: "validate done"}, nil
			}
		}
	}
	claimProcs(t, claimShell, claimFirst, claimLogin, claimShell, claimFirst)
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalStopped || res.Run == "" {
		t.Fatalf("premise: the first drive ends on its signal with a lane left: %+v\n%s", res, w.said())
	}
	c := controlOf(t, w.repo.Root(), res.Run)
	if c.Stop == nil || c.Stop.FromFile || c.Stop.HaltedAt == nil || c.Driver == nil || c.Driver.PID != claimFirst {
		t.Fatalf("premise: a settled signal's mark on a run the first drive drove: %+v %+v", c.Stop, c.Driver)
	}
	run, err := ReadState(w.repo.Root(), res.Run)
	if err != nil {
		t.Fatal(err)
	}
	home, _ := stopWorld(t, false)
	return w, run, home
}

// claimStart runs the next drive from the first terminal, with its own stop and lines.
func claimStart(t *testing.T, w *terminalWorld, home string) TerminalResult {
	t.Helper()
	w.stop = make(chan struct{})
	w.lines = nil
	o := w.options(t)
	o.Options.stopHome = home
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatalf("the start: %v\n%s", err, w.said())
	}
	return res
}

// claimUnsettled leaves the run a signal's mark that has not settled, as an interrupt
// leaves it when an agent was still in its call and the drive was then killed.
func claimUnsettled(t *testing.T, repoRoot, runID string) {
	t.Helper()
	now := time.Now().UTC()
	if err := mutate(repoRoot, runID, func(root *os.Root, _ *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		c.Stop = &RunStop{SetBy: "a signal to the drive", SetAt: now, NotedAt: now}
		return false, writeControl(root, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
}

// claimDriverIs records pid, as the stand-in table reads it, as the run's live driver,
// having claimed the run an hour ago.
func claimDriverIs(t *testing.T, repoRoot, runID string, pid int) {
	t.Helper()
	tab, err := readProcs()
	if err != nil {
		t.Fatal(err)
	}
	h, ok := procOf(tab, pid)
	if !ok {
		t.Fatalf("process %d is not in the stand-in table", pid)
	}
	if err := mutate(repoRoot, runID, func(root *os.Root, _ *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		c.Driver = &LiveDriver{PID: h.PID, StartUnix: h.StartUnix, SlackMS: h.Slack.Milliseconds(), At: time.Now().UTC().Add(-time.Hour)}
		return false, writeControl(root, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
}

// namesNoTypedTakeOver fails the test where a line of the start names a command a person
// would type to take up or step runID, which the start took itself.
func namesNoTypedTakeOver(t *testing.T, w *terminalWorld, runID string) {
	t.Helper()
	for _, l := range w.lines {
		if strings.Contains(l, "resume --take") || strings.Contains(l, "step --run "+runID) {
			t.Fatalf("a line of the start names a typed take-over of %s: %q\n%s", runID, l, w.said())
		}
	}
}

// tookOver fails the test unless the start said it took runID over, and the run no longer
// carries a stop's mark.
func tookOver(t *testing.T, w *terminalWorld, runID string) {
	t.Helper()
	if !slices.ContainsFunc(w.lines, func(l string) bool { return strings.Contains(l, "took over") && strings.Contains(l, runID) }) {
		t.Fatalf("the start does not say it took %s over:\n%s", runID, w.said())
	}
	if c := controlOf(t, w.repo.Root(), runID); c.Stop != nil {
		t.Fatalf("the mark stands on %s: %+v", runID, c.Stop)
	}
}

// leftNamed fails the test unless the start named issue as passed over, its reason holding
// want, and left the mark on runID.
func leftNamed(t *testing.T, w *terminalWorld, res TerminalResult, issue, runID, want string) {
	t.Helper()
	i := slices.IndexFunc(res.PassedOver, func(p Excluded) bool { return p.ID == issue })
	if i < 0 || res.PassedOver[i].Check != CheckNeedsPerson || !strings.Contains(res.PassedOver[i].Reason, want) {
		t.Fatalf("the start does not name %s with %q: %+v\n%s", issue, want, res.PassedOver, w.said())
	}
	if c := controlOf(t, w.repo.Root(), runID); c.Stop == nil {
		t.Fatalf("the start took %s, which it was to leave", runID)
	}
}

// TestADrivesStartFromItsOwnTerminalLandsThreeLanes is the lab's first fixture as a
// drive's start: the run's recorded driver is the login process of the terminal the start
// is typed in, and a signal's mark that never settled stands on its file, which an earlier
// abcd wrote. The start is the only command, and the three lanes land.
func TestADrivesStartFromItsOwnTerminalLandsThreeLanes(t *testing.T) {
	w, run, home := claimInterrupted(t)
	claimNextStart(t)
	claimDriverIs(t, w.repo.Root(), run.RunID, claimLogin)
	claimUnsettled(t, w.repo.Root(), run.RunID)
	stateAtVersion(t, w.repo.Root(), run.RunID, schemaVersionPrevious)
	res := claimStart(t, w, home)
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 3 {
		t.Fatalf("the start lands the three lanes: %+v %v\n%s", res, lanesDone(t, w.repo), w.said())
	}
	tookOver(t, w, run.RunID)
	namesNoTypedTakeOver(t, w, run.RunID)
}

// TestAnUnsettledSignalMarkOnAnEarlierVersionIsSettledAndTaken is the incident's seventh
// event: an interrupt with an agent in its call, then a rebuild, so the run's file states
// the earlier version and its driver is gone. The start settles the mark and takes the run.
func TestAnUnsettledSignalMarkOnAnEarlierVersionIsSettledAndTaken(t *testing.T) {
	w, run, home := claimInterrupted(t)
	claimUnsettled(t, w.repo.Root(), run.RunID)
	stateAtVersion(t, w.repo.Root(), run.RunID, schemaVersionPrevious)
	claimNextStart(t)
	res := claimStart(t, w, home)
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 3 {
		t.Fatalf("the start lands the three lanes: %+v %v\n%s", res, lanesDone(t, w.repo), w.said())
	}
	tookOver(t, w, run.RunID)
	namesNoTypedTakeOver(t, w, run.RunID)
}

// claimArmingWorld is a world whose land stage arms its pull request on the lane's own head
// the first time, taking a signal there, and finishes the lane after.
func claimArmingWorld(t *testing.T) *terminalWorld {
	t.Helper()
	w := newTerminalWorld(t, "shape")
	w.realWorktree()
	armed := false
	for i := range w.stages {
		if w.stages[i].Name == StageLand {
			w.stages[i].Run = func(_ Context, l *Lane) (Outcome, error) {
				if armed {
					return Outcome{Note: "land done"}, nil
				}
				armed = true
				l.PR, l.Landing = 7, &Landing{Pushed: l.HeadSHA, Merge: "armed", Armed: true}
				close(w.stop)
				return Outcome{Stay: true, Note: "pull request #7 is armed"}, nil
			}
		}
	}
	old := parkedStages
	parkedStages = func() Stages { return w.stages }
	t.Cleanup(func() { parkedStages = old })
	return w
}

// TestARunParkedUnderASignalIsTakenAndFinishedOnceMerged: a signal at the arming step
// leaves the run parked at its pull request under the signal's mark. The next start takes
// it over, and once the pull request has merged the drive finishes it.
func TestARunParkedUnderASignalIsTakenAndFinishedOnceMerged(t *testing.T) {
	w := claimArmingWorld(t)
	first, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if first.Ended != TerminalStopped || first.Run == "" {
		t.Fatalf("premise: the first drive ends on the signal with the lane parked: %+v\n%s", first, w.said())
	}
	run, err := ReadState(w.repo.Root(), first.Run)
	if err != nil {
		t.Fatal(err)
	}
	i := parkedAtLand(run)
	if c := controlOf(t, w.repo.Root(), first.Run); i < 0 || c.Stop == nil || c.Stop.FromFile || c.Stop.HaltedAt == nil {
		t.Fatalf("premise: parked under a settled signal's mark: %d %+v", i, c.Stop)
	}
	// The pull request merges: its head reaches the default branch on origin.
	w.repo.Git("push", "-q", "origin", run.Lanes[i].Landing.Pushed+":refs/heads/main")
	home, _ := stopWorld(t, false)
	claimStart(t, w, home)
	tookOver(t, w, first.Run)
	namesNoTypedTakeOver(t, w, first.Run)
	after, err := ReadState(w.repo.Root(), first.Run)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Complete() {
		t.Fatalf("the merged lane is not finished:\n%s", w.said())
	}
}

// TestALiftedPersonsStopIsTakenUp: once the person has lifted their stop, the next start
// takes up the run it halted, reserving it again, and lands it with the rest (decision 1).
func TestALiftedPersonsStopIsTakenUp(t *testing.T) {
	w, run, home := claimInterrupted(t)
	markHalted(t, w.repo.Root(), run.RunID, true)
	claimNextStart(t)
	res := claimStart(t, w, home)
	tookOver(t, w, run.RunID)
	namesNoTypedTakeOver(t, w, run.RunID)
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 3 {
		t.Fatalf("the start lands the three lanes: %+v %v\n%s", res, lanesDone(t, w.repo), w.said())
	}
	if slices.ContainsFunc(res.PassedOver, func(p Excluded) bool { return p.ID == run.Key }) {
		t.Fatalf("the run it took is named as passed over: %+v", res.PassedOver)
	}
}

// TestAPersonsStopThatStandsIsLeftAndNamed: while the person's stop stands the start takes
// nothing and names the run it halted, with what takes it up.
func TestAPersonsStopThatStandsIsLeftAndNamed(t *testing.T) {
	w, run, home := claimInterrupted(t)
	markHalted(t, w.repo.Root(), run.RunID, true)
	if _, err := runstop.Set(home, w.clock.t); err != nil {
		t.Fatal(err)
	}
	claimNextStart(t)
	res := claimStart(t, w, home)
	if res.Ended != TerminalHalted {
		t.Fatalf("the stop ends the start: %+v\n%s", res, w.said())
	}
	leftNamed(t, w, res, run.Key, run.RunID, "lifts the stop")
	if strings.Contains(w.said(), "resume --take") {
		t.Fatalf("the start names a typed take-over:\n%s", w.said())
	}
}

// TestARunWhoseDrainWasReplacedIsTakenWithTheRest: decision 2. The drain that opened the
// halted run is gone and no drain lists the run, and the start takes it up and drives it
// with the checkout's other work.
func TestARunWhoseDrainWasReplacedIsTakenWithTheRest(t *testing.T) {
	w, run, home := claimInterrupted(t)
	if err := os.Remove(filepath.Join(w.repo.Root(), filepath.FromSlash(DrainStateRel))); err != nil {
		t.Fatal(err)
	}
	claimNextStart(t)
	claimStart(t, w, home)
	tookOver(t, w, run.RunID)
	namesNoTypedTakeOver(t, w, run.RunID)
	after, err := ReadState(w.repo.Root(), run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Complete() {
		t.Fatalf("the run the start took is not driven to its end:\n%s", w.said())
	}
	if done := lanesDone(t, w.repo); len(done) != 2 || slices.Contains(done, run.Key) {
		t.Fatalf("the new drain lands the other two issues: %v\n%s", done, w.said())
	}
}

// TestAHaltedRunWhoseLiveDriverIsInAnotherTerminalIsLeftAndNamed: a halted run's take-over
// lets an idle driver that claimed before the halt give way, so the start judges the
// driver itself: one alive in another terminal keeps the run, and the start names who
// holds it and goes on with the other lanes.
func TestAHaltedRunWhoseLiveDriverIsInAnotherTerminalIsLeftAndNamed(t *testing.T) {
	w, run, home := claimInterrupted(t)
	claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive, claimOtherLogin, claimOtherShell)
	claimDriverIs(t, w.repo.Root(), run.RunID, claimOtherLogin)
	markHalted(t, w.repo.Root(), run.RunID, false)
	res := claimStart(t, w, home)
	leftNamed(t, w, res, run.Key, run.RunID, "process "+strconv.Itoa(claimOtherLogin))
	if c := controlOf(t, w.repo.Root(), run.RunID); c.Driver == nil || c.Driver.PID != claimOtherLogin {
		t.Fatalf("the start took the run from its live driver: %+v", c.Driver)
	}
	if done := lanesDone(t, w.repo); len(done) != 2 || slices.Contains(done, run.Key) {
		t.Fatalf("the start goes on with the other two lanes: %v\n%s", done, w.said())
	}
}

// TestAnUnreadableProcessTableLeavesTheRun: whether the recorded driver lives cannot be
// told, so the start leaves the run, names why, and goes on with the other lanes.
func TestAnUnreadableProcessTableLeavesTheRun(t *testing.T) {
	w, run, home := claimInterrupted(t)
	old := readProcs
	readProcs = func() (sessionname.Table, error) { return sessionname.Table{}, os.ErrPermission }
	t.Cleanup(func() { readProcs = old })
	res := claimStart(t, w, home)
	leftNamed(t, w, res, run.Key, run.RunID, "process table")
	if done := lanesDone(t, w.repo); len(done) != 2 || slices.Contains(done, run.Key) {
		t.Fatalf("the start goes on with the other two lanes: %v\n%s", done, w.said())
	}
}
