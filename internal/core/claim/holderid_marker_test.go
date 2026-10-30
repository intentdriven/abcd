package claim

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// A release holds one entry for each pull request (iss-2610290809526033):
// where an act combines two records, it keeps one entry for each, never both.

// quietForge is a forge with #11, #12 and #13 open and armed for nothing,
// as a cut's withdrawals leave them, and its comment writer set.
func quietForge(t *testing.T) *fakeForge {
	t.Helper()
	f := newFakeForge(t, map[int]*fakePR{
		11: {id: "PR_eleven", head: headA},
		12: {id: "PR_twelve", head: headB},
		13: {id: "PR_thirteen", head: headC},
	})
	releaseForge.comments = f
	return f
}

// settleOf is res's settle of id, failing the test when there is none.
func settleOf(t *testing.T, res Result, id string) Settle {
	t.Helper()
	i := slices.IndexFunc(res.Settles, func(s Settle) bool { return s.Subject == id })
	if i < 0 {
		t.Fatalf("settles = %+v; want one of %s", res.Settles, id)
	}
	return res.Settles[i]
}

// tipEntries is the withdrawals origin's id tip records, failing the test
// when it does not read or names a pull request twice.
func tipEntries(t *testing.T, w *world, id string) []Withdrawal {
	t.Helper()
	msg := w.remoteMessage(w.remoteTip(id))
	m, why := parseMessage(id, msg)
	if why != "" {
		t.Fatalf("origin's %s tip does not read (%s):\n%s", id, why, msg)
	}
	return m.Withdrawn
}

// TestARoundTripBetweenTwoHoldersKeepsOneEntryForAPullRequest: maya cuts
// rel-v0.16.0 and withdraws #12; octo takes the lapsed cut over and, #12
// armed again by hand, withdraws it again; maya takes the lapsed cut back.
// Her own newest push records her #12 and the tip octo's: both still to
// settle, both vouched for by their holder's own push, so the earliest
// recorded, hers, is the one her take-over writes, once, and her release
// re-arms it.
func TestARoundTripBetweenTwoHoldersKeepsOneEntryForAPullRequest(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	keepHolderID(t, octo.home, octoKept)
	arm12 := func() {
		f.mu.Lock()
		f.prs[12].queued = true
		f.mu.Unlock()
	}
	arm12()
	if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.set(w.clock().Add(25 * time.Hour))
	arm12()
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("octo's take-over: %v", err)
	}
	if e, _ := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 12); e.By != octoKept {
		t.Fatalf("octo's take-over records %+v; want #12 withdrawn again under octo's id", tipEntries(t, w, "rel-v0.16.0"))
	}
	w.set(w.clock().Add(25 * time.Hour))
	if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("maya's take-over: %v", err)
	}
	ws := tipEntries(t, w, "rel-v0.16.0")
	if len(ws) != 1 || ws[0].Number != 12 || ws[0].By != mayaB || !ws[0].Pending() {
		t.Fatalf("maya's take-over records %+v; want #12 once, hers, the earliest recorded", ws)
	}
	f.reset()
	res, err := maya.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatalf("maya's release: %v", err)
	}
	if s := settleOf(t, res, "rel-v0.16.0"); !slices.Equal(numbers(s.Settled), []string{"#12"}) {
		t.Fatalf("maya's release = %+v; want #12 re-armed", s)
	}
	if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 {
		t.Fatalf("release records %+v; want one entry", ws)
	}
}

// TestALaterUnsettledEntryAnotherLoginPushedNeverDisplacesAVouchedOne:
// maya's cut records #12 under her kept id, pushed by her; mal pushes a
// reservation over it recording #12 under mal's own id, still withdrawn, and
// the tag lands. Both are still to settle and both are vouched for, so the
// earliest recorded, maya's, is the one her settle keeps and re-arms, and
// the tombstone lists #12 once.
func TestALaterUnsettledEntryAnotherLoginPushedNeverDisplacesAVouchedOne(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	mine := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: mayaB, State: WithdrawnPending}
	m := message{Record: "rel-v0.16.0", Holder: mayaB, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{mine}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "maya"
	malID := "h_" + strings.Repeat("7", 32)
	theirs := mine
	theirs.By, theirs.Head = malID, headC
	m.Holder, m.Copy, m.Withdrawn = malID, "0123456789abcdef0123456789abcdef", []Withdrawal{theirs}
	w.pushAs(f, "rel-v0.16.0", "mal", m.text())
	w.tagOrigin("v0.16.0")
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	s := settleOf(t, res, "rel-v0.16.0")
	if len(s.Settled) != 1 || s.Settled[0].By != mayaB || s.Settled[0].Head != headB || s.Settled[0].State != WithdrawnRearmed {
		t.Fatalf("maya's settle = %+v; want her own #12 re-armed at its withdrawn head", s)
	}
	ws := tipEntries(t, w, "rel-v0.16.0")
	if len(ws) != 1 || ws[0].By != mayaB {
		t.Fatalf("tip records %+v; want #12 once, maya's", ws)
	}
}
