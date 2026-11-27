package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
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

// The wired half (spc-2611141617199772 step 4): the terminal loop sets the state at each
// of its waits and hands it to the surface, and writes the file.

// stateDrive is a terminal on root with a clock its waits move, keeping every value its
// State is handed.
type stateDrive struct {
	tm     *terminal
	clk    *clock
	states []DriveState
	lines  []string
}

func newStateDrive(t *testing.T, root string) *stateDrive {
	t.Helper()
	d := &stateDrive{clk: &clock{t: driveT0}}
	d.tm = newTerminal(root, TerminalOptions{
		Options: Options{Now: d.clk.now},
		Say:     func(s string) { d.lines = append(d.lines, s) },
		State:   func(s DriveState) { d.states = append(d.states, s) },
		Sleep:   func(_ context.Context, dd time.Duration) { d.clk.t = d.clk.t.Add(dd) },
	})
	return d
}

// last is the value last handed, failing when none was.
func (d *stateDrive) last(t *testing.T) DriveState {
	t.Helper()
	if len(d.states) == 0 {
		t.Fatalf("no state was handed:\n%s", strings.Join(d.lines, "\n"))
	}
	return d.states[len(d.states)-1]
}

// stateRoot is a git checkout whose layer sets the stuck bounds, so the machine's own layer
// moves no figure a test reads.
func stateRoot(t *testing.T, minutes, refusals int) string {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repoConfig(t, repo, fmt.Sprintf(`{"pace": {"stuck_minutes": %d, "stuck_refusals": %d}}`, minutes, refusals))
	return repo.Root()
}

// wantState fails unless s says state since since, on on, with an until of until (zero
// for none) and the act do.
func wantState(t *testing.T, s DriveState, state string, since time.Time, on string, until time.Time, do string) {
	t.Helper()
	gotUntil := time.Time{}
	if s.Until != nil {
		gotUntil = *s.Until
	}
	if s.State != state || !s.Since.Equal(since) || s.On != on || !gotUntil.Equal(until) || s.Do != do {
		t.Fatalf("state %s since %v on %q until %v do %q,\nwant %s since %v on %q until %v do %q",
			s.State, s.Since, s.On, gotUntil, s.Do, state, since, on, until, do)
	}
}

// Each wait sets the state, its since and its act: the pace window sets an until and is
// waiting before it, past the bound; a look at origin and a cap wait set none, and keep
// the since of the last movement, so a wait whose reason alternates with the pace
// window's line is stalled at the bound; a movement clears the start, and a failure after
// it sets its own; an outage keeps that start; a person's act at the claim is shown and a
// host session's remedy never is.
func TestEachDriveWaitSetsItsStateSinceAndAct(t *testing.T) {
	d := newStateDrive(t, stateRoot(t, 60, 3))
	ctx := context.Background()
	pace := driveAt(90)
	d.tm.waitFor(moveWaitOn(DrainResult{NextEligibleAt: &pace}))
	d.tm.sleep(ctx, 90*time.Minute, "the pace window")
	if len(d.states) != 2 {
		t.Fatalf("a wait of ninety minutes is written at its start and after its first hour: %d values", len(d.states))
	}
	for _, s := range d.states {
		wantState(t, s, DriveWaiting, driveT0, driveOnPace, pace, "")
	}
	if got := d.states[1].Written; !got.Equal(driveAt(60)) {
		t.Fatalf("the second look is written at the first stretch's end: %v", got)
	}
	if next := d.states[0].Next; !next.Equal(driveAt(60).Add(driveStateMargin)) {
		t.Fatalf("a stretch's value is due again at the stretch's end and its margin: %v", next)
	}
	origin := DrainOriginWait{NextProbeAt: driveAt(120)}
	d.tm.waitFor(moveWaitOn(DrainResult{OriginWait: &origin}))
	d.tm.sleep(ctx, 30*time.Minute, "origin")
	wantState(t, d.last(t), DriveStalled, driveT0, driveOnOrigin, time.Time{}, "")
	held := driveAt(200)
	d.tm.waitFor(moveWaitOn(DrainResult{WaitingUntil: &held}))
	d.tm.sleep(ctx, time.Minute, "the cap")
	wantState(t, d.last(t), DriveStalled, driveT0, driveOnHeldWork, time.Time{}, "")

	d.tm.moving("iss-1 at implement")
	moved := d.clk.t
	wantState(t, d.last(t), DriveWorking, moved, "iss-1 at implement", time.Time{}, "")
	d.clk.t = d.clk.t.Add(5 * time.Minute)
	first := d.clk.t
	d.tm.failed(ctx, false, "iss-1 did not move")
	wantState(t, d.last(t), DriveWaiting, first, "a failure, tried again", time.Time{}, "")
	for range 3 {
		d.tm.failed(ctx, false, "iss-1 did not move")
	}
	wantState(t, d.last(t), DriveWaiting, first, "an outage", time.Time{}, "")
	d.tm.moved()
	again := d.clk.t
	d.tm.failed(ctx, false, "iss-1 did not move")
	wantState(t, d.last(t), DriveWaiting, again, "a failure, tried again", time.Time{}, "")

	d.tm.waitFor("iss-1, which another driver holds", time.Time{}, driveAct{Words: "end process 42 (login)", Person: true})
	d.tm.idle(ctx, TerminalWaitFloor, "held by process 42")
	wantState(t, d.last(t), DriveWaiting, again, "iss-1, which another driver holds", time.Time{}, "end process 42 (login)")
	d.tm.waitFor("iss-1 at land", time.Time{}, driveAct{Words: "run `abcd implement receipt --run r1`", Person: true})
	d.tm.sleep(ctx, TerminalWaitFloor, "the receipt")
	wantState(t, d.last(t), DriveWaiting, again, "iss-1 at land", time.Time{}, "")
}

