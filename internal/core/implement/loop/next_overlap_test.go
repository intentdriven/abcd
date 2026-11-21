package loop

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/statusblock"
	"github.com/intentdriven/abcd/internal/gittest"
)

// `build next` passes over an intent whose start-time footprint another
// copy's live reservation of another record shares a path with, as last
// fetched (iss-2611182334466021): its start would be refused at the
// reservation, so the pick never lands on it and moves on to the next
// candidate. The overlap is the start's own: the footprint runFootprint names
// with no step built, judged by claim's one overlap rule (Snapshot.Contention).

// heldIssue is the record another copy's reservation holds in these tests: an
// issue that is no planned intent, so only its footprint can hold one off.
const heldIssue = "iss-2611010000000001"

// overlapRepo is two planned intents on origin: itd-10, the readier by score,
// footprinted on internal/core/intent, and itd-11 on internal/other.
func overlapRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, "- it works\n"), pickSpec("11", "- packages: internal/other\n- tests: the other\n")},
	})
	return repo, onOrigin(t, repo)
}

// quinnHolds reserves id from a copy of Q's with footprint fp, under o's
// clock, and returns that copy.
func quinnHolds(t *testing.T, bare, id string, o claim.Options, fp ...string) string {
	t.Helper()
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: id, Footprint: fp, Lease: time.Hour}, as(t, q, o)); err != nil {
		t.Fatalf("Q reserving %s: %v", id, err)
	}
	return q
}

// excludedAt is the exclusion of id in set, and whether there is one.
func excludedAt(set CandidateSet, id string) (Excluded, bool) {
	i := slices.IndexFunc(set.Excluded, func(e Excluded) bool { return e.ID == id })
	if i < 0 {
		return Excluded{}, false
	}
	return set.Excluded[i], true
}

// TestBuildNextPassesOverAnIntentAnotherCopysFootprintHolds: Q holds a file
// inside itd-10's footprint package under another record; itd-10 is excluded
// at the check `overlap`, its reason naming the path, the record holding it
// and the hold's expiry, and the pick starts itd-11 instead.
func TestBuildNextPassesOverAnIntentAnotherCopysFootprintHolds(t *testing.T) {
	repo, bare := overlapRepo(t)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/core/intent/score.go")
	fetchedBy(t, repo.Root())

	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	e, ok := excludedAt(set, "itd-10")
	if !ok || e.Check != CheckOverlap {
		t.Fatalf("itd-10 is excluded at the overlap check: %+v", set.Excluded)
	}
	for _, want := range []string{`"internal/core/intent/score.go"`, "in the footprint of " + heldIssue, "reserved by " + heldBy(t, quinnEmail) + " until "} {
		if !strings.Contains(e.Reason, want) {
			t.Fatalf("the exclusion names %q: %q", want, e.Reason)
		}
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
		t.Fatalf("only itd-11 is a candidate: %+v", set.Candidates)
	}
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatalf("the pick moves on to the next candidate: %v", err)
	}
	if res.Pick == nil || res.Pick.Chosen.ID != "itd-11" {
		t.Fatalf("the pick takes itd-11: %+v", res.Pick)
	}
	if fp := footprintOf(t, bare, "itd-11"); !slices.Equal(fp, []string{"internal/other"}) {
		t.Fatalf("the picked intent is reserved: %v", fp)
	}
}

// TestAnExpiredOrReleasedReservationExcludesNothing: a hold that has expired,
// or that its holder released, on a path in a candidate's footprint holds
// nothing off; only a live one does. itd-10 shares a path with Q's live hold
// and is passed over; itd-11 shares one with a hold no longer live, and is
// the pick.
func TestAnExpiredOrReleasedReservationExcludesNothing(t *testing.T) {
	for _, lapse := range []string{"expired", "released"} {
		t.Run(lapse, func(t *testing.T) {
			repo, bare := overlapRepo(t)
			quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/core/intent")
			const gone = "iss-2611010000000002"
			switch lapse {
			case "expired":
				quinnHolds(t, bare, gone, clockAt(time.Now().Add(-3*time.Hour)), "internal/other")
			case "released":
				q := quinnHolds(t, bare, gone, claim.Options{}, "internal/other")
				if _, err := claim.Reserve(q, claim.Request{Record: gone, Release: true}, as(t, q, claim.Options{})); err != nil {
					t.Fatalf("Q releasing %s: %v", gone, err)
				}
			}
			fetchedBy(t, repo.Root())

			set, err := candidates(repo.Root(), "")
			if err != nil {
				t.Fatal(err)
			}
			if e, ok := excludedAt(set, "itd-10"); !ok || e.Check != CheckOverlap {
				t.Fatalf("itd-10 is excluded at the overlap check: %+v", set.Excluded)
			}
			if _, ok := excludedAt(set, "itd-11"); ok || len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
				t.Fatalf("the %s hold excludes nothing, so itd-11 is the candidate: %+v", lapse, set)
			}
			res, err := Next(repo.Root(), Options{}, NextOptions{})
			if err != nil || res.Pick == nil || res.Pick.Chosen.ID != "itd-11" {
				t.Fatalf("the pick takes itd-11 past the %s hold: %+v %v", lapse, res.Pick, err)
			}
		})
	}
}

