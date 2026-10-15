package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// notHeld is err's *NotHeld, failing the test when it carries none.
func notHeld(t *testing.T, err error, why string) *NotHeld {
	t.Helper()
	var nh *NotHeld
	if !errors.As(err, &nh) {
		t.Fatalf("err = %v; want a *NotHeld (%s)", err, why)
	}
	if nh.Why != why {
		t.Fatalf("NotHeld.Why = %q; want %q (%v)", nh.Why, why, err)
	}
	if !strings.HasSuffix(err.Error(), nothingPushed) {
		t.Fatalf("a renewal refused says nothing was pushed: %q", err)
	}
	return nh
}

// TestARenewalRenewsOnlyThisCopysOwnReservation is the implement loop's
// renewal (spec 2, step 8): a renew-only act renews this person's reservation
// from this copy, live or lapsed, keeping its footprint, and refuses every
// other tip with a *NotHeld naming why, pushing nothing. It never reserves an
// absent or released record and never takes one over, so a run's renewal
// cannot undo a release (C36) or move another's reservation.
func TestARenewalRenewsOnlyThisCopysOwnReservation(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p2 := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	renew := func(who *participant, id string, live bool) (Result, error) {
		return who.do(Request{Record: id, Renew: true, Live: live})
	}

	if _, err := renew(p, "iss-5", false); err != nil {
		notHeld(t, err, NotHeldAbsent)
	} else {
		t.Fatal("renewing a record nobody reserved reserves it")
	}
	if w.remoteTip("iss-5") != "" {
		t.Fatal("a refused renewal pushed a reservation")
	}

	if _, err := p.reserve("iss-5", "a.go", "b"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(time.Hour))
	res, err := renew(p, "iss-5", true)
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("P renews its own: %+v %v", res, err)
	}
	if strings.Join(res.Footprint, ",") != "a.go,b" || !res.ClaimedAt.Equal(t0.Add(time.Hour)) ||
		!res.ExpiresAt.Equal(t0.Add(3*time.Hour)) {
		t.Fatalf("the renewal keeps the footprint and counts the default lease from now: %+v", res)
	}
	if !strings.Contains(w.remoteMessage(w.remoteTip("iss-5")), "Abcd-Footprint: b\n") {
		t.Fatal("the renewal's commit drops the footprint")
	}
	tip := w.remoteTip("iss-5")

	if _, err := p.do(Request{Record: "iss-5", Renew: true, Footprint: []string{"c.go"}}); !errors.Is(err, ErrRefused) {
		t.Fatalf("a renewal names no footprint of its own: %v", err)
	}
	nh := notHeld(t, func() error { _, err := renew(q, "iss-5", false); return err }(), NotHeldAnotherPerson)
	if nh.Holder != "octo" || !nh.Expiry.Equal(t0.Add(3*time.Hour)) {
		t.Fatalf("the refusal names the holder and the expiry: %+v", nh)
	}
	notHeld(t, func() error { _, err := renew(p2, "iss-5", false); return err }(), NotHeldAnotherCopy)
	if w.remoteTip("iss-5") != tip {
		t.Fatal("a refused renewal moved the branch")
	}

	// Lapsed: renewed when asked for any of one's own, refused when it must be
	// live; another person's lapsed one is never taken over.
	w.set(t0.Add(5 * time.Hour))
	notHeld(t, func() error { _, err := renew(p, "iss-5", true); return err }(), NotHeldLapsed)
	notHeld(t, func() error { _, err := renew(q, "iss-5", false); return err }(), NotHeldAnotherPerson)
	if w.remoteTip("iss-5") != tip {
		t.Fatal("a refused renewal moved the branch")
	}
	if res, err := renew(p, "iss-5", false); err != nil || res.Act != ActRenewed {
		t.Fatalf("P renews its own lapsed reservation: %+v %v", res, err)
	}

	// Released: a renewal never reserves the record again.
	if _, err := p.do(Request{Record: "iss-5", Release: true}); err != nil {
		t.Fatal(err)
	}
	tomb := w.remoteTip("iss-5")
	nh = notHeld(t, func() error { _, err := renew(p, "iss-5", false); return err }(), NotHeldReleased)
	if nh.ReleasedAt.IsZero() || nh.Derived {
		t.Fatalf("a tombstone's release names its time: %+v", nh)
	}
	if w.remoteTip("iss-5") != tomb {
		t.Fatal("a renewal over a tombstone reserved the record again")
	}
}

// TestARenewOnlyActIsNeverRefusedByAnOverlap: a renew-only act keeps the tip's
// own footprint, so it cannot widen it, and it is not judged for overlap. Two
// reservations with overlapping footprints, which a race between two people
// can let in (each reservation is its own ref, so the overlap is not
// compare-and-swapped), each renew; otherwise neither lane could renew or
// land until one lapsed.
func TestARenewOnlyActIsNeverRefusedByAnOverlap(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	// Q's overlapping reservation, as the race lets it in.
	w.pushHostile("iss-2", "abcd reservation: iss-2\n\nAbcd-Reservation: iss-2\nAbcd-Holder: quinn\n"+
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Footprint: a.go\n"+
		"Abcd-Claimed-At: 2026-10-03T14:00:00Z\nAbcd-Expires-At: 2026-10-03T16:00:00Z\n", t0)
	w.set(t0.Add(time.Hour))
	if res, err := p.do(Request{Record: "iss-1", Renew: true}); err != nil || res.Act != ActRenewed {
		t.Fatalf("P's renew-only act over an overlap the race let in = %+v, %v; want it renewed", res, err)
	}
	// A re-reserve naming a footprint is still checked: it could widen.
	if _, err := p.reserve("iss-1", "a.go"); !errors.Is(err, ErrBackOff) {
		t.Fatalf("a re-reserve naming its footprint is checked for overlap: %v", err)
	}
}

// TestRevivingALapsedReservationIsJudgedForOverlap: a renew-only act over one's
// own lapsed tip revives a reservation that held nothing while it lapsed, and
// another may have reserved its paths since, so it is judged for overlap like
// a new one.
func TestRevivingALapsedReservationIsJudgedForOverlap(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "iss-1", Footprint: []string{"a.go"}, Lease: time.Minute}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(2 * time.Minute))
	if _, err := q.reserve("iss-2", "a.go"); err != nil {
		t.Fatalf("Q reserves the path P's lapsed reservation named: %v", err)
	}
	tip := w.remoteTip("iss-1")
	w.set(t0.Add(3 * time.Minute))
	_, err := p.do(Request{Record: "iss-1", Renew: true})
	want := `"a.go" is in the footprint of iss-2, reserved by quinn until 2026-10-03T16:02:00Z; take other work; nothing was pushed`
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("reviving P's lapsed iss-1 over Q's iss-2 = %v; want %q", err, want)
	}
	if got := w.remoteTip("iss-1"); got != tip {
		t.Fatalf("the refused revival moved the branch to %s", got)
	}
}
