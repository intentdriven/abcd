package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// stoppedDrive is a drive stopped because its run's state changed in an agent's call.
func stoppedDrive(t *testing.T) (*terminalWorld, TerminalOptions, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	w := newTerminalWorld(t, "shape", "tamper")
	state := func() string {
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
				t.Setenv(terminalTamperEnv, state())
				return run(c, l)
			}
		}
	}
	o.Options.RunStateWatch = true
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall || res.Resume != "" {
		t.Fatalf("the run stops, and offers no resume: %+v", res)
	}
	for _, want := range []string{"a person's call, never an agent's", "your own", "remove", "--discard", "deletes its branch and worktree and closes its pull request"} {
		if !strings.Contains(res.Why, want) {
			t.Fatalf("the way out says %q: %s", want, res.Why)
		}
	}
	if strings.Contains(res.Why, "only the driver") {
		t.Fatalf("it blames no one: %s", res.Why)
	}
	t.Setenv(terminalTamperEnv, "")
	st, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	return w, o, st.Lanes[0].RunID
}

// TestADriveThatStoppedOnAChangedRunDoesNotCarryOn: what stopped it is kept where a stage
// does not write, so a later drive, a step and a hand-in all refuse the run.
func TestADriveThatStoppedOnAChangedRunDoesNotCarryOn(t *testing.T) {
	w, o, run := stoppedDrive(t)
	w.lines = nil
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalWall || !strings.Contains(res.Why, "--discard") {
		t.Fatalf("a later drive refuses the run, saying how to discard it: %+v %v\n%s", res, err, strings.Join(w.lines, "\n"))
	}
	if strings.Contains(strings.Join(w.lines, "\n"), "land done") {
		t.Fatalf("the later drive went on:\n%s", strings.Join(w.lines, "\n"))
	}
	for name, step := range map[string]func() error{
		"a step": func() error { _, err := advance(w.repo.Root(), run, w.stages, Options{}); return err },
		"a driven step": func() error {
			_, err := Drive(context.Background(), w.repo.Root(), run, w.stages, Options{}, Runners{})
			return err
		},
		"a hand-in": func() error {
			_, err := Receipt(w.repo.Root(), run, filepath.Join(w.repo.Root(), "any"), w.stages, Options{})
			return err
		},
	} {
		if err := step(); err == nil || !strings.Contains(err.Error(), "--discard") {
			t.Errorf("%s is refused, saying how to discard it: %v", name, err)
		}
	}
	st, err := ReadState(w.repo.Root(), run)
	if err != nil {
		t.Fatal(err)
	}
	if !driveStopped(st.RunID) {
		t.Errorf("the stop is kept")
	}
}

// TestDiscardingTheRunLiftsTheStop: the person's discard ends it, and a run that was never
// stopped is not refused.
func TestDiscardingTheRunLiftsTheStop(t *testing.T) {
	w, _, run := stoppedDrive(t)
	st, err := ReadState(w.repo.Root(), run)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture's lane holds no worktree the loop made, which a real one does.
	if err := mutate(w.repo.Root(), run, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Worktree, st.Lanes[0].Branch = "", ""
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Discard(w.repo.Root(), run, st.Lanes[0].ID, Options{}); err != nil {
		t.Fatalf("the discard is the way out: %v", err)
	}
	path, err := driveStopPath(run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("the stop stands: %v", err)
	}
	if err := driveStopRefusal("run-2611051955085311"); err != nil {
		t.Fatalf("another run is not refused: %v", err)
	}
}

// TestAStateStillUnreadStopsTheDriveWithAMark: a changed state can hide behind a read that
// keeps failing, so it is kept against the run as a change is.
func TestAStateStillUnreadStopsTheDriveWithAMark(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	w := newTerminalWorld(t, "shape", "unreadstate")
	var path string
	o := oneLane(t, w)
	for i := range w.stages {
		if w.stages[i].Name == StageBrief {
			run := w.stages[i].Run
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				path = filepath.Join(w.repo.Root(), filepath.FromSlash(StateRelPath(c.State.RunID)))
				t.Setenv(terminalTamperEnv, path)
				return run(c, l)
			}
		}
	}
	o.Options.RunStateWatch = true
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if path != "" {
		_ = os.Chmod(path, 0o600)
	}
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall || !strings.Contains(res.Why, "could not be read") || !strings.Contains(res.Why, "If you know what caused it") {
		t.Fatalf("the drive stops, saying so and how to clear it: %+v", res)
	}
	st, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !driveStopped(st.Lanes[0].RunID) {
		t.Fatal("what the read could not show is not kept against the run")
	}
}

// TestAStateReadRetriesThroughAPassingFailure: one failure is not a change.
func TestAStateReadRetriesThroughAPassingFailure(t *testing.T) {
	root, id, _, _ := startToImplement(t)
	before, err := runFingerprint(root, id)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(fingerprintWait / 2)
		_ = os.Chmod(path, 0o600)
	}()
	after, err := runFingerprint(root, id)
	<-done
	if err != nil || after != before {
		t.Fatalf("the read recovers to the same state: %v", err)
	}
	if driveStopped(id) {
		t.Fatal("a failure that passed is kept against the run")
	}
}

// TestALookThatFailsRefusesTheRun: only a stop known to be absent reads as none.
func TestALookThatFailsRefusesTheRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// A file where the stops' folder goes answers with something other than not-found
	// once a looked-for name lies below a directory no one may search.
	dir := abcdhome.Path(home, driveStopsDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if !driveStopped("run-2611051955085311") {
		t.Fatal("a look that failed reads as no stop")
	}
	err := driveStopRefusal("run-2611051955085311")
	if err == nil || !strings.Contains(err.Error(), "could not check") || strings.Contains(err.Error(), "state changed") {
		t.Fatalf("the refusal says it could not check: %v", err)
	}
}

// TestDiscardAsksTheForgeWhichPullRequest: the stop says the state cannot be trusted, so
// the pull request closed is the one the lane's branch has.
func TestDiscardAsksTheForgeWhichPullRequest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, unreviewedQueueRuleset("MERGE"))
	f.validated(t)
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[st.current()].PR = 99
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := recordDriveStop(f.runID); err != nil {
		t.Fatal(err)
	}
	if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil {
		t.Fatal(err)
	}
	if log := f.ghLog(t); !strings.Contains(log, "pr close 7") || strings.Contains(log, "pr close 99") {
		t.Fatalf("the forge's own pull request is closed:\n%s", log)
	}
}
