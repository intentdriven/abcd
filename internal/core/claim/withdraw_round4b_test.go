package claim

import (
	"errors"
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
	// The record has not yet caught up with the tombstone: it holds no push
	// made after this point.
	held := map[string]bool{}
	for _, l := range strings.Fields(runGit(t, w.bare, "rev-list", RefPrefix+"rel-v0.16.0")) {
		held[l] = true
	}
	f.edit = func(ps []Push) []Push {
		return slices.DeleteFunc(ps, func(p Push) bool { return !held[p.After] })
	}
	f.reset()
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed", got)
	}
}

// TestAReleaseNamesWhatItDoesNotActOnOncePerTip: --release reads the
// holder's own newest push. A commit another login pushed over it is not
// read, and is named once for the tip: the release re-arms only what octo's
// own push recorded, and the act after it names nothing again. An app's push
// of octo's record, or a record behind the tip, leaves that push not known:
// the release is refused, naming why, and nothing is pushed or armed.
func TestAReleaseNamesWhatItDoesNotActOnOncePerTip(t *testing.T) {
	setup := func() (*world, *fakeForge, *participant, string) {
		w := newWorld(t)
		f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}, 14: {id: "PR_fourteen", head: headA}})
		octo := w.clone("1234+octo@users.noreply.github.com", "")
		if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatal(err)
		}
		record := w.remoteTip("rel-v0.16.0")
		m, ws := w.tipWithdrawals("rel-v0.16.0")
		m.Withdrawn = append(ws, Withdrawal{Number: 14, ID: "PR_fourteen", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending})
		m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
		rival := w.pushChild("rel-v0.16.0", record, m.text(), w.clock())
		f.pusherOf[rival] = "mal"
		f.reset()
		w.set(t0.Add(time.Minute))
		return w, f, octo, record
	}
	_, f, octo, _ := setup()
	res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.mutations(), []string{"enablePullRequestAutoMerge #11"}) || f.pr(14).method != "" {
		t.Fatalf("mutations %v; want only octo's #11 re-armed, #14 untouched", f.mutations())
	}
	notes := ""
	for _, x := range res.Settles {
		notes += strings.Join(x.Notes(), "\n")
	}
	if !strings.Contains(notes, "1 commit above "+octo.holderID()+"'s newest push to rel-v0.16.0's branch, pushed by mal (1), are not read") {
		t.Fatalf("notes = %q; want mal's commit named", notes)
	}
	res, err = octo.reserve("iss-8", "o.go")
	if err != nil || len(res.Settles) != 0 {
		t.Fatalf("next act settles = %+v, %v; want nothing named again", res.Settles, err)
	}
	for name, set := range map[string]func(f *fakeForge, record string){
		"app":    func(f *fakeForge, record string) { f.pusherOf[record] = "app" },
		"behind": func(f *fakeForge, _ string) { f.lag = 2 },
	} {
		w, f, octo, record := setup()
		set(f, record)
		tip := w.remoteTip("rel-v0.16.0")
		_, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "abcd acts on none of "+octo.holderID()+"'s withdrawals on rel-v0.16.0") ||
			len(f.mutations()) != 0 || w.remoteTip("rel-v0.16.0") != tip {
			t.Fatalf("%s: release = %v, mutations %v; want refused, naming why, nothing pushed", name, err, f.mutations())
		}
	}
}

// TestAWithdrawalIsVouchedForOnlyWithinTheWalk: octo's newest push within
// the 64 commits read, beneath commits another login pushed, is read and
// re-armed from when the commit beneath it, the reservation, is read too, so
// the walk shows that push first recorded #11; one at the walk's last commit,
// or past it, is not, and nothing is re-armed.
func TestAWithdrawalIsVouchedForOnlyWithinTheWalk(t *testing.T) {
	for _, c := range []struct {
		renewals int
		rearmed  bool
	}{{maxChain - 2, true}, {maxChain - 1, false}, {maxChain, false}} {
		w := newWorld(t)
		f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
		p := w.clone("1234+octo@users.noreply.github.com", "")
		if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatal(err)
		}
		// The reservation and its record, then the renewals over them.
		m, _ := w.tipWithdrawals("rel-v0.16.0")
		top := w.pushChildren("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m, c.renewals)
		for _, l := range strings.Fields(runGit(t, w.bare, "rev-list", "-n", fmt.Sprint(c.renewals), top)) {
			f.pusherOf[l] = "mal"
		}
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

// quinnID is the holder id quinn's copy keeps.
var quinnID = "h_" + strings.Repeat("c", 32)

// TestASettleNamesOnlyItsOwnPersonsWithdrawals: another person's withdrawal
// on an ended release is theirs to settle, so the act names it stranded on
// a reservation, never as this person's.
func TestASettleNamesOnlyItsOwnPersonsWithdrawals(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	w.rewrittenOver("rel-v0.16.0") // octo's #11, octo's own pushes
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = append(ws, Withdrawal{Number: 12, ID: "PR_twelve", Form: FormAutoMerge, Head: headB, By: quinnID, State: WithdrawnPending})
	w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), t0)
	s := settleAfter(t, octo)
	if s.Unplaced != "" || len(s.NotRead) != 0 || strings.Contains(strings.Join(s.Notes(), "\n"), "#12") {
		t.Fatalf("settle = %+v; want quinn's #12 not named as octo's", s)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want only octo's #11", got)
	}
}

// TestAnUnrecordedWithdrawalIsNamedAgainOnANewTip: commits not read are named
// once for each tip, so a new tip names them again.
func TestAnUnrecordedWithdrawalIsNamedAgainOnANewTip(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.rewrittenOver("rel-v0.16.0")
	f.pusherOf[record], f.pusherOf[tomb] = "mal", "mal"
	if s := settleAfter(t, octo); !slices.Equal(s.NotRead, []PushCount{{Pusher: "mal", Commits: 2}}) {
		t.Fatalf("settle = %+v", s)
	}
	if _, err := octo.do(Request{Record: "iss-8", Release: true}); err != nil {
		t.Fatal(err)
	}
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	again := w.pushChild("rel-v0.16.0", tomb, m.text()+"\n", t0)
	f.pusherOf[again] = "mal"
	if s := settleAfter(t, octo); !slices.Equal(s.NotRead, []PushCount{{Pusher: "mal", Commits: 3}}) {
		t.Fatalf("settle on the new tip = %+v; want mal's commits named again", s)
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
	for _, rw := range snap.ReleaseWithdrawals(p.dir, p.self(), at) {
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
	record, tomb := w.rewrittenOver("rel-v0.16.0")
	f.edit = func(ps []Push) []Push {
		i, j := pushIndex(ps, tomb), pushIndex(ps, record)
		return slices.Concat(ps[:i], []Push{{Before: ps[j].Before, After: tomb, Pusher: "octo"}}, ps[j+1:])
	}
	settleAfter(t, octo)
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed", got)
	}
}

// TestTheWalkReadsTheSixtyFourCommitsTheBriefNames: brief 37 and the launch
// check's remedy name the walk's bound, 64 commits; the tests above pin the
// behaviour at whatever bound is set, this pins the bound itself.
func TestTheWalkReadsTheSixtyFourCommitsTheBriefNames(t *testing.T) {
	if maxChain != 64 {
		t.Fatalf("maxChain = %d; brief 37 and the launch check's remedy say 64", maxChain)
	}
}
