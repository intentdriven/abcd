package loop

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// A drain's pace and sub-agents, and re-admission (spc-2610161044461448 step
// 16; iss-2610161015417715, iss-2610161015457468; dcn-2610161040399117): the
// pace a drain runs on reaches every lane it opens, a running drain takes a
// new one rather than refusing it, and an issue it handed back is looked at
// again once its record changes on origin.

// paceOf is the pace run id started on.
func paceOf(t *testing.T, repo interface{ Root() string }, id string) Pace {
	t.Helper()
	st, err := ReadState(repo.Root(), id)
	if err != nil || st.Pace == nil {
		t.Fatalf("run %s's pace: %+v %v", id, st.Pace, err)
	}
	return *st.Pace
}

func paceNumbers(p Pace) [4]int {
	return [4]int{p.WorkMinutes.Value, p.PauseMinutes.Value, p.SubAgents.Value, p.FixRounds.Value}
}

// TestEveryLaneOfADrainRunsOnTheDrainsPace: a drain begun with --pace 120/30
// --sub-agents 4 opens its second lane, in a move given no flag, on that pace
// and ceiling, not on what the configuration resolves.
func TestEveryLaneOfADrainRunsOnTheDrainsPace(t *testing.T) {
	repo := drainRepo(t)
	o := Options{Pace: strp("120/30"), SubAgents: strp("4")}
	first := drainToLane(t, repo, o, DrainOptions{})
	if got := paceNumbers(paceOf(t, repo, first.Start.RunID)); got != [4]int{120, 30, 4, BundledFixRounds} {
		t.Fatalf("the first lane runs on the drain's pace: %v", got)
	}
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackSecondPackage, Reason: "it reaches the site", Home: "the brief"})
	second := drainToLane(t, repo, Options{}, DrainOptions{})
	if second.Start == nil {
		t.Fatalf("the drain opens its second lane: %+v", second)
	}
	p := paceOf(t, repo, second.Start.RunID)
	if got := paceNumbers(p); got != [4]int{120, 30, 4, BundledFixRounds} {
		t.Fatalf("the second lane runs on the drain's pace, not the configuration's: %v", got)
	}
	if p.SubAgents.Origin != "--sub-agents 4" {
		t.Fatalf("the lane's pace names where the drain's came from: %+v", p.SubAgents)
	}
}

// TestARunningDrainTakesANewPaceAndAppliesIt: a running drain given --pace
// 60/15 --sub-agents 3 takes and records them, the lane in progress takes
// them for its next stage, and the next lane opens on them.
func TestARunningDrainTakesANewPaceAndAppliesIt(t *testing.T) {
	repo := drainRepo(t)
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	moved, err := Drain(repo.Root(), Options{Pace: strp("60/15"), SubAgents: strp("3")}, DrainOptions{})
	if err != nil {
		t.Fatalf("a running drain takes a new pace: %v", err)
	}
	if moved.Pace == nil || paceNumbers(*moved.Pace) != [4]int{60, 15, 3, BundledFixRounds} {
		t.Fatalf("the move reports the new pace: %+v", moved.Pace)
	}
	if !strings.Contains(moved.Next, "60/15 minutes, 3 sub-agents") {
		t.Fatalf("next says the drain took the new pace: %s", moved.Next)
	}
	st, _, err := readDrain(repo.Root())
	if err != nil || st.Pace == nil || paceNumbers(*st.Pace) != [4]int{60, 15, 3, BundledFixRounds} {
		t.Fatalf("the drain's state records the new pace: %+v %v", st.Pace, err)
	}
	if got := paceNumbers(paceOf(t, repo, first.Start.RunID)); got != [4]int{60, 15, 3, BundledFixRounds} {
		t.Fatalf("the lane in progress takes the new pace for its next stage: %v", got)
	}
	if note := recordNote(t, repo, first.Start.RunID, StagePace); !strings.Contains(note, "60/15") {
		t.Fatalf("the lane's run record says its pace changed: %q", note)
	}
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackSecondPackage, Reason: "it reaches the site", Home: "the brief"})
	second := drainToLane(t, repo, Options{}, DrainOptions{})
	if second.Start == nil || paceNumbers(paceOf(t, repo, second.Start.RunID)) != [4]int{60, 15, 3, BundledFixRounds} {
		t.Fatalf("the next lane opens on the new pace: %+v", second.Start)
	}
}

// TestAHandedBackIssueWhoseRecordChangedIsJudgedAgain: a yes hands the issue
// back; while its record stands the drain passes it by, and once a person
// changes the record on origin the drain asks the judgement over it again.
func TestAHandedBackIssueWhoseRecordChangedIsJudgedAgain(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	first, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil || first.Judging == nil || first.Judging.Issue != secondIssue {
		t.Fatalf("the drain asks about the documentation issue first: %+v %v", first.Judging, err)
	}
	c.t = c.t.Add(time.Minute)
	yes := judge(t, repo, o, first, JudgementYes, HandBackTrustRule, "it changes who may run the page")
	if r := routeOf(t, yes.HandBacks, secondIssue); r.Blob == "" {
		t.Fatalf("the hand-back records the record's blob on origin: %+v", r)
	}
	c.t = c.t.Add(time.Minute)
	still, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil || still.Judging == nil || still.Judging.Issue != eligibleIssue {
		t.Fatalf("while the record stands the drain moves on: %+v %v", still.Judging, err)
	}

	editOnOrigin(t, bare, secondIssue)
	c.t = c.t.Add(time.Minute)
	again, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Judging == nil || again.Judging.Issue != secondIssue {
		t.Fatalf("once its record changed on origin the issue is judged again: %+v passed %+v", again.Judging, again.Passed)
	}
}

