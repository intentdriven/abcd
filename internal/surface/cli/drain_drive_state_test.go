package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/board"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/statusblock"
)

// The drive's state at the terminal (spc-2611141617199772 step 4): the loop hands each
// value it writes to the surface, which prints a `state:` line when the state's word,
// what it is on or its act changes, and not otherwise.

var stateT0 = time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)

func stateAt(m int) time.Time { return stateT0.Add(time.Duration(m) * time.Minute) }

// driveStates is a stretch of a drive's values: a wait rewritten at each look, a call at
// each tick, then a change of each field the line is printed for.
func driveStates() []loop.DriveState {
	pace := stateAt(30)
	return []loop.DriveState{
		{State: loop.DriveWaiting, Since: stateT0, Until: &pace, On: "the pace window", Written: stateT0},
		{State: loop.DriveWaiting, Since: stateT0, Until: &pace, On: "the pace window", Written: stateAt(10)},
		{State: loop.DriveWaiting, Since: stateT0, Until: &pace, On: "the pace window", Written: stateAt(20)},
		{State: loop.DriveWorking, Since: stateAt(30), On: "iss-1 at implement", Written: stateAt(30)},
		{State: loop.DriveWorking, Since: stateAt(30), On: "iss-1 at implement", Written: stateAt(35)},
		{State: loop.DriveWorking, Since: stateAt(40), On: "iss-1 at validate", Written: stateAt(40)},
		{State: loop.DriveWaiting, Since: stateAt(41), On: "iss-1, which another driver holds", Do: "end process 42 (login)", Written: stateAt(41)},
		{State: loop.DriveWaiting, Since: stateAt(41), On: "iss-1, which another driver holds", Written: stateAt(42)},
		{State: loop.DriveStalled, Since: stateAt(41), On: "iss-1, which another driver holds", Written: stateAt(101)},
		{State: loop.DriveStalled, Since: stateAt(41), On: "iss-1, which another driver holds", Written: stateAt(130)},
	}
}

// wantStateLines are the lines driveStates prints, each composed at its value's written
// time.
var wantStateLines = []string{
	"state: waiting 0m · on the pace window · you: nothing to do",
	"state: working 0m · on iss-1 at implement · you: nothing to do",
	"state: working 0m · on iss-1 at validate · you: nothing to do",
	"state: waiting 0m · on iss-1, which another driver holds · you: end process 42 (login)",
	"state: waiting 1m · on iss-1, which another driver holds · you: nothing to do",
	"state: stalled 1h00m · on iss-1, which another driver holds · you: needs a look",
}

// A `state:` line is printed when the state's word, its on or its act changes, and not
// otherwise: a wait rewritten at each look and a call at each tick add none.
func TestADriveStateLineIsPrintedOnAChangeAndNotOtherwise(t *testing.T) {
	var said []string
	show := driveStateLines(func(s string) { said = append(said, s) })
	for _, s := range driveStates() {
		show(s)
	}
	if strings.Join(said, "\n") != strings.Join(wantStateLines, "\n") {
		t.Fatalf("the state lines:\n%s\nwant:\n%s", strings.Join(said, "\n"), strings.Join(wantStateLines, "\n"))
	}
}

// The drive hands the loop its State, whose lines reach the terminal through the drive's
// own say, as each change comes.
func TestADriveHandsItsStateLinesToTheTerminal(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		if o.State == nil {
			t.Fatal("the drive hands the loop no State")
		}
		for _, s := range driveStates() {
			o.State(s)
		}
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	var got []string
	for _, l := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(l, "state: ") {
			got = append(got, l)
		}
	}
	if strings.Join(got, "\n") != strings.Join(wantStateLines, "\n") {
		t.Fatalf("the terminal's state lines:\n%s\nwant:\n%s\nall:\n%s", strings.Join(got, "\n"), strings.Join(wantStateLines, "\n"), out.String())
	}
}

