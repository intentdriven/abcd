package loop

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A released branch is deleted at the next sweep (adr-2610161940004937), so a
// pending hand-back can outlive the branch it was to be published over. The
// drain publishes it as the branch's first commit instead, which origin takes
// only while the branch is still absent.

// sweep deletes id's reservation branch on origin, as the reservation sweep
// does once the branch is spent.
func sweep(t *testing.T, bare, id string) {
	t.Helper()
	gitIn(t, bare, "update-ref", "-d", claim.RefPrefix+id)
}

// TestAPendingHandBackIsPublishedOverASweptBranch: a yes whose hold was
// renewed by hand keeps its hand-back pending; the person releases the issue
// by hand and the sweep deletes the released branch. The next move publishes
// the hand-back as the branch's first commit, and another clone's drain then
// passes the issue over, naming the hand-back.
func TestAPendingHandBackIsPublishedOverASweptBranch(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil || first.Judging == nil || first.Judging.Issue != secondIssue {
		t.Fatalf("the first move asks over %s: %+v %v", secondIssue, first.Judging, err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	if _, ok := pendingFor(next.PendingHandBacks, secondIssue); !ok {
		t.Fatalf("the yes keeps no pending hand-back: %+v", next.PendingHandBacks)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	sweep(t, bare, secondIssue)

	poll, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if kind, blob := tombstoneTrailers(t, bare, secondIssue); kind != HandBackTrustRule || blob != originBlob(t, bare, secondIssue) {
		t.Fatalf("origin's tip marks %q over %q; want the pending trust-rule hand-back published over the swept branch", kind, blob)
	}
	tomb := refOf(t, bare, secondIssue)
	if r := originReservation(t, bare, claim.RefPrefix+secondIssue); r.parent != "" {
		t.Fatalf("the hand-back is not the branch's first commit: %+v", r)
	}
	if _, ok := pendingFor(poll.PendingHandBacks, secondIssue); ok || !strings.Contains(poll.Next, "published the hand-back of "+secondIssue) {
		t.Fatalf("the poll still keeps the hand-back pending, or does not say it published it: %+v, %q", poll.PendingHandBacks, poll.Next)
	}

	q := otherCopy(t, bare, quinnEmail)
	var res DrainResult
	asHomeOf(t, q, func() { res, err = Drain(q, Options{}, DrainOptions{}) })
	if err != nil {
		t.Fatal(err)
	}
	if p := passedFor(t, res, secondIssue); p.Check != CheckReservation || !strings.Contains(p.Reason, secondIssue+" was handed back by "+heldBy(t, patEmail)+" as trust-rule") {
		t.Fatalf("Quinn's drain passes the hand-back over, naming it: %+v", p)
	}
	if got := refOf(t, bare, secondIssue); got != tomb {
		t.Fatalf("Quinn's drain pushed over the hand-back: %s, was %s", got, tomb)
	}
}

// TestAPendingHandBackOverASweptBranchSomeoneReservedStaysPending: the move
// read the branch absent, and Quinn reserves the issue before the publish
// fetches; the publish leaves Quinn's reservation as it stands and keeps the
// hand-back pending, for the release that ends Quinn's hold.
func TestAPendingHandBackOverASweptBranchSomeoneReservedStaysPending(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	blob := originBlob(t, bare, secondIssue)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	sweep(t, bare, secondIssue)
	// The move's own fetch prunes the swept branch from this checkout too.
	repo.Git("update-ref", "-d", claim.TrackingPrefix+secondIssue)
	now := time.Now()
	view, err := claim.View(repo.Root(), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := view.Lookup(secondIssue); ok {
		t.Fatalf("the view still reads the swept branch")
	}
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	quinns := refOf(t, bare, secondIssue)
	pending := DrainPendingHandBack{Issue: secondIssue, Kind: HandBackTrustRule, Blob: blob, Why: "the hold moved on", At: now}
	st := DrainState{PendingHandBacks: []DrainPendingHandBack{pending},
		HandBacks: []DrainRoute{{Issue: secondIssue, From: DrainFromJudgement, Wrote: "nothing: a flag in this summary"}}}

	if out := publishPending(repo.Root(), &st, &view, now); len(out) != 0 {
		t.Fatalf("the move ended the pending hand-back over Quinn's reservation: %q", out)
	}
	if _, ok := pendingFor(st.PendingHandBacks, secondIssue); !ok {
		t.Fatalf("the hand-back is no longer pending: %+v", st.PendingHandBacks)
	}
	if refOf(t, bare, secondIssue) != quinns {
		t.Fatal("the move pushed over Quinn's reservation")
	}
}

// TestAPendingHandBackWithNoBlobOverASweptBranchIsDropped: a hand-back judged
// while origin's default branch could not be read names no record blob; with
// its branch swept there is nothing to fence it on, so the move drops it,
// publishes nothing, and says why.
func TestAPendingHandBackWithNoBlobOverASweptBranchIsDropped(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	sweep(t, bare, secondIssue)
	// The move's own fetch prunes the swept branch from this checkout too.
	repo.Git("update-ref", "-d", claim.TrackingPrefix+secondIssue)
	now := time.Now()
	view, err := claim.View(repo.Root(), now)
	if err != nil {
		t.Fatal(err)
	}
	pending := DrainPendingHandBack{Issue: secondIssue, Kind: HandBackTrustRule, Why: "the hold moved on", At: now}
	st := DrainState{PendingHandBacks: []DrainPendingHandBack{pending},
		HandBacks: []DrainRoute{{Issue: secondIssue, From: DrainFromJudgement, Wrote: "nothing: a flag in this summary"}}}

	out := publishPending(repo.Root(), &st, &view, now)
	if len(out) != 1 || !strings.Contains(out[0], "dropped the hand-back of "+secondIssue) || !strings.Contains(out[0], "names no record blob") {
		t.Fatalf("the move says %q; want the blob-less hand-back dropped, and why", out)
	}
	if _, ok := pendingFor(st.PendingHandBacks, secondIssue); ok {
		t.Fatalf("the hand-back is still pending: %+v", st.PendingHandBacks)
	}
	if refOf(t, bare, secondIssue) != "" {
		t.Fatal("the move created the swept branch for a hand-back with no blob")
	}
	if r := routeOf(t, st.HandBacks, secondIssue); !strings.Contains(r.Wrote, "dropped the hand-back of "+secondIssue) {
		t.Fatalf("the route wrote %q; want the hand-back said dropped", r.Wrote)
	}
}
