package loop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The review round on spc-2610161044461448 steps 1-18: each test here is one
// finding of the two reviews, watched fail before its fix.

// failingEntropy is a minter's entropy that cannot be read: a start that
// mints its run id from it fails after its reservation is made.
type failingEntropy struct{}

func (failingEntropy) Read([]byte) (int, error) { return 0, errors.New("no entropy") }

// trailerOn is the values of key on origin's reservation tip for id.
func trailerOn(t *testing.T, bare, id, key string) []string {
	t.Helper()
	sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id)
	if sha == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(gitIn(t, bare, "log", "-1", "--format=%B", sha), "\n") {
		if v, ok := strings.CutPrefix(line, key+": "); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestAFailedBuildStartOverAHandBackKeepsIt: a person's `abcd build iss-X`
// over a standing hand-back reserves it, as a person's own reservation may;
// when the start then fails, the release its deferred clean-up pushes keeps
// the hand-back, so no drain judges the issue again while its record stands.
func TestAFailedBuildStartOverAHandBackKeepsIt(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}, Drain: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if res, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true, HandBack: claim.HandBackDesignFinding}, claim.Options{}); err != nil || res.HandBackUnpublished != "" {
		t.Fatalf("the hand-back = %+v, %v", res, err)
	}
	_, err := Start(repo.Root(), secondIssue, Options{Minter: recordid.Minter{Entropy: failingEntropy{}}})
	if err == nil || !strings.Contains(err.Error(), "no entropy") {
		t.Fatalf("the start = %v; want it to fail minting its run id", err)
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); !tomb.released {
		t.Fatalf("the failed start left its reservation holding: %+v", tomb)
	}
	if kind := trailerOn(t, bare, secondIssue, "Abcd-Handed-Back"); len(kind) != 1 || kind[0] != claim.HandBackDesignFinding {
		t.Fatalf("the failed start's release marks hand-back %v; want the design-finding hand-back kept", kind)
	}
}

// refuseReservationPush makes origin refuse every push to id's reservation
// branch, as a hook would, and returns the undo.
func refuseReservationPush(t *testing.T, bare, id string) (undo func()) {
	t.Helper()
	hook := filepath.Join(bare, "hooks", "pre-receive")
	script := fmt.Sprintf("#!/bin/sh\nwhile read old new ref; do [ \"$ref\" = \"%s%s\" ] && { echo 'reservations of %s are closed' >&2; exit 1; }; done\nexit 0\n",
		claim.RefPrefix, id, id)
	if err := os.WriteFile(hook, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Remove(hook); err != nil {
			t.Fatal(err)
		}
	}
}

// pendingFor is the pending hand-back ps names for issue, if any.
func pendingFor(ps []DrainPendingHandBack, issue string) (DrainPendingHandBack, bool) {
	i := slices.IndexFunc(ps, func(p DrainPendingHandBack) bool { return p.Issue == issue })
	if i < 0 {
		return DrainPendingHandBack{}, false
	}
	return ps[i], true
}

