//go:build unix

package loop

// terminal_limit_release_test.go is a drive that lets a lane's issue go for a long
// limit (iss-2611052230565049): released at the threshold, kept below it, reserved
// again on recovery with what it held, dropped when the issue is no longer the lane's,
// and every one of those with what the lane left still to be found.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// partialFile is what leavesWork has the first implementer leave uncommitted.
const partialFile = "partial.txt"

// leavesWork has the lane's first implementer leave an uncommitted file behind.
func (w *terminalWorld) leavesWork(t *testing.T) {
	t.Helper()
	for i := range w.stages {
		if w.stages[i].Name != StageImplement {
			continue
		}
		run, once := w.stages[i].Run, false
		w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
			if !once {
				once = true
				if err := os.WriteFile(filepath.Join(l.Worktree, partialFile), []byte("half done\n"), 0o600); err != nil {
					return Outcome{}, err
				}
			}
			return run(c, l)
		}
	}
}

// firstLaneRun is the run of the drain's first lane.
func (w *terminalWorld) firstLaneRun(t *testing.T) State {
	t.Helper()
	st, _, err := readDrain(w.repo.Root())
	if err != nil || len(st.Lanes) == 0 {
		t.Fatalf("no lane: %+v %v", st.Lanes, err)
	}
	run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

// asidesOf reads every aside saved for the run's first lane.
func asidesOf(t *testing.T, root string, run State) []RestartAside {
	t.Helper()
	rel, err := laneRel(run.RunID, run.Lanes[0].ID, stageRestart)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(rel), AsideDirName, "*", AsideFileName))
	if err != nil {
		t.Fatal(err)
	}
	var out []RestartAside
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var a RestartAside
		if err := json.Unmarshal(raw, &a); err != nil {
			t.Fatal(err)
		}
		out = append(out, a)
	}
	return out
}

// savedFile reports whether an aside of the run's first lane saved name.
func savedFile(t *testing.T, root string, run State, name string) bool {
	t.Helper()
	return slices.ContainsFunc(asidesOf(t, root, run), func(a RestartAside) bool { return slices.Contains(a.Files, name) })
}

// TestALimitAnHourOffLetsTheLaneGoAndReservesItAgain: a reset exactly the threshold
// away releases the issue before the wait begins, so a peer can see it through the
// wait, and renews nothing while it lasts. On recovery the same run reserves it again
// with the footprint it held and lands.
func TestALimitAnHourOffLetsTheLaneGoAndReservesItAgain(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	w.resetIn(t, terminalReleaseAfter)
	var during claim.Reservation
	var key string
	var footprint []string
	first := true
	w.onSleep = func(time.Duration) {
		if first {
			first = false
			run := w.firstLaneRun(t)
			key = run.Key
			during = reservationOf(t, w.repo.Root(), key, w.clock.t)
			footprint = during.Footprint
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if during.State != claim.StateReleased {
		t.Fatalf("the issue is let go before the wait: %+v\n%s", during, w.said())
	}
	if res.Ended != DrainStoppedCap || w.launches(t) != 3 || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the same run lands after the reset: %+v, %d launches\n%s", res, w.launches(t), w.said())
	}
	if n := runCount(t, w.repo); n != 1 {
		t.Fatalf("no second run opens: %d runs", n)
	}
	run := w.firstLaneRun(t)
	var stages []string
	for _, e := range run.Record {
		switch e.Stage {
		case StageDriveReleased, StageReservation, StageReservationRenewed:
			stages = append(stages, e.Stage)
		}
	}
	if !slices.Equal(stages, []string{StageReservation, StageDriveReleased, StageReservation}) {
		t.Fatalf("reserved, let go, reserved again, with no renewal through the wait: %v", stages)
	}
	after := reservationOf(t, w.repo.Root(), key, w.clock.t)
	if after.State != claim.StateLive || after.Copy != claim.CopyThis || !slices.Equal(after.Footprint, footprint) {
		t.Fatalf("reserved again with the same footprint %v: %+v", footprint, after)
	}
}

// TestALimitJustBelowTheThresholdKeepsItsHold: releasing and reserving again costs a
// commit on origin each way, so a short limit keeps what it holds.
func TestALimitJustBelowTheThresholdKeepsItsHold(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	w.resetIn(t, terminalReleaseAfter-time.Minute)
	var during claim.Reservation
	first := true
	w.onSleep = func(time.Duration) {
		if first {
			first = false
			during = reservationOf(t, w.repo.Root(), w.firstLaneRun(t).Key, w.clock.t)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands after the reset: %+v %v\n%s", res, err, w.said())
	}
	if during.State != claim.StateLive {
		t.Fatalf("a short limit keeps its hold: %+v", during)
	}
	if run := w.firstLaneRun(t); slices.ContainsFunc(run.Record, func(e Entry) bool { return e.Stage == StageDriveReleased }) {
		t.Fatalf("nothing was let go: %+v", run.Record)
	}
}

// TestALimitNamingNoResetIsLetGoOnceItHasLastedTheThreshold: with nothing saying how
// long it lasts, the limit keeps its hold through the schedule's first looks, and the
// first turn at or past the threshold lets the issue go.
func TestALimitNamingNoResetIsLetGoOnceItHasLastedTheThreshold(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	start := w.clock.t
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
		if w.clock.t.Sub(start) >= 3*time.Hour {
			return true, nil
		}
		return false, limitedNoReset()
	}
	sawReleased := false
	w.onSleep = func(time.Duration) {
		elapsed := w.clock.t.Sub(start)
		run := w.firstLaneRun(t)
		r := reservationOf(t, w.repo.Root(), run.Key, w.clock.t)
		switch {
		case elapsed < terminalReleaseAfter && r.State != claim.StateLive:
			t.Errorf("%v in, under the threshold, the hold is kept: %+v", elapsed, r)
		case elapsed >= terminalReleaseAfter+30*time.Minute && r.State != claim.StateReleased:
			t.Errorf("%v in, the issue is let go: %+v", elapsed, r)
		}
		sawReleased = sawReleased || r.State == claim.StateReleased
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap || w.launches(t) != 3 {
		t.Fatalf("the lane lands once a probe answers: %+v %v\n%s", res, err, w.said())
	}
	if !sawReleased {
		t.Fatalf("the issue was never let go:\n%s", w.said())
	}
}

// TestWithNoBoundANamedLongResetLetsGoAndNoResetKeeps: with no bound a named reset is
// judged the same, and a limit naming none is found out only by the lane's own launch,
// which needs the reservation, so it keeps its hold up to the run's eight hours.
func TestWithNoBoundANamedLongResetLetsGoAndNoResetKeeps(t *testing.T) {
	t.Run("named", func(t *testing.T) {
		w := newTerminalWorld(t, "shape", "limited")
		w.realWorktree()
		o := w.options(t)
		o.Drain.Max = 1
		w.resetIn(t, 2*time.Hour)
		var during claim.Reservation
		first := true
		w.onSleep = func(time.Duration) {
			if first {
				first = false
				during = reservationOf(t, w.repo.Root(), w.firstLaneRun(t).Key, w.clock.t)
			}
		}
		res, err := RunTerminal(context.Background(), w.repo.Root(), o)
		if err != nil || res.Ended != DrainStoppedCap {
			t.Fatalf("the lane lands after the reset: %+v %v\n%s", res, err, w.said())
		}
		if during.State != claim.StateReleased {
			t.Fatalf("a long named reset lets the issue go: %+v\n%s", during, w.said())
		}
	})
	t.Run("none named", func(t *testing.T) {
		w := newTerminalWorld(t, "shape", "limited", "limited", "limited", "limited", "limited", "limited")
		w.realWorktree()
		o := w.options(t)
		o.Drain.Max = 1
		w.guardSleeps(t, &o)
		w.onSleep = func(time.Duration) {
			run := w.firstLaneRun(t)
			if slices.ContainsFunc(run.Record, func(e Entry) bool { return e.Stage == StageDriveReleased }) {
				t.Errorf("no probe runs, so the hold stays for the launch that asks:\n%s", w.said())
			}
		}
		if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
			t.Fatal(err)
		}
	})
}

// TestAnIssueFixedElsewhereDropsTheLaneAndTheRunGoesOn: the record left open on the
// default branch during the wait, so the lane ends, nothing is reserved for it, no
// hand-back is published on a finished issue, what the lane left stays where it can be
// found, and the next issue is taken.
func TestAnIssueFixedElsewhereDropsTheLaneAndTheRunGoesOn(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.leavesWork(t)
	o := w.bounded(t, 30*24*time.Hour)
	o.Drain.Max = 2
	w.resetIn(t, 5*time.Hour)
	first := true
	w.onSleep = func(time.Duration) {
		if first {
			first = false
			key := w.firstLaneRun(t).Key
			resolveOnOrigin(t, w.bare, key)
			// A real resolve stamps the record as it moves it, so its blob changes too,
			// and finished is still what the hand-back says.
			changeRecordOnOrigin(t, w.bare, key, "resolved")
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	wantDropped(t, w, res, "finished on origin's default branch")
	run := w.firstLaneRun(t)
	if _, err := claim.Fetch(w.repo.Root(), claim.Options{Now: w.clock.now}); err != nil {
		t.Fatal(err)
	}
	if r := reservationOf(t, w.repo.Root(), run.Key, w.clock.t); r.State != claim.StateReleased || r.HandedBack != "" {
		t.Fatalf("nothing is reserved or published over a finished issue: %+v", r)
	}
}

// TestAPeerHoldingTheIssueDropsTheLaneAndTheRunGoesOn: a peer took the released issue,
// or took it and handed it back, during the wait. Either way the drive backs off, as any
// drain does, the lane goes back saying which, nothing is pushed over what the peer left,
// and the run goes on.
func TestAPeerHoldingTheIssueDropsTheLaneAndTheRunGoesOn(t *testing.T) {
	for _, tc := range []struct {
		name, reason, home string
		handsBack          bool
	}{
		{name: "holding it", reason: "back off", home: "the peer who holds it"},
		{name: "handing it back", reason: "handed back by", home: "a hand-back stands", handsBack: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newTerminalWorld(t, "shape", "limited")
			w.realWorktree()
			w.leavesWork(t)
			o := w.bounded(t, 30*24*time.Hour)
			o.Drain.Max = 2
			w.resetIn(t, 5*time.Hour)
			q := otherCopy(t, w.bare, quinnEmail)
			first := true
			peerTip := ""
			w.onSleep = func(time.Duration) {
				if !first {
					return
				}
				first = false
				took := w.firstLaneRun(t).Key
				if got := reservationOf(t, w.repo.Root(), took, w.clock.t); got.State != claim.StateReleased {
					t.Errorf("the peer can see the issue free: %+v", got)
				}
				qo := as(t, q, claim.Options{Now: w.clock.now})
				if _, err := claim.Reserve(q, claim.Request{Record: took, Footprint: []string{"elsewhere.md"}, Lease: 24 * time.Hour}, qo); err != nil {
					t.Error(err)
				}
				if tc.handsBack {
					if _, err := claim.Reserve(q, claim.Request{Record: took, Release: true, HandBack: claim.HandBackSecondPackage}, qo); err != nil {
						t.Error(err)
					}
				}
				peerTip = gitIn(t, w.bare, "rev-parse", claim.RefPrefix+took)
			}
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatal(err)
			}
			wantDropped(t, w, res, tc.reason)
			run := w.firstLaneRun(t)
			if !strings.Contains(run.Lanes[0].HandBack.Home, tc.home) {
				t.Fatalf("the hand-back says who the issue is with (%q): %q", tc.home, run.Lanes[0].HandBack.Home)
			}
			if tip := gitIn(t, w.bare, "rev-parse", claim.RefPrefix+run.Key); tip != peerTip {
				t.Fatalf("nothing is pushed over what the peer left: %s, was %s", tip, peerTip)
			}
			if _, err := claim.Fetch(w.repo.Root(), claim.Options{Now: w.clock.now}); err != nil {
				t.Fatal(err)
			}
			r := reservationOf(t, w.repo.Root(), run.Key, w.clock.t)
			if r.Copy == claim.CopyThis {
				t.Fatalf("what the peer left stays the peer's: %+v", r)
			}
			if tc.handsBack && r.HandedBack != claim.HandBackSecondPackage {
				t.Fatalf("the hand-back stands: %+v", r)
			}
			if !tc.handsBack && (r.State != claim.StateLive || r.HandedBack != "") {
				t.Fatalf("the peer keeps what it took: %+v", r)
			}
		})
	}
}

// TestOnlyADrainBacksOffAHandBackOnResuming: the drive's take-up carries the drain's mark
// and a person's resume of the same run does not: a person may go over a hand-back, and
// is told so, as a person's reservation always may.
func TestOnlyADrainBacksOffAHandBackOnResuming(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(root, claim.Request{Record: run.Key, Release: true}, claim.Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	q := otherCopy(t, bare, quinnEmail)
	qo := as(t, q, claim.Options{Now: c.now})
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Footprint: []string{"elsewhere.md"}}, qo); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Release: true, HandBack: claim.HandBackSecondPackage}, qo); err != nil {
		t.Fatal(err)
	}
	tip := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key)
	r, ok := AsRefusal(reserveResumed(root, &StartResult{RunID: runID}, Options{Now: c.now, drain: true}))
	if !ok || !r.Contention || r.Stage != StageReservation || !strings.Contains(r.Reason, "handed back by") {
		t.Fatalf("a drain backs off the hand-back: %+v", r)
	}
	if got := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key); got != tip {
		t.Fatalf("nothing is pushed by a drain that backs off: %s, was %s", got, tip)
	}
	if err := reserveResumed(root, &StartResult{RunID: runID}, Options{Now: c.now}); err != nil {
		t.Fatalf("a person's resume goes over the hand-back: %v", err)
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	if last := lastReservationEntry(run); last == nil || !strings.Contains(last.Note, "handed back") {
		t.Fatalf("and is told it did: %+v", last)
	}
}

