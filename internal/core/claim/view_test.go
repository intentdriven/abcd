package claim

import (
	"errors"
	"slices"
	"testing"
	"time"
)

// TestTheViewJudgesEveryReservationFromDiskAlone: View lists each reservation
// as last fetched, with its state (live, expired, released by a tombstone or
// by the default branch), this copy's own marked, the view's fetch time, and
// each person's live count; it runs no fetch and no push.
func TestTheViewJudgesEveryReservationFromDiskAlone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-6", "a.go"); err != nil { // lapses below
		t.Fatal(err)
	}
	w.set(t0.Add(3 * time.Hour))
	for _, id := range []string{"iss-5", "iss-7", "iss-8"} {
		if _, err := p.reserve(id, id+".go"); err != nil { // footprints never overlap
			t.Fatal(err)
		}
	}
	if _, err := p.do(Request{Record: "iss-7", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.reserve("itd-9", "c.go"); err != nil {
		t.Fatal(err)
	}
	w.seed.Write(".abcd/work/issues/resolved/iss-8-the-widget.md", "---\nid: iss-8\n---\n# The widget\n")
	w.seed.Commit("resolve iss-8")
	w.seed.Git("push", "-q", "origin", "main")
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}

	log := traced(t)
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range log.subcommands() {
		if c == "fetch" || c == "push" {
			t.Fatalf("the view ran git %s", c)
		}
	}
	if !snap.FetchedAt.Equal(w.clock()) {
		t.Fatalf("the view is dated %s; want the fetch at %s", snap.FetchedAt, w.clock())
	}
	got := map[string]Reservation{}
	for _, r := range snap.Reservations {
		got[r.Record] = r
	}
	for id, want := range map[string][2]string{
		"iss-5": {StateLive, ""}, "iss-6": {StateExpired, ""}, "iss-7": {StateReleased, ReleasedByTombstone},
		"iss-8": {StateReleased, ReleasedByDefaultBranch}, "itd-9": {StateLive, ""},
	} {
		r, ok := got[id]
		if !ok || r.State != want[0] || r.ReleasedBy != want[1] {
			t.Errorf("%s = %+v; want %s %s", id, r, want[0], want[1])
		}
	}
	if got["iss-5"].Copy != CopyThis || got["itd-9"].Copy != "" || !slices.Equal(got["itd-9"].Footprint, []string{"c.go"}) {
		t.Errorf("copies: iss-5 %q, itd-9 %q; want this copy's marked and Q's not", got["iss-5"].Copy, got["itd-9"].Copy)
	}
	if lh := snap.LiveByHolder(); lh["octo"] != 1 || lh["quinn"] != 1 || len(lh) != 2 {
		t.Errorf("live by holder = %v; want octo 1, quinn 1", lh)
	}

	// A checkout with no origin answers ErrNoOrigin, which is no failed fetch.
	p.git("remote", "rename", "origin", "upstream")
	if _, err := Fetch(p.dir, p.opts()); !errors.Is(err, ErrNoOrigin) {
		t.Fatalf("Fetch with no origin = %v; want ErrNoOrigin", err)
	}
}
