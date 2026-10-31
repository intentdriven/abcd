package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestANotEndedSentenceMarksEveryMentionOfAClaimedHolder: each mention of a
// holder the reader only has the trailer's word for is marked claimed.
func TestANotEndedSentenceMarksEveryMentionOfAClaimedHolder(t *testing.T) {
	l := notEndedLine("rel-v0.16.0", mayaB, true)
	if n := strings.Count(l, mayaB+" ("+SenderClaimed+")"); n != strings.Count(l, mayaB) || n < 2 {
		t.Fatalf("%q marks %d of %d mentions claimed; want every one", l, n, strings.Count(l, mayaB))
	}
	if l := notEndedLine("rel-v0.16.0", mayaB, false); strings.Contains(l, SenderClaimed) {
		t.Fatalf("%q marks an unclaimed holder", l)
	}
}

// allClaimed reports whether every mention of id in text is marked claimed,
// and there is at least one.
func allClaimed(text, id string) bool {
	n := strings.Count(text, id)
	return n > 0 && strings.Count(text, id+" ("+SenderClaimed+")") == n
}

// TestEveryLineNamingAnotherHoldersReleaseMarksTheirIDClaimed: to maya, octo's
// cut names octo by a holder id no record of pushes vouches for, so every
// mention of it is marked claimed: in the hold's line, the merge hold's
// reason and its wait, and, once the cut lapses, the lapsed line, the
// listing's lapse sentence and the second release's back-off. To octo,
// whose id it is, none is.
func TestEveryLineNamingAnotherHoldersReleaseMarksTheirIDClaimed(t *testing.T) {
	w := newWorld(t)
	armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	id := octo.keptHolderID()
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	h, err := FetchMergeHold(maya.dir, w.clock(), maya.opts())
	if err != nil || len(h.Holds) != 1 {
		t.Fatalf("maya's merge hold = %+v, %v; want octo's cut", h, err)
	}
	for what, text := range map[string]string{"hold line": h.Holds[0].Line(), "reason": h.Reason(), "wait": h.Waits()} {
		if !allClaimed(text, id) {
			t.Errorf("maya's %s %q; want every mention of octo's id marked claimed", what, text)
		}
	}
	if own, err := JudgedHolds(octo.dir, w.clock(), octo.opts()); err != nil || len(own) != 1 || strings.Contains(own[0].Text(), SenderClaimed) {
		t.Fatalf("octo's own hold = %+v, %v; want octo's id not called claimed", own, err)
	}
	// Past the lease, and still inside the take-over margin.
	w.set(w.clock().Add(24*time.Hour + 2*time.Minute))
	if _, err := Fetch(maya.dir, maya.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(maya.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	lapsed := snap.LapsedFor(w.clock(), maya.self())
	if len(lapsed) != 1 || !allClaimed(lapsed[0].Text(), id) {
		t.Fatalf("maya's lapsed cuts = %+v; want octo's id marked claimed in the line", lapsed)
	}
	if rw := listingOf(t, w, maya, "rel-v0.16.0"); !rw.Lapsed || !allClaimed(strings.Join(rw.Lines(), "\n"), id) {
		t.Fatalf("maya's listing %+v %q; want every mention of octo's id marked claimed", rw, rw.Lines())
	}
	_, err = maya.do(Request{Record: "rel-v0.17.0"})
	if !errors.Is(err, ErrBackOff) || !allClaimed(err.Error(), id) {
		t.Fatalf("maya's second release = %v; want a back-off marking every mention of octo's id claimed", err)
	}
	if l := snap.LapsedFor(w.clock(), octo.self()); len(l) != 1 || strings.Contains(l[0].Text(), SenderClaimed) {
		t.Fatalf("octo's lapsed cut %+v; want octo's id not called claimed", l)
	}
}