// TestAYesWhoseHoldMovedOnPublishesItsHandBackAtTheNextRelease: a yes whose
// hold the move cannot release with its hand-back (renewed by hand since,
// never taken because the person held the issue live, or released by hand
// before the answer) says in its route that the hand-back is not published,
// and why, keeps it pending in the drain's state, and publishes it over the
// issue's next release: a later move's, or the same move's when the release
// is already there.
func TestAYesWhoseHoldMovedOnPublishesItsHandBackAtTheNextRelease(t *testing.T) {
	for _, c := range []struct {
		name   string
		before func(t *testing.T, repo *gittest.Repo)
		after  func(t *testing.T, repo *gittest.Repo)
		// sameMove is true when the yes move itself finds the release.
		sameMove bool
	}{
		{name: "renewed by hand", after: func(t *testing.T, repo *gittest.Repo) {
			if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "never taken", before: func(t *testing.T, repo *gittest.Repo) {
			if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "released by hand", sameMove: true, after: func(t *testing.T, repo *gittest.Repo) {
			if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, bare := reservingDrainRepo(t, "commands/drain.md")
			if c.before != nil {
				c.before(t, repo)
			}
			first, err := Drain(repo.Root(), Options{}, DrainOptions{})
			if err != nil || first.Judging == nil || first.Judging.Issue != secondIssue {
				t.Fatalf("the first move asks over %s: %+v %v", secondIssue, first.Judging, err)
			}
			if c.after != nil {
				c.after(t, repo)
			}
			next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
			r := routeOf(t, next.HandBacks, secondIssue)
			if c.sameMove {
				if kind, blob := tombstoneTrailers(t, bare, secondIssue); kind != HandBackTrustRule || blob != originBlob(t, bare, secondIssue) {
					t.Fatalf("the yes move left origin's tip marking %q over %q; want the trust-rule hand-back published over the release", kind, blob)
				}
				if !strings.Contains(r.Wrote, "published the hand-back") {
					t.Fatalf("the route says the hand-back was published: %q", r.Wrote)
				}
				if _, ok := pendingFor(next.PendingHandBacks, secondIssue); ok {
					t.Fatalf("a published hand-back is still pending: %+v", next.PendingHandBacks)
				}
				return
			}
			if !strings.Contains(r.Wrote, "the hand-back is not yet published on origin") {
				t.Fatalf("the route says the hand-back is not published, and why: %q", r.Wrote)
			}
			if p, ok := pendingFor(next.PendingHandBacks, secondIssue); !ok || p.Kind != HandBackTrustRule || p.Why == "" ||
				p.Blob != originBlob(t, bare, secondIssue) {
				t.Fatalf("the move's pending hand-backs = %+v; want %s's trust-rule hand-back, its blob and why", next.PendingHandBacks, secondIssue)
			}
			if st, _, err := readDrain(repo.Root()); err != nil {
				t.Fatal(err)
			} else if _, ok := pendingFor(st.PendingHandBacks, secondIssue); !ok {
				t.Fatalf("the drain's state keeps no pending hand-back: %+v", st.PendingHandBacks)
			}
			if kind, _ := tombstoneTrailers(t, bare, secondIssue); kind != "" {
				t.Fatalf("the person's live reservation carries a hand-back: %q", kind)
			}

			// The person releases the issue; the next move publishes the
			// hand-back over that release.
			if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
				t.Fatal(err)
			}
			tomb := refOf(t, bare, secondIssue)
			poll, err := Drain(repo.Root(), Options{}, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if kind, blob := tombstoneTrailers(t, bare, secondIssue); kind != HandBackTrustRule || blob != originBlob(t, bare, secondIssue) {
				t.Fatalf("origin's tip marks %q over %q; want the pending trust-rule hand-back published", kind, blob)
			}
			if r := originReservation(t, bare, claim.RefPrefix+secondIssue); r.parent != tomb {
				t.Fatalf("the hand-back is not published over the person's release %s: %+v", tomb, r)
			}
			if _, ok := pendingFor(poll.PendingHandBacks, secondIssue); ok || !strings.Contains(poll.Next, "published the hand-back") {
				t.Fatalf("the poll still keeps the hand-back pending, or does not say it published it: %+v, %q", poll.PendingHandBacks, poll.Next)
			}
		})
	}
}

// TestANoThenAnUnreadablePeerAccountsForTheHold: a no is taken, and the move
// is then refused on a peer it cannot read; the drain's state records the no
// and no judgement awaited, and the hold the no was asked on, which no lane
// opened on and the state no longer records, is released.
func TestANoThenAnUnreadablePeerAccountsForTheHold(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	broken := filepath.Join(t.TempDir(), "wt-broken")
	repo.Git("worktree", "add", "-q", "-b", "lane-broken", broken)
	breakGitFile(t, broken)
	path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementNo, "", "the remedy changes neither"))
	_, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path})
	if r := mustRefusal(t, err); r.Check != CheckPeers {
		t.Fatalf("the move is refused on the peer: %+v", r)
	}
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.Judging != nil || len(st.Judgements) != 1 || st.Judgements[0].Answer != JudgementNo {
		t.Fatalf("the state records the no and awaits nothing: judging %+v, judgements %+v", st.Judging, st.Judgements)
	}
	releasedOver(t, bare, secondIssue, taken)
}

// TestAYesThenARefusedReservationKeepsWhatTheMoveDid: a yes promotes and
// pushes its hand-back, and the next issue's reservation is then refused by
// origin for a reason of its own; the refusal comes after the move's writes,
// so the judgement, the route and the awaited judgement's end are recorded,
// the refusal says the answer need not be given again, and the next move
// routes nothing twice.
func TestAYesThenARefusedReservationKeepsWhatTheMoveDid(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirst(t, repo, bare, Options{}, DrainOptions{})
	undo := refuseReservationPush(t, bare, eligibleIssue)
	path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary"))
	_, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path})
	r := mustRefusal(t, err)
	if r.Contention || !strings.Contains(r.Reason, "reservations of "+eligibleIssue+" are closed") {
		t.Fatalf("the next issue's reservation is refused, not contended: %+v", r)
	}
	if !strings.Contains(r.Reason, "the judgement answer given was taken") {
		t.Fatalf("the refusal says the answer need not be given again: %q", r.Reason)
	}
	releasedOver(t, bare, secondIssue, taken)
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.Judging != nil || len(st.Judgements) != 1 || len(st.HandBacks) != 1 || st.HandBacks[0].Issue != secondIssue {
		t.Fatalf("the state records the yes and its route: judging %+v, judgements %+v, hand-backs %+v", st.Judging, st.Judgements, st.HandBacks)
	}
	undo()
	next, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(slices.DeleteFunc(slices.Clone(next.HandBacks), func(r DrainRoute) bool { return r.Issue != secondIssue })); n != 1 ||
		next.Judging == nil || next.Judging.Issue != eligibleIssue {
		t.Fatalf("the next move routes %s %d time(s) and judges %+v; want it routed once and the next issue asked", secondIssue, n, next.Judging)
	}
}

