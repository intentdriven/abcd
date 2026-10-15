package claim

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// A reservation dated far ahead of the reader's clock is counted from when
// this copy first fetched it, so it lapses a day after that sighting (or an
// unreadable one its default hold after it) rather than holding for a day
// from every reading; one dated a little ahead, within lease.Skew, is judged
// by its own dates as before.

// fiveYears is how far ahead the tips below are dated.
const fiveYears = 5 * 365 * 24 * time.Hour

// reservationText is a readable reservation message for id, held by octo,
// claimed at claimed and expiring at expires.
func reservationText(id string, claimed, expires time.Time) string {
	return "abcd reservation: " + id + "\n\nAbcd-Reservation: " + id + "\nAbcd-Holder: octo\n" +
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Footprint: a.go\n" +
		"Abcd-Claimed-At: " + stamp(claimed) + "\nAbcd-Expires-At: " + stamp(expires) + "\n"
}

// fetchAt fetches the reservations into p at the world's time at.
func (p *participant) fetchAt(at time.Time) {
	p.w.t.Helper()
	p.w.set(at)
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		p.w.t.Fatalf("fetching at %s: %v", at, err)
	}
}

// TestAReservationClaimedFarAheadLapsesADayAfterItWasFirstFetched: a
// reservation claimed five years ahead, with an expiry a day after that,
// holds from this copy's first fetch of it for lease.Max and no longer; a
// later fetch does not restart it, and once lapsed it is taken over as any
// lapsed reservation is.
func TestAReservationClaimedFarAheadLapsesADayAfterItWasFirstFetched(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	ahead := t0.Add(fiveYears)
	bad := w.pushHostile("iss-5", reservationText("iss-5", ahead, ahead.Add(lease.Max)), ahead)
	q.fetchAt(t0)
	q.fetchAt(t0.Add(12 * time.Hour))

	snap, err := View(q.dir, t0.Add(lease.Max-time.Second))
	if err != nil || len(snap.Reservations) != 1 {
		t.Fatalf("view = %+v, %v; want the one reservation", snap, err)
	}
	if r := snap.Reservations[0]; r.State != StateLive || !r.EffectiveExpiry.Equal(t0.Add(lease.Max)) {
		t.Fatalf("a second before a day from the first fetch = %+v; want live until %s", r, t0.Add(lease.Max))
	}
	snap, err = View(q.dir, t0.Add(lease.Max))
	if err != nil || len(snap.Reservations) != 1 || snap.Reservations[0].State != StateExpired {
		t.Fatalf("a day after the first fetch = %+v, %v; want expired", snap, err)
	}

	w.set(t0.Add(lease.Max - time.Second))
	if _, err := q.reserve("iss-5", "b.go"); !errors.Is(err, ErrBackOff) ||
		!strings.Contains(err.Error(), "until "+stamp(t0.Add(lease.Max))) {
		t.Fatalf("Q's reserve within the day = %v; want a back-off naming %s", err, stamp(t0.Add(lease.Max)))
	}
	w.set(t0.Add(lease.Max + TakeoverMargin))
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Commit != bad {
		t.Fatalf("Q's reserve past the day and the margin = %+v, %v; want %s taken over", res, err, bad)
	}
}

// TestAnUnreadableTipDatedFarAheadIsHeldFromItsFirstFetch: an unreadable tip
// whose committer date is five years ahead is held for the default hold from
// when this copy first fetched it, not from every reading.
func TestAnUnreadableTipDatedFarAheadIsHeldFromItsFirstFetch(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	bad := w.pushHostile("iss-5", "abcd reservation: iss-5\n\nnot a reservation\n", t0.Add(fiveYears))
	q.fetchAt(t0)
	q.fetchAt(t0.Add(time.Hour))

	snap, err := View(q.dir, t0.Add(lease.Default-time.Second))
	if err != nil || len(snap.Reservations) != 1 {
		t.Fatalf("view = %+v, %v; want the one tip", snap, err)
	}
	if r := snap.Reservations[0]; r.State != StateUnreadable || !r.EffectiveExpiry.Equal(t0.Add(lease.Default)) {
		t.Fatalf("within the hold = %+v; want unreadable, held until %s", r, t0.Add(lease.Default))
	}
	w.set(t0.Add(lease.Default + TakeoverMargin))
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Commit != bad {
		t.Fatalf("Q's reserve past the hold and the margin = %+v, %v; want %s taken over", res, err, bad)
	}
}

// TestAReservationAheadWithoutAFetchIsCountedFromTheAct: an act that meets a
// far-ahead tip no fetch here recorded counts it from its own fetch, and keeps
// that time, so the next act does not restart it.
func TestAReservationAheadWithoutAFetchIsCountedFromTheAct(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	ahead := t0.Add(fiveYears)
	w.pushHostile("iss-5", reservationText("iss-5", ahead, ahead.Add(lease.Max)), ahead)
	w.set(t0)
	if _, err := q.reserve("iss-5", "b.go"); !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "until "+stamp(t0.Add(lease.Max))) {
		t.Fatalf("Q's first reserve = %v; want a back-off until %s", err, stamp(t0.Add(lease.Max)))
	}
	w.set(t0.Add(20 * time.Hour))
	if _, err := q.reserve("iss-5", "b.go"); !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "until "+stamp(t0.Add(lease.Max))) {
		t.Fatalf("Q's later reserve = %v; want the same back-off, until %s", err, stamp(t0.Add(lease.Max)))
	}
}

// TestAReservationALittleAheadIsJudgedByItsOwnDates: a claim dated within
// lease.Skew of the reader, as two honest clocks may differ, holds until its
// own expiry, and its first fetch changes nothing.
func TestAReservationALittleAheadIsJudgedByItsOwnDates(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	c := t0.Add(lease.Skew)
	w.pushHostile("iss-5", reservationText("iss-5", c, c.Add(lease.Default)), c)
	q.fetchAt(t0.Add(-time.Hour))
	snap, err := View(q.dir, t0)
	if err != nil || len(snap.Reservations) != 1 {
		t.Fatalf("view = %+v, %v; want the one reservation", snap, err)
	}
	if r := snap.Reservations[0]; r.State != StateLive || !r.EffectiveExpiry.Equal(c.Add(lease.Default)) {
		t.Fatalf("a claim %s ahead = %+v; want live until its own expiry %s", lease.Skew, r, c.Add(lease.Default))
	}
}