// Where only some of a lane's holds were left to lapse, a renewal of the run keeps every
// one of them, so the end names each issue with what became of its hold and never says the
// lane's reservation is left to lapse (dcn-2611271952390075).
func TestADriveEndNamesEachHoldWhereOnlySomeWereLeftToLapse(t *testing.T) {
	since := stateT0
	res := loop.TerminalResult{Ended: loop.TerminalUntil, Resume: loop.TerminalResumeCommand,
		SetAside: []loop.SetAsideLane{{Issue: "iss-1", Run: "run-1", Reason: "the landing's record is in no status folder", Since: since, At: since.Add(time.Hour),
			Holds: []loop.SetAsideHold{{Issue: "iss-1", LeftToLapse: true}, {Issue: "iss-2"}, {Issue: "iss-3", LeftToLapse: true}}}}}
	var b bytes.Buffer
	renderDriveEnd(&b, res)
	out := b.String()
	want := "  set aside: iss-1, run run-1, after waiting 1h0m0s: the landing's record is in no status folder; the reservation of iss-1, iss-3 was left to lapse and that of iss-2 could not be, so a renewal of the run keeps each, and each lapses with its lease " + loop.UnlessRunRenewed
	if !strings.Contains(out, want) {
		t.Fatalf("the end does not name the holds per issue:\n%s\nwant:\n%s", out, want)
	}
	if strings.Contains(out, loop.UnlessKeptTakenUp) || strings.Contains(out, "the driver's that holds the run") {
		t.Fatalf("the end says the lane is left to lapse, or a driver's:\n%s", out)
	}
}

// standDriveNow has the status surfaces compose the drive's line at at.
func standDriveNow(t *testing.T, at time.Time) {
	t.Helper()
	old := driveNow
	driveNow = func() time.Time { return at }
	t.Cleanup(func() { driveNow = old })
}

// threeWay is the drive's line as each of the three places prints it for the one value
// s, stored in the checkout at root and read back at the time it was written: the
// drive's terminal, `abcd implement status` and the board's facilitator view, each with
// its own label taken off. The product thinker's view draws the word and the until
// alone (dcn-2611280725169564), so it is not one of the three.
func threeWay(t *testing.T, root string, s loop.DriveState) (terminal, status, boardLine string) {
	t.Helper()
	storeDriveState(t, root, s)
	standDriveNow(t, s.Written)
	show := driveStateLines(func(l string) { terminal = l })
	show(s)
	var out, errb strings.Builder
	renderDriveShown(&out, driveShownIn(root, &errb))
	status = strings.TrimSuffix(out.String(), "\n")
	d := boardDrive(root, &errb)
	if errb.Len() != 0 {
		t.Fatalf("the stored value was not read: %s", errb.String())
	}
	for _, l := range board.Render(board.Input{Status: &statusblock.Block{}, Drive: d.forBoard()}, board.Frame{View: board.Facilitator, Form: board.Text, Width: 200, Rung: board.Mono}) {
		if strings.HasPrefix(l, "  drive:") {
			boardLine = l
		}
	}
	return strings.TrimPrefix(terminal, "state: "), strings.TrimPrefix(status, "drive: "), strings.TrimPrefix(boardLine, "  drive:      ")
}

// The drive's terminal, `abcd implement status` and the board's facilitator view print
// the one line, composed once from one stored value, byte for byte (criterion 6), in
// each state a drive stores.
func TestTheTerminalTheStatusAndTheBoardShowOneDriveLine(t *testing.T) {
	root := managedCheckout(t)
	self := thisProcess(t)
	written := stateAt(90)
	until := stateAt(120)
	for _, s := range []loop.DriveState{
		{State: loop.DriveWorking, Since: stateAt(84), On: "iss-1 at implement"},
		{State: loop.DriveWaiting, Since: stateAt(60), Until: &until, On: "the pace window"},
		{State: loop.DriveWaiting, Since: stateAt(41), On: "iss-1, which another driver holds", Do: "end process 42 (login)"},
		{State: loop.DriveStalled, Since: stateT0, On: "3 lanes handed back in a row, stuck at validate"},
		{State: loop.DriveStopped, Since: stateAt(88), On: "its end: until", Do: "run `abcd drain --drive` to go on"},
	} {
		s.Version, s.Process, s.Written, s.Next = loop.DriveStateVersion, self, written, written.Add(10*time.Minute)
		terminal, status, boardLine := threeWay(t, root, s)
		if terminal == "" || terminal != status || terminal != boardLine {
			t.Fatalf("the three places differ for %s:\nterminal: %q\nstatus:   %q\nboard:    %q", s.State, terminal, status, boardLine)
		}
		if want := loop.DriveLine(s, written); terminal != want {
			t.Fatalf("the line is %q, want %q", terminal, want)
		}
	}
}

