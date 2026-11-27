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

// noteAtSetAside reports, once issue's lane has been set aside, whether this copy left its
// reservation's tip to lapse as the drive said so, the first time.
func noteAtSetAside(t *testing.T, w *terminalWorld, o *TerminalOptions, issue string) func() bool {
	t.Helper()
	var noted *bool
	say := o.Say
	o.Say = func(s string) {
		say(s)
		if noted == nil && strings.HasPrefix(s, "set aside ") && strings.Contains(s, issue) {
			n := claim.LeftToLapse(w.repo.Root(), issue, tipOf(t, w.bare, issue).commit)
			noted = &n
		}
	}
	return func() bool {
		t.Helper()
		if noted == nil {
			t.Fatalf("%s was never set aside:\n%s", issue, w.said())
		}
		return *noted
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
// the lane is set aside, saying why, and the two other lanes land. The live driver is
// still working the run, so its reservation is not left to lapse: no note is written,
// and a passive renewal in this copy renews it as before (dcn-2611270753395749).
// Once the others are done it is tried again, set aside again at once where nothing
// changed, and then tried once an hour until the bound, never spinning.
func TestARunWithALiveDriverInAnotherTerminalIsSetAside(t *testing.T) {
	w, run, home := setAsideBlockedByOther(t)
	root := w.repo.Root()
	var noted, renewed, seen bool
	res, until := setAsideUntil(t, w, home, 4, func(o *TerminalOptions) {
		say := o.Say
		o.Say = func(l string) {
			say(l)
			if seen || !strings.HasPrefix(l, "set aside ") || !strings.Contains(l, run.Key) {
				return
			}
			seen = true
			tip := tipOf(t, w.bare, run.Key)
			noted = claim.LeftToLapse(root, run.Key, tip.commit)
			for _, r := range RenewRunsIn(root, "", tip.claimed.Add(61*time.Minute)) {
				renewed = renewed || (r.RunID == run.RunID && r.Renewed)
			}
		}
	})
	if res.Ended != TerminalUntil {
		t.Fatalf("the drive ends at its bound: %+v\n%s", res, w.said())
	}
	s := setAsideOf(t, w, res, run.RunID)
	waitedAtLeast(t, s, BundledStuckMinutes*time.Minute)
	if !strings.Contains(s.Reason, "process "+strconv.Itoa(claimOtherLogin)) || !strings.Contains(setAsideLine(w, run.RunID), "process "+strconv.Itoa(claimOtherLogin)) {
		t.Fatalf("the set-aside does not say who holds the run: %+v\n%s", s, w.said())
	}
	otherLanesLanded(t, w, run.Key)
	if !seen || noted || s.LeftToLapse || !renewed {
		t.Fatalf("a lane a live driver holds is left to lapse: set aside %v, note %v, LeftToLapse %v, renewed by implement status %v\n%s", seen, noted, s.LeftToLapse, renewed, w.said())
	}
	if !strings.Contains(setAsideLine(w, run.RunID), "left to the driver that holds the run") {
		t.Fatalf("the set-aside does not say the hold is the live driver's: %q", setAsideLine(w, run.RunID))
	}
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
// since nothing in the test renews it, so the try once its block clears holds it again
// before it steps; and where a peer has taken the issue since, the lane is dropped to the
// peer.
func TestATryAfterTheLeaseLapsedReservesAgain(t *testing.T) {
	for _, peer := range []bool{false, true} {
		t.Run(map[bool]string{false: "lapsed", true: "a peer holds it"}[peer], func(t *testing.T) {
			w, run, home := setAsideBlockedByOther(t)
			cleared, stepped, heldAtStep := false, false, claim.StateExpired
			for i := range w.stages {
				inner := w.stages[i].Run
				w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
					if cleared && !stepped && c.State.Key == run.Key {
						stepped = true
						heldAtStep = reservationOf(t, w.repo.Root(), run.Key, w.clock.t).State
					}
					return inner(c, l)
				}
			}
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
			if !stepped || heldAtStep != claim.StateLive {
				t.Fatalf("the try steps the lane holding nothing: stepped %v, its hold %s\n%s", stepped, heldAtStep, w.said())
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
			noted := noteAtSetAside(t, w, &o, eligibleIssue)
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatalf("the drive: %v\n%s", err, w.said())
			}
			if !noted() {
				t.Fatalf("the hold of a lane no driver works is not left to lapse as it is set aside:\n%s", w.said())
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
			if !strings.Contains(s.Reason, "is in no status folder") || !s.LeftToLapse {
				t.Fatalf("the set-aside does not give the landing's reason, or leave its hold to lapse: %+v", s)
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
	// The driver the stop could not get past ends before the drive lets go, so the hold is
	// no one's work (dcn-2611270753395749).
	claimNextStart(t)
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

// TestAKeptHoldALiveDriverWorksKeepsItsRenewals: the signal's stop could not be applied
// because another live driver holds the run, which is still alive as the drive lets go:
// that driver is working the run, so its hold is not left to lapse, the kept line names
// the renewers, and implement status renews it (dcn-2611270753395749).
func TestAKeptHoldALiveDriverWorksKeepsItsRenewals(t *testing.T) {
	w, run, _ := stopUnclaimable(t, schemaVersionPrevious)
	root := w.repo.Root()
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: w.clock.now}, Say: func(s string) { w.lines = append(w.lines, s) }})
	tm.haltOnSignal()
	tm.letGo()
	kept := tipOf(t, w.bare, run.Key)
	if kept.released || claim.LeftToLapse(root, run.Key, kept.commit) {
		t.Fatalf("the hold a live driver works is released, or left to lapse: %+v\n%s", kept, w.said())
	}
	// The run is short of its pull request, so the kept line names the status call with its limit.
	if !strings.Contains(w.said(), unlessShortRunRenewed) {
		t.Fatalf("the kept line does not name the renewers:\n%s", w.said())
	}
	renewed := false
	for _, r := range RenewRunsIn(root, "", kept.claimed.Add(61*time.Minute)) {
		renewed = renewed || (r.RunID == run.RunID && r.Renewed)
	}
	if !renewed {
		t.Fatalf("implement status does not renew the hold a live driver works:\n%s", w.said())
	}
}

// TestALaneGivenBackForItsLookAndNotReachedStaysSetAside: two lanes are set aside, and at
// their try the drive reaches one, whose stage runs past the bound, so the drive ends
// before the other's look. That other lane is still set aside: the end names it so, its
// reservation is kept, not released, and it is left to lapse again.
func TestALaneGivenBackForItsLookAndNotReachedStaysSetAside(t *testing.T) {
	w := newTerminalWorld(t)
	setAsideMinutes(t, w, 10)
	recordGoneAtLand(t, w, eligibleIssue, secondIssue)
	o := w.options(t)
	until := w.clock.t.Add(3 * time.Hour)
	o.Until = until
	tried, reached := false, ""
	say := o.Say
	o.Say = func(l string) {
		say(l)
		if strings.HasPrefix(l, "trying again the lanes set aside") {
			tried = true
		}
	}
	for i := range w.stages {
		run := w.stages[i].Run
		w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
			if tried && reached == "" && (c.State.Key == eligibleIssue || c.State.Key == secondIssue) {
				reached = c.State.Key
				w.clock.t = until.Add(time.Minute)
			}
			return run(c, l)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalUntil || reached == "" {
		t.Fatalf("premise: the bound passes in the stage of the first lane tried: %q %+v\n%s", reached, res, w.said())
	}
	other := map[string]string{eligibleIssue: secondIssue, secondIssue: eligibleIssue}[reached]
	i := slices.IndexFunc(res.SetAside, func(s SetAsideLane) bool { return s.Issue == other })
	if i < 0 {
		t.Fatalf("the lane given back for its look and not reached is not named set aside: %+v\n%s", res.SetAside, w.said())
	}
	tip := tipOf(t, w.bare, other)
	if tip.released || !claim.LeftToLapse(w.repo.Root(), other, tip.commit) || !res.SetAside[i].LeftToLapse {
		t.Fatalf("the lane still set aside is released, or not left to lapse: %+v %+v\n%s", tip, res.SetAside[i], w.said())
	}
}

// TestAStartLetsGoAnUnlistedRunItTookUpThatCannotMove (iss-2611270555396370): the start
// takes up a run no drain lists that an earlier drive's end let go, and its step is
// refused, so it comes back still in progress, and no move of the drive reaches it. The
// start lets it go before the drain's first move opens a lane, so the drive never holds two
// reserved runs and steps one; the run is recorded as let go, for a later start to take up.
func TestAStartLetsGoAnUnlistedRunItTookUpThatCannotMove(t *testing.T) {
	w, run, home := stopLetGoAtBound(t)
	stopLaterStart(t)
	root := w.repo.Root()
	seen := refuseLandOf(t, w, run.Key)
	claimStart(t, w, home)
	if !tookUpLine(w, run.RunID) {
		t.Fatalf("premise: the start takes the run up:\n%s", w.said())
	}
	opened, heldThen := seen()
	if !opened {
		t.Fatalf("premise: the drain opens a lane:\n%s", w.said())
	}
	if heldThen == claim.StateLive {
		t.Fatalf("the drive held the run it took up beside the lane the drain opened:\n%s", w.said())
	}
	after, err := ReadState(root, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if outcome, _, _ := laneOutcome(after); outcome != DrainLaneInProgress || !letGoByDrive(after) {
		t.Fatalf("the run is not left in progress and let go for a later start: %s\n%s", outcome, w.said())
	}
	if !strings.Contains(w.said(), "which no drain lists, is still in progress") {
		t.Fatalf("the let-go is not said:\n%s", w.said())
	}
}

// refuseLandOf has the landing of key's run refuse, and reports the reservation of key
// the first time a stage of any other run is reached, which is the drain's first lane.
func refuseLandOf(t *testing.T, w *terminalWorld, key string) func() (bool, string) {
	t.Helper()
	root := w.repo.Root()
	opened, heldThen := false, claim.StateLive
	for i := range w.stages {
		inner, name := w.stages[i].Run, w.stages[i].Name
		w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
			if c.State.Key == key {
				if name == StageLand {
					return Outcome{}, refuse(string(StageLand), "", l.ID, "the landing cannot go on here", "see the reason")
				}
				return inner(c, l)
			}
			if !opened {
				opened = true
				heldThen = reservationOf(t, root, key, w.clock.t).State
			}
			return inner(c, l)
		}
	}
	return func() (bool, string) { return opened, heldThen }
}

// TestAStartLetsGoAStopMarkedUnlistedRunItTookOverThatCannotMove (iss-2611270555396370):
// the run no drain lists stands under a settled signal's mark, so the start takes it over
// (TakeRun) rather than taking it up, and its step is refused, so it comes back still in
// progress. The start lets it go before the drain's first move opens a lane, as it does a
// run it took up with no mark, and records it as let go, for a later start to take up.
func TestAStartLetsGoAStopMarkedUnlistedRunItTookOverThatCannotMove(t *testing.T) {
	w, run, home := claimReplaced(t)
	if c := controlOf(t, w.repo.Root(), run.RunID); c.Stop == nil {
		t.Fatalf("premise: a mark stands on the run")
	}
	seen := refuseLandOf(t, w, run.Key)
	claimStart(t, w, home)
	tookOver(t, w, run.RunID)
	opened, heldThen := seen()
	if !opened {
		t.Fatalf("premise: the drain opens a lane:\n%s", w.said())
	}
	if heldThen == claim.StateLive {
		t.Fatalf("the drive held the run it took over beside the lane the drain opened:\n%s", w.said())
	}
	after, err := ReadState(w.repo.Root(), run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if outcome, _, _ := laneOutcome(after); outcome != DrainLaneInProgress || !letGoByDrive(after) {
		t.Fatalf("the run is not left in progress and let go for a later start: %s\n%s", outcome, w.said())
	}
	if !strings.Contains(w.said(), "which no drain lists, is still in progress") {
		t.Fatalf("the let-go is not said:\n%s", w.said())
	}
}

// drainRunOf is the run of issue's lane in the checkout's drain, failing the test when no
// lane lists it.
func drainRunOf(t *testing.T, root, issue string) string {
	t.Helper()
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(st.Lanes, func(l DrainLane) bool { return l.Issue == issue })
	if i < 0 {
		t.Fatalf("no lane of the drain lists %s: %+v", issue, st.Lanes)
	}
	return st.Lanes[i].RunID
}

// TestALaneALiveDriverTakesAfterItWasSetAsideKeepsItsRenewals: a lane set aside at the
// landing's missing record has its hold left to lapse. Before its try a person takes its
// run in another terminal, so the try is blocked at the claim and the lane is set aside
// for that live driver. The note left before is taken off as it is, so the driver's
// renewals keep the hold, as the drive says they do (dcn-2611270753395749).
func TestALaneALiveDriverTakesAfterItWasSetAsideKeepsItsRenewals(t *testing.T) {
	w := newTerminalWorld(t)
	claimProcs(t, claimShell, claimDrive, claimLogin, claimShell, claimDrive, claimOtherLogin, claimOtherShell)
	setAsideMinutes(t, w, 10)
	recordGoneAtLand(t, w, eligibleIssue)
	root := w.repo.Root()
	o := w.options(t)
	o.Until = w.clock.t.Add(3 * time.Hour)
	first := noteAtSetAside(t, w, &o, eligibleIssue)
	taken, seen, noted, worded := false, false, true, false
	say := o.Say
	o.Say = func(l string) {
		say(l)
		switch {
		case !taken && strings.HasPrefix(l, "trying again the lanes set aside"):
			taken = true
			claimDriverIs(t, root, drainRunOf(t, root, eligibleIssue), claimOtherLogin)
		case taken && !seen && strings.HasPrefix(l, "set aside ") && strings.Contains(l, eligibleIssue):
			seen = true
			noted = claim.LeftToLapse(root, eligibleIssue, tipOf(t, w.bare, eligibleIssue).commit)
			worded = strings.Contains(l, "left to the driver that holds the run")
		}
	}
	res, err := RunTerminal(context.Background(), root, o)
	if err != nil {
		t.Fatalf("the drive: %v\n%s", err, w.said())
	}
	if !first() || !taken || !seen || !worded {
		t.Fatalf("premise: set aside left to lapse, then set aside for the live driver that took the run: %v %v %v %v\n%s", first(), taken, seen, worded, w.said())
	}
	if noted {
		t.Fatalf("the note left before stands on a hold the drive says a live driver's renewals keep:\n%s", w.said())
	}
	if s := setAsideOf(t, w, res, drainRunOf(t, root, eligibleIssue)); s.LeftToLapse {
		t.Fatalf("the end names the live driver's hold left to lapse: %+v", s)
	}
}

// TestALaneTakenUpAgainIsNotNamedLeftToLapseAtAKill: a lane set aside, its hold left to
// lapse, is tried again and taken up, which takes the note off, and the drive is killed in
// its stage. The end names the hold's lapse without saying it was left to lapse: a renewal
// in this copy keeps it again.
func TestALaneTakenUpAgainIsNotNamedLeftToLapseAtAKill(t *testing.T) {
	w := newTerminalWorld(t)
	setAsideMinutes(t, w, 10)
	recordGoneAtLand(t, w, eligibleIssue)
	root := w.repo.Root()
	o := w.options(t)
	o.Until = w.clock.t.Add(3 * time.Hour)
	first := noteAtSetAside(t, w, &o, eligibleIssue)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	tried, killed := false, false
	say := o.Say
	o.Say = func(l string) {
		say(l)
		if strings.HasPrefix(l, "trying again the lanes set aside") {
			tried = true
		}
	}
	for i := range w.stages {
		run := w.stages[i].Run
		w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
			if tried && !killed && c.State.Key == eligibleIssue {
				killed = true
				cancel()
			}
			return run(c, l)
		}
	}
	res, err := RunTerminal(ctx, root, o)
	if err != nil {
		t.Fatalf("the drive: %v\n%s", err, w.said())
	}
	if !first() || !killed || res.Ended != TerminalKilled || res.Issue != eligibleIssue || res.Lapses == nil {
		t.Fatalf("premise: the lane set aside is taken up again and the drive killed in its stage, its hold live: %v %v %+v\n%s", first(), killed, res, w.said())
	}
	tip := tipOf(t, w.bare, eligibleIssue)
	if tip.released || claim.LeftToLapse(root, eligibleIssue, tip.commit) {
		t.Fatalf("premise: the take-up took the note off a live hold: %+v\n%s", tip, w.said())
	}
	if res.LeftToLapse {
		t.Fatalf("the end names a hold taken up again as left to lapse: %+v\n%s", res, w.said())
	}
}
