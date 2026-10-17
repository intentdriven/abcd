package loop

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// `abcd drain --take-over` (spc-2610161044461448 step 15;
// iss-2610161015422306; dcn-2610161040398791): a hosted session that
// restarted lost its local tier, so its new copy reads its own reservations as
// another copy's. Without the flag the drain passes such an issue over and
// says the flag moves it; with it, the drain takes this person's reservation
// over from the lost copy as it reaches the issue, and never another
// person's.

// TestADrainTakesOverThisPersonsReservationFromAnotherCopy: P's lost copy
// holds the documentation issue the drain order takes first. A plain move
// passes it over naming `abcd drain --take-over`; a move with the flag
// reserves it for this copy, taking it over, and asks its judgement; the lane
// a no opens starts on that reservation.
func TestADrainTakesOverThisPersonsReservationFromAnotherCopy(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	lost := otherCopy(t, bare, patEmail)
	if _, err := claim.Reserve(lost, claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	held := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+secondIssue)

	plain, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(plain.Passed, func(e Excluded) bool { return e.ID == secondIssue })
	if i < 0 {
		t.Fatalf("without --take-over the issue your other copy holds is passed over: %+v", plain.Passed)
	}
	if why := plain.Passed[i].Reason; !strings.Contains(why, "abcd drain --take-over") {
		t.Fatalf("the pass-over names the flag the drain accepts: %q", why)
	}
	if got := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+secondIssue); got != held {
		t.Fatal("a plain move pushes nothing over your other copy's reservation")
	}

	took, err := Drain(repo.Root(), Options{TakeOver: true}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if took.Judging == nil || took.Judging.Issue != secondIssue || took.Judging.Hold == nil {
		t.Fatalf("with --take-over the drain reserves the issue and asks its judgement: %+v", took.Judging)
	}
	snap, err := claim.View(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r, ok := snap.Lookup(secondIssue)
	if !ok || r.Copy != claim.CopyThis || r.State != claim.StateLive || r.Holder != "octo" {
		t.Fatalf("the reservation is this copy's now: %+v", r)
	}
	res := judge(t, repo, Options{TakeOver: true}, took, JudgementNo, "", "an internal page fix")
	if res.Start == nil || res.Lane == nil || res.Lane.Issue != secondIssue {
		t.Fatalf("the no opens the lane on the reservation taken over: %+v", res)
	}
}

// TestADrainTakeOverNeverMovesAnotherPersonsReservation: --take-over moves
// this person's own reservation alone; another person's is still passed over,
// with nothing pushed.
func TestADrainTakeOverNeverMovesAnotherPersonsReservation(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	held := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+secondIssue)
	res, err := Drain(repo.Root(), Options{TakeOver: true}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(res.Passed, func(e Excluded) bool { return e.ID == secondIssue })
	if i < 0 || strings.Contains(res.Passed[i].Reason, "--take-over") {
		t.Fatalf("another person's reservation is passed over, naming no take-over: %+v", res.Passed)
	}
	if got := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+secondIssue); got != held {
		t.Fatal("nothing is pushed over another person's reservation")
	}
}
