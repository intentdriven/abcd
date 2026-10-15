package claim

import (
	"testing"
	"time"
)

// TestHeldByNamesAnotherCopysReservationWithTheViewsAge: the renders name a
// reservation another copy holds, with its holder, its effective expiry and
// the view's age, read from disk alone; this copy's own, a released one and an
// expired one hold nothing.
func TestHeldByNamesAnotherCopysReservationWithTheViewsAge(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	for _, id := range []string{"iss-5", "iss-6"} {
		if _, err := p.reserve(id, id+".go"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := p.do(Request{Record: "iss-6", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	at := t0.Add(12 * time.Minute)
	snap, err := View(q.dir, at)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := snap.HeldBy("iss-5", at), "reserved by octo until 2026-10-03T16:00:00Z, as of the last fetch 12 minutes ago"; got != want {
		t.Errorf("HeldBy(iss-5) = %q; want %q", got, want)
	}
	for _, id := range []string{"iss-6", "iss-7"} {
		if got := snap.HeldBy(id, at); got != "" {
			t.Errorf("HeldBy(%s) = %q; a released or absent reservation holds nothing", id, got)
		}
	}
	late := t0.Add(3 * time.Hour)
	if snap, err = View(q.dir, late); err != nil {
		t.Fatal(err)
	}
	if got := snap.HeldBy("iss-5", late); got != "" {
		t.Errorf("HeldBy(iss-5) past its expiry = %q; want nothing", got)
	}
	mine, err := View(p.dir, at)
	if err != nil {
		t.Fatal(err)
	}
	if got := mine.HeldBy("iss-5", at); got != "" {
		t.Errorf("this copy's own reservation reads as held: %q", got)
	}
}

// TestTheViewsAgeIsUnknownWithoutAStamp: a view no abcd fetch dated says so
// rather than giving an age.
func TestTheViewsAgeIsUnknownWithoutAStamp(t *testing.T) {
	at := t0.Add(90 * time.Minute)
	for _, c := range []struct {
		fetched time.Time
		want    string
	}{
		{time.Time{}, "as of a fetch of unknown age"},
		{at.Add(-20 * time.Second), "as of the last fetch under a minute ago"},
		{at.Add(-time.Minute), "as of the last fetch 1 minute ago"},
		{at.Add(-90 * time.Minute), "as of the last fetch 90 minutes ago"},
		{at.Add(-5 * time.Hour), "as of the last fetch 5 hours ago"},
		{at.Add(-72 * time.Hour), "as of the last fetch 3 days ago"},
	} {
		if got := (Snapshot{FetchedAt: c.fetched}).AsOf(at); got != c.want {
			t.Errorf("AsOf with a fetch at %s = %q; want %q", c.fetched, got, c.want)
		}
	}
}

// TestLineForNamesTheRecordsReservationInEveryState: the record render's line
// for one record, by state, with whose copy it is when it is the reader's own;
// a view no abcd fetch stamped, holding nothing for the record, is unknown,
// never none; a checkout with no remote named origin has no line.
func TestLineForNamesTheRecordsReservationInEveryState(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")

	if l, ok := LineFor(q.dir, "iss-5", t0, q.opts()); !ok || l.State != LineUnknown ||
		l.Text != "unknown, nothing has been fetched here; `abcd peers` fetches" {
		t.Fatalf("a never-fetched view = %+v, %v; want unknown", l, ok)
	}
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []*participant{q, laptop} {
		if _, err := Fetch(c.dir, c.opts()); err != nil {
			t.Fatal(err)
		}
	}
	at := t0.Add(12 * time.Minute)
	const asOf = ", as of the last fetch 12 minutes ago"
	for _, c := range []struct {
		who  *participant
		id   string
		text string
	}{
		{q, "iss-5", "live, reserved by octo until 2026-10-03T16:00:00Z" + asOf},
		{laptop, "iss-5", "live, reserved by octo (another copy of yours) until 2026-10-03T16:00:00Z" + asOf},
		{q, "iss-6", "none" + asOf},
	} {
		l, ok := LineFor(c.who.dir, c.id, at, c.who.opts())
		if !ok || l.Text != c.text || l.AgeSeconds == nil || *l.AgeSeconds != 720 {
			t.Errorf("LineFor(%s) = %+v, %v; want %q", c.id, l, ok, c.text)
		}
	}
	if l, _ := LineFor(p.dir, "iss-5", t0, p.opts()); l.Copy != LineThisCopy || l.Holder != "octo" || l.State != StateLive {
		t.Errorf("the holder's own line = %+v; want this copy's live reservation", l)
	}

	q.git("remote", "rename", "origin", "upstream")
	if l, ok := LineFor(q.dir, "iss-5", at, q.opts()); ok {
		t.Errorf("a checkout with no origin has a line: %+v", l)
	}
}
