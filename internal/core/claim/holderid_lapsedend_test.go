package claim

import (
	"testing"
	"time"
)

// A cut its holder left ends at that holder's next act once its lease has
// lapsed past TakeoverMargin with no tag on origin (dcn-2610271926369425),
// whatever it still holds (iss-2610290809526033): an entry under the marker
// is settled as left to be re-armed by hand, and another holder's entry is
// carried unchanged on the tombstone, for their own copy to settle once the
// release has ended. Neither is acted on by the holder's copy, and a new cut
// then reserves.

// lapsedPastTheMargin moves w's clock past the lapse of the release maya
// just reserved, and past TakeoverMargin beyond it.
func lapsedPastTheMargin(w *world) {
	w.set(w.clock().Add(25 * time.Hour))
}

// TestALapsedCutHoldingOnlyAMarkerEntryEndsAtItsHoldersNextAct: maya takes
// over octo's lapsed cut made before the holder id, so octo's #11 is written
// under the marker; her cut lapses past the margin, and she runs a plain
// reserve. Its settle ends her cut: a tombstone records #11 manual under the
// marker, #11 is never acted on, and a new release cut reserves.
func TestALapsedCutHoldingOnlyAMarkerEntryEndsAtItsHoldersNextAct(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	m := message{Record: "rel-v0.16.0", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{octoPreID}, AssistedBy: "abcd:0.0.0"}
	w.pushTipAt("rel-v0.16.0", m.text(), w.clock())
	before := allCommits(t, w)
	w.set(w.clock().Add(2 * time.Hour))
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("maya's take-over: %v", err)
	}
	if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 || ws[0].By != preIDMarker || !ws[0].Pending() {
		t.Fatalf("take-over records %+v; want only #11, under the marker, still withdrawn", ws)
	}
	lapsedPastTheMargin(w)
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	s := settleOf(t, res, "rel-v0.16.0")
	if s.LapsedAt == nil {
		t.Fatalf("maya's settle = %+v; want it to end her lapsed cut", s)
	}
	wantByHand(t, s, "#11")
	tip, ws := w.tipWithdrawals("rel-v0.16.0")
	if !tip.released() || len(ws) != 1 || ws[0].By != preIDMarker || ws[0].State != WithdrawnManual {
		t.Fatalf("tip %+v records %+v; want a tombstone with #11 manual under the marker", tip, ws)
	}
	if got := actedOn(f, 11); len(got) != 0 {
		t.Fatalf("#11 was acted on: %v; want it never re-armed", got)
	}
	pat := w.clone("4321+pat@users.noreply.github.com", "")
	if _, err := pat.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatalf("a new cut after the lapsed one ended: %v; want it reserved", err)
	}
	noLoginPushedSince(t, w, f, before)
}

// TestALapsedCutHoldingOnlyAnotherHoldersEntryEndsAtItsHoldersNextAct:
// maya takes over octo's lapsed cut, which records #11 under octo's kept id;
// her cut lapses past the margin, and she runs a plain reserve. Its settle
// ends her cut with a tombstone carrying octo's #11 unchanged, still
// withdrawn, which her copy does not act on; a new release cut reserves; and
// octo's next act finds #11 through his own push and re-arms it.
func TestALapsedCutHoldingOnlyAnotherHoldersEntryEndsAtItsHoldersNextAct(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	theirs := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: octoKept, State: WithdrawnPending}
	m := message{Record: "rel-v0.16.0", Holder: octoKept, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{theirs}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "octo"
	w.set(w.clock().Add(2 * time.Hour))
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("maya's take-over: %v", err)
	}
	if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 || ws[0] != theirs {
		t.Fatalf("take-over records %+v; want octo's #11 carried unchanged", ws)
	}
	lapsedPastTheMargin(w)
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	if s := settleOf(t, res, "rel-v0.16.0"); s.LapsedAt == nil || len(s.Settled)+len(s.ByHand) != 0 {
		t.Fatalf("maya's settle = %+v; want it to end her lapsed cut, settling nothing", s)
	}
	tip, ws := w.tipWithdrawals("rel-v0.16.0")
	if !tip.released() || len(ws) != 1 || ws[0] != theirs {
		t.Fatalf("tip %+v records %+v; want a tombstone carrying octo's #11 unchanged", tip, ws)
	}
	if got := actedOn(f, 11); len(got) != 0 {
		t.Fatalf("#11 was acted on: %v; want it left for octo's own copy", got)
	}
	pat := w.clone("4321+pat@users.noreply.github.com", "")
	if _, err := pat.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatalf("a new cut after the lapsed one ended: %v; want it reserved", err)
	}
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	keepHolderID(t, octo.home, octoKept)
	res, err = octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatalf("octo's act: %v", err)
	}
	if s := settleOf(t, res, "rel-v0.16.0"); len(s.Settled) != 1 || s.Settled[0].By != octoKept || s.Settled[0].State != WithdrawnRearmed {
		t.Fatalf("octo's settle = %+v; want his own #11 re-armed", s)
	}
}
