package loop

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

var driveT0 = time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)

func driveAt(m int) time.Time { return driveT0.Add(time.Duration(m) * time.Minute) }

// A wait with no named end reads waiting short of the stuck bound's minutes
// from when anything last moved, and stalled from the bound on (criterion 8).
func TestADriveWaitIsStalledOnlyPastTheBound(t *testing.T) {
	bound := 60 * time.Minute
	if got := stallWord(driveT0, time.Time{}, bound, driveAt(59)); got != DriveWaiting {
		t.Fatalf("59 minutes into a wait with no end: %q, want %q", got, DriveWaiting)
	}
	if got := stallWord(driveT0, time.Time{}, bound, driveAt(60)); got != DriveStalled {
		t.Fatalf("60 minutes into a wait with no end: %q, want %q", got, DriveStalled)
	}
	if got := stallWord(driveT0, time.Time{}, 0, driveAt(BundledStuckMinutes)); got != DriveStalled {
		t.Fatalf("no bound given reads the bundled stuck minutes: %q, want %q", got, DriveStalled)
	}
	if got := stallWord(time.Time{}, time.Time{}, bound, driveAt(600)); got != DriveWaiting {
		t.Fatalf("a wait with no start yet: %q, want %q", got, DriveWaiting)
	}
}

// A wait whose named end is still ahead is never stalled before it, however long
// the drive has gone without moving; once the end has passed the time rule holds.
func TestADriveWaitIsNeverStalledBeforeItsNamedEnd(t *testing.T) {
	bound := 60 * time.Minute
	if got := stallWord(driveT0, driveAt(605), bound, driveAt(600)); got != DriveWaiting {
		t.Fatalf("ten hours in with the named end five minutes ahead: %q, want %q", got, DriveWaiting)
	}
	if got := stallWord(driveT0, driveAt(595), bound, driveAt(600)); got != DriveStalled {
		t.Fatalf("ten hours in with the named end passed: %q, want %q", got, DriveStalled)
	}
	w := waitState(driveWait{Start: driveT0, Until: driveAt(605), On: "the pace window"}, bound, driveAt(600))
	if w.Until == nil || !w.Until.Equal(driveAt(605)) {
		t.Fatalf("a wait with its end ahead carries it as Until: %v", w.Until)
	}
	w = waitState(driveWait{Start: driveT0, Until: driveAt(595), On: "the pace window"}, bound, driveAt(600))
	if w.Until != nil {
		t.Fatalf("an end already passed is no Until: %v", w.Until)
	}
}

// The prototype's first error, written as a test: a wait whose reason alternates
// with the pace window's line keeps the since of the last movement, and is
// stalled at the bound.
func TestADriveSinceSurvivesAlternatingReasons(t *testing.T) {
	bound := 60 * time.Minute
	steps := []struct {
		at    int
		w     driveWait
		state string
	}{
		{30, driveWait{Start: driveT0, On: "an outage"}, DriveWaiting},
		{45, driveWait{Start: driveT0, Until: driveAt(50), On: "the pace window"}, DriveWaiting},
		{52, driveWait{Start: driveT0, On: "a look at origin"}, DriveWaiting},
		{61, driveWait{Start: driveT0, On: "an outage"}, DriveStalled},
		{70, driveWait{Start: driveT0, On: "a look at origin"}, DriveStalled},
	}
	for _, s := range steps {
		got := waitState(s.w, bound, driveAt(s.at))
		if got.State != s.state || !got.Since.Equal(driveT0) {
			t.Fatalf("at +%dm on %s: %s since %v, want %s since %v", s.at, s.w.On, got.State, got.Since, s.state, driveT0)
		}
	}
}

// While every lane left is set aside the drive retries them each hour, and that
// retry is a named end: the line reads waiting however long it lasts (decision 9).
func TestTheSetAsideRetryReadsWaitingHoweverLong(t *testing.T) {
	bound := 60 * time.Minute
	for h := 0; h < 48; h++ {
		for _, m := range []int{0, 1, 30, 59} {
			now := driveAt(h*60 + m)
			next := driveAt((h + 1) * 60)
			got := waitState(driveWait{Start: driveT0, Until: next, On: "every lane left is set aside"}, bound, now)
			if got.State != DriveWaiting || got.Until == nil || !got.Until.Equal(next) {
				t.Fatalf("hour %d minute %d of the retries: %s until %v, want waiting until %v", h, m, got.State, got.Until, next)
			}
		}
	}
}