// An outage on a fixture drive: its start is set at the first failure after the lane
// moved, every wait of it keeps that start and names no end, and it reads waiting short
// of the stuck bound's minutes and stalled past them; the drive's end is stopped, and
// the file holds the last value with the drive's own process.
func TestAnOutageReadsStalledFromItsFirstFailure(t *testing.T) {
	w := newTerminalWorld(t, append([]string{"shape"}, repeated("none", 40)...)...)
	repoConfig(t, w.repo, `{"pace": {"stuck_minutes": 60}}`)
	o := w.options(t)
	o.Drain.Max = 1
	var states []DriveState
	o.State = func(s DriveState) { states = append(states, s) }
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	var since time.Time
	waiting, stalled := 0, 0
	for _, s := range states {
		if s.On != "an outage" && s.On != "a failure, tried again" {
			continue
		}
		if since.IsZero() {
			since = s.Since
		}
		if !s.Since.Equal(since) || s.Until != nil {
			t.Fatalf("a wait of the outage moved its start or named an end: %+v, first since %v", s, since)
		}
		want := DriveWaiting
		if s.Written.Sub(s.Since) >= time.Hour {
			want = DriveStalled
		}
		if s.State != want {
			t.Fatalf("%v into the outage it reads %s, want %s", s.Written.Sub(s.Since), s.State, want)
		}
		if s.State == DriveWaiting {
			waiting++
		} else {
			stalled++
		}
	}
	if waiting == 0 || stalled == 0 {
		t.Fatalf("the outage read waiting %d times and stalled %d times\n%s", waiting, stalled, w.said())
	}
	last := states[len(states)-1]
	if last.State != DriveStopped || last.Process.PID != os.Getpid() {
		t.Fatalf("the drive's end: %+v", last)
	}
	data, err := os.ReadFile(filepath.Join(w.repo.Root(), filepath.FromSlash(DriveStateRel)))
	if err != nil {
		t.Fatal(err)
	}
	var stored DriveState
	if err := json.Unmarshal(data, &stored); err != nil || stored.State != DriveStopped || stored.Version != DriveStateVersion || !stored.Written.Equal(last.Written) {
		t.Fatalf("the file does not hold the last value: %v %+v", err, stored)
	}
}

// A rate limit whose reset the provider names sets that reset as the wait's until and
// reads waiting before it, however long past the bound; one that names none sets no
// until, keeps the start of its first limit, and reads stalled past the bound.
func TestARateLimitSetsAnUntilOnlyWhereItNamesAReset(t *testing.T) {
	t.Run("a named reset", func(t *testing.T) {
		w := newTerminalWorld(t, "shape", "limited")
		w.realWorktree()
		repoConfig(t, w.repo, `{"pace": {"stuck_minutes": 60}}`)
		reset := w.clock.t.Add(3 * time.Hour).UTC().Truncate(time.Second)
		t.Setenv(terminalResetEnv, strconv.FormatInt(reset.Unix(), 10))
		o := w.options(t)
		o.Drain.Max = 1
		var limited []DriveState
		o.State = func(s DriveState) {
			if s.On == driveOnLimit {
				limited = append(limited, s)
			}
		}
		if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
			t.Fatal(err)
		}
		if len(limited) == 0 {
			t.Fatalf("no state named the limit\n%s", w.said())
		}
		for _, s := range limited {
			if s.State != DriveWaiting || s.Until == nil || !s.Until.Equal(reset) {
				t.Fatalf("a limit with a named reset: %+v, want waiting until %v", s, reset)
			}
		}
		if last := limited[len(limited)-1]; last.Written.Sub(last.Since) < time.Hour {
			t.Fatalf("premise: the wait outlasts the bound: %+v", last)
		}
	})
	t.Run("no reset", func(t *testing.T) {
		w := newTerminalWorld(t, append([]string{"shape"}, repeated("limited", 60)...)...)
		w.realWorktree()
		repoConfig(t, w.repo, `{"pace": {"stuck_minutes": 60}}`)
		o := w.options(t)
		o.Drain.Max = 1
		var limited []DriveState
		o.State = func(s DriveState) {
			if s.On == driveOnLimit {
				limited = append(limited, s)
			}
		}
		if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
			t.Fatal(err)
		}
		stalled := false
		for _, s := range limited {
			if s.Until != nil || !s.Since.Equal(limited[0].Since) {
				t.Fatalf("a limit with no reset named an end or moved its start: %+v", s)
			}
			stalled = stalled || s.State == DriveStalled
		}
		if !stalled {
			t.Fatalf("a limit with no reset never read stalled: %+v", limited)
		}
	})
}

