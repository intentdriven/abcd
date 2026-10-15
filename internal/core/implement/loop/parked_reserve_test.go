package loop

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A lane parked at land holds its reservation while it waits on a person's
// merge (spc-2610031753359849 spec 2, the final review of 2026-10-14): every
// drain move renews a parked lane's reservation at half its lease, and a
// fence before a merge that waits on a person's approval renews for a day.

// drainAt makes one drain move at root with o and, when it awaits the host
// judgement, answers no, as drainToLane does for a fixture repository.
func drainAt(t *testing.T, root string, o Options) DrainResult {
	t.Helper()
	res, err := Drain(root, o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil {
		return res
	}
	data, err := json.Marshal(judgementAnswer(res.Judging, JudgementNo, "", "the remedy changes neither what a user sees nor a trust boundary"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(res.Judging.Answer))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Drain(root, o, DrainOptions{Judgement: path})
	if err != nil {
		t.Fatalf("the host's no on %s: %v", res.Judging.Issue, err)
	}
	return out
}

// TestADrainKeepsItsOpenPullRequestLanesHeld: a drain lane parked at land, its
// pull request left open for a person, is fenced for a day and then renewed by
// each drain move at half its lease, keeping the day, so across a wait of a
// day and a half it stays this copy's, and another account's drain passes the
// issue over rather than taking a lapsed tip.
func TestADrainKeepsItsOpenPullRequestLanesHeld(t *testing.T) {
	f := drainLandFixtureWith(t, unreviewedQueueRuleset("MERGE"))
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.parkIssueLane(t)
	if l.Landing.Armed {
		t.Fatalf("the pull request is left open for a person: %+v", l.Landing)
	}
	ref := claim.RefPrefix + secondIssue
	parked := originReservation(t, f.bare, ref)
	if parked.expires.Sub(parked.claimed) != 24*time.Hour {
		t.Fatalf("a lane left open for a person is fenced for a day: %+v", parked)
	}

	start := parked.claimed
	var at time.Time
	for _, after := range []time.Duration{12*time.Hour + 30*time.Minute, 25 * time.Hour, 37*time.Hour + 30*time.Minute} {
		at = start.Add(after)
		now := at
		drainAt(t, f.repo.Root(), Options{Now: func() time.Time { return now }})
		tip := originReservation(t, f.bare, ref)
		if tip.holder != "octo" || tip.released || tip.claimed.Before(at) || tip.expires.Sub(tip.claimed) != 24*time.Hour {
			t.Fatalf("the drain move at +%s renews the parked lane's reservation: %+v", after, tip)
		}
	}

	// Another account, its clock at the end of the wait.
	q := otherCopy(t, f.bare, quinnEmail)
	if err := os.WriteFile(filepath.Join(q, ".git", "info", "exclude"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"AGENTS.md"}},
		claim.Options{Now: func() time.Time { return at }}); !errors.Is(err, claim.ErrBackOff) {
		t.Fatalf("another account reserving the parked issue after a day and a half = %v; want the back-off", err)
	}
	qres := drainAt(t, q, Options{Now: func() time.Time { return at }})
	i := slices.IndexFunc(qres.Passed, func(e Excluded) bool { return e.ID == secondIssue })
	if i < 0 || qres.Passed[i].Check != CheckReservation || !strings.Contains(qres.Passed[i].Reason, "reserved by octo") {
		t.Fatalf("another account's drain passes the parked issue over: passed %+v, lane %+v", qres.Passed, qres.Lane)
	}
}

// TestTheFenceHoldsADayWhenTheMergeWaitsOnACodeOwner: a merge the ruleset
// gates behind a person's approval (here a code-owner review, with a
// CODEOWNERS file naming an owner) can wait overnight, so the fence renews the
// reservation with the longest lease, a day, not the queue's check timeout.
func TestTheFenceHoldsADayWhenTheMergeWaitsOnACodeOwner(t *testing.T) {
	ruleset := `{"bypass_actors":[],"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
		`"enforcement":"active","name":"main review","rules":[{"type":"deletion"},` +
		`{"parameters":{"required_approving_review_count":0,"require_code_owner_review":true},"type":"pull_request"},` +
		`{"parameters":{"merge_method":"MERGE","grouping_strategy":"ALLGREEN","check_response_timeout_minutes":90},"type":"merge_queue"}],"target":"branch"}` + "\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-review.json": ruleset,
		".github/CODEOWNERS":                   "* @octo\n",
	})
	f.toArm(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	f.step(t)
	tip := originReservation(t, f.bare, reservedRef)
	if tip.expires.Sub(tip.claimed) != 24*time.Hour {
		t.Fatalf("the fence renews for a day before a code owner's review: %s (%+v)", tip.expires.Sub(tip.claimed), tip)
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("the merge is armed behind the review: %+v", l.Landing)
	}
}

// TestAStepWhileAnImplementerWorksRenewsTheReservation: a long implementer
// stage keeps its reservation while the run is stepped, since every step
// renews at half the lease before it looks at the stage, even one that moves
// nothing. A run nobody steps for longer than its lease loses it.
func TestAStepWhileAnImplementerWorksRenewsTheReservation(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	stepTo(t, f.repo, f.runID, f.stages, StageImplement)
	if res, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-61*time.Minute), "internal/widget")
	before := originReservation(t, f.bare, reservedRef)
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		if r, ok := AsRefusal(err); !ok || !r.Contention {
			t.Fatalf("a step while the implementer works: %v", err)
		}
	}
	if after := originReservation(t, f.bare, reservedRef); after.parent != before.commit || after.claimed.Before(now) {
		t.Fatalf("the step renews while the implementer works: before %+v, after %+v", before, after)
	}
}

// codeOwnerRuleset gates the default branch behind a code owner's review and
// a merge queue.
const codeOwnerRuleset = `{"bypass_actors":[],"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
	`"enforcement":"active","name":"main review","rules":[{"type":"deletion"},` +
	`{"parameters":{"required_approving_review_count":0,"require_code_owner_review":true},"type":"pull_request"},` +
	`{"parameters":{"merge_method":"MERGE","grouping_strategy":"ALLGREEN","check_response_timeout_minutes":90},"type":"merge_queue"}],"target":"branch"}` + "\n"

// TestARenewalKeepsTheDayTheFenceGave: a drain move past half a day-long
// fence lease renews with the tip's own term, never the default, so a pull
// request waiting overnight on a code owner stays held a day from the renewal.
func TestARenewalKeepsTheDayTheFenceGave(t *testing.T) {
	f := drainLandFixtureWith(t, codeOwnerRuleset)
	f.repo.Write(".github/CODEOWNERS", "* @octo\n")
	f.repo.Commit("the code owners")
	f.repo.Git("push", "-q", "origin", "main")
	f.repo.Git("fetch", "-q", "origin")
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	f.armIssueLane(t)
	ref := claim.RefPrefix + secondIssue
	fenced := originReservation(t, f.bare, ref)
	if fenced.expires.Sub(fenced.claimed) != 24*time.Hour {
		t.Fatalf("the fence renews for a day behind a code owner: %+v", fenced)
	}
	at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)
	drainAt(t, f.repo.Root(), Options{Now: func() time.Time { return at }})
	tip := originReservation(t, f.bare, ref)
	if tip.parent != fenced.commit || tip.claimed.Before(at) || tip.expires.Sub(tip.claimed) != 24*time.Hour {
		t.Fatalf("the drain's renewal keeps the day: fenced %+v, renewed %+v (%s)", fenced, tip, tip.expires.Sub(tip.claimed))
	}
}

// TestTheFenceHoldsADayWhenThePullRequestIsLeftForAPerson: a pull request the
// landing leaves open for a person to merge, under a mirror with no queue or
// with a queue no approval gates, waits on that person as long as a review
// would, so the fence renews for a day there too. Only a checkout with no
// mirror at all keeps the default lease (C44, decision 29).
func TestTheFenceHoldsADayWhenThePullRequestIsLeftForAPerson(t *testing.T) {
	for name, ruleset := range map[string]string{
		"no merge queue":     noQueueRuleset,
		"a queue unreviewed": unreviewedQueueRuleset("MERGE"),
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, ruleset)
			f.toArm(t)
			now := time.Now().UTC().Truncate(time.Second)
			reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
			f.step(t)
			tip := originReservation(t, f.bare, reservedRef)
			if tip.expires.Sub(tip.claimed) != 24*time.Hour {
				t.Fatalf("the fence renews for a day before a person's merge: %s", tip.expires.Sub(tip.claimed))
			}
			if l := currentLane(t, f.repo, f.runID); l.Landing == nil || l.Landing.Armed || l.Landing.Merge == "" {
				t.Fatalf("the pull request is left open for a person: %+v", l.Landing)
			}
		})
	}
}