// A remedy is shown only where it is a person's act at a terminal: one written for
// a host session, such as handing back a receipt, is never the act (F-21).
func TestAHostSessionsRemedyIsNeverTheDriveAct(t *testing.T) {
	cases := []struct {
		act  driveAct
		want string
	}{
		{driveAct{Words: "start an agent by hand and run `abcd implement receipt`"}, ""},
		{driveAct{Words: "run `abcd implement receipt --run r1`", Person: true}, ""},
		{driveAct{Words: "run `abcd implement step --run r1`", Person: true}, ""},
		{driveAct{Words: "lift the stop with `abcd implement stop --lift`"}, ""},
		{driveAct{Words: "lift the stop with `abcd implement stop --lift`", Person: true}, "lift the stop with `abcd implement stop --lift`"},
		{driveAct{Person: true}, ""},
	}
	for _, c := range cases {
		if got := c.act.shown(); got != c.want {
			t.Fatalf("%+v shown as %q, want %q", c.act, got, c.want)
		}
	}
	w := waitState(driveWait{Start: driveT0, On: "iss-1 at land", Act: driveAct{Words: "hand back the receipt with `abcd implement receipt`"}}, time.Hour, driveAt(5))
	if w.Do != "" {
		t.Fatalf("a host session's remedy became the act: %q", w.Do)
	}
}

// The one composing function: <state> <elapsed> · on <what> · you: <act>, where no
// act reads "nothing to do" while working or waiting and "needs a look" while
// stalled or stopped.
func TestTheDriveLineSaysTheActWords(t *testing.T) {
	now := driveAt(164)
	cases := []struct {
		s    DriveState
		want string
	}{
		{DriveState{State: DriveWorking, Since: driveAt(159), On: "iss-1 at implement"}, "working 5m · on iss-1 at implement · you: nothing to do"},
		{DriveState{State: DriveWaiting, Since: driveAt(124), On: "the pace window"}, "waiting 40m · on the pace window · you: nothing to do"},
		{DriveState{State: DriveStalled, Since: driveT0, On: "outage"}, "stalled 2h44m · on outage · you: needs a look"},
		{DriveState{State: DriveStopped, Since: driveAt(104)}, "stopped 1h00m · you: needs a look"},
		{DriveState{State: DriveWaiting, Since: driveAt(154), On: "a stop", Do: "lift the stop"}, "waiting 10m · on a stop · you: lift the stop"},
		{DriveState{State: DriveStalled, Since: driveAt(160), On: "iss-2\x1b[31m at land", Do: "look\u2028here"}, "stalled 4m · on iss-2?[31m at land · you: look?here"},
	}
	for _, c := range cases {
		if got := DriveLine(c.s, now); got != c.want {
			t.Fatalf("line of %+v:\n got %q\nwant %q", c.s, got, c.want)
		}
	}
}

// The file is written whole, by rename, with a version of its own, mode 0600, and
// carries the drive's process, when it was written and when it will be again.
func TestTheDriveStateFileIsWholeAndVersioned(t *testing.T) {
	repo := t.TempDir()
	home, _ := os.UserHomeDir()
	in := DriveState{
		State:   DriveWaiting,
		Since:   driveT0,
		On:      "iss-1 at land in " + filepath.Join(home, "x") + "\x07",
		Lanes:   DriveLanes{Done: 2, InFlight: 1, SetAside: 1, HandedBack: 3},
		Process: DriveProcess{PID: 4242, StartUnix: driveT0.Add(-time.Hour).Unix(), SlackMS: 250},
	}
	stored, err := writeDriveState(repo, in, driveAt(7), terminalRenewEvery)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, filepath.FromSlash(DriveStateRel))
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
		t.Fatalf("the state file is %v, want a regular file of mode 0600", fi.Mode())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got DriveState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("the state file is not whole JSON: %v\n%s", err, data)
	}
	if got.Version != DriveStateVersion || got.Version < 1 {
		t.Fatalf("version %d, want %d", got.Version, DriveStateVersion)
	}
	if got.Process != in.Process || got.Lanes != in.Lanes {
		t.Fatalf("process %+v lanes %+v, want %+v %+v", got.Process, got.Lanes, in.Process, in.Lanes)
	}
	if !got.Written.Equal(driveAt(7)) || !got.Next.Equal(driveAt(7).Add(terminalRenewEvery)) {
		t.Fatalf("written %v next %v, want %v and %v", got.Written, got.Next, driveAt(7), driveAt(7).Add(terminalRenewEvery))
	}
	if strings.Contains(got.On, "\x07") || (home != "" && strings.Contains(got.On, home)) {
		t.Fatalf("On stored with a control code or the home: %q", got.On)
	}
	if got.On != stored.On || DriveLine(got, driveAt(9)) != DriveLine(stored, driveAt(9)) {
		t.Fatalf("the value handed back differs from the one stored: %+v vs %+v", stored, got)
	}
	// A second write replaces the first whole and leaves no temporary file.
	if _, err := writeDriveState(repo, DriveState{State: DriveStopped, Since: driveAt(8), Process: in.Process}, driveAt(8), terminalRenewEvery); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != filepath.Base(path) {
		t.Fatalf("the logs folder holds %v, want the state file alone", ents)
	}
	data, _ = os.ReadFile(path)
	got = DriveState{}
	if err := json.Unmarshal(data, &got); err != nil || got.State != DriveStopped || got.On != "" {
		t.Fatalf("the second write did not replace the first whole: %v %+v", err, got)
	}
}