// changeRecordOnOrigin adds a line to id's record in folder on origin's main.
func changeRecordOnOrigin(t *testing.T, bare, id, folder string) {
	t.Helper()
	pushFromOtherCopy(t, bare, "change "+id, func(dir string) {
		matches, _ := filepath.Glob(filepath.Join(dir, ".abcd", "work", "issues", folder, id+"-*.md"))
		if len(matches) != 1 {
			t.Fatalf("origin holds no %s record of %s", folder, id)
		}
		f, err := os.OpenFile(matches[0], os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.WriteString("\nWhat this asks has changed.\n"); err != nil {
			t.Fatal(err)
		}
	})
}

// TestARecordChangedWhileLetGoDropsTheLane: a person changed what the issue asks during
// the wait, so the fix under way is to what it asked before and is no longer wanted.
// The lane goes back saying so, with nothing reserved.
func TestARecordChangedWhileLetGoDropsTheLane(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.leavesWork(t)
	o := w.bounded(t, 30*24*time.Hour)
	o.Drain.Max = 2
	w.resetIn(t, 5*time.Hour)
	first := true
	tip := ""
	w.onSleep = func(time.Duration) {
		if !first {
			return
		}
		first = false
		key := w.firstLaneRun(t).Key
		tip = gitIn(t, w.bare, "rev-parse", claim.RefPrefix+key)
		changeRecordOnOrigin(t, w.bare, key, "open")
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	wantDropped(t, w, res, "changed while the issue was let go")
	run := w.firstLaneRun(t)
	if strings.Contains(run.Lanes[0].HandBack.Reason, "no longer wanted") {
		t.Fatalf("a change to the record says only that it changed: %q", run.Lanes[0].HandBack.Reason)
	}
	if !strings.Contains(run.Lanes[0].HandBack.Home, "whoever changed its record") {
		t.Fatalf("the hand-back says who the issue is with: %q", run.Lanes[0].HandBack.Home)
	}
	if got := gitIn(t, w.bare, "rev-parse", claim.RefPrefix+run.Key); got != tip {
		t.Fatalf("nothing is reserved over a record that moved on: %s, was %s", got, tip)
	}
}

// TestARecordChangedBetweenDrivesDropsTheLane: a drive's end lets its lanes go as a limit
// does, so the next drive compares the record in the same way before it goes on.
func TestARecordChangedBetweenDrivesDropsTheLane(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 20*time.Minute)
	w.resetIn(t, 30*time.Minute)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalUntil {
		t.Fatalf("the bound ends the drive: %+v %v\n%s", res, err, w.said())
	}
	run := w.firstLaneRun(t)
	if !letGoByDrive(run) {
		t.Fatalf("the drive's end let the issue go:\n%s", w.said())
	}
	changeRecordOnOrigin(t, w.bare, run.Key, "open")
	again := w.bounded(t, 24*time.Hour)
	if _, err := RunTerminal(context.Background(), w.repo.Root(), again); err != nil {
		t.Fatal(err)
	}
	run = w.firstLaneRun(t)
	if hb := run.Lanes[0].HandBack; hb == nil || !strings.Contains(hb.Reason, "changed while the issue was let go") {
		t.Fatalf("the lane goes back saying the record changed: %+v\n%s", hb, w.said())
	}
}

// TestARecordChangedBeforeATakeOverDropsTheLane: a signal halted the run during the wait,
// so the next drive takes it over rather than up, and compares the record all the same.
func TestARecordChangedBeforeATakeOverDropsTheLane(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	w.resetIn(t, 5*time.Hour)
	first := true
	w.onSleep = func(time.Duration) {
		if first {
			first = false
			// Changed before the signal: the stop's own release must not note it as
			// what the lane was let go over.
			changeRecordOnOrigin(t, w.bare, w.firstLaneRun(t).Key, "open")
			close(w.stop)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalStopped {
		t.Fatalf("the signal ends the drive: %+v %v\n%s", res, err, w.said())
	}
	run := w.firstLaneRun(t)
	w.onSleep = nil
	again := w.bounded(t, 24*time.Hour)
	again.Stop = make(chan struct{})
	if _, err := RunTerminal(context.Background(), w.repo.Root(), again); err != nil {
		t.Fatal(err)
	}
	run = w.firstLaneRun(t)
	if hb := run.Lanes[0].HandBack; hb == nil || !strings.Contains(hb.Reason, "changed while the issue was let go") {
		t.Fatalf("the lane goes back saying the record changed: %+v\n%s", hb, w.said())
	}
}

// TestARecordChangedAfterASignalDropsTheLane: a signal reaches a drive that is not waiting
// on a limit, so the stop's own release is the one that notes the record, and the next
// drive's take-over compares against it.
func TestARecordChangedAfterASignalDropsTheLane(t *testing.T) {
	w := newTerminalWorld(t, append([]string{"shape"}, repeated("none", 3)...)...)
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	first := true
	w.onSleep = func(time.Duration) {
		if first {
			first = false
			close(w.stop)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalStopped {
		t.Fatalf("the signal ends the drive: %+v %v\n%s", res, err, w.said())
	}
	run := w.firstLaneRun(t)
	if c := controlOf(t, w.repo.Root(), run.RunID); c.Stop == nil || c.Stop.HaltedAt == nil || c.Stop.FromFile {
		t.Fatalf("the signal halted the run: %+v", c.Stop)
	}
	if letGoByDrive(run) {
		t.Fatalf("no limit let this run go: %+v", run.Record)
	}
	changeRecordOnOrigin(t, w.bare, run.Key, "open")
	tip := gitIn(t, w.bare, "rev-parse", claim.RefPrefix+run.Key)
	w.onSleep = nil
	again := w.bounded(t, 24*time.Hour)
	again.Stop = make(chan struct{})
	if _, err := RunTerminal(context.Background(), w.repo.Root(), again); err != nil {
		t.Fatal(err)
	}
	run = w.firstLaneRun(t)
	if hb := run.Lanes[0].HandBack; hb == nil || !strings.Contains(hb.Reason, "changed while the issue was let go") {
		t.Fatalf("the lane goes back saying the record changed: %+v\n%s", hb, w.said())
	}
	if got := gitIn(t, w.bare, "rev-parse", claim.RefPrefix+run.Key); got != tip {
		t.Fatalf("nothing is reserved over the changed record: %s, was %s", got, tip)
	}
}

// TestADrainResumingALaneLetGoComparesTheRecord: a drain that has ended leaves its lane let
// go. A new drain passes the issue over while the run stands, and any drain that does
// resume it, through the start a concurrent build can race it to, compares the record: the
// lane goes back, nothing is reserved, and the issue is free for a fresh run.
func TestADrainResumingALaneLetGoComparesTheRecord(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if !letGoByDrive(run) {
		t.Fatalf("the lane is let go: %+v", run.Record)
	}
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	ended := c.t
	st.Stopped, st.EndedAt = DrainStoppedOrigin, &ended
	if err := writeDrain(root, st); err != nil {
		t.Fatal(err)
	}
	changeRecordOnOrigin(t, bare, run.Key, "open")
	tip := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key)
	move, err := Drain(root, Options{Now: c.now}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(move.Passed, func(p Excluded) bool { return p.ID == run.Key && p.Check == CheckRun }) {
		t.Fatalf("a new drain passes over an issue whose run stands: %+v", move.Passed)
	}
	_, err = Start(root, run.Key, Options{Now: c.now, drain: true})
	r, ok := AsRefusal(err)
	if !ok || !strings.Contains(r.Reason, "changed while the issue was let go") {
		t.Fatalf("the drain's start is refused, saying the record changed: %v", err)
	}
	if got := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key); got != tip {
		t.Fatalf("nothing is reserved: %s, was %s", got, tip)
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	if hb := run.Lanes[0].HandBack; hb == nil || !strings.Contains(hb.Home, "whoever changed its record") {
		t.Fatalf("the start drops the lane: %+v", hb)
	}
	runs, err := Runs(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, live := liveRun(runs, run.Key, true, false); live {
		t.Fatal("the dropped run is no drain's to resume")
	}
}

// TestAPersonsTakeOverSaysTheRecordChanged: a person goes on over a changed record, as over
// a hand-back, and is told, since the brief reads the record at the lane's base and the
// agent restarted could otherwise work to the old text.
func TestAPersonsTakeOverSaysTheRecordChanged(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	w.resetIn(t, 30*time.Minute)
	home := stopOnFirstWait(t, w, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalHalted {
		t.Fatalf("the person's stop halts the run: %+v %v\n%s", res, err, w.said())
	}
	if _, err := runstop.Lift(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	changeRecordOnOrigin(t, w.bare, res.Issue, "open")
	took, err := TakeRun(w.repo.Root(), res.Run, Options{stopHome: home, Now: w.clock.now})
	if err != nil {
		t.Fatalf("a person's take-over goes ahead: %v", err)
	}
	for _, want := range []string{"changed while the issue was let go", "the lane's base", "a person's own, so it was made"} {
		if !strings.Contains(took.Reservation, want) {
			t.Fatalf("the take-over says %q: %q", want, took.Reservation)
		}
	}
	run := w.firstLaneRun(t)
	if run.Lanes[0].HandBack != nil {
		t.Fatalf("nothing is dropped: %+v", run.Lanes[0].HandBack)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateLive || r.Copy != claim.CopyThis {
		t.Fatalf("the take-over reserves again: %+v", r)
	}
}

// TestAStateLockAtATakeUpWaits: another invocation holding the run lock passes. The lane
// waits a turn with nothing written on it, and nothing is pushed to origin for a
// reservation the run could not have recorded.
func TestAStateLockAtATakeUpWaits(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	waits := 0
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}, Sleep: func(context.Context, time.Duration) { waits++ }})
	tm.letGoForLimit(runID)
	before, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	ref := claim.RefPrefix + before.Key
	commits := gitIn(t, bare, "rev-list", "--count", ref)
	was := lockTimeout
	lockTimeout = 100 * time.Millisecond
	t.Cleanup(func() { lockTimeout = was })
	var done bool
	var ended string
	lock := filepath.Join(root, filepath.FromSlash(RunRelDir), lockFileName)
	if err := fsutil.WithFileLock(lock, time.Second, func() error {
		done, ended = tm.takeUp(context.Background(), runID)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !done || ended != "" || waits == 0 {
		t.Fatalf("the lane waits a turn and the drive goes on: %v %q, %d wait(s)", done, ended, waits)
	}
	if got := gitIn(t, bare, "rev-list", "--count", ref); got != commits {
		t.Fatalf("nothing is pushed while the lock is held: %s commits, was %s", got, commits)
	}
	after, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	was2, _ := json.Marshal(before)
	now, _ := json.Marshal(after)
	if string(was2) != string(now) {
		t.Fatalf("nothing is written for what passes:\n%s\n%s", was2, now)
	}
}

// TestATakeOverRestartsARoleALimitMarked: a role the limit's release marked has no agent, so
// the take-over restarts it and never says it may still be in its call.
func TestATakeOverRestartsARoleALimitMarked(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	laneAwaiting(t, root, runID, nil, c.t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	markHalted(t, root, runID, true)
	home, _ := stopWorld(t, false)
	took, err := TakeRun(root, runID, Options{Now: c.now, stopHome: home})
	if err != nil {
		t.Fatalf("the take-over goes ahead: %v", err)
	}
	if text := strings.Join(took.Left, "\n"); strings.Contains(text, "may still be in its call") {
		t.Fatalf("a role with no agent is not in a call: %s", text)
	}
	if len(took.Restarted) != 1 {
		t.Fatalf("the marked role is restarted: %+v", took)
	}
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range run.Lanes[0].Awaits {
		if a.Kept != "" {
			t.Fatalf("no mark stands after the restart: %+v", a)
		}
	}
}

// TestAChangeTheLaneNeverFetchedIsNoChangeWhileLetGo: a record that changed before the
// release, and that this checkout first fetches in the release, did not change while the
// issue was let go, so the lane goes on.
func TestAChangeTheLaneNeverFetchedIsNoChangeWhileLetGo(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	changeRecordOnOrigin(t, bare, run.Key, "open")
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	if done, ended := tm.takeUp(context.Background(), runID); done || ended != "" {
		t.Fatalf("the lane is taken up: %v %q", done, ended)
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	if run.Lanes[0].HandBack != nil {
		t.Fatalf("nothing changed while it was let go: %+v", run.Lanes[0].HandBack)
	}
}

// TestAFailedReleaseIsTriedAgainWhenTheRecordMovedOn: the release never reached origin and
// nothing renewed the tip, so a lane dropped for a changed record tries the release once
// more rather than leaving its own tip holding off everyone until the lease runs out.
func TestAFailedReleaseIsTriedAgainWhenTheRecordMovedOn(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	failures := 0
	claimReserve = func(repo string, req claim.Request, o claim.Options) (claim.Result, error) {
		if req.Release && failures == 0 {
			failures++
			return claim.Result{}, errors.New("no route to origin")
		}
		return was(repo, req, o)
	}
	var said []string
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}, Say: func(line string) { said = append(said, line) }})
	tm.letGoForLimit(runID)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if !letGoByDrive(run) || reservationOf(t, root, run.Key, c.t).State != claim.StateLive {
		t.Fatalf("the release failed and the tip stands: %+v", run.Record)
	}
	changeRecordOnOrigin(t, bare, run.Key, "open")
	if _, err := claim.Fetch(root, claim.Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	if done, ended := tm.takeUp(context.Background(), runID); !done || ended != "" {
		t.Fatalf("the lane is dropped: %v %q", done, ended)
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	if run.Lanes[0].HandBack == nil {
		t.Fatalf("the lane goes back: %+v", run.Lanes[0])
	}
	if r := reservationOf(t, root, run.Key, c.t); r.State != claim.StateReleased {
		t.Fatalf("the tip is released on the second try: %+v", r)
	}
	if reason := run.Lanes[0].HandBack.Reason; !strings.Contains(reason, "released the reservation") {
		t.Fatalf("the hand-back says what the second try released: %q", reason)
	}
	if text := strings.Join(said, "\n"); !strings.Contains(text, "released the reservation") {
		t.Fatalf("and so does the screen: %s", text)
	}
	wantNoFailedRelease(t, root, runID, c)
}

// wantNoFailedRelease fails t unless the listing reads the run's latest release as one
// that reached origin.
func wantNoFailedRelease(t *testing.T, root, runID string, c *clock) {
	t.Helper()
	l, err := ListRuns(root, runID, Options{Now: c.now})
	if err != nil || len(l.Runs) != 1 {
		t.Fatalf("the run is listed: %+v %v", l, err)
	}
	if l.Runs[0].ReleaseFailed {
		t.Fatalf("the listing reads the release that reached origin: %+v", l.Runs[0])
	}
}

// TestAHandReservationMadeWhileLetGoStaysOnTheDrop: a person changed the record and reserved
// the issue by hand from this copy during the wait. The lane goes back and that hold stays,
// whether the run's own release reached origin or not, and the drop says it stands.
func TestAHandReservationMadeWhileLetGoStaysOnTheDrop(t *testing.T) {
	for _, tc := range []struct {
		name          string
		releaseFailed bool
	}{{"over the run's tombstone", false}, {"over the tip its failed release left", true}} {
		t.Run(tc.name, func(t *testing.T) {
			root, bare, runID, c := laneForPeerCheck(t)
			was := claimReserve
			t.Cleanup(func() { claimReserve = was })
			failures := 0
			claimReserve = func(repo string, req claim.Request, o claim.Options) (claim.Result, error) {
				if tc.releaseFailed && req.Release && failures == 0 {
					failures++
					return claim.Result{}, errors.New("no route to origin")
				}
				return was(repo, req, o)
			}
			var said []string
			tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}, Say: func(line string) { said = append(said, line) }})
			tm.letGoForLimit(runID)
			run, err := ReadState(root, runID)
			if err != nil {
				t.Fatal(err)
			}
			changeRecordOnOrigin(t, bare, run.Key, "open")
			if _, err := claim.Reserve(root, claim.Request{Record: run.Key, Footprint: []string{"commands/drain.md"}}, claim.Options{Now: c.now}); err != nil {
				t.Fatal(err)
			}
			tip := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key)
			if done, ended := tm.takeUp(context.Background(), runID); !done || ended != "" {
				t.Fatalf("the lane is dropped: %v %q", done, ended)
			}
			if got := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key); got != tip {
				t.Fatalf("the hand reservation is left as it stands: %s, was %s", got, tip)
			}
			if run, err = ReadState(root, runID); err != nil {
				t.Fatal(err)
			}
			hb := run.Lanes[0].HandBack
			if hb == nil || !strings.Contains(hb.Reason, "stands") {
				t.Fatalf("the hand-back says the reservation stands: %+v", hb)
			}
			if text := strings.Join(said, "\n"); !strings.Contains(text, "stands") {
				t.Fatalf("and so does the screen: %s", text)
			}
		})
	}
}

// TestAFailedReleaseOnWakingIsCompared: a move renews nothing a drive let go, so the lane
// reaches the take-up, whose comparison drops it and tries the release again.
func TestAFailedReleaseOnWakingIsCompared(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.leavesWork(t)
	o := w.bounded(t, 30*24*time.Hour)
	o.Drain.Max = 2
	w.resetIn(t, 5*time.Hour)
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	failures := 0
	claimReserve = func(repo string, req claim.Request, o claim.Options) (claim.Result, error) {
		if req.Release && failures == 0 {
			failures++
			return claim.Result{}, errors.New("no route to origin")
		}
		return was(repo, req, o)
	}
	first := true
	w.onSleep = func(time.Duration) {
		if first {
			first = false
			changeRecordOnOrigin(t, w.bare, w.firstLaneRun(t).Key, "open")
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if failures != 1 {
		t.Fatalf("the release failed once: %d\n%s", failures, w.said())
	}
	wantDropped(t, w, res, "changed while the issue was let go")
}

// TestARecordDeletedWhileLetGoDropsTheLane: the note proves the record stood at the release,
// so a branch that can be read and holds no file for it is a change. The lane, which has
// nothing to resolve yet, goes back saying the record is gone, and the run goes on.
func TestARecordDeletedWhileLetGoDropsTheLane(t *testing.T) {
	for _, tc := range []struct {
		name string
		pull bool
	}{{"on origin", false}, {"on origin and pulled into the checkout", true}} {
		t.Run(tc.name, func(t *testing.T) {
			w := newTerminalWorld(t, "shape", "limited")
			w.realWorktree()
			w.leavesWork(t)
			o := w.bounded(t, 30*24*time.Hour)
			o.Drain.Max = 2
			w.resetIn(t, 5*time.Hour)
			first := true
			w.onSleep = func(time.Duration) {
				if first {
					first = false
					deleteRecordOnOrigin(t, w.bare, w.firstLaneRun(t).Key)
					if tc.pull {
						gitIn(t, w.repo.Root(), "pull", "-q", "--ff-only", "origin", "main")
					}
				}
			}
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatal(err)
			}
			wantDropped(t, w, res, "is gone from origin's default branch")
			if got := w.firstLaneRun(t).Lanes[0].Resolves; len(got) != 0 {
				t.Fatalf("the lane had nothing to resolve: %+v", got)
			}
		})
	}
}

// deleteRecordOnOrigin removes id's open record from origin's main.
func deleteRecordOnOrigin(t *testing.T, bare, id string) {
	t.Helper()
	pushFromOtherCopy(t, bare, "delete "+id, func(dir string) {
		matches, _ := filepath.Glob(filepath.Join(dir, ".abcd", "work", "issues", "open", id+"-*.md"))
		if len(matches) != 1 {
			t.Fatalf("origin holds no open record of %s", id)
		}
		gitIn(t, dir, "rm", "-q", matches[0])
	})
}

// TestAPersonIsToldTheRecordIsGone: a person's act goes ahead and says so.
func TestAPersonIsToldTheRecordIsGone(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	deleteRecordOnOrigin(t, bare, run.Key)
	start := StartResult{RunID: runID}
	if err := reserveResumed(root, &start, Options{Now: c.now}); err != nil {
		t.Fatalf("a person goes ahead: %v", err)
	}
	if !strings.Contains(start.Reservation, "is gone from origin's default branch") {
		t.Fatalf("and is told the record is gone: %q", start.Reservation)
	}
}

// TestARecordThatCannotBeComparedWaits: when the comparison cannot be made, whether the fetch
// failed or the branch cannot be read, a drain waits with nothing pushed and nothing dropped,
// and a person goes ahead and is told it was not compared.
func TestARecordThatCannotBeComparedWaits(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(t *testing.T, root string)
	}{
		{"the fetch fails", func(t *testing.T, root string) {
			was := recordFetch
			t.Cleanup(func() { recordFetch = was })
			recordFetch = func(string, claim.Options) (string, error) { return "", errors.New("no route to origin") }
		}},
		{"the branch cannot be read", func(t *testing.T, root string) {
			was := recordFetch
			t.Cleanup(func() { recordFetch = was })
			recordFetch = func(string, claim.Options) (string, error) { return "", nil }
			gitIn(t, root, "update-ref", "-d", "refs/remotes/origin/main")
			gitIn(t, root, "symbolic-ref", "-d", "refs/remotes/origin/HEAD")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, bare, runID, c := laneForPeerCheck(t)
			tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}, Sleep: func(context.Context, time.Duration) {}})
			tm.letGoForLimit(runID)
			run, err := ReadState(root, runID)
			if err != nil {
				t.Fatal(err)
			}
			gitIn(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
			tc.arrange(t, root)
			tip := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key)
			if done, ended := tm.takeUp(context.Background(), runID); !done || ended != "" {
				t.Fatalf("the lane waits and the drive goes on: %v %q", done, ended)
			}
			if got := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key); got != tip {
				t.Fatalf("nothing is pushed: %s, was %s", got, tip)
			}
			if run, err = ReadState(root, runID); err != nil {
				t.Fatal(err)
			}
			if run.Lanes[0].HandBack != nil || !letGoByDrive(run) {
				t.Fatalf("the lane stays let go: %+v", run.Lanes[0].HandBack)
			}
			start := StartResult{RunID: runID}
			if err := reserveResumed(root, &start, Options{Now: c.now}); err != nil {
				t.Fatalf("a person goes ahead: %v", err)
			}
			if !strings.Contains(start.Reservation, "could not be compared") {
				t.Fatalf("and is told it was not compared: %q", start.Reservation)
			}
		})
	}
}