// TestANoWhoseLaneReservationIsRefusedKeepsTheJudgement: a no opens the lane
// on no hold, the person having released it by hand, and origin refuses the
// lane's own reservation; the move is refused after its writes, so the no is
// recorded and not asked again, and once origin takes the push the next move
// opens the lane with no judgement asked.
func TestANoWhoseLaneReservationIsRefusedKeepsTheJudgement(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, _ := askFirst(t, repo, bare, Options{}, DrainOptions{})
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	undo := refuseReservationPush(t, bare, secondIssue)
	path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementNo, "", "the remedy changes neither"))
	_, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path})
	if r := mustRefusal(t, err); r.Stage != StageReservation || r.Contention || !strings.Contains(r.Reason, "the judgement answer given was taken") {
		t.Fatalf("the lane's reservation refuses the move after the answer was taken: %+v", r)
	}
	st, _, err := readDrain(repo.Root())
	if err != nil || st.Judging != nil || len(st.Judgements) != 1 || st.Judgements[0].Answer != JudgementNo {
		t.Fatalf("the state records the no: %+v %+v %v", st.Judging, st.Judgements, err)
	}
	undo()
	next, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if next.Judging != nil || next.Lane == nil || next.Lane.Issue != secondIssue {
		t.Fatalf("the next move opens %s's lane without asking again: %+v", secondIssue, next)
	}
}

// failingForge makes origin read as a github.com repository whose forge does
// not answer, so every act that asks the fork question names it skipped.
func failingForge(t *testing.T) {
	t.Helper()
	restore := claim.SwapForkCheckForTest(
		func(string) ([]string, error) { return []string{"https://github.com/octo/abcd.git"}, nil },
		func(_, _, _ string) (github.Repo, error) { return github.Repo{}, errors.New("unreachable") })
	t.Cleanup(restore)
}

// forkWarning is how a run record names a fork check its reservation could
// not make.
const forkWarning = "not checked: whether origin is a fork (octo/abcd: "

// TestALaneOnAJudgementHoldNamesItsForkCheckInALaterMove: the no was taken
// over a hold whose fork check could not be made, by a move whose lane did not
// open; the later move that opens the lane on that hold, given no answer
// itself, still names the skipped check, from the recorded judgement.
func TestALaneOnAJudgementHoldNamesItsForkCheckInALaterMove(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	failingForge(t)
	first, _ := askFirst(t, repo, bare, Options{}, DrainOptions{})
	if first.Judging.Hold.ForkUnchecked == "" {
		t.Fatal("the judgement's hold names no skipped fork check")
	}
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	j := *st.Judging
	st.Judgements = append(st.Judgements, DrainJudgement{Issue: j.Issue, RemedySHA256: j.RemedySHA256, Answer: JudgementNo,
		Reason: "the remedy changes neither", Applied: true, At: time.Now().UTC(), Hold: j.Hold})
	st.Judging = nil
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil || res.Start == nil || res.Lane == nil || res.Lane.Issue != secondIssue {
		t.Fatalf("the move opens %s's lane on the hold: %+v %v", secondIssue, res, err)
	}
	if !strings.HasPrefix(res.Start.ForkUnchecked, "octo/abcd: ") {
		t.Fatalf("the start names the skipped check: %q", res.Start.ForkUnchecked)
	}
	if note := recordNote(t, repo, res.Start.RunID, StageReservation); !strings.Contains(note, forkWarning) {
		t.Fatalf("the run record names the skipped check: %q", note)
	}
}