// TestAReceiptRenewsAtHalfLease: an implementer that runs past half the lease
// with no step between keeps the reservation, since handing its receipt back
// renews it as a step would.
func TestAReceiptRenewsAtHalfLease(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	stepTo(t, f.repo, f.runID, f.stages, StageImplement)
	if res, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-61*time.Minute), "internal/widget")
	before := originReservation(t, f.bare, reservedRef)
	l := currentLane(t, f.repo, f.runID)
	dir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := laneCommit(t, f.repo, l, "one.txt")
	if _, err := Receipt(f.repo.Root(), f.runID, writeReceipt(t, dir, goodReceipt(t, f.runID, l, dir, sha)), f.stages, Options{}); err != nil {
		t.Fatalf("the implementer's receipt: %v", err)
	}
	if after := originReservation(t, f.bare, reservedRef); after.parent != before.commit || after.claimed.Before(now) {
		t.Fatalf("the receipt renews the reservation: before %+v, after %+v", before, after)
	}
	if recordNote(t, f.repo, f.runID, StageReservationRenewed) == "" {
		t.Fatal("the run record logs the receipt's renewal")
	}
}

// TestAStepNamesAReservationAnotherHoldsNow: when the view shows the run's
// reservation held by another person or copy, a step says so in its answer
// and logs it once in the run record, rather than passing over it in silence.
func TestAStepNamesAReservationAnotherHoldsNow(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-130*time.Minute), "internal/widget")
	q := otherCopy(t, f.bare, quinnEmail)
	if res, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/gadget"}}, clockAt(now.Add(-time.Minute))); err != nil || res.Act != claim.ActTakenOver {
		t.Fatalf("Q takes the lapsed reservation over: %+v %v", res, err)
	}
	if _, err := claim.Fetch(f.repo.Root(), clockAt(now)); err != nil {
		t.Fatal(err)
	}
	res := f.step(t)
	if !strings.Contains(res.Next, "the reservation on itd-10 is held by quinn") {
		t.Fatalf("the step names who holds the reservation now: %q", res.Next)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationLost); !strings.Contains(note, "quinn") {
		t.Fatalf("the run record logs the lost reservation: %q", note)
	}
	f.step(t)
	if n := countStage(t, f, StageReservationLost); n != 1 {
		t.Fatalf("the lost reservation is logged once, not every step: %d entries", n)
	}
}

// countStage counts the run record's entries at stage.
func countStage(t *testing.T, f *landFixture, stage string) int {
	t.Helper()
	n := 0
	for _, s := range recordStages(t, f.repo, f.runID) {
		if s == stage {
			n++
		}
	}
	return n
}

// TestAFailingRenewalIsLoggedOnceWhileOriginIsDown: while origin does not
// answer, every step's renewal fails, and the run record logs it once until a
// renewal succeeds, so an offline spell does not grow the record a line per
// step; each step's answer still names it.
func TestAFailingRenewalIsLoggedOnceWhileOriginIsDown(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-61*time.Minute), "internal/widget")
	if err := os.Rename(f.bare, f.bare+".gone"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if res, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err == nil && !strings.Contains(res.Next, "could not renew") {
			t.Fatalf("each step names the failed renewal: %q", res.Next)
		}
	}
	if n := countStage(t, f, StageReservationRenewFailed); n != 1 {
		t.Fatalf("the failed renewal is logged once while origin is down: %d entries", n)
	}
}
