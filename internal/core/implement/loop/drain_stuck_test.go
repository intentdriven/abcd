package loop

import (
	"fmt"
	"os"
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
// differ, at one stage, for sixty minutes hand the lane back as stuck; at
// thirty minutes the lane is still in progress.
func TestSixtyMinutesRefusingAtOneStageHandTheLaneBackAsStuck(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	stages := briefRefusing(func(lane string, n int) *Refusal {
		return refuse(string(StageBrief), "", lane, fmt.Sprintf("the brief cannot be rendered, try %d", n), "settle it")
	})
	refuseStep(t, repo, runID, stages, o)
	c.t = c.t.Add(30 * time.Minute)
	refuseStep(t, repo, runID, stages, o)
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Lane == nil || res.Lane.RunID != runID || len(res.Routed) != 0 {
		t.Fatalf("thirty minutes refusing leave the lane in progress: %+v", res)
	}
	c.t = c.t.Add(31 * time.Minute)
	refuseStep(t, repo, runID, stages, o)
	res, err = Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := wantStuck(t, res, bare, runID, "try 3")
	if !strings.Contains(r.Reason, "61 minutes") {
		t.Fatalf("the hand-back names how long the lane refused: %q", r.Reason)
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