// TestATakeOverOfARunLetGoReservesAgain: a person takes over a run a drive let go with no stop
// on it. The take-over reserves and compares, as it does after a stop, since nothing later
// would and the lane would go on holding nothing.
func TestATakeOverOfARunLetGoReservesAgain(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	changeRecordOnOrigin(t, bare, run.Key, "open")
	home, _ := stopWorld(t, false)
	took, err := TakeRun(root, runID, Options{Now: c.now, stopHome: home})
	if err != nil {
		t.Fatalf("the take-over goes ahead: %v", err)
	}
	if !strings.Contains(took.Reservation, "changed while the issue was let go") {
		t.Fatalf("it says the record changed: %q", took.Reservation)
	}
	if r := reservationOf(t, root, run.Key, c.t); r.State != claim.StateLive || r.Copy != claim.CopyThis {
		t.Fatalf("it reserves again: %+v", r)
	}
}

// TestAListingSaysAFailedReleasesTipStands: the run let go and the tip still holds, so the
// listing says neither that the run holds it nor that nothing does.
func TestAListingSaysAFailedReleasesTipStands(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	claimReserve = func(string, claim.Request, claim.Options) (claim.Result, error) {
		return claim.Result{}, errors.New("no route to origin")
	}
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	l, err := ListRuns(root, runID, Options{Now: c.now})
	if err != nil || len(l.Runs) != 1 {
		t.Fatalf("the run is listed: %+v %v", l, err)
	}
	if l.Runs[0].Reserved || !l.Runs[0].ReleaseFailed {
		t.Fatalf("the listing says the failed release's tip stands: %+v", l.Runs[0])
	}
}

