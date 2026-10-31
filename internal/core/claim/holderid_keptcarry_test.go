package claim

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// An entry naming the reader's kept holder id that the chain does not vouch
// for as theirs (iss-2610290809526033) is carried by a push of theirs only
// over a commit that holds it exactly as it stands, so no push of theirs
// is ever the first record of an entry they did not withdraw.

// octoKept is the holder id octo's home keeps in these tests.
var octoKept = "h_" + strings.Repeat("5", 32)

// unvouchedKeptEntry is #12 recorded withdrawn under octo's kept id on a
// reservation maya pushed: an entry octo never withdrew.
var unvouchedKeptEntry = Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: octoKept, State: WithdrawnPending}

// keptEntryOnMayasLapsedCut is rel-v0.16.0 reserved under maya's id, in one
// commit the forge's record says maya pushed, recording unvouchedKeptEntry,
// lapsed past the take-over margin; #12 is armed for nothing, and octo's home
// keeps octoKept.
func keptEntryOnMayasLapsedCut(t *testing.T) (*world, *fakeForge, *participant) {
	t.Helper()
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{12: {id: "PR_twelve", head: headB}})
	releaseForge.comments = f
	m := message{Record: "rel-v0.16.0", Holder: mayaB, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{unvouchedKeptEntry}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "maya"
	w.set(w.clock().Add(2 * time.Hour))
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	keepHolderID(t, octo.home, octoKept)
	return w, f, octo
}

// actedOn12 is each call that re-armed or withdrew #12, or commented on it.
func actedOn12(f *fakeForge) []string {
	var out []string
	for _, m := range f.mutations() {
		if strings.HasSuffix(m, " #12") {
			out = append(out, m)
		}
	}
	for _, c := range f.posted() {
		if strings.HasPrefix(c, "#12 ") {
			out = append(out, c)
		}
	}
	return out
}

// TestARenewalOverACommitWithoutAnUnvouchedKeptEntryDropsIt: maya's lapsed
// cut records #12 under octo's kept id. octo takes it over, which carries
// #12 over maya's commit as it stands; quinn then pushes a tombstone that
// ends nothing and carries no entry, and octo renews over it. The renewal
// does not carry #12, so no push of octo's is its first record: no reading
// of octo's (their own record, the entries vouched and acted on, the
// peers listing) counts it as theirs, an admission of it is refused, and
// neither the release nor a later settle re-arms or comments on it.
func TestARenewalOverACommitWithoutAnUnvouchedKeptEntryDropsIt(t *testing.T) {
	w, f, octo := keptEntryOnMayasLapsedCut(t)
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("octo's take-over: %v", err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); !slices.Contains(ws, unvouchedKeptEntry) {
		t.Fatalf("the take-over records %+v; want #12 carried over maya's commit as it stands", ws)
	}
	tomb := message{Record: "rel-v0.16.0", Holder: quinnID, Copy: "1f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock(), ReleasedAt: w.clock(), AssistedBy: "abcd:0.0.0"}
	w.pushAs(f, "rel-v0.16.0", "quinn", tomb.text())
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("octo's renewal: %+v, %v; want it renewed", res, err)
	}
	tip := w.remoteTip("rel-v0.16.0")
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 12) != "" {
		t.Fatalf("the renewal records %+v; want #12 not carried over quinn's commit, which does not hold it", ws)
	}

	self := octo.self()
	ws := []Withdrawal{unvouchedKeptEntry}
	own, vouched := readerRecord(octo.dir, "rel-v0.16.0", tip, self, &ws)
	if slices.ContainsFunc(own.mine(self), func(w Withdrawal) bool { return w.Number == 12 }) ||
		slices.ContainsFunc(ws, func(w Withdrawal) bool { return w.Number == 12 && actsOn(self, vouched)(w) }) {
		t.Fatalf("octo's record holds %+v, vouched %+v; want #12 neither theirs nor acted on", own.mine(self), vouched)
	}
	if rw := listingOf(t, w, octo, "rel-v0.16.0"); slices.Contains(numbers(rw.Due), "#12") || slices.Contains(numbers(rw.Held), "#12") {
		t.Fatalf("octo's listing = %+v; want #12 neither due nor held as theirs", rw)
	}
	f.reset()
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 12}); err == nil {
		t.Fatal("octo admitted #12; want the admission refused")
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatalf("octo's release: %v", err)
	}
	w.tagOrigin("v0.16.0")
	if _, err := SettleEndedReleases(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	if _, err := octo.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	if got := actedOn12(f); len(got) != 0 {
		t.Fatalf("octo acted on #12: %v; want it never re-armed or commented on", got)
	}
}
