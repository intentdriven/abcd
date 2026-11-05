package loop

// drive_stop.go keeps that a drive stopped on a changed run state. The run's own
// directory is what the change reached, so the fact lies under the person's machine-scoped
// home, keyed on the run, where a stage's file tools do not write. Until a person discards
// the run's lanes or restores its state, nothing goes on from it: what it now says of a
// verdict or a stage is not known to be the driver's.

import (
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

// driveStopped reports whether a drive stopped on run runID stands.
func driveStopped(runID string) bool {
	path, err := driveStopPath(runID)
	if err != nil {
		return false
	}
	_, err = os.Lstat(path)
	return err == nil
}

// driveStopRemedy says how a person lifts the stop on run runID.
func driveStopRemedy(runID string) string {
	path, _ := driveStopPath(runID)
	return fmt.Sprintf("discard each of its lanes with `abcd implement step --run %s --discard <lane>`, or restore the run's state yourself and remove %s", runID, fsutil.RedactHome(path))
}

// driveStopRefusal is the refusal of a move on a run a drive stopped on, nil for any other.
func driveStopRefusal(runID string) error {
	if !ValidRunID(runID) || !driveStopped(runID) {
		return nil
	}
	return refuse("state", "", "", "a drive stopped on "+runID+" because its state changed while an agent was in its call, so nothing goes on from what it now says",
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
