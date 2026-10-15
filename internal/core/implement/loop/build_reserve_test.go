package loop

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// `build next` and `build <itd-N>` honour another person's reservation
// (spc-2610031753359849 spec 2, step 12): the peers check reads the
// reservations as last fetched, from disk alone, and counts another copy's
// as a holder, naming it with the view's age; this copy's own is never one.

// onOrigin puts repo on a local bare origin, as P's checkout, and returns the
// bare repository's path.
func onOrigin(t *testing.T, repo *gittest.Repo) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	asPat(t, repo)
	return bare
}

// fetchedBy fetches the reservations into root, as `abcd peers` does, so its
// view holds what origin holds now.
func fetchedBy(t *testing.T, root string) {
	t.Helper()
	if _, err := claim.Fetch(root, claim.Options{}); err != nil {
		t.Fatalf("fetching the reservations: %v", err)
	}
}

// TestBuildNextExcludesAnotherPersonsReservationWithTheViewsAge is C38: an
// intent another person reserved, in the view as last fetched, is not a
// candidate; `build next` lists it under excluded at the peers check, naming
// the holder, the expiry and the view's age, and picks among the rest.
func TestBuildNextExcludesAnotherPersonsReservationWithTheViewsAge(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, gwt), pickSpec("11", fpSmall)},
	})
	bare := onOrigin(t, repo)
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/core/intent"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())

	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
		t.Fatalf("only the intent nobody else reserved is a candidate: %+v", set.Candidates)
	}
	i := slices.IndexFunc(set.Excluded, func(e Excluded) bool { return e.ID == "itd-10" })
	if i < 0 {
		t.Fatalf("Q's intent is excluded: %+v", set.Excluded)
	}
	e := set.Excluded[i]
	if e.Check != CheckPeers || !strings.Contains(e.Reason, "reserved by quinn until ") ||
		!strings.Contains(e.Reason, "as of the last fetch under a minute ago") {
		t.Fatalf("the exclusion names the holder, the expiry and the view's age at the peers check: %+v", e)
	}
}

// TestBuildRefusesAnotherPersonsReservationAtThePeersCheck is C39: `build
// <itd-N>` over an intent another person reserved, as last fetched, is
// refused at the peers check as contention, naming the holder, and writes
// nothing.
func TestBuildRefusesAnotherPersonsReservationAtThePeersCheck(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	bare := onOrigin(t, repo)
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())
	held := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", reservedRef)

	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Stage != "check" || r.Check != CheckPeers || !r.Contention {
		t.Fatalf("want the peers check as contention: %+v", r)
	}
	if !strings.Contains(r.Reason, "reserved by quinn until ") || !strings.Contains(r.Reason, "as of the last fetch") {
		t.Fatalf("the refusal names Q's reservation and the view's age: %q", r.Reason)
	}
	runTierAbsent(t, repo.Root())
	if got := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", reservedRef); got != held {
		t.Fatalf("Q's reservation moved: %s", got)
	}
}

// TestTheRunsOwnReservationIsNeverAHolder: this copy's own reservation on the
// intent is not a peer's, so the peers check passes over it; this person's
// other copy is a holder, named as theirs on another computer.
func TestTheRunsOwnReservationIsNeverAHolder(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	bare := onOrigin(t, repo)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	chk, err := Check(repo.Root(), "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if row := chk.Checks[len(chk.Checks)-1]; row.Name != CheckPeers || !row.OK {
		t.Fatalf("this copy's own reservation is no peer's: %+v", row)
	}

	// P's second computer takes it over once it lapses; P's view then
	// names it as P's other copy.
	p2 := otherCopy(t, bare, patEmail)
	if _, err := claim.Reserve(p2, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}, TakeOver: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())
	chk, err = Check(repo.Root(), "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	row := chk.Checks[len(chk.Checks)-1]
	if row.Name != CheckPeers || row.OK || !strings.Contains(row.Detail, "your copy on another computer") {
		t.Fatalf("P's other copy holds it, named as P's: %+v", row)
	}
}
