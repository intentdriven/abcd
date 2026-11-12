package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// A drive that takes up a lane whose run's file states another schema version, which it
// may not save, passes that lane over and goes on with its other work
// (iss-2611120957377169): nothing of the lane is saved, claimed or reserved and no agent
// runs for it, the drive says once what it left and why, and a drive left with only such
// lanes ends on an outcome of its own while the drain stays open.

// passedOverWorld is a terminal world whose drive cannot find its own process in the
// process table, so it claims no run, and whose lane at index k is left, as soon as the
// move that opens it returns, as a drive leaves a lane it lets go and with its file at
// the version before this abcd's: the take-up's save is then the judge's to refuse.
type passedOverWorld struct {
	*terminalWorld
	k     int
	runID string
	// state and control are the left lane's files as the fixture left them; control is
	// nil where the run has none.
	state, control []byte
	moves          int
}

func newPassedOverWorld(t *testing.T, k int) (*passedOverWorld, TerminalOptions) {
	t.Helper()
	w := &passedOverWorld{terminalWorld: newTerminalWorld(t), k: k}
	procsOf(t, hostOne, hostTwo)
	o := w.options(t)
	o.Moved = func() {
		w.moves++
		if w.runID != "" {
			return
		}
		st, _, err := readDrain(w.repo.Root())
		if err != nil {
			t.Fatal(err)
		}
		if len(st.Lanes) <= k || st.Lanes[k].Outcome != DrainLaneInProgress {
			return
		}
		w.leave(t, st.Lanes[k].RunID)
	}
	return w, o
}

// leave lets the lane of runID go as a drive's end lets it go (letGo), then writes its file
// at the version before this abcd's.
func (w *passedOverWorld) leave(t *testing.T, runID string) {
	t.Helper()
	root := w.repo.Root()
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	now := w.clock.now()
	clause, failed := releaseReservation(root, run.Key, "", now)
	if err := mutateReservation(root, runID, func(st *State) (bool, error) {
		noteRecordReleased(root, st, failed, now)
		st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleased, Note: clause})
		st.UpdatedAt = now
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	w.runID = runID
	w.state = stateAtVersion(t, root, runID, schemaVersionPrevious)
	w.control = controlBytes(t, root, runID)
}

// controlBytes is the run's control record as it stands, nil where there is none.
func controlBytes(t *testing.T, root, runID string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(controlRel(runID))))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// wantUntouched fails unless the left lane's files are as the fixture left them.
func (w *passedOverWorld) wantUntouched(t *testing.T) {
	t.Helper()
	if w.runID == "" {
		t.Fatalf("premise: the fixture left a lane:\n%s", strings.Join(w.lines, "\n"))
	}
	root := w.repo.Root()
	if after := readStateBytes(t, root, w.runID); !bytes.Equal(w.state, after) {
		t.Fatalf("the drive rewrote the run it passed over:\n%s", after)
	}
	if after := controlBytes(t, root, w.runID); !bytes.Equal(w.control, after) {
		t.Fatalf("the drive rewrote the control record of the run it passed over:\nbefore %s\nafter %s", w.control, after)
	}
}

