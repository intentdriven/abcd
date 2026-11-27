package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A drive sets aside a lane it cannot take and goes on (spc-2611141617199772 step 3;
// itd-2611141610197753 decisions 3, 4, 5, 8 and 9), against the terminal world: a lane
// that waits the stuck bound's minutes on what the drive cannot end is set aside, its
// reservation left to lapse, and the other lanes land; once no other lane is in progress
// it is tried again, then once an hour, until the drive's end.

// setAsideLine is the drive's line naming runID set aside, "" when it said none.
func setAsideLine(w *terminalWorld, runID string) string {
	for _, l := range w.lines {
		if strings.HasPrefix(l, "set aside ") && strings.Contains(l, "run "+runID) {
			return l
		}
	}
	return ""
}

// setAsideOf is the lane of runID among res's set aside, failing the test when it is not.
func setAsideOf(t *testing.T, w *terminalWorld, res TerminalResult, runID string) SetAsideLane {
	t.Helper()
	i := slices.IndexFunc(res.SetAside, func(s SetAsideLane) bool { return s.Run == runID })
	if i < 0 {
		t.Fatalf("the end does not name run %s set aside: %+v\n%s", runID, res.SetAside, w.said())
	}
	return res.SetAside[i]
}

// otherLanesLanded fails the test unless the two issues other than issue landed.
func otherLanesLanded(t *testing.T, w *terminalWorld, issue string) {
	t.Helper()
	if done := lanesDone(t, w.repo); len(done) != 2 || slices.Contains(done, issue) {
		t.Fatalf("the two other lanes land: %v\n%s", done, w.said())
	}
}

// tipAtSetAside reads the reservation of issue on origin as the drive says it set its lane
// aside, the first time, so a test can tell whether anything was pushed for it since.
func tipAtSetAside(t *testing.T, w *terminalWorld, o *TerminalOptions, issue string) func() reservation {
	t.Helper()
	var got *reservation
	say := o.Say
	o.Say = func(s string) {
		say(s)
		if got == nil && strings.HasPrefix(s, "set aside ") && strings.Contains(s, issue) {
			r := tipOf(t, w.bare, issue)
			got = &r
		}
	}
	return func() reservation {
		t.Helper()
		if got == nil {
			t.Fatalf("%s was never set aside:\n%s", issue, w.said())
		}
		return *got
	}
}

// leftToLapseAt fails the test unless nothing was pushed for issue since its lane was set
// aside, and this copy left that tip to lapse.
func leftToLapseAt(t *testing.T, w *terminalWorld, issue string, at reservation) {
	t.Helper()
	if now := tipOf(t, w.bare, issue); now.commit != at.commit || now.released {
		t.Fatalf("a push was made for %s after it was set aside: %+v, was %+v", issue, now, at)
	}
	if !claim.LeftToLapse(w.repo.Root(), issue, at.commit) {
		t.Fatalf("%s's reservation is not left to lapse at %s", issue, at.commit)
	}
}

// setAsideBlockedByOther is a world whose lane's run the next drive cannot claim at its
// opening: the first drive was killed with the lane in flight, and its recorded driver,
// alive in a second terminal, took it since. It returns the run and the stop home.
func setAsideBlockedByOther(t *testing.T) (*terminalWorld, State, string) {
	t.Helper()
	w, run, home := stopKilled(t)
	claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive, claimOtherLogin, claimOtherShell)
	claimDriverIs(t, w.repo.Root(), run.RunID, claimOtherLogin)
	return w, run, home
}

// setAsideUntil runs the next start under a bound hours after the clock, returning the
// result and the bound.
func setAsideUntil(t *testing.T, w *terminalWorld, home string, hours int, set func(*TerminalOptions)) (TerminalResult, time.Time) {
	t.Helper()
	var until time.Time
	res, err := claimStartWith(t, w, home, func(o *TerminalOptions) {
		until = w.clock.t.Add(time.Duration(hours) * time.Hour)
		o.Until = until
		if set != nil {
			set(o)
		}
	})
	if err != nil {
		t.Fatalf("the start: %v\n%s", err, w.said())
	}
	return res, until
}

// waitedAtLeast fails the test unless s waited the bound before it was set aside, and no
// longer than the bound and the longest wait between looks.
func waitedAtLeast(t *testing.T, s SetAsideLane, bound time.Duration) {
	t.Helper()
	if waited := s.At.Sub(s.Since); waited < bound || waited > bound+terminalIdleCap {
		t.Fatalf("set aside after %s; want the bound %s", waited, bound)
	}
}

