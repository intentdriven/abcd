package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if res.Ended != TerminalWall || !strings.Contains(res.Why, "--discard") {
		t.Fatalf("the run stops, saying how to discard it: %+v", res)
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