// During an agent's call the state is rewritten at each renewal tick, from the tick's
// own goroutine, each due again a tick and its margin on.
func TestTheDriveStateIsRewrittenAtEachRenewalTick(t *testing.T) {
	was := terminalRenewTick
	terminalRenewTick = 20 * time.Millisecond
	t.Cleanup(func() { terminalRenewTick = was })
	w := newTerminalWorld(t, "shape", "slow")
	o := w.options(t)
	o.Drain.Max = 1
	var mu sync.Mutex
	working := 0
	o.State = func(s DriveState) {
		mu.Lock()
		defer mu.Unlock()
		if s.State == DriveWorking && strings.Contains(s.On, " at "+string(StageImplement)) {
			working++
			if !s.Next.Equal(s.Written.Add(terminalRenewTick + driveStateMargin)) {
				t.Errorf("a tick's value is due again at %v, want a tick and its margin after %v", s.Next, s.Written)
			}
		}
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, w.said())
	}
	mu.Lock()
	defer mu.Unlock()
	if working < 3 {
		t.Fatalf("the state was written %d times on the lane's call, want it rewritten at its ticks\n%s", working, w.said())
	}
}

// While every lane left is set aside, the drive's wait ends at its next hourly retry
// (decision 9): it reads waiting until then, however long nothing has moved.
func TestEveryLaneSetAsideWaitsUntilTheRetry(t *testing.T) {
	d := newStateDrive(t, stateRoot(t, 60, 3))
	d.tm.stillSince = driveAt(-600)
	d.tm.setAside = []SetAsideLane{{Issue: "iss-1", Run: "r1"}}
	d.tm.nextRetry = driveAt(60)
	if ended := d.tm.awaitSetAside(context.Background()); ended != "" {
		t.Fatalf("the wait ended the drive: %s", ended)
	}
	for _, s := range d.states {
		wantState(t, s, DriveWaiting, driveAt(-600), "1 lane set aside", driveAt(60), "")
		if s.Lanes.SetAside != 1 {
			t.Fatalf("the count of lanes set aside: %+v", s.Lanes)
		}
	}
}