// TestARunWithALiveDriverInAnotherTerminalIsSetAside: the lane's run is held by a live
// driver in another terminal, which the drive may not take. At the stuck bound's minutes
// the lane is set aside, saying why, and the two other lanes land; nothing is pushed for
// it after, and its reservation is left to lapse. Once the others are done it is tried
// again, set aside again at once where nothing changed, and then tried once an hour until
// the bound, never spinning.
func TestARunWithALiveDriverInAnotherTerminalIsSetAside(t *testing.T) {
	w, run, home := setAsideBlockedByOther(t)
	var tip func() reservation
	res, until := setAsideUntil(t, w, home, 4, func(o *TerminalOptions) { tip = tipAtSetAside(t, w, o, run.Key) })
	if res.Ended != TerminalUntil {
		t.Fatalf("the drive ends at its bound: %+v\n%s", res, w.said())
	}
	s := setAsideOf(t, w, res, run.RunID)
	waitedAtLeast(t, s, BundledStuckMinutes*time.Minute)
	if !strings.Contains(s.Reason, "process "+strconv.Itoa(claimOtherLogin)) || !strings.Contains(setAsideLine(w, run.RunID), "process "+strconv.Itoa(claimOtherLogin)) {
		t.Fatalf("the set-aside does not say who holds the run: %+v\n%s", s, w.said())
	}
	otherLanesLanded(t, w, run.Key)
	leftToLapseAt(t, w, run.Key, tip())
	if c := controlOf(t, w.repo.Root(), run.RunID); c.Driver == nil || c.Driver.PID != claimOtherLogin {
		t.Fatalf("the drive took the run from its live driver: %+v", c.Driver)
	}
	tries := 0
	for _, l := range w.lines {
		if strings.HasPrefix(l, "trying again the lanes set aside") {
			tries++
		}
	}
	// Tried as the other lanes are done, then hourly: no more than once an hour of the
	// time that was left to the bound.
	if left := int(until.Sub(s.At) / time.Hour); tries < 2 || tries > left+1 {
		t.Fatalf("tried again %d times in the %d hours left; want once as the others were done and then hourly:\n%s", tries, left, w.said())
	}
	if !strings.Contains(w.said(), "set aside "+run.Key+"'s lane, run "+run.RunID+", again") {
		t.Fatalf("a try blocked as before does not set the lane aside again at once:\n%s", w.said())
	}
	if !strings.Contains(w.said(), "waiting ") || !strings.Contains(w.said(), "every lane left in progress is set aside") {
		t.Fatalf("the drive does not say it waits on its lanes set aside:\n%s", w.said())
	}
	if res.Resume != TerminalResumeCommand {
		t.Fatalf("the end does not name the start that tries the lane again: %+v", res)
	}
}

// TestARunItsOwnTerminalHoldsWithAProcessBesideTheDriveIsSetAside: the recorded driver is
// the login process of the drive's own terminal, but a helper runs beneath it besides the
// drive, so the claim leaves the run: at the bound the lane is set aside, naming the
// process that kept it, and the two other lanes land.
func TestARunItsOwnTerminalHoldsWithAProcessBesideTheDriveIsSetAside(t *testing.T) {
	w, run, home := stopKilled(t)
	claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive, claimHelper)
	claimDriverIs(t, w.repo.Root(), run.RunID, claimLogin)
	res, _ := setAsideUntil(t, w, home, 2, nil)
	s := setAsideOf(t, w, res, run.RunID)
	waitedAtLeast(t, s, BundledStuckMinutes*time.Minute)
	if !strings.Contains(s.Reason, strconv.Itoa(claimHelper)) {
		t.Fatalf("the set-aside does not name the process beside the drive: %+v", s)
	}
	otherLanesLanded(t, w, run.Key)
}

// TestABlockThatClearsJustAfterTheSetAsideLands: the driver in the other terminal ends
// just after the lane is set aside, so the try once the other lanes are done claims the
// run and lands it, and nothing is left set aside.
func TestABlockThatClearsJustAfterTheSetAsideLands(t *testing.T) {
	w, run, home := setAsideBlockedByOther(t)
	res, _ := setAsideUntil(t, w, home, 4, func(o *TerminalOptions) {
		say := o.Say
		o.Say = func(s string) {
			say(s)
			if strings.HasPrefix(s, "set aside ") {
				claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive)
			}
		}
	})
	if done := lanesDone(t, w.repo); len(done) != 3 || !slices.Contains(done, run.Key) {
		t.Fatalf("the lane set aside lands once its block clears: %v\n%s", done, w.said())
	}
	if len(res.SetAside) != 0 || res.Ended != DrainStoppedCap {
		t.Fatalf("nothing is left set aside, and the drive ends at its cap: %+v\n%s", res, w.said())
	}
	tip := tipOf(t, w.bare, run.Key)
	if claim.LeftToLapse(w.repo.Root(), run.Key, tip.commit) {
		t.Fatal("the lane taken up again is still left to lapse")
	}
}

