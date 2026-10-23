package claim

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestAReleaseReArmsWhileTheRecordLagsItsOwnTombstone: the forge's record of
// pushes usually reaches the tombstone --release just pushed a moment later,
// so a record whose newest push is the tip the release was written over
// still settles it, and the release re-arms.
func TestAReleaseReArmsWhileTheRecordLagsItsOwnTombstone(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.lag = 1 // the record has not yet caught up with the tombstone
	f.reset()
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed", got)
	}
}

// TestAReleaseNamesWhatItDoesNotActOnOncePerTip: --release names each
// withdrawal it leaves alone (an app's push recorded it, no reservation of
// the holder's recorded it, or the record does not settle it yet), and the
// act after it, over the same tip, does not name the first two again.
func TestAReleaseNamesWhatItDoesNotActOnOncePerTip(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}, 14: {id: "PR_fourteen", head: headA}})
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.pusherOf[w.remoteTip("rel-v0.16.0")] = "app" // the commit recording #11
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = append(ws, Withdrawal{Number: 14, ID: "PR_fourteen", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending})
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[planted] = "mal"
	f.reset()
	w.set(t0.Add(time.Minute))
	res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	var s Settle
	for _, x := range res.Settles {
		if x.Subject == "rel-v0.16.0" && x.Commit != "" {
			s = x
		}
	}
	if !slices.Equal(numbersOf(s.AppPushed), []int{11}) || !slices.Equal(numbersOf(s.Unrecorded), []int{14}) || len(f.mutations()) != 0 {
		t.Fatalf("release settles = %+v, mutations %v; want #11 app-pushed and #14 unrecorded, nothing armed", res.Settles, f.mutations())
	}
	notes := strings.Join(s.Notes(), "\n")
	if !strings.Contains(notes, "an app pushed the commit that recorded them") || !strings.Contains(notes, "#14 on rel-v0.16.0 recorded as withdrawn by octo") {
		t.Fatalf("notes = %q", notes)
	}
	// Over the same tip, the next act names neither again.
	res, err = octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range res.Settles {
		if len(x.AppPushed)+len(x.Unrecorded) > 0 {
			t.Fatalf("next act settles = %+v; want nothing named again for the same tip", res.Settles)
		}
	}
	// A record behind the tombstone and its parent leaves a fresh release's
	// withdrawal not yet verified, named.
	w2 := newWorld(t)
	f2 := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
	p := w2.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f2.lag = 2
	f2.reset()
	res, err = p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil || len(res.Settles) != 1 || !slices.Equal(numbersOf(res.Settles[0].Unverified), []int{11}) ||
		!strings.Contains(strings.Join(res.Settles[0].Notes(), "\n"), "#11 on rel-v0.16.0 not yet verified") || len(f2.mutations()) != 0 {
		t.Fatalf("release = %+v, %v, mutations %v; want #11 named not yet verified", res.Settles, err, f2.mutations())
	}
}

// TestAWithdrawalIsVouchedForOnlyWithinTheWalk: a first record within the
// 64 commits read is vouched for; one the walk stops on with more beneath it
// is not.
func TestAWithdrawalIsVouchedForOnlyWithinTheWalk(t *testing.T) {
	for _, c := range []struct {
		renewals int
		rearmed  bool
	}{{maxChain - 2, true}, {maxChain - 1, false}} {
		w := newWorld(t)
		f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
		p := w.clone("1234+octo@users.noreply.github.com", "")
		if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatal(err)
		}
		// The reservation and its record, then the renewals over them.
		m, _ := w.tipWithdrawals("rel-v0.16.0")
		w.pushChildren("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m, c.renewals)
		w.tagOrigin("v0.16.0")
		f.reset()
		res, err := p.reserve("iss-8", "o.go")
		if err != nil {
			t.Fatal(err)
		}
		got := slices.Contains(f.mutations(), "enablePullRequestAutoMerge #11")
		if got != c.rearmed {
			t.Fatalf("%d renewals: re-armed %v (settles %+v); want %v", c.renewals, got, res.Settles, c.rearmed)
		}
	}
}

