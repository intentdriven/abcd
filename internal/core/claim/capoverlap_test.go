package claim

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// reserveTen has p reserve iss-1 to iss-10, each with its own file.
func reserveTen(t *testing.T, p *participant) {
	t.Helper()
	for i := 1; i <= 10; i++ {
		if _, err := p.reserve(fmt.Sprintf("iss-%d", i), fmt.Sprintf("f%d.go", i)); err != nil {
			t.Fatalf("reservation %d: %v", i, err)
		}
	}
}

// TestTheEleventhReservationIsRefusedNamingTheCap is C47: a reservation that
// would take its holder past ten live reservations is refused, naming the cap
// and the ten, and nothing is pushed.
func TestTheEleventhReservationIsRefusedNamingTheCap(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	reserveTen(t, p)
	_, err := p.reserve("iss-11", "f11.go")
	want := "you hold 10 live reservations, the most one person may hold: iss-1, iss-10, iss-2, iss-3, iss-4, iss-5, " +
		"iss-6, iss-7, iss-8, iss-9; release one (abcd reserve <id> --release) first; nothing was pushed"
	if !errors.Is(err, ErrRefused) || err.Error() != want {
		t.Fatalf("the eleventh reservation = %v; want %q", err, want)
	}
	if tip := w.remoteTip("iss-11"); tip != "" {
		t.Fatalf("a reservation over the cap reached origin: %s", tip)
	}
	// Another person's count is their own: Q is not refused by P's ten.
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := q.reserve("iss-11", "f11.go"); err != nil {
		t.Fatalf("Q's first reservation, beside P's ten: %v", err)
	}
}

// TestTheCapNeverRefusesARenewalOrATakeOverOfOnesOwnCopy: a renewal and a
// take-over of one's own other copy add no reservation, so a holder at the
// cap makes both.
func TestTheCapNeverRefusesARenewalOrATakeOverOfOnesOwnCopy(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	reserveTen(t, p)
	w.set(t0.Add(time.Hour))
	if res, err := p.reserve("iss-1", "f1.go"); err != nil || res.Act != ActRenewed {
		t.Fatalf("a renewal at the cap = %+v, %v; want it renewed", res, err)
	}
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := laptop.do(Request{Record: "iss-2", Footprint: []string{"f2.go"}, TakeOver: true})
	if err != nil || res.Act != ActTakenOver {
		t.Fatalf("a take-over of one's own other copy at the cap = %+v, %v; want it taken over", res, err)
	}
}

// TestTheCapCountsNeitherReleasedNorExpiredReservations: a tombstone and a
// lapsed lease hold nothing, so they leave room under the cap.
func TestTheCapCountsNeitherReleasedNorExpiredReservations(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	for i := 1; i <= 9; i++ {
		if _, err := p.reserve(fmt.Sprintf("iss-%d", i), fmt.Sprintf("f%d.go", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := p.do(Request{Record: "iss-10", Footprint: []string{"f10.go"}, Lease: time.Minute}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-1", Release: true}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(10 * time.Minute)) // iss-10 has lapsed: eight live
	for _, id := range []string{"iss-11", "iss-12"} {
		if _, err := p.reserve(id, id+".go"); err != nil {
			t.Fatalf("%s, with a released and an expired reservation beside eight live ones: %v", id, err)
		}
	}
	if _, err := p.reserve("iss-13", "iss-13.go"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 10 live reservations") {
		t.Fatalf("the reservation past ten live ones = %v; want the cap refusal", err)
	}
}

// TestAnOverlappingFootprintIsRefusedNamingTheSharedPath is C48: a footprint
// sharing a path with another live reservation, whoever holds it, is refused
// with the back-off, naming that reservation, its holder and the shared path.
// Two paths are shared when equal or when one is a parent directory of the
// other, segment by segment; the path named is the deeper one, which lies in
// both footprints.
func TestAnOverlappingFootprintIsRefusedNamingTheSharedPath(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "commands/reserve.md", "internal/core/claim"); err != nil {
		t.Fatal(err)
	}
	const held = ", reserved by octo until 2026-10-03T16:00:00Z; take other work; nothing was pushed"
	for name, c := range map[string]struct {
		who  *participant
		id   string
		fp   []string
		path string
	}{
		"an equal path":                     {q, "iss-2", []string{"docs/a.md", "internal/core/claim"}, "internal/core/claim"},
		"a file under a reserved directory": {q, "iss-3", []string{"internal/core/claim/claim.go"}, "internal/core/claim/claim.go"},
		"a directory over a reserved file":  {q, "iss-4", []string{"commands"}, "commands/reserve.md"},
		"the holder's own other record":     {p, "iss-5", []string{"internal/core/claim/view.go"}, "internal/core/claim/view.go"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := c.who.reserve(c.id, c.fp...)
			want := `"` + c.path + `" is in the footprint of iss-1` + held
			if !errors.Is(err, ErrBackOff) || err.Error() != want {
				t.Fatalf("reserve %v = %v; want %q", c.fp, err, want)
			}
			if tip := w.remoteTip(c.id); tip != "" {
				t.Fatalf("an overlapping reservation reached origin: %s", tip)
			}
		})
	}
	// Segment by segment: a sibling whose name merely begins the same is not
	// shared.
	if _, err := q.reserve("iss-9", "internal/core/claimx", "commands-old/reserve.md"); err != nil {
		t.Fatalf("a footprint that shares only a name's beginning: %v", err)
	}
}

// TestARenewalIsNeverWidenedPastAnotherLiveReservation: a renewal re-checks its
// footprint against the view, so it cannot widen onto another reservation.
func TestARenewalIsNeverWidenedPastAnotherLiveReservation(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.reserve("iss-2", "b.go"); err != nil {
		t.Fatal(err)
	}
	tip := w.remoteTip("iss-1")
	_, err := p.reserve("iss-1", "a.go", "b.go")
	want := `"b.go" is in the footprint of iss-2,` + " reserved by quinn until 2026-10-03T16:00:00Z; take other work; nothing was pushed"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("a widening renewal = %v; want %q", err, want)
	}
	if got := w.remoteTip("iss-1"); got != tip {
		t.Fatalf("the refused renewal moved the branch to %s", got)
	}
	if res, err := p.reserve("iss-1", "a.go"); err != nil || res.Act != ActRenewed {
		t.Fatalf("a renewal over its own footprint = %+v, %v; want it renewed", res, err)
	}
}

// TestNeitherReleasedNorExpiredReservationsOverlap: a tombstone's and a lapsed
// lease's footprints hold no path.
func TestNeitherReleasedNorExpiredReservationsOverlap(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "iss-1", Footprint: []string{"a.go"}, Lease: time.Minute}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.reserve("iss-2", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-2", Release: true}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(10 * time.Minute))
	if _, err := q.reserve("iss-3", "a.go", "b.go"); err != nil {
		t.Fatalf("a footprint over an expired and a released reservation: %v", err)
	}
}
