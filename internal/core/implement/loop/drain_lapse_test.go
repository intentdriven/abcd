package loop

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/lease"
)

// A judgement request's bound (spc-2610161044461448 step 2): the drain takes
// its judgement hold once per request, so a host that polls without answering
// never renews it; a request whose hold lapses unanswered is dropped and the
// hold released, and the issue is asked again only after the drain's other
// eligible work.

// TestALapsedJudgementRequestIsDroppedAndAskedAgainOnlyAfterOtherWork: polls
// without an answer push nothing, even when the move's view cannot be read;
// once the hold has lapsed the request is dropped, the hold released, and the
// next issue asked instead; once that one is handed back, the lapsed issue is
// asked again on a new hold.
func TestALapsedJudgementRequestIsDroppedAndAskedAgainOnlyAfterOtherWork(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	// A window longer than the lease, so the pace never holds the moves.
	pace := "600/0"
	_, taken := askFirst(t, repo, bare, Options{Now: c.now, Pace: &pace}, DrainOptions{})
	pushes := countPushes(t, repo)

	c.t = c.t.Add(30 * time.Minute)
	was := drainView
	drainView = func(string, time.Time) (claim.Snapshot, error) { return claim.Snapshot{}, errors.New("unreadable") }
	poll, err := Drain(repo.Root(), o, DrainOptions{})
	drainView = was
	if err != nil {
		t.Fatal(err)
	}
	if poll.Judging == nil || poll.Judging.Issue != secondIssue || pushes() != 0 {
		t.Fatalf("a poll without an answer still awaits %s and pushes nothing: judging %+v, %d push(es)", secondIssue, poll.Judging, pushes())
	}
	heldStill(t, bare, secondIssue, taken)

	c.t = c.t.Add(lease.Default)
	lapsed, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	releasedOver(t, bare, secondIssue, taken)
	if lapsed.Judging == nil || lapsed.Judging.Issue != eligibleIssue {
		t.Fatalf("once the request lapses the next issue is asked: %+v", lapsed.Judging)
	}
	if !strings.Contains(lapsed.Next, secondIssue+" was not answered before") {
		t.Fatalf("the move says the request lapsed: %q", lapsed.Next)
	}
	if st, _, _ := readDrain(repo.Root()); len(st.Lapsed) != 1 || st.Lapsed[0].Issue != secondIssue {
		t.Fatalf("the drain records the lapsed request: %+v", st.Lapsed)
	}

	c.t = c.t.Add(claim.TakeoverMargin)
	again := judge(t, repo, o, lapsed, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	if again.Judging == nil || again.Judging.Issue != secondIssue || again.Judging.Hold == nil || again.Judging.Hold.Commit == taken {
		t.Fatalf("with the other work taken and the cool-down past, the lapsed issue is asked again on a new hold: %+v", again.Judging)
	}
	if st, _, _ := readDrain(repo.Root()); len(st.Lapsed) != 0 {
		t.Fatalf("an issue asked again is no longer lapsed, so it does not sort last for the rest of the drain: %+v", st.Lapsed)
	}
	judge(t, repo, o, again, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	if st, _, _ := readDrain(repo.Root()); len(st.Lapsed) != 0 {
		t.Fatalf("an answered issue is not lapsed: %+v", st.Lapsed)
	}
}

// TestALapsedIssueWaitsOutItsCoolDownBeforeItIsAskedAgain: the one eligible
// issue's request lapses; the move that drops it releases the hold and does
// not ask again, nor does any move before the cool-down has passed, so
// another account has a window to take the issue; the drain does not end
// meanwhile, and names when it asks again; once the cool-down has passed,
// the issue is asked again on a new hold and is no longer lapsed.
func TestALapsedIssueWaitsOutItsCoolDownBeforeItIsAskedAgain(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	onlyIssueEligible(t, bare)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	pace := "600/0"
	_, taken := askFirst(t, repo, bare, Options{Now: c.now, Pace: &pace}, DrainOptions{})
	o := Options{Now: c.now}

	c.t = c.t.Add(lease.Default)
	dropped := c.t
	for _, at := range []time.Time{dropped, dropped.Add(claim.TakeoverMargin - time.Second)} {
		c.t = at
		res, err := Drain(repo.Root(), o, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		releasedOver(t, bare, secondIssue, taken)
		if res.Judging != nil || res.Stopped != "" {
			t.Fatalf("at %s the lapsed issue is not asked again and the drain goes on: %+v", at, res)
		}
		if e := passedFor(t, res, secondIssue); e.Check != CheckLapsed {
			t.Fatalf("the lapsed issue is passed over for its cool-down: %+v", e)
		}
		if until := stamp(dropped.Add(claim.TakeoverMargin)); !strings.Contains(res.Next, until) {
			t.Fatalf("the move names when the issue is asked again (%s): %q", until, res.Next)
		}
	}
	c.t = dropped.Add(claim.TakeoverMargin)
	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judging.Issue != secondIssue || res.Judging.Hold == nil || res.Judging.Hold.Commit != refOf(t, bare, secondIssue) {
		t.Fatalf("once the cool-down has passed the issue is asked again on a new hold: %+v", res.Judging)
	}
	if st, _, _ := readDrain(repo.Root()); len(st.Lapsed) != 0 {
		t.Fatalf("the issue asked again is pruned from the lapsed requests: %+v", st.Lapsed)
	}
}

// onlyIssueEligible raises the drain fixture's bug issue past the rule's
// severities on origin, so the documentation issue is the only eligible work.
func onlyIssueEligible(t *testing.T, bare string) {
	t.Helper()
	pushFromOtherCopy(t, bare, "raise the bug", func(dir string) { setSeverity(t, dir, eligibleIssue, "major") })
}

// nothingLive fails when origin holds any reservation still live at now.
func nothingLive(t *testing.T, bare string, now time.Time) {
	t.Helper()
	for _, ref := range strings.Fields(gitIn(t, bare, "for-each-ref", "--format=%(refname)", claim.RefPrefix)) {
		if r := originReservation(t, bare, ref); !r.released && now.Before(r.expires) {
			t.Fatalf("origin still holds %s live: %+v", ref, r)
		}
	}
}

// pollUntilAsked polls the drain at step until it awaits a judgement again,
// at most n times, and returns that move.
func pollUntilAsked(t *testing.T, repo string, c *clock, step time.Duration, n int) DrainResult {
	t.Helper()
	for range n {
		c.t = c.t.Add(step)
		res, err := Drain(repo, Options{Now: c.now}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Judging != nil {
			return res
		}
	}
	t.Fatalf("the drain never asked again in %d poll(s)", n)
	return DrainResult{}
}

// TestALapsedOnlyIssueIsReleasedThenAskedAgainOnTheDrainsOwnHold: the one
// eligible issue's request lapses unanswered; the drain releases its lapsed
// reservation by a tombstone over that commit before anything else in the
// move, asks again on a reservation it records as its own, and a yes then
// releases that one, leaving nothing live.
func TestALapsedOnlyIssueIsReleasedThenAskedAgainOnTheDrainsOwnHold(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	onlyIssueEligible(t, bare)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	pace := "600/0"
	_, taken := askFirst(t, repo, bare, Options{Now: c.now, Pace: &pace}, DrainOptions{})
	updates := refUpdates(t, bare)

	c.t = c.t.Add(lease.Default)
	if _, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	again := pollUntilAsked(t, repo.Root(), c, lease.Skew, 3)
	pushed := updates(claim.RefPrefix + secondIssue)
	if len(pushed) == 0 {
		t.Fatal("the lapse pushed nothing")
	}
	if r := reservationCommit(t, bare, pushed[0]); !r.released || r.parent != taken {
		t.Fatalf("the lapse's first push = %+v; want a tombstone over the lapsed %s", r, taken)
	}
	h := again.Judging.Hold
	if again.Judging.Issue != secondIssue || h == nil || h.Commit != refOf(t, bare, secondIssue) {
		t.Fatalf("the issue is asked again on the drain's own hold, the tip: %+v", again.Judging)
	}
	judge(t, repo, Options{Now: c.now}, again, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	releasedOver(t, bare, secondIssue, h.Commit)
	nothingLive(t, bare, c.t)
}

// TestPollingWithoutAnAnswerNeverHoldsTheIssuePastOneLease: the one eligible
// issue is polled every four minutes, unanswered, across four leases; every
// reservation of it origin shows live is the hold the awaited judgement
// records, none outlives one lease, each is released by the poll that finds it
// lapsed, and the issue is asked again on a new hold only once a cool-down of
// at least the take-over margin has passed since that release.
func TestPollingWithoutAnAnswerNeverHoldsTheIssuePastOneLease(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	onlyIssueEligible(t, bare)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	pace := "600/0"
	askFirst(t, repo, bare, Options{Now: c.now, Pace: &pace}, DrainOptions{})
	const step = 4 * time.Minute
	holds := map[string]bool{}
	// lapsedAt is when the last hold seen lapsed: a new hold is claimed no
	// sooner than the cool-down after it, since the request is dropped no
	// sooner than the lapse.
	var lapsedAt time.Time
	for end := c.t.Add(4 * lease.Default); c.t.Before(end); {
		c.t = c.t.Add(step)
		res, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		r := originReservation(t, bare, claim.RefPrefix+secondIssue)
		if r.released {
			continue
		}
		if !holds[r.commit] && !lapsedAt.IsZero() && r.claimed.Sub(lapsedAt) < claim.TakeoverMargin {
			t.Fatalf("at %s the issue was reserved again %s after its last hold lapsed; want a cool-down of at least %s",
				c.t, r.claimed.Sub(lapsedAt), claim.TakeoverMargin)
		}
		lapsedAt = r.expires
		if !c.t.Before(r.expires) || r.expires.Sub(r.claimed) > lease.Default {
			t.Fatalf("at %s origin holds %s past its lease: %+v", c.t, secondIssue, r)
		}
		if res.Judging == nil || res.Judging.Hold == nil || res.Judging.Hold.Commit != r.commit {
			t.Fatalf("at %s origin's live reservation %s is not the awaited judgement's hold: %+v", c.t, r.commit, res.Judging)
		}
		holds[r.commit] = true
	}
	if len(holds) < 3 {
		t.Fatalf("four leases of polling took %d hold(s); want the issue asked again after each lapse", len(holds))
	}
}

// TestAnExpiredReservationOfThisPersonsIsTheDrainsHoldOnceRenewed: this
// person reserved the issue by hand and let it lapse; the drain's push for the
// judgement renews it, and that push is the drain's hold, recorded and
// released by a yes.
func TestAnExpiredReservationOfThisPersonsIsTheDrainsHoldOnceRenewed(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}},
		clockAt(c.t.Add(-lease.Default-time.Hour))); err != nil {
		t.Fatal(err)
	}
	lapsed := refOf(t, bare, secondIssue)
	first, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	h := first.Judging.Hold
	if first.Judging.Issue != secondIssue || h == nil || h.Commit != refOf(t, bare, secondIssue) {
		t.Fatalf("the drain records its renewal of the lapsed reservation as its hold: %+v", first.Judging)
	}
	if r := originReservation(t, bare, claim.RefPrefix+secondIssue); r.parent != lapsed {
		t.Fatalf("the drain's push renews the lapsed %s: %+v", lapsed, r)
	}
	judge(t, repo, Options{Now: c.now}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	releasedOver(t, bare, secondIssue, h.Commit)
}

// TestALiveReservationOfThisPersonsIsNeverTheDrainsHold: this person holds
// the issue live by hand and the move's view cannot be read, so the drain's
// push renews it; that renewal is the person's, not the drain's, and a yes
// pushes nothing over it.
func TestALiveReservationOfThisPersonsIsNeverTheDrainsHold(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	was := drainView
	t.Cleanup(func() { drainView = was })
	drainView = func(string, time.Time) (claim.Snapshot, error) { return claim.Snapshot{}, errors.New("unreadable") }
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Judging == nil || first.Judging.Issue != secondIssue || first.Judging.Hold != nil {
		t.Fatalf("a renewal of the person's live reservation records no hold of the drain's: %+v", first.Judging)
	}
	mine := refOf(t, bare, secondIssue)
	judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	heldStill(t, bare, secondIssue, mine)
}