// TestAStopsReleaseSettlesAFailedRelease: the limit's release did not reach origin, and a
// stop's later release did, so the listing reads the tombstone that stands.
func TestAStopsReleaseSettlesAFailedRelease(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	claimReserve = func(string, claim.Request, claim.Options) (claim.Result, error) {
		return claim.Result{}, errors.New("no route to origin")
	}
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	claimReserve = was
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if failedReleaseTip(run) == "" {
		t.Fatalf("the limit's release failed: %+v", run.Record)
	}
	if clause, failed := releaseReservation(root, run.Key, "", c.now()); failed {
		t.Fatalf("the stop's release reaches origin: %s", clause)
	}
	markHalted(t, root, runID, false)
	wantNoFailedRelease(t, root, runID, c)
}

// TestAKillNamesOnlyThisCopysLapse: a peer holds the issue the lane let go, so the end of a
// kill names no lapse, which would be the peer's expiry.
func TestAKillNamesOnlyThisCopysLapse(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Footprint: []string{"elsewhere.md"}}, as(t, q, claim.Options{Now: c.now})); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Fetch(root, claim.Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	if r := reservationOf(t, root, run.Key, c.t); r.Copy == claim.CopyThis || r.State != claim.StateLive {
		t.Fatalf("the peer holds it: %+v", r)
	}
	if res := tm.ended(TerminalKilled); res.Run != runID || res.Lapses != nil {
		t.Fatalf("the kill names the run and no lapse of its own: %+v", res)
	}
}

