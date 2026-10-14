package loop

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The drain's reservation per issue (spc-2610031753359849 spec 2, step 14):
// in a checkout with a remote named origin, each issue is reserved as its lane
// starts, with its found_at path as the footprint (C37); an issue with no
// found_at path is handed back with the reason and the drain moves on.

// reservingDrainRepo is a drain repository on a local bare origin, P's
// checkout: the bug eligibleIssue names found_at, and the documentation
// issue, which the drain order takes first, names found_at only when
// docFoundAt is not "".
func reservingDrainRepo(t *testing.T, docFoundAt string) (*gittest.Repo, string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	issueRepoRule(t, repo)
	repo.Write("commands/drain.md", "# drain\n")
	fileIssueAt(t, repo, eligibleIssue, capture.SeverityMinor, capture.Category("bug"), issueRemedy, "AGENTS.md:3")
	fileIssueAt(t, repo, secondIssue, capture.SeverityMinor, capture.Category("documentation"), "Name the flag in the page.", docFoundAt)
	repo.Commit("the ledger")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	asPat(t, repo)
	return repo, bare
}

// footprintOf is the footprint origin's reservation on id names.
func footprintOf(t *testing.T, bare, id string) []string {
	t.Helper()
	sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id)
	if sha == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(gitIn(t, bare, "log", "-1", "--format=%B", sha), "\n") {
		if v, ok := strings.CutPrefix(line, "Abcd-Footprint: "); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestDrainReservesEachIssueWithItsFoundAt is C37: the drain reserves the
// issue whose lane it opens, as this person from this copy, with the path its
// found_at names (a line locator dropped) as the footprint, and the run it
// starts records the reservation, so `implement step` renews, fences and
// tombstones it.
func TestDrainReservesEachIssueWithItsFoundAt(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	if fp := footprintOf(t, bare, secondIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the reservation's footprint is the issue's found_at path: %v", fp)
	}
	if fp := footprintOf(t, bare, eligibleIssue); fp != nil {
		t.Fatalf("an issue whose lane has not opened is not reserved: %v", fp)
	}
	st, err := ReadState(repo.Root(), first.Start.RunID)
	if err != nil || st.ReservedAt == nil {
		t.Fatalf("the run records its reservation: %+v %v", st.ReservedAt, err)
	}
	if note := recordNote(t, repo, first.Start.RunID, StageReservation); !strings.Contains(note, "reserved "+secondIssue) {
		t.Fatalf("the run record says what it reserved: %q", note)
	}
	snap, err := claim.View(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Reservations) != 1 || snap.Reservations[0].Copy != claim.CopyThis || snap.Reservations[0].Holder != "octo" {
		t.Fatalf("the reservation is this copy's, P's: %+v", snap.Reservations)
	}

	// A line locator is dropped: the bug's found_at is AGENTS.md:3.
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	second := drainToLane(t, repo, Options{}, DrainOptions{})
	if second.Start == nil || second.Lane == nil || second.Lane.Issue != eligibleIssue {
		t.Fatalf("the drain moves to the bug's lane: %+v", second)
	}
	if fp := footprintOf(t, bare, eligibleIssue); !slices.Equal(fp, []string{"AGENTS.md"}) {
		t.Fatalf("the footprint is found_at's path without its locator: %v", fp)
	}
}

// TestADrainHandsBackAnIssueWithNoFoundAtAndMovesOn: an issue whose found_at
// names no repository-relative path cannot be reserved; the drain hands it
// back with the reason, asks no judgement over it, reserves nothing for it,
// and opens the next issue's lane.
func TestADrainHandsBackAnIssueWithNoFoundAtAndMovesOn(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "")
	res := drainToLane(t, repo, Options{}, DrainOptions{})
	r := routeOf(t, res.HandBacks, secondIssue)
	if want := secondIssue + " has no found_at path to reserve it with; handed back"; r.Reason != want || r.From != DrainFromReservation {
		t.Fatalf("the hand-back names the reason: %+v; want %q", r, want)
	}
	if !strings.Contains(DrainSummaryLine(r), "found_at") {
		t.Fatalf("the summary line names found_at: %s", DrainSummaryLine(r))
	}
	if fp := footprintOf(t, bare, secondIssue); fp != nil {
		t.Fatalf("an issue handed back is not reserved: %v", fp)
	}
	for _, j := range res.Judgements {
		if j.Issue == secondIssue {
			t.Fatalf("no judgement is asked over an issue handed back for its found_at: %+v", j)
		}
	}
	if res.Lane == nil || res.Lane.Issue != eligibleIssue {
		t.Fatalf("the drain moves on to the next issue: %+v", res.Lane)
	}
	if fp := footprintOf(t, bare, eligibleIssue); !slices.Equal(fp, []string{"AGENTS.md"}) {
		t.Fatalf("the next issue is reserved: %v", fp)
	}
}

// TestADrainPassesOverAnIssueAnotherPersonReserved: another person's live
// reservation is the back-off; the drain passes over that issue naming the
// holder, starts no run for it and pushes nothing over it.
func TestADrainPassesOverAnIssueAnotherPersonReserved(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "AGENTS.md")
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"docs"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	held := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+secondIssue)
	res := drainToLane(t, repo, Options{}, DrainOptions{})
	i := slices.IndexFunc(res.Passed, func(e Excluded) bool { return e.ID == secondIssue })
	if i < 0 || res.Passed[i].Check != CheckReservation || !strings.Contains(res.Passed[i].Reason, "is reserved by quinn") {
		t.Fatalf("the drain passes over Q's issue naming Q: %+v", res.Passed)
	}
	if got := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+secondIssue); got != held {
		t.Fatalf("Q's reservation moved: %s", got)
	}
	runs, err := Runs(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(runs, func(st State) bool { return st.Key == secondIssue }) {
		t.Fatal("a run was started for an issue another person holds")
	}
}

// TestADrainWithoutOriginReservesNothing: a checkout with no remote named
// origin has nowhere to reserve: the drain opens the lane as before, hands
// back no issue for want of a found_at, and the run records no reservation.
func TestADrainWithoutOriginReservesNothing(t *testing.T) {
	repo := drainRepo(t)
	res := drainToLane(t, repo, Options{}, DrainOptions{})
	if res.Start == nil || len(res.HandBacks) != 0 {
		t.Fatalf("the lane opens and nothing is handed back: %+v", res)
	}
	st, err := ReadState(repo.Root(), res.Start.RunID)
	if err != nil || st.ReservedAt != nil {
		t.Fatalf("the run holds no reservation: %+v %v", st.ReservedAt, err)
	}
}