// TestThisCopysOwnReservationExcludesNothing: a reservation this copy holds
// on another record is this copy's own work, as the start's reservation reads
// it, so a path it shares with a candidate's footprint holds nothing off.
func TestThisCopysOwnReservationExcludesNothing(t *testing.T) {
	repo, bare := overlapRepo(t)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/core/intent")
	const own = "iss-2611010000000003"
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: own, Footprint: []string{"internal/other/x.go"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())

	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := excludedAt(set, "itd-10"); !ok || e.Check != CheckOverlap {
		t.Fatalf("itd-10 is excluded at the overlap check: %+v", set.Excluded)
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
		t.Fatalf("this copy's own hold excludes nothing, so itd-11 is the candidate: %+v", set)
	}
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil || res.Pick == nil || res.Pick.Chosen.ID != "itd-11" {
		t.Fatalf("the pick takes itd-11 beside this copy's own hold: %+v %v", res.Pick, err)
	}
}

// TestARefusedStartOfThePickNamesTheIntentPicked: Q's hold reached origin
// after this checkout last fetched, so the view passes itd-10 and the pick
// chooses it; the start's reservation, which fetches, backs off, and the
// refusal names itd-10 beside the path and its holder.
func TestARefusedStartOfThePickNamesTheIntentPicked(t *testing.T) {
	repo, bare := overlapRepo(t)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/core/intent")

	_, err := Next(repo.Root(), Options{}, NextOptions{})
	r := mustRefusal(t, err)
	if r.Stage != StageReservation || !r.Contention {
		t.Fatalf("the start backs off at the reservation: %+v", r)
	}
	if !strings.Contains(r.Reason, "itd-10") || !strings.Contains(r.Reason, "in the footprint of "+heldIssue) {
		t.Fatalf("the refusal names the intent picked and the hold: %q", r.Reason)
	}
	runTierAbsent(t, repo.Root())
}

// TestTheStatusHeadPassesOverAnOverlapBuildNextExcludes: the board's next-up
// head follows the pick past an intent whose footprint another copy's live
// hold shares, so it names itd-11, the intent `build next` would start.
func TestTheStatusHeadPassesOverAnOverlapBuildNextExcludes(t *testing.T) {
	repo, bare := overlapRepo(t)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/core/intent")
	fetchedBy(t, repo.Root())

	b, err := statusblock.Read(repo.Root(), StatusLanes, StatusPeers)
	if err != nil {
		t.Fatal(err)
	}
	var head string
	for _, r := range b.Now {
		if r.NextUp {
			head = r.ID
		}
	}
	if head != "itd-11" {
		t.Fatalf("the head is itd-11, the intent the pick starts: %q in %+v", head, b.Now)
	}
}

// TestTheStatusHeadPassesOverAFootprintBuildNextExcludes: itd-10 is the readier
// but its spec names its package by prose, so build next excludes it at
// `footprint` and starts itd-11; the board's head is itd-11 too.
func TestTheStatusHeadPassesOverAFootprintBuildNextExcludes(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", "- packages: this spec (the live captures)\n- tests: the first\n")},
		"11": {pickIntent("11", "", settledQuestions, "- it works\n"), pickSpec("11", "- packages: internal/other\n- tests: the other\n")},
	})
	bare := onOrigin(t, repo)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/unrelated")
	fetchedBy(t, repo.Root())

	set, err := candidatesAt(repo.Root(), "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := excludedAt(set, "itd-10"); !ok || e.Check != CheckFootprint {
		t.Fatalf("build next excludes itd-10 at footprint: %+v", set.Excluded)
	}
	b, err := statusblock.Read(repo.Root(), StatusLanes, StatusPeers)
	if err != nil {
		t.Fatal(err)
	}
	var head string
	for _, r := range b.Now {
		if r.NextUp {
			head = r.ID
		}
	}
	if head != "itd-11" {
		t.Fatalf("the head is itd-11, the intent the pick starts: %q in %+v", head, b.Now)
	}
}
