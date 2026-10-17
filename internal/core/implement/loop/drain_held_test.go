package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// Reserving before judging (spc-2610161044461448 step 2; iss-2610161015387318;
// dcn-2610161040368890): the drain reserves an issue before it asks any
// judgement over it, so the reservation push is the compare-and-swap between
// accounts; an issue or a footprint another copy holds in the move's view is
// passed over, naming the holder, with no push and no judgement; a yes
// releases the reservation taken before it, and a no opens the lane on it.

// countFetches makes every fetch repo makes from origin append a line to a
// file, through the upload-pack its remote names, and returns the count
// reader.
func countFetches(t *testing.T, repo *gittest.Repo) func() int {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "fetches")
	script := filepath.Join(dir, "upload-pack")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho fetch >> '"+log+"'\nexec git upload-pack \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "remote.origin.uploadpack", script)
	return func() int {
		data, err := os.ReadFile(log)
		if err != nil {
			return 0
		}
		return strings.Count(string(data), "fetch\n")
	}
}

// refOf is origin's reservation branch tip for id, "" when it has none.
func refOf(t *testing.T, bare, id string) string {
	t.Helper()
	return gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id)
}

// passedFor is the pass-over a move names for id.
func passedFor(t *testing.T, res DrainResult, id string) Excluded {
	t.Helper()
	i := slices.IndexFunc(res.Passed, func(e Excluded) bool { return e.ID == id })
	if i < 0 {
		t.Fatalf("the move passes over %s: %+v", id, res.Passed)
	}
	return res.Passed[i]
}

// TestAHeldIssueGetsNoPushAndNoJudgement: the documentation issue, which the
// order takes first, is reserved by Q; the move passes it over naming Q, asks
// no judgement over it and pushes nothing over Q's reservation, and reserves
// and asks over the next issue instead.
func TestAHeldIssueGetsNoPushAndNoJudgement(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"docs"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	held := refOf(t, bare, secondIssue)
	pushes := countPushes(t, repo)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// An unchanged ref alone would also be what a refused push leaves: the
	// move pushes once, the next issue's reservation, and nothing over Q's.
	if n := pushes(); n != 1 {
		t.Fatalf("the move pushed %d time(s); want one, the next issue's reservation", n)
	}
	if p := passedFor(t, res, secondIssue); p.Check != CheckReservation || !strings.Contains(p.Reason, secondIssue+" is reserved by quinn") ||
		!strings.Contains(p.Reason, "no judgement") {
		t.Fatalf("the pass-over names Q and says nothing was asked: %+v", p)
	}
	if got := refOf(t, bare, secondIssue); got != held {
		t.Fatalf("Q's reservation moved: %s, was %s", got, held)
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("the judgement is asked over the next issue, not the held one: %+v", res.Judging)
	}
	if fp := footprintOf(t, bare, eligibleIssue); !slices.Equal(fp, []string{"AGENTS.md"}) {
		t.Fatalf("the next issue is reserved before its judgement: %v", fp)
	}
}

// TestAHeldFootprintGetsNoPushAndNoJudgement: an issue whose found_at lies in
// another person's reservation of another record is passed over the same way,
// naming the record and its holder.
func TestAHeldFootprintGetsNoPushAndNoJudgement(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: "iss-2609300000000999", Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p := passedFor(t, res, secondIssue); p.Check != CheckReservation ||
		!strings.HasPrefix(p.Reason, `"commands/drain.md" is in the footprint of iss-2609300000000999, reserved by quinn`) {
		t.Fatalf("the pass-over names the footprint's record and holder: %+v", p)
	}
	if refOf(t, bare, secondIssue) != "" {
		t.Fatal("an issue whose footprint is held was reserved")
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("the judgement is asked over the next issue: %+v", res.Judging)
	}
}

// TestAJudgementYesReleasesTheReservationTakenBeforeIt: the first move
// reserves the issue it asks over, from this copy; a yes routes the issue and
// pushes a tombstone over that reservation, and the route says so.
func TestAJudgementYesReleasesTheReservationTakenBeforeIt(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Judging == nil || first.Judging.Issue != secondIssue {
		t.Fatalf("the move asks over the documentation issue: %+v", first.Judging)
	}
	if r := originReservation(t, bare, claim.RefPrefix+secondIssue); r.commit == "" || r.released || r.holder != "octo" {
		t.Fatalf("the issue is reserved, as P, before its judgement is asked: %+v", r)
	}
	if runCount(t, repo) != 0 {
		t.Fatal("a run was started before the judgement")
	}
	next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	if r := originReservation(t, bare, claim.RefPrefix+secondIssue); !r.released {
		t.Fatalf("a yes releases the reservation taken before it: %+v", r)
	}
	if r := routeOf(t, next.HandBacks, secondIssue); !strings.Contains(r.Wrote, "released the reservation on "+secondIssue) {
		t.Fatalf("the route says the reservation was released: %q", r.Wrote)
	}
}

// TestAJudgementNoOpensTheLaneOnTheReservationTakenBeforeIt: a no opens the
// lane on the reservation the drain took before it asked, with no second push,
// and the run records that reservation, so its steps renew and fence it.
func TestAJudgementNoOpensTheLaneOnTheReservationTakenBeforeIt(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	taken := refOf(t, bare, secondIssue)
	if taken == "" {
		t.Fatal("the issue is reserved before its judgement is asked")
	}
	next := judge(t, repo, Options{}, first, JudgementNo, "", "the remedy changes neither")
	if next.Start == nil || next.Lane == nil || next.Lane.Issue != secondIssue {
		t.Fatalf("a no opens the issue's lane: %+v", next)
	}
	if got := refOf(t, bare, secondIssue); got != taken {
		t.Fatalf("the lane pushed again over the reservation it opened on: %s, was %s", got, taken)
	}
	st, err := ReadState(repo.Root(), next.Start.RunID)
	if err != nil || st.ReservedAt == nil {
		t.Fatalf("the run records the reservation it opened on: %+v %v", st.ReservedAt, err)
	}
	if note := recordNote(t, repo, next.Start.RunID, StageReservation); !strings.Contains(note, secondIssue) || !strings.Contains(note, "before its judgement") {
		t.Fatalf("the run record names the reservation taken before the judgement: %q", note)
	}
}

// TestPassingOverHeldIssuesAddsNoFetch: with a no stored over an issue Q has
// since reserved, and the only other issue Q's too, the move fetches once, its
// plan's fetch: neither held issue is sent to a reservation act, which would
// fetch again, nor judged.
func TestPassingOverHeldIssuesAddsNoFetch(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// This copy's reservation goes, and Q takes both issues.
	_, _ = claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{})
	q := otherCopy(t, bare, quinnEmail)
	for id, fp := range map[string]string{secondIssue: "commands/drain.md", eligibleIssue: "AGENTS.md"} {
		if _, err := claim.Reserve(q, claim.Request{Record: id, Footprint: []string{fp}}, claim.Options{}); err != nil {
			t.Fatal(err)
		}
	}
	fetches := countFetches(t, repo)
	next := judge(t, repo, Options{}, first, JudgementNo, "", "the remedy changes neither")
	if n := fetches(); n != 1 {
		t.Fatalf("the move fetched %d time(s); passing over held issues adds no fetch to the plan's one", n)
	}
	passedFor(t, next, secondIssue)
	passedFor(t, next, eligibleIssue)
	if next.Judging != nil || next.Start != nil || runCount(t, repo) != 0 {
		t.Fatalf("no held issue is judged or started: %+v", next)
	}
}
