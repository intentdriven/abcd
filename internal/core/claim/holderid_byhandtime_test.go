package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// The line naming an entry under the marker, which its author re-arms by
// hand, is printed while the release is still being cut (the stranded note of
// a take-over or renewal, an admission's refusal) and after it has ended (the
// settle that ends it, the listing of an ended release) (iss-2610290809526033).
// While the cut goes on it says the re-arm waits for the release to end, so no
// reader tells the author to re-arm mid-cut; once the release has ended it
// names no time.

const untilTheEnd = "once the release ends"

// TestTheByHandLineSaysWhenWhileTheCutGoesOnAndNotAfter: octo takes over
// quinn's lapsed cut, which records #12 under a login, so his take-over
// writes it under the marker. The take-over's stranded note and the refusal
// of his admission of #12 say its author re-arms it once the release ends;
// the settle of his release, and the listing of the ended release, name no
// time.
func TestTheByHandLineSaysWhenWhileTheCutGoesOnAndNotAfter(t *testing.T) {
	w := newWorld(t)
	quietForge(t)
	m := message{Record: "rel-v0.16.0", Holder: "quinn", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{mayaPreID}, AssistedBy: "abcd:0.0.0"}
	w.pushTipAt("rel-v0.16.0", m.text(), w.clock())
	w.set(w.clock().Add(2 * time.Hour))
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	keepHolderID(t, octo.home, octoKept)
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatalf("octo's take-over: %v", err)
	}
	byHand := func(what, l string, timed bool) {
		t.Helper()
		if !strings.Contains(l, "#12") || !strings.Contains(l, "by hand") {
			t.Fatalf("%s = %q; want #12 named to be re-armed by hand", what, l)
		}
		if strings.Contains(l, untilTheEnd) != timed {
			t.Fatalf("%s = %q; want %q in it: %v", what, l, untilTheEnd, timed)
		}
		noLogin(t, what, l)
	}
	byHand("the take-over's stranded note", StrandedNote("rel-v0.16.0", res.Stranded, octoKept, true, false), true)
	_, err = octo.do(Request{Record: "rel-v0.16.0", Admit: 12})
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("octo's admission of #12: %v; want it refused", err)
	}
	byHand("the admission's refusal", err.Error(), true)
	res, err = octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatalf("octo's release: %v", err)
	}
	s := settleOf(t, res, "rel-v0.16.0")
	byHand("the release's note", strings.Join(s.Notes(), "\n"), false)
	for _, x := range s.Settled {
		if x.preID() {
			byHand("the release's settled line", SettledLine(s.Subject, x), false)
		}
	}
	rw := listingOf(t, w, octo, "rel-v0.16.0")
	if !rw.Ended || len(rw.ByHand) == 0 {
		t.Fatalf("the listing = %+v; want the ended release naming #12 by hand", rw)
	}
	byHand("the ended release's listing", strings.Join(rw.Lines(), "\n"), false)
	marker := []Withdrawal{{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: preIDMarker, State: WithdrawnPending}}
	byHand("an ended release's stranded line", StrandedNote("rel-v0.16.0", marker, octoKept, true, true), false)
	byHand("a cut's stranded line", StrandedNote("rel-v0.16.0", marker, octoKept, true, false), true)
}
