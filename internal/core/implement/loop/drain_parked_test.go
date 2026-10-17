package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// Parked lanes are read from their pull requests (spc-2610161044461448 step 8;
// iss-2610161015381750; dcn-2610161040380948): every drain move reads every
// lane parked at land in one forge listing, hands a closed pull request's lane
// back as closed with its reservation released, stops renewing a conflicting
// one and names it, and renews an open one and names it as waiting.

// stubPR is one pull request as the stub forge's listing answers it.
type stubPR struct {
	Number           int    `json:"number"`
	HeadRefName      string `json:"headRefName"`
	State            string `json:"state"`
	Mergeable        string `json:"mergeable"`
	MergeStateStatus string `json:"mergeStateStatus"`
}

// listPRs makes the stub forge list prs.
func (f *landFixture) listPRs(t *testing.T, prs ...stubPR) {
	t.Helper()
	data, err := json.Marshal(prs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.gh, "pr.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// parkedListings counts the parked-lane listings in the stub forge's log.
func (f *landFixture) parkedListings(t *testing.T) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(f.ghLog(t), "\n") {
		if strings.HasPrefix(line, "pr list") && strings.Contains(line, "--state all") {
			n++
		}
	}
	return n
}

// parkedOf is the parked reading of runID in res.
func parkedOf(t *testing.T, res DrainResult, runID string) DrainParked {
	t.Helper()
	i := slices.IndexFunc(res.Parked, func(p DrainParked) bool { return p.RunID == runID })
	if i < 0 {
		t.Fatalf("no parked reading of %s in %+v", runID, res.Parked)
	}
	return res.Parked[i]
}

// parkDrainLane opens the drain's documentation lane and parks it at land with
// its pull request left open for a person, the fence holding it a day.
func parkDrainLane(t *testing.T) (*landFixture, Lane) {
	t.Helper()
	f := drainLandFixtureWith(t, unreviewedQueueRuleset("MERGE"))
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	return f, f.parkIssueLane(t)
}

// TestEveryParkedLaneIsReadInOneForgeListing: two lanes parked at land, the
// drain's and a person's own build, are both read from one listing of the
// forge's pull requests, and each open one is named as waiting.
func TestEveryParkedLaneIsReadInOneForgeListing(t *testing.T) {
	f, drained := parkDrainLane(t)
	drainRun := f.runID

	// A person's own build of the other issue, parked at land too, with its
	// own pull request.
	for _, name := range []string{"pr.json", "body.md"} {
		_ = os.Remove(filepath.Join(f.gh, name))
	}
	start, err := Start(f.repo.Root(), eligibleIssue, Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	f.runID, f.issue = start.RunID, eligibleIssue
	f.parkIssueLane(t)
	if err := mutate(f.repo.Root(), start.RunID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].PR = 8
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	own := currentLane(t, f.repo, start.RunID)
	f.listPRs(t,
		stubPR{Number: 7, HeadRefName: drained.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"},
		stubPR{Number: 8, HeadRefName: own.Branch, State: "OPEN", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"})
	before := f.parkedListings(t)

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.parkedListings(t) - before; got != 1 {
		t.Fatalf("the move reads its parked lanes in one forge listing, not %d:\n%s", got, f.ghLog(t))
	}
	for _, run := range []string{drainRun, start.RunID} {
		if p := parkedOf(t, res, run); p.State != ParkedWaiting {
			t.Fatalf("an open pull request's lane is waiting: %+v", p)
		}
	}
	if !strings.Contains(res.Next, "#7") || !strings.Contains(res.Next, "#8") || !strings.Contains(res.Next, "waiting") {
		t.Fatalf("next names both parked lanes as waiting: %s", res.Next)
	}
}

// TestAClosedPullRequestReleasesAndHandsBackItsLane: a parked lane whose pull
// request the forge lists closed is handed back as closed, its reservation
// released with a tombstone marking it closed, and the drain routes the
// hand-back and records the lane as handed back.
func TestAClosedPullRequestReleasesAndHandsBackItsLane(t *testing.T) {
	f, l := parkDrainLane(t)
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "CLOSED", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"})

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedClosed || p.PR != 7 {
		t.Fatalf("the closed pull request's lane is read as closed: %+v", p)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if hb := st.Lanes[0].HandBack; st.Lanes[0].Stage != StageHandedBack || hb == nil || hb.Kind != claim.HandBackClosed {
		t.Fatalf("the run's lane is handed back as closed: %+v", st.Lanes[0])
	}
	if tomb := originReservation(t, f.bare, claim.RefPrefix+secondIssue); !tomb.released {
		t.Fatalf("the lane's reservation is released: %+v", tomb)
	}
	if kind, _ := tombstoneTrailers(t, f.bare, secondIssue); kind != claim.HandBackClosed {
		t.Fatalf("the tombstone marks the hand-back closed: %q", kind)
	}
	i := slices.IndexFunc(res.Lanes, func(dl DrainLane) bool { return dl.RunID == f.runID })
	if i < 0 || res.Lanes[i].Outcome != DrainLaneHandedBack {
		t.Fatalf("the drain records its lane as handed back: %+v", res.Lanes)
	}
	if r := routeOf(t, res.Routed, secondIssue); r.Kind != claim.HandBackClosed || r.Route != RouteHome || r.From != "lane" {
		t.Fatalf("the drain routes the closed lane's hand-back home: %+v", r)
	}
	if !strings.Contains(res.Next, "#7") || !strings.Contains(res.Next, "closed") {
		t.Fatalf("next names the closed pull request: %s", res.Next)
	}
	again, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(again.Parked, func(p DrainParked) bool { return p.RunID == f.runID }) ||
		slices.ContainsFunc(again.Routed, func(r DrainRoute) bool { return r.Issue == secondIssue }) {
		t.Fatalf("a lane handed back as closed is neither read nor routed again: %+v %+v", again.Parked, again.Routed)
	}
	// implement step --discard takes the closed lane, and closes nothing.
	if _, err := Discard(f.repo.Root(), f.runID, st.Lanes[0].ID, Options{}); err != nil {
		t.Fatalf("--discard accepts a lane handed back as closed: %v", err)
	}
	if strings.Contains(f.ghLog(t), "pr close") {
		t.Fatalf("the discard closes no pull request already closed:\n%s", f.ghLog(t))
	}
	if _, err := os.Stat(l.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the discard removes the closed lane's worktree: %v", err)
	}
}

// TestAConflictingPullRequestIsNotRenewedAndIsNamed: a parked lane whose pull
// request the forge lists conflicting is not renewed past half its lease, and
// next names it.
func TestAConflictingPullRequestIsNotRenewedAndIsNamed(t *testing.T) {
	f, l := parkDrainLane(t)
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "CONFLICTING", MergeStateStatus: "DIRTY"})
	ref := claim.RefPrefix + secondIssue
	fenced := originReservation(t, f.bare, ref)
	at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)

	res, err := Drain(f.repo.Root(), Options{Now: func() time.Time { return at }}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if tip := originReservation(t, f.bare, ref); tip.commit != fenced.commit {
		t.Fatalf("a conflicting pull request's lane is not renewed: fenced %+v, now %+v", fenced, tip)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedConflicting {
		t.Fatalf("the lane is read as conflicting: %+v", p)
	}
	if !strings.Contains(res.Next, "#7") || !strings.Contains(res.Next, "conflict") {
		t.Fatalf("next names the conflicting pull request: %s", res.Next)
	}
	if st, err := ReadState(f.repo.Root(), f.runID); err != nil || st.Lanes[0].HandBack != nil {
		t.Fatalf("a conflicting lane is not handed back by the drain: %+v %v", st.Lanes[0], err)
	}
}

// TestAnOpenParkedLaneIsRenewedAndNamedAsWaiting: a parked lane whose pull
// request is open and mergeable, and whose head is not yet on the default
// branch, is renewed past half its lease and named as waiting, not skipped.
func TestAnOpenParkedLaneIsRenewedAndNamedAsWaiting(t *testing.T) {
	f, l := parkDrainLane(t)
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
	ref := claim.RefPrefix + secondIssue
	fenced := originReservation(t, f.bare, ref)
	at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)

	res, err := Drain(f.repo.Root(), Options{Now: func() time.Time { return at }}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if tip := originReservation(t, f.bare, ref); tip.parent != fenced.commit || tip.released || tip.claimed.Before(at) {
		t.Fatalf("an open pull request's lane is renewed: fenced %+v, now %+v", fenced, tip)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedWaiting || p.PR != 7 {
		t.Fatalf("the lane is read as waiting: %+v", p)
	}
	if !strings.Contains(res.Next, "#7") || !strings.Contains(res.Next, "waiting") {
		t.Fatalf("next names the waiting pull request: %s", res.Next)
	}
}

// TestAParkedLaneTheForgeDoesNotAnswerForIsKeptHeld: only an answer the
// forge gave changes a parked lane (dcn-2610170728074867). A listing that
// fails, one that does not parse, one that does not show the lane's pull
// request, and one whose pull request of that number has another head branch
// each leave the lane unread: renewed past half its lease, not handed back,
// and named as kept held.
func TestAParkedLaneTheForgeDoesNotAnswerForIsKeptHeld(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, f *landFixture, l Lane)
		says  string
	}{
		{name: "a listing that fails", says: "could not be read from the forge", setup: func(t *testing.T, f *landFixture, _ Lane) {
			if err := os.WriteFile(filepath.Join(f.gh, "refuse-list"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "a listing that does not parse", says: "does not parse", setup: func(t *testing.T, f *landFixture, _ Lane) {
			if err := os.WriteFile(filepath.Join(f.gh, "pr.json"), []byte("not json"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "a pull request the listing does not show", says: "is not among the forge's listing", setup: func(t *testing.T, f *landFixture, _ Lane) {
			f.listPRs(t, stubPR{Number: 8, HeadRefName: "someone/else", State: "CLOSED"})
		}},
		{name: "a pull request with another head branch", says: "is not among the forge's listing", setup: func(t *testing.T, f *landFixture, _ Lane) {
			f.listPRs(t, stubPR{Number: 7, HeadRefName: "someone/else", State: "CLOSED"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := parkDrainLane(t)
			tc.setup(t, f, l)
			ref := claim.RefPrefix + secondIssue
			fenced := originReservation(t, f.bare, ref)
			at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)

			res, err := Drain(f.repo.Root(), Options{Now: func() time.Time { return at }}, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if p := parkedOf(t, res, f.runID); p.State != ParkedUnread || !strings.Contains(p.Note, tc.says) || !strings.Contains(p.Note, "kept held") {
				t.Fatalf("the lane is read as unread and kept held: %+v", p)
			}
			if tip := originReservation(t, f.bare, ref); tip.parent != fenced.commit || tip.released || tip.claimed.Before(at) {
				t.Fatalf("the lane's reservation is renewed: fenced %+v, now %+v", fenced, tip)
			}
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			if st.Lanes[0].HandBack != nil || st.Lanes[0].Stage != StageLand {
				t.Fatalf("the lane stays parked at land, not handed back: %+v", st.Lanes[0])
			}
		})
	}
}
