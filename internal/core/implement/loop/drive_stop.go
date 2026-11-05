package loop

// drive_stop.go keeps that a drive stopped on a run state changed or unreadable. The run's own
// directory is what the change reached, so the fact lies under the person's machine-scoped
// home, keyed on the run, where a stage's file tools do not write. Until a person discards
// the run's lanes or restores its state, nothing goes on from it: what it now says of a
// verdict or a stage is not known to be the driver's.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// driveStopsDir is the folder under the home that holds one file per stopped run.
const driveStopsDir = "drive-stops"

// driveStopPath is where the stop on run runID is kept.
func driveStopPath(runID string) (string, error) {
	home, err := abcdhome.MachineHome()
	if err != nil {
		return "", err
	}
	return abcdhome.Path(home, driveStopsDir, runID), nil
}

// recordDriveStop keeps that a drive stopped on run runID.
func recordDriveStop(runID string) error {
	path, err := driveStopPath(runID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), abcdhome.DirMode); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, []byte(errRunStateChanged.Error()+"\n"), 0o600)
}

// lookForDriveStop reports whether a drive stopped on run runID stands, and any failure
// to tell: only a stop known to be absent reads as none.
func lookForDriveStop(runID string) (bool, error) {
	path, err := driveStopPath(runID)
	if err != nil {
		return false, err
	}
	_, err = os.Lstat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	}
	return false, err
}

// driveStopped reports whether run runID is to be refused: a stop stands, or none can be
// told apart from one.
func driveStopped(runID string) bool {
	stopped, err := lookForDriveStop(runID)
	return stopped || err != nil
}

// driveStopRemedy says how a person lifts the stop on run runID.
func driveStopRemedy(runID string) string {
	path, _ := driveStopPath(runID)
	where := fsutil.RedactHome(path)
	return fmt.Sprintf("If you know what caused it, such as your own `abcd implement step`, remove %s. Otherwise discard each lane with `abcd implement step --run %s --discard <lane>`, which deletes its branch and worktree and closes its pull request, or restore the run's state yourself and remove that file", where, runID)
}

// driveStopRefusal is the refusal of a move on a run a drive stopped on, or whose stop
// could not be looked for, nil for any other.
func driveStopRefusal(runID string) error {
	if !ValidRunID(runID) {
		return nil
	}
	stopped, err := lookForDriveStop(runID)
	if err != nil {
		return refuse("state", "", "", "abcd could not check whether a drive stopped on "+runID+" ("+fsutil.RedactHome(err.Error())+"), so nothing goes on from it",
			"make the home readable, with HOME set, then run it again")
	}
	if !stopped {
		return nil
	}
	return refuse("state", "", "", "a drive stopped on "+runID+" because its state changed or could not be read after an agent's call, so nothing goes on from what it now says",
		driveStopRemedy(runID))
}

// liftDriveStop removes the stop once no lane of st is left to go on.
func liftDriveStop(st State) {
	for _, l := range st.Lanes {
		if l.Stage != StageDone && l.Stage != StageDiscarded && l.Stage != StageHandedBack {
			return
		}
	}
	if path, err := driveStopPath(st.RunID); err == nil {
		_ = os.Remove(path)
	}
}