// TestADrainsWideningBacksOffAReservationOverAHandBack: a person reserved over a peer's
// hand-back, and a drain's resume that widens that reservation is a drain's act, so it
// backs off where a person's own widening goes ahead.
func TestADrainsWideningBacksOffAReservationOverAHandBack(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(root, claim.Request{Record: run.Key, Release: true}, claim.Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	q := otherCopy(t, bare, quinnEmail)
	qo := as(t, q, claim.Options{Now: c.now})
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Footprint: []string{"elsewhere.md"}}, qo); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Release: true, HandBack: claim.HandBackSecondPackage}, qo); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(root, claim.Request{Record: run.Key, Footprint: []string{"commands/drain.md"}}, claim.Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	wider := []string{"commands/drain.md", "commands/wider.md"}
	_, err = reserveAgain(root, run.Key, wider, false, true, false, "", c.t)
	if r, ok := AsRefusal(err); !ok || !r.Contention || !strings.Contains(r.Reason, "made over a hand-back") {
		t.Fatalf("the drain's widening backs off: %v", err)
	}
	if _, err := reserveAgain(root, run.Key, wider, false, false, false, "", c.t); err != nil {
		t.Fatalf("a person's own widening goes ahead: %v", err)
	}
}

// TestADriveTakingOverBacksOffAHandBack: a run this drive's own signal halted is taken over
// by the next drive, and that take-over reserves as a drain's act too.
func TestADriveTakingOverBacksOffAHandBack(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	markHalted(t, root, runID, false)
	if _, err := claim.Reserve(root, claim.Request{Record: run.Key, Release: true}, claim.Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	q := otherCopy(t, bare, quinnEmail)
	qo := as(t, q, claim.Options{Now: c.now})
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Footprint: []string{"elsewhere.md"}}, qo); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Release: true, HandBack: claim.HandBackSecondPackage}, qo); err != nil {
		t.Fatal(err)
	}
	tip := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	done, ended := tm.takeUp(context.Background(), runID)
	if !done || ended != "" {
		t.Fatalf("the lane is dropped and the drive goes on: %v %q", done, ended)
	}
	if got := gitIn(t, bare, "rev-parse", claim.RefPrefix+run.Key); got != tip {
		t.Fatalf("nothing is pushed over the hand-back: %s, was %s", got, tip)
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	if hb := run.Lanes[0].HandBack; hb == nil || !strings.Contains(hb.Home, "a hand-back stands") {
		t.Fatalf("the lane goes back saying so: %+v", hb)
	}
}