// TestAFinishedHandedBackRunNoLongerPassesItsIssueOver: a lane hands its
// issue back; once a person changes the record on origin the drain judges it
// afresh, though the old run's directory remains and an old no stands over
// the same remedy, and its no opens a new run rather than resuming the old.
func TestAFinishedHandedBackRunNoLongerPassesItsIssueOver(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the documentation issue's lane opens: %+v", first)
	}
	handBackLaneOf(t, repo, first.Start.RunID, o, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	c.t = c.t.Add(time.Minute)
	routed, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	routeOf(t, routed.Routed, secondIssue)

	editOnOrigin(t, bare, secondIssue)
	c.t = c.t.Add(time.Minute)
	again, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(again.Passed, func(e Excluded) bool { return e.ID == secondIssue }) {
		t.Fatalf("the finished run no longer passes the issue over: %+v", again.Passed)
	}
	if again.Judging == nil || again.Judging.Issue != secondIssue {
		t.Fatalf("the issue is judged afresh, its old no notwithstanding: %+v", again.Judging)
	}
	c.t = c.t.Add(time.Minute)
	opened := judge(t, repo, o, again, JudgementNo, "", "an internal page fix")
	if opened.Start == nil || opened.Start.RunID == first.Start.RunID || opened.Lane.Issue != secondIssue {
		t.Fatalf("the no opens a new run for the issue: %+v", opened.Start)
	}
}

// TestADoneRunWithAnOpenPullRequestPassesItsIssueOverInANewDrain: a lane
// that landed its pull request, still open, is a run that is over but not
// handed back; a new drain on origin passes its issue over rather than
// judging it again and opening a second run and a second pull request.
func TestADoneRunWithAnOpenPullRequestPassesItsIssueOverInANewDrain(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	first := drainToLane(t, repo, o, DrainOptions{Max: 1})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the documentation issue's lane opens: %+v", first)
	}
	landLaneOf(t, repo, first.Start.RunID, 41)
	c.t = c.t.Add(time.Minute)
	ended, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil || ended.Stopped != DrainStoppedCap {
		t.Fatalf("the drain stops at its cap: %+v %v", ended.Stopped, err)
	}

	c.t = c.t.Add(time.Minute)
	again, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !again.Started {
		t.Fatalf("a new drain begins: %+v", again)
	}
	if again.Judging != nil && again.Judging.Issue == secondIssue {
		t.Fatalf("the issue whose pull request is open is judged again: %+v", again.Judging)
	}
	if e := passedFor(t, again, secondIssue); e.Check != CheckRun || !strings.Contains(e.Reason, first.Start.RunID) {
		t.Fatalf("the issue is passed over naming its run: %+v", e)
	}
	if again.Start != nil || runCount(t, repo) != 1 {
		t.Fatalf("no second run opens for the issue: %+v", again.Start)
	}
}

// readmittedRuns hands secondIssue's lane back, changes its record on origin
// and lets the drain open a new run for it: the handed-back run and the newer
// one, oldest first.
func readmittedRuns(t *testing.T) (repo *gittest.Repo, o Options, old, newer string) {
	t.Helper()
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o = Options{Now: c.now}
	first := drainToLane(t, repo, o, DrainOptions{})
	handBackLaneOf(t, repo, first.Start.RunID, o, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	c.t = c.t.Add(time.Minute)
	if _, err := Drain(repo.Root(), o, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	editOnOrigin(t, bare, secondIssue)
	c.t = c.t.Add(time.Minute)
	again, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	c.t = c.t.Add(time.Minute)
	opened := judge(t, repo, o, again, JudgementNo, "", "an internal page fix")
	if opened.Start == nil || opened.Start.RunID == first.Start.RunID {
		t.Fatalf("the no opens a new run for the issue: %+v", opened.Start)
	}
	return repo, o, first.Start.RunID, opened.Start.RunID
}

// TestBuildResumesTheNewerRunOfAReadmittedIssue: once a handed-back issue is
// re-admitted and a new run opened for it, `abcd build <iss>` resumes that
// newer run, not the one handed back.
func TestBuildResumesTheNewerRunOfAReadmittedIssue(t *testing.T) {
	repo, o, old, newer := readmittedRuns(t)
	res, err := Start(repo.Root(), secondIssue, o)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Resumed || res.RunID != newer {
		t.Fatalf("build resumes the newer run %s, not the handed-back %s: %+v", newer, old, res)
	}
}

// TestAStepWithNoRunNamedAddressesTheNewerRunOfAReadmittedIssue: a bare
// `abcd implement step` addresses the newer run of a re-admitted issue, the
// handed-back run it supersedes no longer counting as one in progress.
func TestAStepWithNoRunNamedAddressesTheNewerRunOfAReadmittedIssue(t *testing.T) {
	repo, _, old, newer := readmittedRuns(t)
	got, err := Resolve(repo.Root(), "")
	if err != nil {
		t.Fatalf("one run is in progress, the handed-back %s superseded: %v", old, err)
	}
	if got != newer {
		t.Fatalf("Resolve addresses %s, want the newer run %s", got, newer)
	}
}

// TestARecordReadWithNoRunNamedAddressesTheNewerRunOfAReadmittedIssue: so
// does a run record read with no run named.
func TestARecordReadWithNoRunNamedAddressesTheNewerRunOfAReadmittedIssue(t *testing.T) {
	repo, _, old, newer := readmittedRuns(t)
	got, err := LatestRun(repo.Root())
	if err != nil || got != newer {
		t.Fatalf("the record read addresses the newer run %s, not the handed-back %s: %q %v", newer, old, got, err)
	}
}