// TestATryAfterTheLeaseLapsedReservesAgain: the lane's hold lapses while it is set aside,
// since nothing renews it, so the try once its block clears reserves it again before it
// steps; and where a peer has taken the issue since, the lane is dropped to the peer.
func TestATryAfterTheLeaseLapsedReservesAgain(t *testing.T) {
	for _, peer := range []bool{false, true} {
		t.Run(map[bool]string{false: "lapsed", true: "a peer holds it"}[peer], func(t *testing.T) {
			w, run, home := setAsideBlockedByOther(t)
			cleared := false
			res, _ := setAsideUntil(t, w, home, 8, func(o *TerminalOptions) {
				say := o.Say
				o.Say = func(s string) {
					say(s)
					if cleared || !strings.HasPrefix(s, "set aside ") {
						return
					}
					cleared = true
					// The other terminal ends, and the hold lapses before the next try.
					claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive)
					w.clock.t = tipOf(t, w.bare, run.Key).expires.Add(10 * time.Minute)
					if peer {
						q := otherCopy(t, w.bare, quinnEmail)
						if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Footprint: []string{"commands/drain.md"}}, as(t, q, claimOptions(w.clock.t))); err != nil {
							t.Fatalf("the peer reserves: %v", err)
						}
					}
				}
			})
			if !cleared {
				t.Fatalf("the lane was never set aside:\n%s", w.said())
			}
			if peer {
				if !strings.Contains(w.said(), run.Key+", run "+run.RunID+", is dropped") {
					t.Fatalf("the lane whose issue a peer holds is not dropped:\n%s", w.said())
				}
				if tip := tipOf(t, w.bare, run.Key); tip.holder == heldBy(t, patEmail) {
					t.Fatalf("the drive pushed over the peer's hold: %+v", tip)
				}
				return
			}
			if !tookUpLine(w, run.RunID) {
				t.Fatalf("the try does not reserve the lapsed lane again:\n%s", w.said())
			}
			if done := lanesDone(t, w.repo); len(done) != 3 || len(res.SetAside) != 0 {
				t.Fatalf("the lane reserved again lands: %v %+v\n%s", done, res.SetAside, w.said())
			}
		})
	}
}

// setAsideMinutes sets pace.stuck_minutes in the repository's layer.
func setAsideMinutes(t *testing.T, w *terminalWorld, minutes int) {
	t.Helper()
	repoConfig(t, w.repo, fmt.Sprintf(`{"pace": {"stuck_minutes": %d}}`, minutes))
}

// recordGoneAtLand has the landing of each issue of issues, the first time it is reached,
// remove the issue's record from the checkout and from origin's default branch and then
// send the lane back for a fresh round, so leaving the landing reserves its whole found_at
// again and finds no record (runFootprint's missing-record refusal, the lab's F-14).
func recordGoneAtLand(t *testing.T, w *terminalWorld, issues ...string) {
	t.Helper()
	gone := map[string]bool{}
	for i := range w.stages {
		if w.stages[i].Name != StageLand {
			continue
		}
		run := w.stages[i].Run
		w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
			if !slices.Contains(issues, c.State.Key) {
				return run(c, l)
			}
			if !gone[c.State.Key] {
				gone[c.State.Key] = true
				paths, _ := filepath.Glob(filepath.Join(w.repo.Root(), ".abcd", "work", "issues", "*", c.State.Key+"*"))
				if len(paths) == 0 {
					t.Errorf("no record of %s to remove", c.State.Key)
				}
				for _, p := range paths {
					rel, _ := filepath.Rel(w.repo.Root(), p)
					w.repo.Git("rm", "-q", filepath.ToSlash(rel))
				}
				w.repo.Commit("the record goes")
				w.repo.Git("push", "-q", "origin", "main")
				w.repo.Git("fetch", "-q", "origin")
			}
			return Outcome{Goto: StageImplement, Note: "a fresh round"}, nil
		}
	}
}

