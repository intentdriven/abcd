package cli

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/board"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The drive's state on the status surfaces (spc-2611141617199772 step 5;
// itd-2611141610197753 criteria 6 and 7): the board and `abcd implement status` show
// the line the drive's own terminal prints, composed by the one function
// (loop.DriveLine) from the one stored value, read through the loop
// (loop.ReadDriveState), which judges whether the drive that wrote it still runs.
// These readers format and print; they decide nothing.

// readDriveState is the loop's reader of the stored state; a test stands one in.
var readDriveState = loop.ReadDriveState

// driveNow is the time a status surface composes the drive's line at; a test stands
// one in.
var driveNow = time.Now

// driveShown is the drive's state as a status surface shows it, in --json too: the
// value as read, a gone drive read as stopped, and the line composed from it.
type driveShown struct {
	State loop.DriveState `json:"state"`
	Line  string          `json:"line"`
	// until is the wait's named end as the board's product view words it, ""
	// for none.
	until string
}

// forBoard is the drive's state as the board draws it, nil where there is none: the
// product thinker's view draws the word and the until alone, the facilitator's the
// whole line (dcn-2611280725169564).
func (d *driveShown) forBoard() *board.Drive {
	if d == nil {
		return nil
	}
	return &board.Drive{State: d.State.State, Until: d.until, Line: d.Line}
}

// driveUntil is a named end as the product view words it, in the person's own zone:
// the clock time where it falls on the same day as now ("14:00"), and the day before
// it otherwise ("Nov 29 14:00"); "" for none.
func driveUntil(u *time.Time, now time.Time, loc *time.Location) string {
	if u == nil {
		return ""
	}
	at, here := u.In(loc), now.In(loc)
	if at.YearDay() == here.YearDay() && at.Year() == here.Year() {
		return at.Format("15:04")
	}
	return at.Format("Jan 2 15:04")
}

// boardDrive is the drive's state for the board, in the checkout containing cwd:
// nil outside a checkout abcd manages, as the board's status block is, and nil where
// there is no line to show (driveShownIn).
func boardDrive(cwd string, stderr io.Writer) *driveShown {
	root, err := gitutil.CheckoutRoot(cwd, statusBlockStore)
	if err != nil || !ahoy.Managed(root) {
		return nil
	}
	return driveShownIn(root, stderr)
}

// driveShownIn reads the drive's state stored in the checkout at root and composes its
// line now. A file that is not there is the ordinary case of a checkout no drive ran
// in: nil, and nothing said on either stream (dcn-2611280725158656). One that is there
// but cannot be read, states a newer version or carries a control code shows no line
// either: nil, with why on stderr, the reason masked and its home redacted, so the
// surface itself never fails on it.
func driveShownIn(root string, stderr io.Writer) *driveShown {
	now := driveNow()
	s, err := readDriveState(root, now)
	switch {
	case errors.Is(err, loop.ErrNoDriveState):
		return nil
	case err != nil:
		fmt.Fprintf(stderr, "abcd: no drive state line — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return nil
	}
	return &driveShown{State: s, Line: loop.DriveLine(s, now), until: driveUntil(s.Until, now, time.Local)}
}

// renderDriveShown prints the drive's line for `abcd implement status`, above the
// runs, under its own label; nothing where there is none to show.
func renderDriveShown(w io.Writer, d *driveShown) {
	if d != nil {
		fmt.Fprintf(w, "drive: %s\n", d.Line)
	}
}
