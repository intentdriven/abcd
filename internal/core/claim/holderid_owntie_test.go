package claim

import (
	"slices"
	"testing"
	"time"
)

// An entry in the reader's own record, as their own newest push records it,
// is kept over every other entry for its pull request where an act combines
// that record with the tip it is written over (iss-2610290809526033): another
// login's entry never displaces the reader's own, whether it is recorded
// earlier on the branch or still to settle beside a kept one of theirs. The
// order of unsettled, vouched and earliest recorded weighs only entries that
// are not the reader's.

// malID is the holder id mal's own reservations carry.
var malID = copyAID

// fiveForge is a forge with #5 open and its merge queued, and its comment
// writer set.
func fiveForge(t *testing.T) *fakeForge {
	t.Helper()
	f := newFakeForge(t, map[int]*fakePR{5: {id: "PR_five", head: headA, queued: true}})
	releaseForge.comments = f
	return f
}

// TestAnEarlierEntryAnotherLoginReAddsNeverDisplacesTheReadersOwn: mal's
// lapsed cut records #5 under mal's id; maya takes it over and withdraws #5
// again under her kept id; mal pushes his entry for #5 back over hers,
// unchanged; the tag lands. mal's entry is recorded earlier on the branch
// than maya's, and his own push vouches for it, but it is not hers, so her
// settle keeps her own #5, re-arms it, and lists #5 once, under her id.
func TestAnEarlierEntryAnotherLoginReAddsNeverDisplacesTheReadersOwn(t *testing.T) {
	w := newWorld(t)
	f := fiveForge(t)
	theirs := Withdrawal{Number: 5, ID: "PR_five", Form: FormQueue, Head: headA, By: malID, State: WithdrawnPending}
	m := message{Record: "rel-v0.16.0", Holder: malID, Copy: "0123456789abcdef0123456789abcdef", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{theirs}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "mal"
	w.set(w.clock().Add(25 * time.Hour))
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("maya's take-over: %v", err)
	}
	if e, _ := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 5); e.By != mayaB || !e.Pending() || f.pr(5).queued {
		t.Fatalf("maya's take-over records %+v, #5 %+v; want #5 withdrawn again under her kept id", tipEntries(t, w, "rel-v0.16.0"), f.pr(5))
	}
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	w.pushAs(f, "rel-v0.16.0", "mal", m.text())
	w.tagOrigin("v0.16.0")
	f.reset()
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	s := settleOf(t, res, "rel-v0.16.0")
	if len(s.Settled) != 1 || s.Settled[0].By != mayaB || s.Settled[0].State != WithdrawnRearmed {
		t.Fatalf("maya's settle = %+v; want her own #5 re-armed", s)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enqueuePullRequest #5"}) {
		t.Fatalf("mutations = %v; want #5 queued again, once", got)
	}
	if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 || ws[0].By != mayaB || ws[0].State != WithdrawnRearmed {
		t.Fatalf("tip records %+v; want #5 once, maya's, re-armed", ws)
	}
}

// TestTheReadersKeptEntryOutlivesAnotherLoginsUnsettledOne: maya's own
// newest push records #5 kept, the forge having refused its withdrawal, and
// the record her next write is combined with records #5 under mal's id, still
// withdrawn. The write keeps her own kept #5 over his unsettled one, carries
// it on her own reservation, and her copy still acts on it, so the renewal
// tries the withdrawal again.
func TestTheReadersKeptEntryOutlivesAnotherLoginsUnsettledOne(t *testing.T) {
	self := Self{ID: mayaB, Login: "maya"}
	kept := Withdrawal{Number: 5, ID: "PR_five", Form: FormAutoSquash, Head: headA, By: mayaB, State: WithdrawnKept}
	theirs := kept
	theirs.By, theirs.State = malID, WithdrawnPending
	own := ownRecord{found: true, m: message{Withdrawn: []Withdrawal{kept}}, first: func(Withdrawal) bool { return true }}
	k := &asker{self: self, complete: true}
	ws := merged(k, []Withdrawal{theirs}, []Withdrawal{theirs}, own, self)
	if !slices.Equal(ws, []Withdrawal{kept}) {
		t.Fatalf("merged = %+v; want maya's own kept #5 alone", ws)
	}
	ws = carried(ws, true)
	if !slices.Equal(ws, []Withdrawal{kept}) || !actsOn(self, k.vouched(ws, own))(kept) {
		t.Fatalf("carried = %+v; want maya's kept #5 carried and acted on, so the renewal tries it again", ws)
	}
}

// TestAStaleMarkerEntryLosesToTheReadersOwnSettledOne: maya's cut, made
// before the holder id, records #11 and #12 under her login; her tombstone
// records #12 re-armed and #11 still withdrawn; mal pushes a commit over it
// carrying #12 under the marker, still withdrawn; the tag lands. Her settle
// keeps her own re-armed #12 over the marker entry, re-arms #11, and leaves
// nothing by hand: #12 is not acted on again, and the tombstone lists it
// once, under her kept id.
func TestAStaleMarkerEntryLosesToTheReadersOwnSettledOne(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	eleven := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "maya", State: WithdrawnPending}
	twelve := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: "maya", State: WithdrawnPending}
	m := message{Record: "rel-v0.16.0", Holder: "maya", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{eleven, twelve}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "maya"
	rearmed := twelve
	rearmed.State = WithdrawnRearmed
	m.ReleasedAt, m.Withdrawn = w.clock(), []Withdrawal{eleven, rearmed}
	w.pushAs(f, "rel-v0.16.0", "maya", m.text())
	marker := twelve
	marker.By = preIDMarker
	stale := message{Record: "rel-v0.16.0", Holder: malID, Copy: "0123456789abcdef0123456789abcdef", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{marker}, AssistedBy: "abcd:0.0.0"}
	w.pushAs(f, "rel-v0.16.0", "mal", stale.text())
	w.tagOrigin("v0.16.0")
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	s := settleOf(t, res, "rel-v0.16.0")
	if !slices.Equal(numbers(s.Settled), []string{"#11"}) || len(s.ByHand) != 0 {
		t.Fatalf("maya's settle = %+v; want #11 re-armed and nothing left by hand", s)
	}
	if got := actedOn(f, 12); len(got) != 0 {
		t.Fatalf("#12 was acted on: %v; want it left as maya's own record settled it", got)
	}
	ws := tipEntries(t, w, "rel-v0.16.0")
	if e, _ := withdrawalOf(ws, 12); len(ws) != 2 || e.By != mayaB || e.State != WithdrawnRearmed {
		t.Fatalf("tip records %+v; want #12 once, maya's, re-armed", ws)
	}
}