// TestGhActingAsNoGitHubUsernameTouchesNothing: an account gh names that is
// not a GitHub username is refused as definitely not the holder.
func TestGhActingAsNoGitHubUsernameTouchesNothing(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f.logins[p.dir] = "not a login"
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil || len(f.mutations()) != 0 || !strings.Contains(res.ForgeUnchecked, "not a GitHub username") {
		t.Fatalf("reserve = %+v, %v, mutations %v; want nothing withdrawn and why", res.ForgeUnchecked, err, f.mutations())
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err == nil {
		t.Fatal("admit was not refused")
	}
}

// TestAReservationWithdrawsOnlyWhatItsTombstoneCanRecord: past 40 pull
// requests, or past what a tombstone any holder writes can carry, a merge is
// not withdrawn, and is named.
func TestAReservationWithdrawsOnlyWhatItsTombstoneCanRecord(t *testing.T) {
	for name, c := range map[string]struct {
		n, idLen int
	}{"past forty": {maxWithdrawals + 1, 8}, "past the cap": {30, 100}} {
		prs := map[int]*fakePR{}
		for i := 1; i <= c.n; i++ {
			id := fmt.Sprintf("PR_%0*d", c.idLen-3, i)
			prs[100000000+i] = &fakePR{id: id, head: headA, method: "REBASE"}
		}
		w := newWorld(t)
		f := newFakeForge(t, prs)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		res, err := p.do(Request{Record: "rel-v0.16.0"})
		if err != nil {
			t.Fatal(err)
		}
		_, ws := w.tipWithdrawals("rel-v0.16.0")
		room := 0
		for _, ff := range res.ForgeFailures {
			if strings.Contains(ff.Reason, "no room left to record it") {
				room++
			}
		}
		if room == 0 || len(ws)+room != c.n || len(ws) > maxWithdrawals || len(f.mutations()) != len(ws) {
			t.Fatalf("%s: recorded %d, %d named without room, mutations %d; want the rest named", name, len(ws), room, len(f.mutations()))
		}
		if !p.prepared(t).tombstoneFits(ws) {
			t.Fatalf("%s: what was recorded does not fit a tombstone", name)
		}
	}
}

// prepared is the participant's act prepared for rel-v0.16.0.
func (p *participant) prepared(t *testing.T) *prepared {
	t.Helper()
	pr, err := prepare(p.dir, Request{Record: "rel-v0.16.0"}, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	return pr
}

// TestASettleNamesOnlyItsOwnPersonsWithdrawals: another person's withdrawal
// on an ended release is theirs to settle, so the act names it stranded on
// a reservation, never as this person's unrecorded.
func TestASettleNamesOnlyItsOwnPersonsWithdrawals(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	w.forgedOver("rel-v0.16.0") // octo's #11, octo's own pushes
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = append(ws, Withdrawal{Number: 12, ID: "PR_twelve", Form: FormAutoMerge, Head: headB, By: "quinn", State: WithdrawnPending})
	w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), t0)
	s := settleAfter(t, octo)
	if slices.Contains(numbersOf(s.Unrecorded), 12) || slices.Contains(numbersOf(s.Unverified), 12) || slices.Contains(numbersOf(s.AppPushed), 12) {
		t.Fatalf("settle = %+v; want quinn's #12 not named as octo's", s)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want only octo's #11", got)
	}
}

// TestAnUnrecordedWithdrawalIsNamedAgainOnANewTip: once for each tip, so a
// new tip names it again.
func TestAnUnrecordedWithdrawalIsNamedAgainOnANewTip(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.forgedOver("rel-v0.16.0")
	f.pusherOf[record], f.pusherOf[tomb] = "mal", "mal"
	if s := settleAfter(t, octo); len(s.Unrecorded) != 1 {
		t.Fatalf("settle = %+v", s)
	}
	if _, err := octo.do(Request{Record: "iss-8", Release: true}); err != nil {
		t.Fatal(err)
	}
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	again := w.pushChild("rel-v0.16.0", tomb, m.text()+"\n", t0)
	f.pusherOf[again] = "mal"
	if s := settleAfter(t, octo); len(s.Unrecorded) != 1 {
		t.Fatalf("settle on the new tip = %+v; want #11 named again", s)
	}
}

// TestALapsedReleaseHoldingOnlySettledEntriesIsNotListedLapsed: the listing
// names a lapse only while the release still holds something unsettled.
func TestALapsedReleaseHoldingOnlySettledEntriesIsNotListedLapsed(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
	f.refuse["disablePullRequestAutoMerge #11"] = "GraphQL: no permission"
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnKept {
		t.Fatalf("recorded %+v; want #11 kept", ws)
	}
	at := w.clock().Add(48 * time.Hour)
	snap, err := View(p.dir, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, rw := range snap.ReleaseWithdrawals(p.dir, "octo", at) {
		if rw.Lapsed {
			t.Fatalf("listing = %+v; want no lapse named for a release holding only a kept entry", rw)
		}
	}
}

// TestAnAdmittedMergeTheSettleCannotArmIsNamedForItsOwner: an admitted pull
// request closed during the cut is named at the settle, as a withdrawn one
// is.
func TestAnAdmittedMergeTheSettleCannotArmIsNamedForItsOwner(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.prs[11].state = "CLOSED"
	f.mu.Unlock()
	res, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	named := false
	for _, s := range res.Settles {
		for _, wd := range s.Settled {
			named = named || wd.Number == 11 && wd.State == WithdrawnClosed
		}
	}
	if !named {
		t.Fatalf("settles = %+v; want #11 named closed", res.Settles)
	}
}

// TestAPersonsOwnPushOfSeveralCommitsVouchesForEach: one push of octo's that
// carried the withdrawal record and the tombstone over it vouches for the
// record, so it is re-armed.
func TestAPersonsOwnPushOfSeveralCommitsVouchesForEach(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.forgedOver("rel-v0.16.0")
	f.edit = func(ps []Push) []Push {
		i, j := pushIndex(ps, tomb), pushIndex(ps, record)
		return slices.Concat(ps[:i], []Push{{Before: ps[j].Before, After: tomb, Pusher: "octo"}}, ps[j+1:])
	}
	settleAfter(t, octo)
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed", got)
	}
}
