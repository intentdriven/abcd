package loop

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A drain waits at the cap and when only held work is left
// (spc-2610161044461448 step 6; iss-2610161015371794, iss-2610161015422836;
// dcn-2610161040364312): at the holder's cap of live reservations, counted in
// the move's view before any reservation or judgement, or when every eligible
// issue left is held by another, the move asks nothing, writes waiting_until
// fifteen minutes on, apart from the pace window, names what it waits on and
// keeps its judgements; it stops as empty only when nothing eligible is left,
// held or not.

// TestADrainAtTheCapWaitsAndNamesThePullRequests: Pat holds ten live
// reservations, one of them a lane whose pull request waits to merge; the
// drain's next move asks no judgement, does not refuse, writes waiting_until
// fifteen minutes on without touching the pace window, names the pull
// request and advises waiting for a merge rather than releasing one.
func TestADrainAtTheCapWaitsAndNamesThePullRequests(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	f.parkIssueLane(t)
	for i := 1; i <= claim.MaxLive-1; i++ {
		id := fmt.Sprintf("iss-26093000000002%02d", i)
		if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: id, Footprint: []string{fmt.Sprintf("other/f%d.go", i)}}, claim.Options{}); err != nil {
			t.Fatalf("Pat's reservation %d: %v", i, err)
		}
	}
	before, _, err := readDrain(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	from := time.Now().UTC().Truncate(time.Second)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a drain at the cap does not refuse: %v", err)
	}
	to := time.Now().UTC()
	if res.Judging != nil || res.Start != nil || res.Complete || res.Stopped != "" {
		t.Fatalf("at the cap the move asks no judgement, opens nothing and does not end: %+v", res)
	}
	if res.WaitingUntil == nil || res.WaitingUntil.Before(from.Add(15*time.Minute)) || res.WaitingUntil.After(to.Add(15*time.Minute)) {
		t.Fatalf("the move writes waiting_until fifteen minutes on: %v", res.WaitingUntil)
	}
	st, _, err := readDrain(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.WaitingUntil == nil || !st.WaitingUntil.Equal(*res.WaitingUntil) || st.Waiting != DrainWaitCap {
		t.Fatalf("the drain's state keeps the wait: %v %q", st.WaitingUntil, st.Waiting)
	}
	if st.NextEligibleAt != nil || !st.WindowStartedAt.Equal(*before.WindowStartedAt) {
		t.Fatalf("the wait leaves the pace window as it was: %v %v", st.NextEligibleAt, st.WindowStartedAt)
	}
	for _, want := range []string{"10 live reservations", "#7", secondIssue, "wait for one of their pull requests to merge"} {
		if !strings.Contains(res.Next, want) {
			t.Fatalf("next names %q: %s", want, res.Next)
		}
	}
	if len(st.Judgements) != len(before.Judgements) || len(st.Judgements) == 0 {
		t.Fatalf("the drain keeps its judgements: %d, before %d", len(st.Judgements), len(before.Judgements))
	}
}

// TestADrainWithOnlyHeldWorkLeftWaitsAndKeepsItsJudgements: once Pat's lane
// is handed back and the one eligible issue left is Quinn's, Pat's drain does
// not stop: it waits, says when it looks again and keeps every judgement;
// once Quinn releases the issue, the next move asks the judgement over it,
// and once that issue is handed back too, nothing eligible is left and the
// drain stops as empty.
func TestADrainWithOnlyHeldWorkLeftWaitsAndKeepsItsJudgements(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	q := otherCopy(t, bare, quinnEmail)
	quinn := claim.Options{Now: c.now}
	if _, err := claim.Reserve(q, claim.Request{Record: eligibleIssue, Footprint: []string{"AGENTS.md"}}, quinn); err != nil {
		t.Fatalf("Quinn reserves the bug: %v", err)
	}
	handBackLaneOf(t, repo, first.Start.RunID, o, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Complete || res.Stopped != "" || res.Judging != nil {
		t.Fatalf("with only held work left the drain neither ends nor asks: %+v", res)
	}
	if res.WaitingUntil == nil || !res.WaitingUntil.Equal(c.t.Add(15*time.Minute)) {
		t.Fatalf("the drain says when it looks again: %v", res.WaitingUntil)
	}
	if !strings.Contains(res.Next, eligibleIssue) || !strings.Contains(res.Next, "reserved by quinn") {
		t.Fatalf("next names the hold it waits on: %s", res.Next)
	}
	if len(res.Judgements) != 1 {
		t.Fatalf("the drain keeps its judgements: %+v", res.Judgements)
	}
	// Still held at a later move before the wait ends: the wait stands as written.
	c.t = c.t.Add(5 * time.Minute)
	again, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Complete || again.WaitingUntil == nil || !again.WaitingUntil.Equal(*res.WaitingUntil) {
		t.Fatalf("a move before the wait ends keeps it: %+v", again.WaitingUntil)
	}

	if _, err := claim.Reserve(q, claim.Request{Record: eligibleIssue, Release: true}, quinn); err != nil {
		t.Fatalf("Quinn releases the bug: %v", err)
	}
	c.t = c.t.Add(15 * time.Minute)
	asked, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if asked.Judging == nil || asked.Judging.Issue != eligibleIssue || asked.WaitingUntil != nil {
		t.Fatalf("once the hold is gone the drain asks the judgement over the issue and waits no more: %+v", asked)
	}
	done := judge(t, repo, o, asked, JudgementYes, HandBackTrustRule, "the fix turns on a trust rule")
	if !done.Complete || done.Stopped != DrainStoppedEmpty {
		t.Fatalf("with nothing eligible left, held or not, the drain stops as empty: %+v", done)
	}
}