// TestADroppedLaneSaysWhoTheIssueIsWith: each cause of backing off names what is true of
// it, and only a peer who holds the issue is called one.
func TestADroppedLaneSaysWhoTheIssueIsWith(t *testing.T) {
	for _, tc := range []struct {
		name, whose string
		arrange     func(t *testing.T, root, bare, key string, c *clock)
	}{
		{"a peer", "the peer who holds it", func(t *testing.T, root, bare, key string, c *clock) {
			q := otherCopy(t, bare, quinnEmail)
			if _, err := claim.Reserve(q, claim.Request{Record: key, Footprint: []string{"elsewhere.md"}}, as(t, q, claim.Options{Now: c.now})); err != nil {
				t.Fatal(err)
			}
		}},
		{"this person's other copy", "your other copy", func(t *testing.T, root, bare, key string, c *clock) {
			p := otherCopy(t, bare, gitIn(t, root, "config", "user.email"))
			if _, err := claim.Reserve(p, claim.Request{Record: key, Footprint: []string{"elsewhere.md"}}, as(t, p, claim.Options{Now: c.now})); err != nil {
				t.Fatal(err)
			}
		}},
		{"a hand-back", "a hand-back stands", func(t *testing.T, root, bare, key string, c *clock) {
			q := otherCopy(t, bare, quinnEmail)
			qo := as(t, q, claim.Options{Now: c.now})
			if _, err := claim.Reserve(q, claim.Request{Record: key, Footprint: []string{"elsewhere.md"}}, qo); err != nil {
				t.Fatal(err)
			}
			if _, err := claim.Reserve(q, claim.Request{Record: key, Release: true, HandBack: claim.HandBackSecondPackage}, qo); err != nil {
				t.Fatal(err)
			}
		}},
		{"an unreadable tip", "cannot be read", func(t *testing.T, root, bare, key string, c *clock) {
			dir := filepath.Join(t.TempDir(), "ahead")
			gitIn(t, "", "clone", "-q", bare, dir)
			tree := gitIn(t, dir, "hash-object", "-w", "-t", "tree", "/dev/null")
			sha := gitIn(t, dir, "-c", "user.name=Quinn", "-c", "user.email=quinn@example.invalid", "-c", "commit.gpgsign=false",
				"commit-tree", tree, "-m", "abcd reservation: "+key, "-m", "not a reservation")
			gitIn(t, dir, "push", "-q", "-f", "origin", sha+":"+claim.RefPrefix+key)
		}},
		{"a finished issue", "finished on the default branch", func(t *testing.T, root, bare, key string, c *clock) {
			resolveOnOrigin(t, bare, key)
		}},
		{"another record's footprint", "the reservation the refusal names", func(t *testing.T, root, bare, key string, c *clock) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, bare, runID, c := laneForPeerCheck(t)
			run, err := ReadState(root, runID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := claim.Reserve(root, claim.Request{Record: run.Key, Release: true}, claim.Options{Now: c.now}); err != nil {
				t.Fatal(err)
			}
			tc.arrange(t, root, bare, run.Key, c)
			if _, err := claim.Fetch(root, claim.Options{Now: c.now}); err != nil {
				t.Fatal(err)
			}
			tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
			tm.dropLane(run, &Refusal{Reason: "the refusal's own words"})
			if run, err = ReadState(root, runID); err != nil {
				t.Fatal(err)
			}
			hb := run.Lanes[0].HandBack
			if hb == nil || !strings.Contains(hb.Home, tc.whose) {
				t.Fatalf("the hand-back says %q: %+v", tc.whose, hb)
			}
			if tc.whose != "the peer who holds it" && strings.Contains(hb.Home, "peer") {
				t.Fatalf("no peer is named where none holds it: %q", hb.Home)
			}
		})
	}
}

// wantDropped fails unless the first lane went back holding nothing, naming where its
// work is, with its worktree, branch and saved work standing, and the next lane landed.
func wantDropped(t *testing.T, w *terminalWorld, res TerminalResult, says string) {
	t.Helper()
	root := w.repo.Root()
	run := w.firstLaneRun(t)
	lane := run.Lanes[0]
	if lane.HandBack == nil || lane.Stage != StageHandedBack {
		t.Fatalf("the lane goes back: %+v\n%s", lane, w.said())
	}
	if run.ReservedAt != nil {
		t.Fatalf("the run holds nothing: %v", run.ReservedAt)
	}
	if !strings.Contains(lane.HandBack.Reason, says) {
		t.Fatalf("the reason says what happened (%q): %q", says, lane.HandBack.Reason)
	}
	rel, err := laneRel(run.RunID, lane.ID, stageRestart)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lane.HandBack.Home, rel+"/"+AsideDirName) || !strings.Contains(lane.HandBack.Home, lane.Branch) {
		t.Fatalf("the hand-back names where the work is: %q", lane.HandBack.Home)
	}
	if !savedFile(t, root, run, partialFile) {
		t.Fatalf("the lane's work is saved: %+v", asidesOf(t, root, run))
	}
	if fi, err := os.Stat(lane.Worktree); err != nil || !fi.IsDir() {
		t.Fatalf("the worktree stands: %v", err)
	}
	gitIn(t, root, "rev-parse", "--verify", "refs/heads/"+lane.Branch)
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the run goes on to the next issue and lands it: %+v\n%s", res, w.said())
	}
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes) != 2 || st.Lanes[0].Outcome != DrainLaneHandedBack || len(st.HandBacks) != 1 {
		t.Fatalf("one lane dropped, one landed: %+v %+v", st.Lanes, st.HandBacks)
	}
}

// TestWorkLeftAtTheReleaseIsSavedAndSurvives: what stands uncommitted in the lane's
// worktree when the threshold passes is saved aside by the stop's own save, the
// worktree is left at its last commit, and the save outlives the lane landing.
func TestWorkLeftAtTheReleaseIsSavedAndSurvives(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	start := w.clock.t
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
		if w.clock.t.Sub(start) >= 2*time.Hour {
			return true, nil
		}
		return false, limitedNoReset()
	}
	first := true
	stoodAfter := true
	var worktree string
	w.onSleep = func(time.Duration) {
		run := w.firstLaneRun(t)
		if first {
			first = false
			worktree = run.Lanes[0].Worktree
			if err := os.WriteFile(filepath.Join(worktree, partialFile), []byte("half done\n"), 0o600); err != nil {
				t.Error(err)
			}
		}
		if letGoByDrive(run) {
			_, err := os.Stat(filepath.Join(worktree, partialFile))
			stoodAfter = err == nil
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands: %+v %v\n%s", res, err, w.said())
	}
	run := w.firstLaneRun(t)
	if !savedFile(t, w.repo.Root(), run, partialFile) {
		t.Fatalf("the work is saved aside: %+v\n%s", asidesOf(t, w.repo.Root(), run), w.said())
	}
	if stoodAfter {
		t.Fatalf("the worktree is reset once the work is saved:\n%s", w.said())
	}
}

