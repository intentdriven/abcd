package loop

import (
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestBuildNextPassesOverAnIntentThisCopyHoldsUnderAnOldHolderID is
// iss-2611190011141331: this copy reserved a file inside itd-10's footprint
// package, then its home was emptied, so the holder id it reserved under is no
// longer the reader's own. The start's reservation backs off over that hold, as
// over another holder's, so `build next` excludes itd-10 at `overlap`, naming
// the path, the record and the old holder, and picks itd-11; starting itd-10
// itself is refused at the reservation alike.
func TestBuildNextPassesOverAnIntentThisCopyHoldsUnderAnOldHolderID(t *testing.T) {
	repo, _ := overlapRepo(t)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: heldIssue, Footprint: []string{"internal/core/intent/score.go"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	old := heldBy(t, patEmail)
	if err := os.Remove(abcdhome.Path(personHome(t, patEmail), "holder-id")); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())

	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	e, ok := excludedAt(set, "itd-10")
	if !ok || e.Check != CheckOverlap {
		t.Fatalf("itd-10 is excluded at the overlap check: %+v", set)
	}
	for _, want := range []string{`"internal/core/intent/score.go"`, "in the footprint of " + heldIssue, "reserved by " + old + " until "} {
		if !strings.Contains(e.Reason, want) {
			t.Fatalf("the exclusion names %q: %q", want, e.Reason)
		}
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
		t.Fatalf("only itd-11 is a candidate: %+v", set.Candidates)
	}

	_, err = Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageReservation || !r.Contention {
		t.Fatalf("the start backs off at the reservation: %+v", r)
	}
	if !strings.Contains(r.Reason, "in the footprint of "+heldIssue) || !strings.Contains(r.Reason, old) {
		t.Fatalf("the refusal names the hold and its old holder: %q", r.Reason)
	}
}

// TestBuildNextPassesNothingOverForThisCopysHoldUnderItsOwnHolderID: the same
// hold, under the holder id the reader still keeps, is this copy's own work,
// so itd-10 is no exclusion and is the pick, beside the hold.
func TestBuildNextPassesNothingOverForThisCopysHoldUnderItsOwnHolderID(t *testing.T) {
	repo, _ := overlapRepo(t)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: heldIssue, Footprint: []string{"internal/core/intent/score.go"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())

	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := excludedAt(set, "itd-10"); ok {
		t.Fatalf("this copy's own hold excludes nothing: %+v", e)
	}
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil || res.Pick == nil || res.Pick.Chosen.ID != "itd-10" {
		t.Fatalf("the pick takes itd-10 beside this copy's own hold: %+v %v", res.Pick, err)
	}
}