// TestTheLandingsMissingRecordIsSetAside: the landing's refusal of a record in no status
// folder names no lane, so the stuck bounds never count it; the drive sets the lane aside
// at the bound's minutes, under --until and with no bound, the two other lanes land, and
// nothing is left for a person to clear.
func TestTheLandingsMissingRecordIsSetAside(t *testing.T) {
	for _, bounded := range []bool{true, false} {
		t.Run(map[bool]string{true: "under --until", false: "with no bound"}[bounded], func(t *testing.T) {
			w := newTerminalWorld(t)
			setAsideMinutes(t, w, 30)
			recordGoneAtLand(t, w, eligibleIssue)
			o := w.options(t)
			if bounded {
				o.Until = w.clock.t.Add(3 * time.Hour)
			}
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatalf("the drive: %v\n%s", err, w.said())
			}
			want := map[bool]string{true: TerminalUntil, false: TerminalWaited}[bounded]
			if res.Ended != want {
				t.Fatalf("the drive ends %q: %+v\n%s", want, res, w.said())
			}
			i := slices.IndexFunc(res.SetAside, func(s SetAsideLane) bool { return s.Issue == eligibleIssue })
			if i < 0 {
				t.Fatalf("the lane whose record went is not set aside: %+v\n%s", res.SetAside, w.said())
			}
			s := res.SetAside[i]
			waitedAtLeast(t, s, 30*time.Minute)
			if !strings.Contains(s.Reason, "is in no status folder") {
				t.Fatalf("the set-aside does not give the landing's reason: %+v", s)
			}
			otherLanesLanded(t, w, eligibleIssue)
			run, err := ReadState(w.repo.Root(), s.Run)
			if err != nil {
				t.Fatal(err)
			}
			if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress {
				t.Fatalf("the lane set aside is handed back: %s", outcome)
			}
		})
	}
}

