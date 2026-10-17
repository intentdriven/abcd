package loop

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A drain keeps its lane's lease every move, and the landing checks the
// reservation before its push (spc-2610161044461448 step 7;
// iss-2610161015393523; dcn-2610161040374343): a drain move, waiting or not,
// renews the lane in progress at half its lease though no step of the lane
// ran, and a lane whose reservation is now another person's pushes nothing,
// opens no pull request and is handed back.

// TestAWaitingDrainMoveRenewsTheLaneInProgress: the drain is paused by its
// pace window, the lane in progress is past half its lease and no step has
// run; the drain's move opens nothing and renews the lane's reservation.
func TestAWaitingDrainMoveRenewsTheLaneInProgress(t *testing.T) {
	t0 := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	c := &clock{t: t0}
	o := Options{Now: c.now}
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	ref := claim.RefPrefix + secondIssue
	before := originReservation(t, bare, ref)
	st, live, err := readDrain(repo.Root())
	if err != nil || !live {
		t.Fatalf("a live drain: %v", err)
	}
	until := t0.Add(4 * time.Hour)
	st.NextEligibleAt = &until
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	c.t = t0.Add(61 * time.Minute)
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Start != nil || !strings.Contains(res.Next, "nothing before") {
		t.Fatalf("the paused drain opens nothing: %+v", res)
	}
	after := originReservation(t, bare, ref)
	if after.commit == before.commit || !after.claimed.Equal(c.t) || after.holder != "octo" || after.released {
		t.Fatalf("the waiting move renews the lane's reservation: before %+v, after %+v", before, after)
	}
	if note := recordNote(t, repo, first.Start.RunID, StageReservationRenewed); !strings.Contains(note, "renewed the reservation on "+secondIssue) {
		t.Fatalf("the lane's run record logs the renewal: %q", note)
	}
}

// TestALaneWhoseReservationIsAnothersPushesNothingAndIsHandedBack: between the
// records and the push, the issue's reservation becomes Quinn's; the push step
// refuses, pushes nothing and opens no pull request, and the drain's next move
// hands the lane back at once.
func TestALaneWhoseReservationIsAnothersPushesNothingAndIsHandedBack(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	repo := f.repo
	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: f.issue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, repo, f.runID, f.stages, StageLand)
	f.step(t)
	f.step(t)
	l = currentLane(t, repo, f.runID)
	if l.Landing == nil || !l.Landing.RecordsDone || l.Landing.Pushed != "" {
		t.Fatalf("the lane is at its push: %+v", l.Landing)
	}
	preflighted(t, l, l.HeadSHA)

	// The reservation becomes Quinn's.
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatalf("Pat's release: %v", err)
	}
	q := otherCopy(t, f.bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"AGENTS.md"}}, claim.Options{}); err != nil {
		t.Fatalf("Quinn's reservation: %v", err)
	}

	_, err := advance(repo.Root(), f.runID, f.stages, Options{})
	r, ok := AsRefusal(err)
	if !ok || !strings.Contains(r.Reason, "quinn") || !strings.Contains(r.Reason, "does not push") {
		t.Fatalf("the push refuses on Quinn's reservation: %v", err)
	}
	if got := gitIn(t, f.bare, "for-each-ref", "--format=%(refname)", "refs/heads/"+l.Branch); got != "" {
		t.Fatalf("nothing is pushed: origin holds %s", got)
	}
	l = currentLane(t, repo, f.runID)
	if l.PR != 0 || l.Landing.Pushed != "" {
		t.Fatalf("no pull request is opened: %+v", l)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	route := routeOf(t, res.Routed, secondIssue)
	if route.Kind != claim.HandBackStuck || !strings.Contains(route.Reason, "quinn") {
		t.Fatalf("the drain hands the lane back at once: %+v", route)
	}
	if tip := originReservation(t, f.bare, claim.RefPrefix+secondIssue); tip.holder != "quinn" || tip.released {
		t.Fatalf("Quinn's reservation is left as it stands: %+v", tip)
	}
}