// TestALaneOnALapsedHoldAsksTheForkQuestionAgain: the hold a judgement was
// asked on lapses before the no; the lane's own reservation renews it, which
// makes it the drain's hold afresh, so it asks the fork question again and
// names the check it could not make. A first move whose reservation renews
// this copy's own lapsed reservation does the same for the judgement's hold.
func TestALaneOnALapsedHoldAsksTheForkQuestionAgain(t *testing.T) {
	t.Run("lapsed before the no", func(t *testing.T) {
		repo, bare := reservingDrainRepo(t, "commands/drain.md")
		failingForge(t)
		c := &clock{t: time.Now().UTC().Truncate(time.Second)}
		o := Options{Now: c.now}
		// A window longer than the lease, so the pace never holds the moves.
		pace := "600/0"
		first, _ := askFirst(t, repo, bare, Options{Now: c.now, Pace: &pace}, DrainOptions{})
		c.t = c.t.Add(3 * time.Hour)
		path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementNo, "", "the remedy changes neither"))
		res, err := Drain(repo.Root(), o, DrainOptions{Judgement: path})
		if err != nil || res.Start == nil || res.Lane == nil || res.Lane.Issue != secondIssue {
			t.Fatalf("the no opens %s's lane: %+v %v", secondIssue, res, err)
		}
		if !strings.HasPrefix(res.Start.ForkUnchecked, "octo/abcd: ") {
			t.Fatalf("the lane on the renewed hold names the skipped check: %q", res.Start.ForkUnchecked)
		}
		if note := recordNote(t, repo, res.Start.RunID, StageReservation); !strings.Contains(note, forkWarning) {
			t.Fatalf("the run record names the skipped check: %q", note)
		}
	})
	t.Run("a lapsed own reservation renewed for the judgement", func(t *testing.T) {
		repo, _ := reservingDrainRepo(t, "commands/drain.md")
		now := time.Now().UTC().Truncate(time.Second)
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}, Drain: true},
			claim.Options{Now: func() time.Time { return now.Add(-3 * time.Hour) }}); err != nil {
			t.Fatal(err)
		}
		failingForge(t)
		first, err := Drain(repo.Root(), Options{Now: func() time.Time { return now }}, DrainOptions{})
		if err != nil || first.Judging == nil || first.Judging.Issue != secondIssue || first.Judging.Hold == nil {
			t.Fatalf("the move asks over %s on the renewed hold: %+v %v", secondIssue, first.Judging, err)
		}
		if !strings.HasPrefix(first.Judging.Hold.ForkUnchecked, "octo/abcd: ") {
			t.Fatalf("the judgement's renewed hold names the skipped check: %q", first.Judging.Hold.ForkUnchecked)
		}
	})
}

// TestADrainWithNoViewBacksOffAHandBackAtItsReservation: the move's view of
// the reservations cannot be read, so the hand-back another clone published
// is not passed over from it; the drain's own reservation backs off the
// hand-back instead, the issue is passed over naming it, nothing is pushed
// over it and no judgement is asked over it.
func TestADrainWithNoViewBacksOffAHandBackAtItsReservation(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}, Drain: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Release: true, HandBack: claim.HandBackUserVisible}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	tomb := refOf(t, bare, secondIssue)
	was := drainView
	t.Cleanup(func() { drainView = was })
	drainView = func(string, time.Time) (claim.Snapshot, error) {
		return claim.Snapshot{}, errors.New("the tracking refs are unreadable")
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p := passedFor(t, res, secondIssue); p.Check != CheckReservation || !strings.Contains(p.Reason, "was handed back by quinn as user-visible") {
		t.Fatalf("the pass-over names the hand-back: %+v", p)
	}
	if refOf(t, bare, secondIssue) != tomb {
		t.Fatal("the drain pushed over the hand-back")
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("the judgement is asked over the next issue: %+v", res.Judging)
	}
}

// TestARefusedReleaseIsNeverLoggedAsReleased: a release the claim package
// refuses for a reason of the tip's (a message it may not write) is logged
// as a failed release, which the drain routing the hand-back tries again,
// never as released.
func TestARefusedReleaseIsNeverLoggedAsReleased(t *testing.T) {
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	calls := 0
	claimReserve = func(string, claim.Request, claim.Options) (claim.Result, error) {
		calls++
		return claim.Result{}, fmt.Errorf("the reservation message would be 4104 bytes, over the 4096-byte cap: %w", claim.ErrRefused)
	}
	now := time.Now()
	st := State{Key: "iss-2609300000000101", ReservedAt: &now}
	releaseOnHandBack(t.TempDir(), &st, HandBack{Kind: HandBackDesignFinding}, now)
	if n := len(st.Record); n != 1 || st.Record[0].Stage != StageReservationReleaseFailed {
		t.Fatalf("the run record logs %+v; want one reservation_release_failed entry", st.Record)
	}
	releaseHandedBack(t.TempDir(), st, HandBack{Kind: HandBackDesignFinding}, now)
	if calls != 2 {
		t.Fatalf("the drain routing the hand-back released %d time(s) in all; want it to try again", calls)
	}
}