// TestALaneLetGoKeepsWhatItHeld: a path the lane widened to beyond the found_at, which
// its branch may change, is reserved again with it.
func TestALaneLetGoKeepsWhatItHeld(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	held := reservationOf(t, root, run.Key, c.t)
	wide := append(slices.Clone(held.Footprint), "docs/widened.md")
	if _, err := claim.Reserve(root, claim.Request{Record: run.Key, Footprint: wide}, claim.Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	if r := reservationOf(t, root, run.Key, c.t); r.State != claim.StateReleased {
		t.Fatalf("the issue is let go: %+v", r)
	}
	if done, ended := tm.takeUp(context.Background(), runID); done || ended != "" {
		t.Fatalf("the lane is taken up: %v %q", done, ended)
	}
	r := reservationOf(t, root, run.Key, c.t)
	if r.State != claim.StateLive || !slices.Contains(r.Footprint, "docs/widened.md") {
		t.Fatalf("reserved again with the path it widened to: %+v", r)
	}
}

// TestALaneLetGoKeepsWhatItWidenedToOverADeletedTip: the tip is deleted while the
// lane is let go, so nothing on origin names the path the lane widened to, and the
// reservation made again names it all the same.
func TestALaneLetGoKeepsWhatItWidenedToOverADeletedTip(t *testing.T) {
	f := newNarrowFixture(t)
	if _, err := f.widen(t, Options{}, "README.txt"); err != nil {
		t.Fatal(err)
	}
	root := f.repo.Root()
	tm := newTerminal(root, TerminalOptions{})
	tm.letGoForLimit(f.runID)
	if tip := f.tip(t); !tip.released {
		t.Fatalf("the issue is let go: %+v", tip)
	}
	gitIn(t, f.bare, "update-ref", "-d", claim.RefPrefix+secondIssue)
	if _, err := claim.Fetch(root, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if done, ended := tm.takeUp(context.Background(), f.runID); done || ended != "" {
		t.Fatalf("the lane is taken up: %v %q", done, ended)
	}
	if tip := f.tip(t); tip.released {
		t.Fatalf("the issue is reserved again: %+v", tip)
	}
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"README.txt", "docs"}) {
		t.Fatalf("the reservation names the found_at and the path the lane widened to: %q", fp)
	}
}

// laneAwaiting puts a role out on the run's lane, its receipt in the lane's directory.
func laneAwaiting(t *testing.T, root, runID string, running *RunnerClaim, now time.Time) string {
	t.Helper()
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := laneRel(runID, run.Lanes[0].ID, stageRestart)
	if err != nil {
		t.Fatal(err)
	}
	receipt := rel + "/validator-1.json"
	if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits = []Await{{Role: RoleAuditor, Receipt: receipt, Since: now, Running: running}}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return receipt
}

// TestARoleNotStartedIsToldAgainOnRecovery: a role out with no agent is marked as the
// stop marks one, so a stop would read no agent there, and is told again when the lane
// is reserved again, so the lane is not left waiting on a mark nothing clears.
func TestARoleNotStartedIsToldAgainOnRecovery(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	laneAwaiting(t, root, runID, nil, c.t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if !letGoByDrive(run) || run.Lanes[0].Awaits[0].Kept == "" {
		t.Fatalf("let go, with the role marked: %+v", run.Lanes[0].Awaits)
	}
	c.t = c.t.Add(5 * time.Hour)
	if done, ended := tm.takeUp(context.Background(), runID); done || ended != "" {
		t.Fatalf("the lane is taken up: %v %q", done, ended)
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	a := run.Lanes[0].Awaits[0]
	if a.Kept != "" || !a.Since.Equal(c.t) {
		t.Fatalf("the role is told again: %+v", a)
	}
}

// TestAReleaseThatFailedIsToldAgainOnceHeld: a release that did not reach origin leaves
// the tip standing, and a renewal then reads the lane as held rather than let go. What
// the release marked is told again all the same, so the lane does not stall on a mark
// and go back as stuck.
func TestAReleaseThatFailedIsToldAgainOnceHeld(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	laneAwaiting(t, root, runID, nil, c.t)
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	failures := 0
	claimReserve = func(repo string, req claim.Request, o claim.Options) (claim.Result, error) {
		if req.Release {
			failures++
			return claim.Result{}, errors.New("no route to origin")
		}
		return was(repo, req, o)
	}
	var said []string
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}, Say: func(line string) { said = append(said, line) }})
	tm.letGoForLimit(runID)
	if failures != 1 {
		t.Fatalf("the release was asked once and failed: %d", failures)
	}
	if text := strings.Join(said, "\n"); !strings.Contains(text, "did not reach origin") {
		t.Fatalf("the failure is said as one: %s", text)
	}
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if !letGoByDrive(run) || run.Lanes[0].Awaits[0].Kept == "" {
		t.Fatalf("the run is let go with the role marked: %+v", run.Lanes[0].Awaits)
	}
	if r := reservationOf(t, root, run.Key, c.t); r.State != claim.StateLive {
		t.Fatalf("the tip stands: %+v", r)
	}
	c.t = c.t.Add(90 * time.Minute)
	if renewed := RenewRunsIn(root, runID, c.t); len(renewed) == 0 {
		t.Fatal("the standing tip is renewed by a session waiting in this checkout")
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	if letGoByDrive(run) {
		t.Fatalf("the renewal reads the lane as held: %+v", run.Record)
	}
	if done, ended := tm.takeUp(context.Background(), runID); done || ended != "" {
		t.Fatalf("the lane goes on: %v %q", done, ended)
	}
	if run, err = ReadState(root, runID); err != nil {
		t.Fatal(err)
	}
	if a := run.Lanes[0].Awaits[0]; a.Kept != "" || !a.Since.Equal(c.t) {
		t.Fatalf("the role is told again: %+v", a)
	}
}

// TestAListingSaysALaneLetGoHoldsNothing: the run keeps its reservation's date for its
// take-up, and the listing says what is true while it is let go and again once it is held.
func TestAListingSaysALaneLetGoHoldsNothing(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	reserved := func() bool {
		t.Helper()
		l, err := ListRuns(root, runID, Options{Now: c.now})
		if err != nil || len(l.Runs) != 1 {
			t.Fatalf("the run is listed: %+v %v", l, err)
		}
		return l.Runs[0].Reserved
	}
	if !reserved() {
		t.Fatal("a run holding its reservation says so")
	}
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	if reserved() {
		t.Fatal("a lane let go lists no reservation")
	}
	if done, ended := tm.takeUp(context.Background(), runID); done || ended != "" {
		t.Fatalf("the lane is taken up: %v %q", done, ended)
	}
	if !reserved() {
		t.Fatal("a lane taken up lists its reservation again")
	}
}