// A symlinked logs folder is refused, as the drive's log refuses it, and nothing
// is written where it points.
func TestADriveStateWriteRefusesASymlinkedLogsFolder(t *testing.T) {
	repo := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(repo, ".abcd", ".work.local", "logs")); err != nil {
		t.Fatal(err)
	}
	_, err := writeDriveState(repo, DriveState{State: DriveWorking, Since: driveT0}, driveT0, terminalRenewEvery)
	if !errors.Is(err, fsutil.ErrNotRealDir) {
		t.Fatalf("a symlinked logs folder: %v, want ErrNotRealDir", err)
	}
	if ents, _ := os.ReadDir(elsewhere); len(ents) != 0 {
		t.Fatalf("written through the link: %v", ents)
	}
}

// Three lane hand-backs in a row with one key read stalled, since the first of
// them; hand-backs before any lane neither count nor clear; another key starts a
// new streak; a landed lane clears it (T7).
func TestTheHandBackStreakReadsStalled(t *testing.T) {
	stuck := &LaneRefused{Stage: "validate", Check: "lint", Reason: "3 findings in run r1"}
	lane := func(issue string) DrainRoute { return DrainRoute{Issue: issue, From: "lane", Kind: HandBackStuck} }
	var s handBackStreak
	s.note(lane("iss-1"), stuck, driveAt(1))
	s.note(DrainRoute{Issue: "iss-9", From: "judgement", Kind: "decision"}, nil, driveAt(2))
	s.note(lane("iss-2"), &LaneRefused{Stage: "validate", Check: "lint", Reason: "7 findings in run r2, 40 minutes"}, driveAt(3))
	if s.stalled(3) {
		t.Fatal("stalled at two hand-backs")
	}
	s.note(DrainRoute{Issue: "iss-8", From: "reservation", Kind: "found_at"}, nil, driveAt(4))
	s.note(DrainRoute{Issue: "iss-7", From: "field", Kind: "rule"}, nil, driveAt(5))
	s.note(lane("iss-3"), stuck, driveAt(6))
	if !s.stalled(3) {
		t.Fatalf("not stalled at three hand-backs with one key: %+v", s)
	}
	if s.stalled(5) {
		t.Fatal("a raised bound still read stalled at three")
	}
	working := DriveState{State: DriveWorking, Since: driveAt(6), On: "iss-4 at implement"}
	got := s.over(working, 3)
	if got.State != DriveStalled || !got.Since.Equal(driveAt(1)) || got.Do != "" {
		t.Fatalf("the streak over working: %+v", got)
	}
	if want := "3 lanes handed back in a row, stuck at validate (lint)"; got.On != want {
		t.Fatalf("the streak's on: %q, want %q", got.On, want)
	}
	if !strings.HasSuffix(DriveLine(got, driveAt(10)), "you: needs a look") {
		t.Fatalf("the streak's line: %q", DriveLine(got, driveAt(10)))
	}
	stopped := DriveState{State: DriveStopped, Since: driveAt(9)}
	if got := s.over(stopped, 3); got.State != DriveStopped {
		t.Fatalf("the streak over a stopped drive: %+v", got)
	}
	// Another key starts a new streak at one.
	s.note(lane("iss-4"), &LaneRefused{Stage: "land", Check: "preflight"}, driveAt(11))
	if s.stalled(1) == false || s.stalled(2) {
		t.Fatalf("another key did not start a new streak at one: %+v", s)
	}
	s.note(lane("iss-5"), &LaneRefused{Stage: "land", Check: "preflight"}, driveAt(12))
	s.note(lane("iss-6"), &LaneRefused{Stage: "land", Check: "preflight"}, driveAt(13))
	if !s.stalled(3) {
		t.Fatal("not stalled at three of the new key")
	}
	s.landed()
	if s.stalled(1) {
		t.Fatalf("a landed lane did not clear the streak: %+v", s)
	}
	if got := s.over(working, 3); got.State != DriveWorking {
		t.Fatalf("a cleared streak changed the state: %+v", got)
	}
	// A kind that differs is another key.
	s.note(lane("iss-10"), stuck, driveAt(20))
	s.note(DrainRoute{Issue: "iss-11", From: "lane", Kind: HandBackClosed}, stuck, driveAt(21))
	if s.stalled(2) {
		t.Fatalf("two kinds counted as one: %+v", s)
	}
	// No bound given reads the bundled count.
	for i := range BundledStuckRefusals - 1 {
		s.note(DrainRoute{Issue: "iss-2" + string(rune('0'+i)), From: "lane", Kind: HandBackClosed}, stuck, driveAt(22+i))
	}
	if !s.stalled(0) {
		t.Fatalf("no bound did not read the bundled %d: %+v", BundledStuckRefusals, s)
	}
}
