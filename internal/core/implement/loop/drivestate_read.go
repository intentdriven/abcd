package loop

// drivestate_read.go reads a drive's stored state back for a status surface
// (spc-2611141617199772 step 5; itd-2611141610197753 criteria 6 and 7). The
// drive writes the value (drivestate.go); `abcd implement status` and the
// board read it here and only format what this returns, so the one place that
// decides whether a drive is still there is the loop, as for a run's driver
// (claimDriverMade).
//
// A file that is not there, cannot be read, states a newer version, or carries
// a control code shows no line: ReadDriveState returns an error that says why,
// and the surface prints it on stderr. A drive that ended without writing its
// last line is told by its recorded process: one that is gone, or whose id is
// held by a process that started at another time, reads stopped, not heard
// from since the value was written, and never working.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxDriveStateBytes bounds the read: the drive writes a few hundred bytes, so a
// file past this is not one it wrote.
const maxDriveStateBytes = 64 << 10

// ErrNoDriveState is ReadDriveState's answer where no drive has written its state in
// the checkout: the ordinary case outside a drive, said so a reader knows why no line
// is shown.
var ErrNoDriveState = errors.New("no drive has written its state in this checkout (" + DriveStateRel + ")")

// ReadDriveState reads the drive's state stored under repoRoot and judges it at now.
// The value comes back as stored, its words with the home redacted, except where the
// drive that wrote it is gone (driveGone): then it reads stopped, since it was
// written, on the drive not heard from since then, with no act, which the line shows
// as needing a look. A value the drive stored as stopped is its own last line and is
// kept. Where there is no line to show the error says why: ErrNoDriveState for no
// file, or a refusal naming the file and what is wrong with it.
func ReadDriveState(repoRoot string, now time.Time) (DriveState, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return DriveState{}, fmt.Errorf("the checkout cannot be opened to read %s: %w", DriveStateRel, err)
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, DriveStateRel, maxDriveStateBytes)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return DriveState{}, ErrNoDriveState
	case errors.Is(err, fsutil.ErrNotRegular):
		return DriveState{}, fmt.Errorf("%s is not a regular file, so it is not one a drive wrote", DriveStateRel)
	case errors.Is(err, fsutil.ErrTooBig):
		return DriveState{}, fmt.Errorf("%s is larger than %d bytes, so it is not one a drive wrote", DriveStateRel, maxDriveStateBytes)
	case err != nil:
		return DriveState{}, fmt.Errorf("%s cannot be read: %w", DriveStateRel, err)
	}
	s, err := parseDriveState(data)
	if err != nil {
		return DriveState{}, err
	}
	if s.State != DriveStopped && driveGone(s.Process, s.Next, now) {
		s = stoppedUnheard(s)
	}
	return s, nil
}

// parseDriveState is the stored value in data, checked as a reader takes it: a
// version this abcd reads, one of the four words, a time it was written, and no
// control code in any word, since the drive stores each word masked (driveText) and a
// file holding one was not written by it. The words come back with the home
// redacted.
func parseDriveState(data []byte) (DriveState, error) {
	var head struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return DriveState{}, fmt.Errorf("%s is not a drive's state: %s", DriveStateRel, termsafe.Sanitize(err.Error()))
	}
	switch {
	case head.Version > DriveStateVersion:
		return DriveState{}, fmt.Errorf("%s states version %d, newer than the %d this abcd reads: a newer abcd wrote it", DriveStateRel, head.Version, DriveStateVersion)
	case head.Version < 1:
		return DriveState{}, fmt.Errorf("%s states no version, so it is not one a drive wrote", DriveStateRel)
	}
	var s DriveState
	if err := json.Unmarshal(data, &s); err != nil {
		return DriveState{}, fmt.Errorf("%s is not a drive's state: %s", DriveStateRel, termsafe.Sanitize(err.Error()))
	}
	for _, f := range []struct{ name, v string }{{"state", s.State}, {"on", s.On}, {"do", s.Do}} {
		if termsafe.SanitizeLine(f.v) != f.v {
			return DriveState{}, fmt.Errorf("%s carries a control code in its %s, so it is not one a drive wrote", DriveStateRel, f.name)
		}
	}
	switch s.State {
	case DriveWorking, DriveWaiting, DriveStalled, DriveStopped:
	default:
		return DriveState{}, fmt.Errorf("%s states %q, which is none of %s, %s, %s or %s", DriveStateRel, s.State, DriveWorking, DriveWaiting, DriveStalled, DriveStopped)
	}
	if s.Written.IsZero() {
		return DriveState{}, fmt.Errorf("%s names no time it was written, so it is not one a drive wrote", DriveStateRel)
	}
	s.On, s.Do = driveText(s.On), driveText(s.Do)
	return s, nil
}

// driveGone reports whether the drive whose process is p, and which said it would
// write again by next, is gone at now. A process recorded with its start is judged
// by the machine's process table (readProcs, then Table.Alive with the slack the
// record keeps, as a run's driver is): gone where its id is not in the table, or is
// held by a process that started at another time. A record with no start, or a table
// that cannot be read, falls back to next: a drive past the time it said it would
// write again is gone, so it can read working for no longer than its own margin.
func driveGone(p DriveProcess, next, now time.Time) bool {
	if p.PID > 0 && p.StartUnix > 0 {
		if tab, err := readProcs(); err == nil {
			return !tab.Alive(LiveDriver{PID: p.PID, StartUnix: p.StartUnix, SlackMS: p.SlackMS}.host())
		}
	}
	return next.IsZero() || now.After(next)
}

// stoppedUnheard is s as a reader shows a drive that is gone: stopped since it was
// last written, on the drive not heard from since then, with no until and no act.
// The lanes and the process are kept as written.
func stoppedUnheard(s DriveState) DriveState {
	s.State, s.Since, s.Until, s.Do = DriveStopped, s.Written, nil, ""
	s.On = "the drive, not heard from since " + s.Written.UTC().Format(time.RFC3339)
	return s
}