// TestPaceStuckMinutesMovesTheBound: pace.stuck_minutes set in the repository's layer is
// the bound a lane waits before it is set aside, as it is the stuck bound's.
func TestPaceStuckMinutesMovesTheBound(t *testing.T) {
	w := newTerminalWorld(t)
	setAsideMinutes(t, w, 5)
	recordGoneAtLand(t, w, eligibleIssue)
	o := w.options(t)
	o.Until = w.clock.t.Add(2 * time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(res.SetAside, func(s SetAsideLane) bool { return s.Issue == eligibleIssue })
	if i < 0 {
		t.Fatalf("the lane is not set aside: %+v\n%s", res, w.said())
	}
	if waited := res.SetAside[i].At.Sub(res.SetAside[i].Since); waited < 5*time.Minute || waited >= BundledStuckMinutes*time.Minute {
		t.Fatalf("set aside after %s; want the five minutes the layer sets", waited)
	}
}

// TestTwoLanesBlockedAtOnceAreBothSetAside: two lanes each meet the landing's missing
// record: each is set aside in turn, the third lands, and both are tried again and set
// aside again until the bound.
func TestTwoLanesBlockedAtOnceAreBothSetAside(t *testing.T) {
	w := newTerminalWorld(t)
	setAsideMinutes(t, w, 10)
	recordGoneAtLand(t, w, eligibleIssue, secondIssue)
	o := w.options(t)
	o.Until = w.clock.t.Add(3 * time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.SetAside) != 2 || res.Ended != TerminalUntil {
		t.Fatalf("both lanes are left set aside at the bound: %+v\n%s", res, w.said())
	}
	if done := lanesDone(t, w.repo); !slices.Equal(done, []string{thirdIssue}) {
		t.Fatalf("the third lane lands: %v\n%s", done, w.said())
	}
	for _, id := range []string{eligibleIssue, secondIssue} {
		if !strings.Contains(w.said(), "set aside "+id+"'s lane") || !strings.Contains(w.said(), id+"'s lane, run ") {
			t.Fatalf("%s is not set aside and tried again:\n%s", id, w.said())
		}
	}
}

// TestADrainWithACapOfOneWaitsOnItsSetAsideLane: with a cap of one, the one lane is set
// aside and the cap is reached, so no other lane opens; under decision 9 the drive does not
// end there: it waits, says so, tries the lane once an hour, and ends at its bound.
func TestADrainWithACapOfOneWaitsOnItsSetAsideLane(t *testing.T) {
	w := newTerminalWorld(t)
	setAsideMinutes(t, w, 10)
	recordGoneAtLand(t, w, eligibleIssue, secondIssue, thirdIssue)
	o := w.options(t)
	o.Drain.Max = 1
	o.Until = w.clock.t.Add(4 * time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalUntil || len(res.SetAside) != 1 || res.Last.Counts.LanesOpened != 1 {
		t.Fatalf("the drive waits on its one lane set aside until its bound: %+v\n%s", res, w.said())
	}
	tries := 0
	for _, l := range w.lines {
		if strings.HasPrefix(l, "trying again the lanes set aside") {
			tries++
		}
	}
	if tries < 3 || tries > 5 {
		t.Fatalf("tried again %d times in four hours; want once at the cap and then hourly:\n%s", tries, w.said())
	}
	waits := 0
	for _, l := range w.lines {
		if strings.HasPrefix(l, "waiting ") && strings.Contains(l, "every lane left in progress is set aside") {
			waits++
		}
	}
	if waits == 0 {
		t.Fatalf("the wait on the lane set aside is not said as waiting:\n%s", w.said())
	}
}

// TestAMachineRefusalThatNamesNoLaneSetsNothingAside is the test of what must not count: a
// review-set refusal and a configuration refusal name no lane and are the machine's, not
// the lane's. They are logged on no lane, the lane is neither handed back nor set aside
// however long they last, and the drive waits on them until its bound.
func TestAMachineRefusalThatNamesNoLaneSetsNothingAside(t *testing.T) {
	for _, stage := range []string{StageReview, StageConfig} {
		t.Run(stage, func(t *testing.T) {
			w := newTerminalWorld(t)
			setAsideMinutes(t, w, 5)
			for i := range w.stages {
				if w.stages[i].Name == StageLand {
					w.stages[i].Run = func(Context, *Lane) (Outcome, error) {
						return Outcome{}, refuse(stage, "", "", "the "+stage+" cannot be read on this machine", "repair it")
					}
				}
			}
			o := w.options(t)
			o.Until = w.clock.t.Add(2 * time.Hour)
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ended != TerminalUntil || len(res.SetAside) != 0 || strings.Contains(w.said(), "set aside ") {
				t.Fatalf("a machine's refusal set a lane aside: %+v\n%s", res, w.said())
			}
			if res.Run == "" {
				t.Fatalf("the lane is not left in progress: %+v", res)
			}
			run, err := ReadState(w.repo.Root(), res.Run)
			if err != nil {
				t.Fatal(err)
			}
			if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress {
				t.Fatalf("the lane went back for a machine's refusal: %s", outcome)
			}
			if i := run.current(); i < 0 || run.Lanes[i].Refused != nil {
				t.Fatalf("the refusal was logged on the lane: %+v", run.Lanes)
			}
		})
	}
}

// TestALaterStartTakesUpAKeptHoldNoDrainListsOnceItsDriverIsGone answers the question step
// 2 left to this step: a run no drain lists whose hold a drive's end kept, its stop not
// applied, carries no mark and is reserved. The hold is left to lapse, so no passive
// renewal keeps it; a later start, the driver that held it gone, takes it up without
// reserving it again while its hold is live, and drives it to its end.
func TestALaterStartTakesUpAKeptHoldNoDrainListsOnceItsDriverIsGone(t *testing.T) {
	w, run, home := stopUnclaimable(t, schemaVersionPrevious)
	root := w.repo.Root()
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(DrainStateRel))); err != nil {
		t.Fatal(err)
	}
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: w.clock.now}, Say: func(s string) { w.lines = append(w.lines, s) }})
	tm.stepped = []string{run.RunID}
	tm.haltOnSignal()
	tm.letGo()
	kept := tipOf(t, w.bare, run.Key)
	if kept.released || !claim.LeftToLapse(root, run.Key, kept.commit) {
		t.Fatalf("premise: the drive's end keeps the hold and leaves it to lapse: %+v\n%s", kept, w.said())
	}
	if !strings.Contains(w.said(), "a drive's next start takes it up once its driver is gone") {
		t.Fatalf("the kept line does not say a start takes the run up:\n%s", w.said())
	}
	if got := RenewRunsIn(root, "", kept.claimed.Add(61*time.Minute)); len(got) != 0 {
		t.Fatalf("implement status renewed the kept hold: %+v", got)
	}

	claimNextStart(t)
	claimStart(t, w, home)
	after, err := ReadState(root, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if outcome, _, _ := laneOutcome(after); outcome != DrainLaneDone {
		t.Fatalf("the kept run is not taken up and driven to its end: %s\n%s", outcome, w.said())
	}
	if tookUpLine(w, run.RunID) {
		t.Fatalf("the start reserved again a hold still live:\n%s", w.said())
	}
}
