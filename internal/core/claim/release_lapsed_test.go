package claim

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// TestALapsedReleaseIsNamedAsHoldingNoMerge (iss-2610241104098581): a release
// reservation that expired with no tag on origin is a cut its holder stopped
// renewing, and it holds no merge. The view names it as lapsed, with its
// holder, its last renewal and how a cut resumes, so no session keeps work
// waiting on a holder that is gone; a tag on origin, or a release, ends it.
func TestALapsedReleaseIsNamedAsHoldingNoMerge(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	live := t0.Add(time.Hour)
	snap, err := View(q.dir, live)
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.Lapsed(live); len(got) != 0 {
		t.Fatalf("a live release is a hold, not a lapsed cut: %+v", got)
	}

	after := t0.Add(lease.Max + time.Hour)
	if holds, err := releaseHolds(q.dir, after, q.opts()); err != nil || len(holds) != 0 {
		t.Fatalf("an expired release holds no merge: %+v %v", holds, err)
	}
	if snap, err = View(q.dir, after); err != nil {
		t.Fatal(err)
	}
	lapsed := snap.Lapsed(after)
	if len(lapsed) != 1 {
		t.Fatalf("the expired release is named as lapsed: %+v", lapsed)
	}
	l := lapsed[0]
	if l.Release != "v0.16.0" || l.Subject != "rel-v0.16.0" || l.Holder != "octo" ||
		!l.RenewedAt.Equal(t0) || !l.LapsedAt.Equal(t0.Add(lease.Max)) {
		t.Fatalf("lapsed = %+v; want v0.16.0 by octo, renewed %s, lapsed %s", l, t0, t0.Add(lease.Max))
	}
	text := l.Text()
	for _, want := range []string{"release v0.16.0", "@octo", "lapsed at " + stamp(t0.Add(lease.Max)),
		"last renewed " + stamp(t0), "no tag v0.16.0 on origin", "no merge is held",
		"From " + stamp(t0.Add(lease.Max+TakeoverMargin)) + ", any copy reserves it again with `abcd reserve rel-v0.16.0`",
		"`abcd reserve rel-v0.16.0 --release`, to withdraw it"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the lapsed line %q does not say %q", text, want)
		}
	}

	w.tagOrigin("v0.16.0")
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	if snap, err = View(q.dir, after); err != nil {
		t.Fatal(err)
	}
	if got := snap.Lapsed(after); len(got) != 0 {
		t.Fatalf("a release whose tag is on origin is finished, not lapsed: %+v", got)
	}
}