// The stall by hand-backs, wired through the drive's moves: three lane hand-backs in a
// row with one key turn the state stalled since the first, on the count and kind; a
// judgement between them neither breaks nor extends the run; a bundle's lane routing its
// every issue counts once; a lane that reaches its pull request clears it; a person's act
// is never hidden by it; and pace.stuck_refusals set in a layer moves the count.
func TestThreeLaneHandBacksInARowReadStalled(t *testing.T) {
	root := stateRoot(t, 60, 3)
	if err := writeDrain(root, DrainState{SchemaVersion: DrainSchemaVersion, Lanes: []DrainLane{{Issue: "iss-10", RunID: "run-2611270000000010", Bundle: []string{"iss-11", "iss-12"}}}}); err != nil {
		t.Fatal(err)
	}
	d := newStateDrive(t, root)
	ctx := context.Background()
	lane := func(issue string, m int) DrainRoute {
		at := driveAt(m)
		return DrainRoute{Issue: issue, From: "lane", Kind: HandBackStuck, At: &at}
	}
	look := func() DriveState {
		d.tm.waitFor(driveOnNextMove, time.Time{}, driveAct{})
		d.tm.sleep(ctx, TerminalWaitFloor, "the next move")
		return d.last(t)
	}
	d.tm.sayMove(DrainResult{Routed: []DrainRoute{lane("iss-1", 1)}})
	d.tm.sayMove(DrainResult{Routed: []DrainRoute{{Issue: "iss-9", From: "judgement", Kind: "decision"}}})
	d.tm.sayMove(DrainResult{Routed: []DrainRoute{lane("iss-10", 2), lane("iss-11", 2), lane("iss-12", 2)}})
	if s := look(); s.State != DriveWaiting {
		t.Fatalf("two lanes handed back, a bundle's counted once, read %s: %+v", s.State, s)
	}
	d.tm.sayMove(DrainResult{Routed: []DrainRoute{lane("iss-3", 3)}})
	s := look()
	wantState(t, s, DriveStalled, driveAt(1), "3 lanes handed back in a row, stuck", time.Time{}, "")
	d.tm.waitFor("iss-4, which another driver holds", time.Time{}, driveAct{Words: "end process 42 (login)", Person: true})
	d.tm.sleep(ctx, TerminalWaitFloor, "held")
	if s := d.last(t); s.State != DriveWaiting || s.Do != "end process 42 (login)" {
		t.Fatalf("the streak hid a person's act: %+v", s)
	}
	d.tm.sayMove(DrainResult{Counts: DrainCounts{PullRequests: 1}})
	if s := look(); s.State != DriveWaiting {
		t.Fatalf("a lane that reached its pull request did not clear the streak: %+v", s)
	}

	two := stateRoot(t, 60, 2)
	d = newStateDrive(t, two)
	d.tm.sayMove(DrainResult{Routed: []DrainRoute{lane("iss-1", 1)}})
	d.tm.sayMove(DrainResult{Routed: []DrainRoute{lane("iss-2", 2)}})
	if s := look(); s.State != DriveStalled || !strings.HasPrefix(s.On, "2 lanes handed back") {
		t.Fatalf("pace.stuck_refusals of 2 in a layer did not move the count: %+v", s)
	}
}

// Where only some holds of a lane set aside could be left to lapse, a renewal of the run
// keeps every one of them, so the lane is not named left to lapse: each issue is named
// with whether its own hold was, the run is not counted among those left to lapse, and
// the set-aside's line names them issue by issue (dcn-2611271952390075).
func TestASetAsideLaneWithSomeHoldsLeftNamesThemPerIssue(t *testing.T) {
	repo, _ := bundleRepo(t)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	first := drainToLane(t, repo, Options{Now: c.now}, DrainOptions{Skip: []string{bundleTwo, bundleThree}, SkipReason: "the lead's bundle"})
	if first.Lane == nil {
		t.Fatalf("the drain opens a lane: %+v", first)
	}
	runID := first.Lane.RunID
	for _, id := range []string{bundleTwo, bundleThree} {
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: id, Footprint: []string{bundleFiles[id]}, Drain: true}, clockAt(c.t)); err != nil {
			t.Fatalf("reserving %s: %v", id, err)
		}
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		for _, id := range []string{bundleTwo, bundleThree} {
			when := c.t
			st.Bundle = append(st.Bundle, Member{Issue: id, Footprint: []string{bundleFiles[id]}, ReservedAt: &when})
			st.Record = append(st.Record, Entry{At: c.t, Stage: StageReservation, Note: "reserved " + id, Issue: id})
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	// The second member's note cannot be written: a folder stands where it goes.
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd", ".work.local", "reservation", "lapse", bundleTwo, "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	var lines []string
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}, Say: func(s string) { lines = append(lines, s) }})
	tm.setLaneAside(SetAsideLane{Issue: bundleLead, Run: runID, Reason: "the landing's record is gone", Since: c.t, At: c.t, site: blockRecord}, false)
	if len(tm.setAside) != 1 {
		t.Fatalf("the lane is not set aside: %+v", tm.setAside)
	}
	s := tm.setAside[0]
	left, kept := SplitHolds(s.Holds)
	if s.LeftToLapse || !slices.Equal(left, []string{bundleLead, bundleThree}) || !slices.Equal(kept, []string{bundleTwo}) {
		t.Fatalf("the lane's holds: left to lapse %v, left %v, kept %v\n%s", s.LeftToLapse, left, kept, strings.Join(lines, "\n"))
	}
	if tm.lapsing[runID] {
		t.Fatal("a run only some of whose holds were left is counted as left to lapse")
	}
	run, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := claim.View(repo.Root(), c.t)
	if err != nil {
		t.Fatal(err)
	}
	if leftToLapse(repo.Root(), run, snap) {
		t.Fatal("premise: a renewal of the run reads it as left to lapse")
	}
	want := "the reservation of " + bundleLead + ", " + bundleThree + " is left to lapse and that of " + bundleTwo + " could not be, so a renewal of the run keeps each"
	if said := strings.Join(lines, "\n"); !strings.Contains(said, want) || strings.Contains(said, "its reservation is left to lapse") {
		t.Fatalf("the set-aside's line does not name the holds per issue:\n%s", said)
	}
}