// TestAStopAfterTheReleaseNamesWhatTheLimitSaved: the worktree is clean by the time the
// stop reads it, so the stop has nothing of its own to save, and its page still says where
// the lane's work is.
func TestAStopAfterTheReleaseNamesWhatTheLimitSaved(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.leavesWork(t)
	o := w.bounded(t, 30*24*time.Hour)
	w.resetIn(t, 5*time.Hour)
	home := stopOnFirstWait(t, w, &o)
	folder := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := runstop.WriteSetting(home, folder, ""); err != nil {
		t.Fatal(err)
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalHalted {
		t.Fatalf("the stop halts the run: %+v %v\n%s", res, err, w.said())
	}
	run := w.firstLaneRun(t)
	saved := asidesOf(t, w.repo.Root(), run)
	i := slices.IndexFunc(saved, func(a RestartAside) bool { return slices.Contains(a.Files, partialFile) })
	if i < 0 {
		t.Fatalf("the limit saved the work: %+v", saved)
	}
	c := controlOf(t, w.repo.Root(), res.Run)
	if c.Stop == nil || len(c.Stop.Left) != 1 {
		t.Fatalf("the stop says what it left: %+v", c.Stop)
	}
	if l := c.Stop.Left[0]; l.Aside != saved[i].Path || l.Files != len(saved[i].Files) {
		t.Fatalf("the stop names the aside at %s: %+v", saved[i].Path, l)
	}
	page := pageOf(t, folder, w.clock.t)
	if !strings.Contains(page, saved[i].Path) {
		t.Fatalf("the page names the aside at %s:\n%s", saved[i].Path, page)
	}
}

// TestAnAgentInItsCallKeepsTheHold: a limit met by one role says nothing of another
// still running, so nothing goes while it does.
func TestAnAgentInItsCallKeepsTheHold(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	laneAwaiting(t, root, runID, &RunnerClaim{Runner: runner.Claude, Session: "s", Until: c.t.Add(time.Hour)}, c.t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if letGoByDrive(run) || run.Lanes[0].Awaits[0].Kept != "" {
		t.Fatalf("the hold stands: %+v", run.Lanes[0].Awaits)
	}
	if r := reservationOf(t, root, run.Key, c.t); r.State != claim.StateLive {
		t.Fatalf("the reservation stands: %+v", r)
	}
}

// TestAParkedLaneKeepsItsReservation: a lane waiting at its pull request was no limit's
// to interrupt, and neither is one with an open pull request that went back to work.
func TestAParkedLaneKeepsItsReservation(t *testing.T) {
	t.Run("parked", func(t *testing.T) {
		f, _ := parkDrainLane(t)
		ref := claim.RefPrefix + secondIssue
		fenced := originReservation(t, f.bare, ref)
		at := fenced.claimed
		tm := newTerminal(f.repo.Root(), TerminalOptions{Options: Options{Now: func() time.Time { return at }}})
		tm.letGoForLimit(f.runID)
		if tip := originReservation(t, f.bare, ref); tip.commit != fenced.commit || tip.released {
			t.Fatalf("a parked lane keeps its reservation: %+v", tip)
		}
	})
	t.Run("an open pull request", func(t *testing.T) {
		root, _, runID, c := laneForPeerCheck(t)
		if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
			st.Lanes[0].PR = 7
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
		tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
		tm.letGoForLimit(runID)
		run, err := ReadState(root, runID)
		if err != nil {
			t.Fatal(err)
		}
		if letGoByDrive(run) {
			t.Fatalf("a lane with an open pull request keeps its reservation: %+v", run.Record)
		}
	})
}

// TestWorkThatCannotBeSavedKeepsTheHold: a worktree git cannot read holds work saved
// nowhere, and a peer is not let in over it: the reservation stays, renewed, however
// long the limit.
func TestWorkThatCannotBeSavedKeepsTheHold(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	o := w.bounded(t, 30*24*time.Hour)
	w.resetIn(t, 5*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap && res.Ended != TerminalWall {
		t.Logf("%+v %v\n%s", res, err, w.said())
	}
	run := w.firstLaneRun(t)
	if letGoByDrive(run) || slices.ContainsFunc(run.Record, func(e Entry) bool { return e.Stage == StageDriveReleased }) {
		t.Fatalf("the hold stands while the work cannot be saved:\n%s", w.said())
	}
	if !slices.ContainsFunc(run.Record, func(e Entry) bool { return e.Stage == StageReservationRenewed }) {
		t.Fatalf("and is renewed through the wait:\n%s", w.said())
	}
}

// TestAStopDuringTheWaitHolds: the issue already let go, a person's stop ends the wait
// at once and halts the run; nothing is reserved after the lift, a new drive names it
// and passes over it, and the take-over it waits for reserves again and re-tells it.
func TestAStopDuringTheWaitHolds(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	o.Drain.Max = 3
	w.resetIn(t, 5*time.Hour)
	home := stopOnFirstWait(t, w, &o)
	set := w.onSleep
	letGoFirst := false
	w.onSleep = func(d time.Duration) {
		letGoFirst = letGoByDrive(w.firstLaneRun(t))
		set(d)
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if !letGoFirst {
		t.Fatalf("the issue was let go before the stop:\n%s", w.said())
	}
	if res.Ended != TerminalHalted || w.launches(t) != 2 || len(w.sleeps) != 1 {
		t.Fatalf("the wait ends on the stop, with no launch after it: %+v, %d launches, %v\n%s", res, w.launches(t), w.sleeps, w.said())
	}
	c := controlOf(t, w.repo.Root(), res.Run)
	if c.Stop == nil || !c.Stop.FromFile || c.Stop.HaltedAt == nil {
		t.Fatalf("the run took the person's stop: %+v", c.Stop)
	}
	if c.Stop.Released == noReservationHeld {
		t.Fatalf("a run let go is not one that held nothing, which no take-over reserves: %+v", c.Stop)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("the issue stays released: %+v", r)
	}
	if _, err := runstop.Lift(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	again := w.options(t)
	again.Stop = make(chan struct{})
	again.Drain.Max = 0
	again.Options.stopHome = home
	next, err := RunTerminal(context.Background(), w.repo.Root(), again)
	if err != nil {
		t.Fatal(err)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateReleased {
		t.Fatalf("a new drive reserved a run a person's stop halted: %+v\n%s", r, w.said())
	}
	if !slices.ContainsFunc(next.PassedOver, func(p Excluded) bool {
		return p.ID == res.Issue && strings.Contains(p.Reason, "implement resume --take "+res.Run)
	}) {
		t.Fatalf("the drive names the run and its take-over: %+v", next.PassedOver)
	}
	if _, err := TakeRun(w.repo.Root(), res.Run, Options{stopHome: home, Now: w.clock.now}); err != nil {
		t.Fatalf("the take-over it waits for: %v", err)
	}
	if r := reservationOf(t, w.repo.Root(), res.Issue, w.clock.t); r.State != claim.StateLive {
		t.Fatalf("the take-over reserves again: %+v", r)
	}
}

// TestASignalDuringTheWaitIsTakenOverByTheNextDrive: a signal is that terminal's
// person's, so a later drive takes the run over, reserves it and lands it.
func TestASignalDuringTheWaitIsTakenOverByTheNextDrive(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	w.resetIn(t, 5*time.Hour)
	letGoFirst := false
	first := true
	w.onSleep = func(time.Duration) {
		if first {
			first = false
			letGoFirst = letGoByDrive(w.firstLaneRun(t))
			close(w.stop)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalStopped {
		t.Fatalf("the signal ends it: %+v %v\n%s", res, err, w.said())
	}
	if !letGoFirst {
		t.Fatalf("the issue was let go before the signal:\n%s", w.said())
	}
	w.clock.t = w.clock.t.Add(5 * time.Hour)
	w.onSleep = nil
	again := w.bounded(t, 24*time.Hour)
	again.Stop = make(chan struct{})
	next, err := RunTerminal(context.Background(), w.repo.Root(), again)
	if err != nil || next.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("a later drive lands the lane: %+v %v\n%s", next, err, w.said())
	}
}

// TestALaneLetGoReadsAsNoPersonsHalt: the release is the drive's, so nothing says a
// person has it: the bound ending the wait leaves a lane a later drive takes up, not
// one it passes over for a take-over.
func TestALaneLetGoReadsAsNoPersonsHalt(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 24*time.Hour)
	w.resetIn(t, 72*time.Hour)
	checked := false
	w.onSleep = func(time.Duration) {
		if checked {
			return
		}
		checked = true
		run := w.firstLaneRun(t)
		if !letGoByDrive(run) {
			t.Errorf("the issue is let go for the wait:\n%s", w.said())
		}
		if why := waitsUntaken(w.repo.Root(), run); why != "" {
			t.Errorf("nothing says a person has it: %q", why)
		}
		if haltedByPerson(w.repo.Root(), run.RunID) {
			t.Error("no person halted it")
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalUntil || res.Resume != TerminalResumeCommand {
		t.Fatalf("the bound ends the run, naming its resume: %+v %v\n%s", res, err, w.said())
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop != nil {
		t.Fatalf("no stop is marked: %+v", c.Stop)
	}
	w.clock.t = w.clock.t.Add(72 * time.Hour)
	w.onSleep = nil
	next, err := RunTerminal(context.Background(), w.repo.Root(), w.bounded(t, 24*time.Hour))
	if err != nil || next.Ended != DrainStoppedCap || len(next.PassedOver) != 0 || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("a later drive lands it, passing over nothing: %+v %v\n%s", next, err, w.said())
	}
}