// wantNamed fails unless the drive said once that it passed the left lane over, with the
// judge's reason, and its result names the lane with that reason.
func (w *passedOverWorld) wantNamed(t *testing.T, res TerminalResult) {
	t.Helper()
	all := strings.Join(w.lines, "\n")
	said := 0
	for _, l := range w.lines {
		if strings.Contains(l, "passed over") && strings.Contains(l, w.runID) {
			said++
			if !strings.Contains(l, leftLine(w.runID, schemaVersionPrevious)) {
				t.Fatalf("the line does not say what was left and why: %s", l)
			}
		}
	}
	if said != 1 {
		t.Fatalf("the drive says once what it passed over, not %d times:\n%s", said, all)
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		PassedLanes []struct{ Issue, Run, Reason string } `json:"passed_lanes"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.PassedLanes) != 1 || got.PassedLanes[0].Run != w.runID || !strings.Contains(got.PassedLanes[0].Reason, leftLine(w.runID, schemaVersionPrevious)) {
		t.Fatalf("the ending does not name the lane passed over: %s", raw)
	}
	if res.Run == w.runID {
		t.Fatalf("the ending names the lane passed over as the lane it left to resume: %+v", res)
	}
}

// drainLive reports whether the checkout's drain is still open.
func drainLive(t *testing.T, root string) bool {
	t.Helper()
	_, live, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	return live
}

// TestADrivePassesOverTheMiddleLaneItMayNotSaveAndWorksTheOthers: of three lanes, the
// middle one's run is left at another version the drive may not save. The first and the
// third are worked to their landing, the middle one's files are as they were, no agent
// runs for it after its judge, and the drive ends naming it, the drain left open.
func TestADrivePassesOverTheMiddleLaneItMayNotSaveAndWorksTheOthers(t *testing.T) {
	w, o := newPassedOverWorld(t, 1)

	res, err := RunTerminal(context.Background(), w.repo.Root(), o)

	if err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	w.wantUntouched(t)
	st, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes) != 3 || st.Lanes[1].RunID != w.runID {
		t.Fatalf("premise: three lanes, the middle one left: %+v", st.Lanes)
	}
	want := []string{st.Lanes[0].Issue, st.Lanes[2].Issue}
	slices.Sort(want)
	if got := lanesDone(t, w.repo); !slices.Equal(got, want) {
		t.Fatalf("the other two lanes are worked to their end: %v\n%s", got, strings.Join(w.lines, "\n"))
	}
	if res.Ended != "passed-lanes" {
		t.Fatalf("the drive ends on the lanes it passed over, not %q: %+v\n%s", res.Ended, res, strings.Join(w.lines, "\n"))
	}
	w.wantNamed(t, res)
	// A judge for each of three issues and an implementer for the two lanes worked.
	if got := w.launches(t); got != 5 {
		t.Fatalf("five agents ran, the middle lane's judge and none for its lane, got %d", got)
	}
	if !drainLive(t, w.repo.Root()) {
		t.Fatal("the drain stopped with a lane in progress a later drive is to take up")
	}
}

// TestADriveLeftWithOnlyLanesItPassedOverEndsInBoundedMoves: a drive whose only lane left
// is one it passed over ends on that, by its cap or as nothing else is eligible, in two
// moves: the one that opened the lane and the one that found nothing else to do, with no
// wait between them.
func TestADriveLeftWithOnlyLanesItPassedOverEndsInBoundedMoves(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*TerminalOptions)
	}{
		{"at its cap", func(o *TerminalOptions) { o.Drain.Max = 1 }},
		{"nothing else eligible", func(o *TerminalOptions) {
			o.Drain.Max = 0
			o.Drain.Skip, o.Drain.SkipReason = []string{secondIssue, thirdIssue}, "not this run"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, o := newPassedOverWorld(t, 0)
			tc.set(&o)

			res, err := RunTerminal(context.Background(), w.repo.Root(), o)

			if err != nil {
				t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
			}
			if res.Ended != "passed-lanes" {
				t.Fatalf("the drive ends on the lane it passed over, not %q\n%s", res.Ended, strings.Join(w.lines, "\n"))
			}
			if w.moves != 2 || len(w.sleeps) != 0 {
				t.Fatalf("the drive ends in two moves and no wait, not %d moves and waits %v\n%s", w.moves, w.sleeps, strings.Join(w.lines, "\n"))
			}
			w.wantUntouched(t)
			w.wantNamed(t, res)
			if got := w.launches(t); got != 1 {
				t.Fatalf("only the lane's judge ran, got %d launches", got)
			}
			if !drainLive(t, w.repo.Root()) {
				t.Fatal("the drain stopped with a lane in progress a later drive is to take up")
			}
		})
	}
}

// TestAMoveLeavesALanePassedOverAndALaterDrivesMoveTakesItUp: a move given the lane passed
// over picks no lane and touches none of its files; a move without it, a later drive's,
// picks the lane, to take up through its own claim and judge.
func TestAMoveLeavesALanePassedOverAndALaterDrivesMoveTakesItUp(t *testing.T) {
	w, o := newPassedOverWorld(t, 0)
	o.Drain.Max = 1
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	w.wantUntouched(t)
	at := w.clock.now().Add(time.Minute)
	opts := Options{Now: func() time.Time { return at }, Runners: o.Options.Runners}

	res, err := Drain(w.repo.Root(), opts, DrainOptions{PassedLanes: []string{w.runID}})

	if err != nil {
		t.Fatal(err)
	}
	if res.Lane != nil || len(res.passedOnly) != 1 || res.passedOnly[0].RunID != w.runID {
		t.Fatalf("the move picks the lane passed over: lane %+v, passed %+v", res.Lane, res.passedOnly)
	}
	if res.Stopped != "" || !strings.Contains(res.Next, w.runID) {
		t.Fatalf("the move stops the drain, or does not name the lane: %q %q", res.Stopped, res.Next)
	}
	w.wantUntouched(t)

	later, err := Drain(w.repo.Root(), opts, DrainOptions{})

	if err != nil {
		t.Fatal(err)
	}
	if later.Lane == nil || later.Lane.RunID != w.runID {
		t.Fatalf("a later drive's move does not pick the lane up: %+v", later.Lane)
	}
}
