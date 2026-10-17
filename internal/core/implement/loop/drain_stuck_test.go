package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A drain lane that cannot finish hands itself back (spc-2610161044461448
// step 5; iss-2610161015368630; dcn-2610161040382067): each refused step of
// an issue run is logged on its lane, and a lane that has refused the same way
// three times, or has refused at one stage for sixty minutes, is handed back
// as stuck by the drain's next move, which releases its reservation with the
// tombstone and takes up the next issue. A wait by design is no refusal, and a
// refusal that cannot change hands the lane back at once.

// briefRefusing is DefaultStages with the brief stage's body replaced by one
// that refuses with refusal(n), n counting its calls from 1.
func briefRefusing(refusal func(lane string, n int) *Refusal) Stages {
	n := 0
	stages := DefaultStages()
	for i := range stages {
		if stages[i].Name == StageBrief {
			stages[i].Run = func(c Context, lane *Lane) (Outcome, error) {
				n++
				return Outcome{}, refusal(lane.ID, n)
			}
		}
	}
	return stages
}

// sameRefusal refuses the same way every time.
func sameRefusal(lane string, _ int) *Refusal {
	return refuse(string(StageBrief), "", lane, "the brief cannot be rendered: the record names no remedy", "write the remedy")
}

// stuckDrain is a reserving drain with its first lane, the documentation
// issue's, cut and at its brief, on the clock c.
func stuckDrain(t *testing.T, c *clock) (*gittest.Repo, string, string) {
	t.Helper()
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	o := Options{Now: c.now}
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	runID := first.Start.RunID
	if _, err := advance(repo.Root(), runID, DefaultStages(), o); err != nil {
		t.Fatalf("the worktree stage: %v", err)
	}
	return repo, bare, runID
}

// refuseStep advances runID through stages and wants the step refused.
func refuseStep(t *testing.T, repo *gittest.Repo, runID string, stages Stages, o Options) {
	t.Helper()
	if _, err := advance(repo.Root(), runID, stages, o); err == nil {
		t.Fatal("the step was not refused")
	}
}

// wantStuck wants res to have handed secondIssue's lane back as stuck,
// released its reservation with a stuck tombstone, and asked the judgement
// over the next issue in the same move.
func wantStuck(t *testing.T, res DrainResult, bare, runID, naming string) DrainRoute {
	t.Helper()
	r := routeOf(t, res.Routed, secondIssue)
	if r.From != "lane" || r.Kind != claim.HandBackStuck || r.Route != RouteHome || !strings.Contains(r.Reason, naming) {
		t.Fatalf("the lane is handed back as stuck, naming the refusal (%q): %+v", naming, r)
	}
	if !strings.Contains(r.Home, "--discard lane-1") || !strings.Contains(r.Home, runID) {
		t.Fatalf("the route names the discard of the stuck lane's run: %q", r.Home)
	}
	if !strings.Contains(r.Wrote, "released the reservation on "+secondIssue) {
		t.Fatalf("the route says the reservation was released: %q", r.Wrote)
	}
	if kind, _ := tombstoneTrailers(t, bare, secondIssue); kind != claim.HandBackStuck {
		t.Fatalf("the tombstone names the hand-back as stuck: %q", kind)
	}
	for _, l := range res.Lanes {
		if l.RunID == runID && l.Outcome != DrainLaneHandedBack {
			t.Fatalf("the stuck lane reads as handed back: %+v", l)
		}
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("the same move takes up the next eligible issue: %+v", res.Judging)
	}
	return r
}

// TestThreeIdenticalRefusalsHandTheLaneBackAsStuck: a lane whose step has
// refused the same way three times is handed back as stuck by the drain's
// next move, its reservation released with the tombstone, and the next issue
// is taken up in that move; two refusals are not enough.
func TestThreeIdenticalRefusalsHandTheLaneBackAsStuck(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	stages := briefRefusing(sameRefusal)
	refuseStep(t, repo, runID, stages, o)
	refuseStep(t, repo, runID, stages, o)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if r := st.Lanes[0].Refused; r == nil || r.Count != 2 || r.Stage != string(StageBrief) {
		t.Fatalf("each refused step is logged on its lane with its stage and a count: %+v", r)
	}
	if note := recordNote(t, repo, runID, StageRefused); !strings.Contains(note, "the record names no remedy") {
		t.Fatalf("the run record logs the refusal: %q", note)
	}
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Lane == nil || res.Lane.RunID != runID || len(res.Routed) != 0 {
		t.Fatalf("two identical refusals leave the lane in progress: %+v", res)
	}
	refuseStep(t, repo, runID, stages, o)
	res, err = Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	wantStuck(t, res, bare, runID, "the record names no remedy")
	st, err = ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if hb := st.Lanes[0].HandBack; st.Lanes[0].Stage != StageHandedBack || hb == nil || hb.Kind != claim.HandBackStuck {
		t.Fatalf("the run records the lane handed back as stuck: %+v", st.Lanes[0])
	}
}

// TestSixtyMinutesRefusingAtOneStageHandTheLaneBackAsStuck: refusals that
// differ, at one stage, coming every ten minutes for sixty minutes hand the
// lane back as stuck; at thirty minutes the lane is still in progress.
func TestSixtyMinutesRefusingAtOneStageHandTheLaneBackAsStuck(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	stages := briefRefusing(func(lane string, n int) *Refusal {
		return refuse(string(StageBrief), "", lane, fmt.Sprintf("the brief cannot be rendered, try %d", n), "settle it")
	})
	refuseStep(t, repo, runID, stages, o)
	for range 3 {
		c.t = c.t.Add(10 * time.Minute)
		refuseStep(t, repo, runID, stages, o)
	}
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Lane == nil || res.Lane.RunID != runID || len(res.Routed) != 0 {
		t.Fatalf("thirty minutes refusing leave the lane in progress: %+v", res)
	}
	for _, step := range []time.Duration{10, 10, 11} {
		c.t = c.t.Add(step * time.Minute)
		refuseStep(t, repo, runID, stages, o)
	}
	res, err = Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := wantStuck(t, res, bare, runID, "try 7")
	if !strings.Contains(r.Reason, "61 minutes") {
		t.Fatalf("the hand-back names how long the lane refused: %q", r.Reason)
	}
}

// TestOnlyTimeRefusalsKeptComingCountsTowardTheBound: a lane that refused
// once and then sat idle for an hour is not stuck, though a move came that
// renews it; a refusal more than fifteen minutes after the last starts the
// time afresh, so two refusals an hour apart with one more two minutes later
// are not an hour of refusing either.
func TestOnlyTimeRefusalsKeptComingCountsTowardTheBound(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, _, runID := stuckDrain(t, c)
	stages := briefRefusing(func(lane string, n int) *Refusal {
		return refuse(string(StageBrief), "", lane, fmt.Sprintf("the brief cannot be rendered, try %d", n), "settle it")
	})
	inProgress := func(what string) {
		t.Helper()
		res, err := Drain(repo.Root(), o, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Lane == nil || res.Lane.RunID != runID || len(res.Routed) != 0 {
			t.Fatalf("%s leaves the lane in progress: %+v", what, res)
		}
	}
	refuseStep(t, repo, runID, stages, o)
	c.t = c.t.Add(61 * time.Minute)
	inProgress("one refusal and an idle hour")
	refuseStep(t, repo, runID, stages, o)
	c.t = c.t.Add(2 * time.Minute)
	refuseStep(t, repo, runID, stages, o)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if r := st.Lanes[0].Refused; r == nil || !r.Since.Equal(c.t.Add(-2*time.Minute)) || !r.Last.Equal(c.t) {
		t.Fatalf("the refusal after the idle hour starts the time afresh: %+v", r)
	}
	inProgress("two minutes of refusing after an idle hour")
}

// TestAStepThatSucceedsBeforeTheHandBackKeepsTheLane: the drain reads a lane
// stuck, and before it takes the run's lock a step running at the same time
// succeeds; the hand-back, judged again under the lock, leaves the lane in
// progress and its reservation holding.
func TestAStepThatSucceedsBeforeTheHandBackKeepsTheLane(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	stages := briefRefusing(sameRefusal)
	for range 3 {
		refuseStep(t, repo, runID, stages, o)
	}
	read, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	b := stuckBounds{refusals: PaceValue{Value: BundledStuckRefusals}, minutes: PaceValue{Value: BundledStuckMinutes}}
	if stuckReason(read, b) == "" {
		t.Fatal("the drain's read finds the lane stuck")
	}
	if _, err := advance(repo.Root(), runID, DefaultStages(), o); err != nil {
		t.Fatalf("the step at the same time succeeds: %v", err)
	}
	st, err := handBackStuck(repo.Root(), runID, b, c.t)
	if err != nil {
		t.Fatal(err)
	}
	if l := st.Lanes[0]; l.HandBack != nil || l.Stage == StageHandedBack || l.Refused != nil {
		t.Fatalf("a lane whose step succeeded since the drain read it is not handed back: %+v", l)
	}
	if tip := originReservation(t, bare, claim.RefPrefix+secondIssue); tip.released {
		t.Fatalf("its reservation is not released: %+v", tip)
	}
}

// TestAWaitByDesignIsNotStuck: a lane that has waited ninety minutes for its
// preflight receipt, or through an outage, is not handed back as stuck: a
// wait is not a refusal.
func TestAWaitByDesignIsNotStuck(t *testing.T) {
	for _, tc := range []struct {
		name string
		wait func(lane string, since time.Time) *Refusal
	}{
		{"the preflight receipt", func(lane string, since time.Time) *Refusal {
			r := contend(string(StageLand), "", lane, checkWaitText(since)+": no preflight receipt names the lane's head", "run the preflight")
			r.checkWait = since
			return r
		}},
		{"an outage", func(lane string, _ time.Time) *Refusal {
			return netFailure(StageBrief, lane, "git fetch", fmt.Errorf("fatal: unable to access: Could not resolve host: github.com"))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
			c := &clock{t: start}
			o := Options{Now: c.now}
			repo, _, runID := stuckDrain(t, c)
			stages := briefRefusing(func(lane string, _ int) *Refusal { return tc.wait(lane, start) })
			for range 7 {
				refuseStep(t, repo, runID, stages, o)
				c.t = c.t.Add(15 * time.Minute)
			}
			st, err := ReadState(repo.Root(), runID)
			if err != nil {
				t.Fatal(err)
			}
			if st.Lanes[0].Refused != nil {
				t.Fatalf("a wait is logged as no refusal: %+v", st.Lanes[0].Refused)
			}
			res, err := Drain(repo.Root(), o, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Lane == nil || res.Lane.RunID != runID || len(res.Routed) != 0 {
				t.Fatalf("a lane waiting ninety minutes on %s is still in progress: %+v", tc.name, res)
			}
		})
	}
}

// TestABriefRefusedForAnIssueFinishedAtItsBaseHandsBackAtOnce: a lane whose
// issue is already resolved at the base its worktree was cut from is refused
// at its brief, and since that cannot change, the drain's next move hands it
// back as stuck at once, without waiting for the bound.
func TestABriefRefusedForAnIssueFinishedAtItsBaseHandsBackAtOnce(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	runID := first.Start.RunID
	resolveOnOrigin(t, bare, secondIssue)
	repo.Git("fetch", "-q", "origin")
	if _, err := advance(repo.Root(), runID, DefaultStages(), o); err != nil {
		t.Fatalf("the worktree stage, cut from origin's default branch: %v", err)
	}
	refuseStep(t, repo, runID, DefaultStages(), o)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if r := st.Lanes[0].Refused; r == nil || r.Count != 1 || r.Final == "" {
		t.Fatalf("a refusal that cannot change is logged as final: %+v", r)
	}
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := routeOf(t, res.Routed, secondIssue)
	if r.Kind != claim.HandBackStuck || !strings.Contains(r.Reason, "is not open at the lane's base") || !strings.Contains(r.Reason, "cannot change") {
		t.Fatalf("the lane is handed back at once as stuck, naming the refusal that cannot change: %+v", r)
	}
}

// TestDiscardAcceptsALaneHandedBackAsStuck: `implement step --discard` takes
// a lane handed back as stuck: its worktree and branch are removed, its stage
// is discarded and the run is over, so no run directory is removed by hand.
func TestDiscardAcceptsALaneHandedBackAsStuck(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, _, runID := stuckDrain(t, c)
	stages := briefRefusing(sameRefusal)
	for range 3 {
		refuseStep(t, repo, runID, stages, o)
	}
	if _, err := Drain(repo.Root(), o, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	wt := st.Lanes[0].Worktree
	if _, err := Discard(repo.Root(), runID, "lane-1", o); err != nil {
		t.Fatalf("--discard over a stuck lane: %v", err)
	}
	st, err = ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Stage != StageDiscarded || !st.Complete() {
		t.Fatalf("the stuck lane is discarded and its run is over: %+v", st.Lanes[0])
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Fatalf("the stuck lane's worktree is removed: %v", err)
	}
	if out := repo.Git("branch", "--list", st.Lanes[0].Branch); strings.TrimSpace(out) != "" {
		t.Fatalf("the stuck lane's branch is deleted: %q", out)
	}
}

// TestADiscardedStuckLaneLetsItsIssueBeTakenUpOnceItsRecordChanges: a lane
// handed back as stuck and then discarded, as the route says to, leaves a run
// that is over and was handed back, not one whose landing settles the issue:
// once a person changes the record on origin, the drain judges the issue
// afresh and its no opens a new run.
func TestADiscardedStuckLaneLetsItsIssueBeTakenUpOnceItsRecordChanges(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	stages := briefRefusing(sameRefusal)
	for range 3 {
		refuseStep(t, repo, runID, stages, o)
	}
	c.t = c.t.Add(time.Minute)
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	wantStuck(t, res, bare, runID, "the record names no remedy")
	if _, err := Discard(repo.Root(), runID, "lane-1", o); err != nil {
		t.Fatalf("--discard over a stuck lane: %v", err)
	}

	editOnOrigin(t, bare, secondIssue)
	c.t = c.t.Add(time.Minute)
	again, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range again.Passed {
		if e.ID == secondIssue {
			t.Fatalf("the discarded stuck run passes its issue over: %+v", e)
		}
	}
	if again.Judging == nil || again.Judging.Issue != secondIssue {
		t.Fatalf("once its record changed on origin the issue is judged afresh: %+v passed %+v next %q routed %+v stopped %q", again.Judging, again.Passed, again.Next, again.Routed, again.Stopped)
	}
	c.t = c.t.Add(time.Minute)
	opened := judge(t, repo, o, again, JudgementNo, "", "an internal page fix")
	if opened.Start == nil || opened.Start.RunID == runID || opened.Lane == nil || opened.Lane.Issue != secondIssue {
		t.Fatalf("the no opens a new lane for the issue: %+v", opened.Start)
	}
}

// TestAJudgementStampedInTheSecondOfItsHandBackIsStale: the no that opened a
// lane and the move that handed the lane back fall in one second (the clock
// never moves here). A judgement taken no later than the hand-back judged the
// record before it changed, so once it changes on origin the issue is judged
// afresh rather than reopened on the old no.
func TestAJudgementStampedInTheSecondOfItsHandBackIsStale(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	stages := briefRefusing(sameRefusal)
	for range 3 {
		refuseStep(t, repo, runID, stages, o)
	}
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	wantStuck(t, res, bare, runID, "the record names no remedy")
	if _, err := Discard(repo.Root(), runID, "lane-1", o); err != nil {
		t.Fatalf("--discard over a stuck lane: %v", err)
	}
	editOnOrigin(t, bare, secondIssue)
	again, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Judging == nil || again.Judging.Issue != secondIssue {
		t.Fatalf("an answer stamped in the hand-back's second is stale, so the issue is judged afresh: judging %+v start %+v next %q", again.Judging, again.Start, again.Next)
	}
}

// TestTheStuckBoundsAreReadFromThePacesLayers: the repository's
// pace.stuck_refusals and pace.stuck_minutes set the bounds.
func TestTheStuckBoundsAreReadFromThePacesLayers(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	repoConfig(t, repo, `{"pace": {"stuck_refusals": 1, "stuck_minutes": 600}}`)
	refuseStep(t, repo, runID, briefRefusing(sameRefusal), o)
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := wantStuck(t, res, bare, runID, "the record names no remedy")
	if !strings.Contains(r.Reason, "1 time") {
		t.Fatalf("the hand-back names the bound the repository set: %q", r.Reason)
	}
}

// TestALaneRefusedIsVersion13s: a lane's `refused` is state version 13's.
// Version 12 is its strict subset and reads as the current version, so a run
// a version-12 abcd wrote carries on; a file of version 12 or lower carrying
// it is refused naming the version and the key (12 and 11 here: this run
// holds a reservation, which version 10 never wrote).
func TestALaneRefusedIsVersion13s(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	repo, _, runID := stuckDrain(t, c)
	refuseStep(t, repo, runID, briefRefusing(sameRefusal), Options{Now: c.now})
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID)))
	current, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	as := func(version int, keepRefused bool) {
		t.Helper()
		if err := os.WriteFile(statePath, current, 0o600); err != nil {
			t.Fatal(err)
		}
		editState(t, repo.Root(), runID, func(doc map[string]any) {
			doc["schema_version"] = version
			if !keepRefused {
				delete(doc["lanes"].([]any)[0].(map[string]any), "refused")
			}
		})
	}
	if SchemaVersion != schemaVersionUnrefused+1 {
		t.Fatalf("the current version is the one after the version before `refused`: %d, %d", SchemaVersion, schemaVersionUnrefused)
	}
	for _, version := range []int{schemaVersionUnrefused, schemaVersionUnanswered} {
		as(version, true)
		_, err := ReadState(repo.Root(), runID)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, fmt.Sprintf("schema version %d", version)) || !strings.Contains(r.Reason, "`refused` on lane-1") {
			t.Errorf("version %d carrying `refused` is refused naming the version and the key: %+v", version, r)
		}
	}
	as(12, false)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatalf("a version-12 file reads: %v", err)
	}
	if st.SchemaVersion != 13 || st.Lanes[0].Refused != nil || st.Lanes[0].Stage != StageBrief {
		t.Fatalf("a version-12 file reads as version 13, its lane as it was: version %d, %+v", st.SchemaVersion, st.Lanes[0])
	}
	as(13, true)
	if st, err := ReadState(repo.Root(), runID); err != nil || st.Lanes[0].Refused == nil {
		t.Fatalf("a version-13 file reads with its lane's `refused`: %+v %v", st.Lanes, err)
	}
}