// runOut runs the CLI with args and returns its stdout, failing on a nonzero exit or on
// anything said on stderr.
func runOut(t *testing.T, args ...string) string {
	t.Helper()
	var out, errb strings.Builder
	if code := Run(args, &out, &errb); code != 0 || errb.Len() != 0 {
		t.Fatalf("abcd %s exited %d:\n%s%s", strings.Join(args, " "), code, out.String(), errb.String())
	}
	return out.String()
}

// The same at the commands themselves: `abcd implement status`, with --run too, prints
// the drive's line above the runs, and the board's facilitator view as its drive: row,
// byte for byte the line the drive's terminal printed for the stored value, which each
// --json carries under drive; the product thinker's view shows the word alone.
func TestTheStatusCommandAndTheBareBoardPrintTheDrivesLine(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	writeRunState(t, root, "itd-7")
	setBannerTTY(t, false)
	setBoardWidth(t, 200)
	written := stateAt(90)
	s := loop.DriveState{Version: loop.DriveStateVersion, State: loop.DriveWaiting, Since: stateAt(41), On: "iss-1, which another driver holds",
		Do: "end process 42 (login)", Process: thisProcess(t), Written: written, Next: written.Add(10 * time.Minute)}
	storeDriveState(t, root, s)
	standDriveNow(t, written)
	var terminal string
	driveStateLines(func(l string) { terminal = strings.TrimPrefix(l, "state: ") })(s)
	if terminal != "waiting 49m · on iss-1, which another driver holds · you: end process 42 (login)" {
		t.Fatalf("the terminal's line: %q", terminal)
	}
	for _, args := range [][]string{{"implement", "status"}, {"implement", "status", "--run", "run-2609290000000001"}} {
		if out := runOut(t, args...); !strings.HasPrefix(out, "drive: "+terminal+"\n") {
			t.Fatalf("abcd %s does not open on the drive's line %q:\n%s", strings.Join(args, " "), terminal, out)
		}
	}
	if out := runOut(t, "--view", "facilitator"); !strings.Contains(out, "\n  drive:      "+terminal+"\n") {
		t.Fatalf("the facilitator view's drive: row is not the drive's line %q:\n%s", terminal, out)
	}
	if out := runOut(t); !strings.Contains(out, "\n● building: drive waiting\n") || strings.Contains(out, "iss-1") {
		t.Fatalf("the product view does not show the drive's word alone:\n%s", out)
	}
	for _, args := range [][]string{{"--json"}, {"implement", "status", "--json"}} {
		var got struct {
			Drive *struct {
				Line  string          `json:"line"`
				State loop.DriveState `json:"state"`
			} `json:"drive"`
		}
		out := runOut(t, args...)
		if err := json.Unmarshal([]byte(out), &got); err != nil || got.Drive == nil || got.Drive.Line != terminal || got.Drive.State.State != loop.DriveWaiting {
			t.Fatalf("abcd %s --json does not carry the drive's line (%v):\n%s", strings.Join(args, " "), err, out)
		}
	}
}

// With no state file the status command and the board show no drive and say nothing
// about one on either stream (dcn-2611280725158656); with a file of a newer abcd they
// show none and say why on stderr.
func TestTheStatusCommandAndTheBoardOnADriveStateFileOrNone(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	setBannerTTY(t, false)
	for _, newer := range []bool{false, true} {
		if newer {
			storeDriveStateBody(t, root, `{"version":99,"state":"working","written":"2026-11-25T10:10:00Z"}`)
		}
		for _, args := range [][]string{{}, {"--view", "facilitator"}, {"--json"}, {"implement", "status"}, {"implement", "status", "--json"}} {
			var out, errb strings.Builder
			if code := Run(args, &out, &errb); code != 0 {
				t.Fatalf("abcd %s exited %d:\n%s%s", strings.Join(args, " "), code, out.String(), errb.String())
			}
			if strings.Contains(out.String(), `"drive"`) || strings.Contains(out.String(), "drive: ") || strings.Contains(out.String(), "building: drive") {
				t.Fatalf("abcd %s shows a drive:\n%s", strings.Join(args, " "), out.String())
			}
			said := strings.Contains(errb.String(), "drive")
			if !newer && said {
				t.Fatalf("abcd %s says something of a drive with no state file:\n%s", strings.Join(args, " "), errb.String())
			}
			if newer && !strings.Contains(errb.String(), "abcd: no drive state line — "+loop.DriveStateRel+" states version 99") {
				t.Fatalf("abcd %s does not say why a newer file shows no line:\n%s", strings.Join(args, " "), errb.String())
			}
		}
	}
}
