package claim

import (
	"errors"
	"os"
	"path/filepath"
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

// TestAReservationALittleAheadIsJudgedByItsOwnDates: a claim dated exactly
// lease.Skew ahead of this copy's fetch, as two honest clocks may differ, is
// judged by its own dates and nothing is kept for it, while one dated a second
// further ahead is counted from that fetch and its time is kept. Each has a
// lease of lease.Max, so the two rules give different expiries.
func TestAReservationALittleAheadIsJudgedByItsOwnDates(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	c5, c6 := t0.Add(lease.Skew), t0.Add(lease.Skew+time.Second)
	within := w.pushTipAt("iss-5", reservationText("iss-5", c5, c5.Add(lease.Max)), c5)
	past := w.pushTipAt("iss-6", reservationText("iss-6", c6, c6.Add(lease.Max)), c6)
	q.fetchAt(t0)

	if kept(q, within) {
		t.Fatalf("a time was kept for the tip dated %s ahead of the fetch; want none", lease.Skew)
	}
	if !kept(q, past) {
		t.Fatalf("no time was kept for the tip dated %s ahead of the fetch; want the fetch's", lease.Skew+time.Second)
	}
	snap, err := View(q.dir, t0.Add(lease.Max+time.Minute))
	if err != nil || len(snap.Reservations) != 2 {
		t.Fatalf("view = %+v, %v; want the two reservations", snap, err)
	}
	got := map[string]Reservation{}
	for _, r := range snap.Reservations {
		got[r.Record] = r
	}
	if r := got["iss-5"]; r.State != StateLive || !r.EffectiveExpiry.Equal(c5.Add(lease.Max)) {
		t.Fatalf("a claim %s ahead = %+v; want live until its own expiry %s", lease.Skew, r, c5.Add(lease.Max))
	}
	if r := got["iss-6"]; r.State != StateExpired || !r.EffectiveExpiry.Equal(t0.Add(lease.Max)) {
		t.Fatalf("a claim %s ahead = %+v; want expired a day after the fetch, %s", lease.Skew+time.Second, r, t0.Add(lease.Max))
	}
}

// pushTipAt pushes a commit with msg on the empty tree, dated at, as the
// reservation branch for id on origin.
func (w *world) pushTipAt(id, msg string, at time.Time) string {
	w.t.Helper()
	return w.pushHostile(id, msg, at)
}

// kept reports whether p's local tier keeps a first-fetch time for commit.
func kept(p *participant, commit string) bool {
	_, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(undatedRelDir), commit))
	return err == nil
}

// TestAReservationAheadWhenFirstFetchedStaysCountedFromThatFetch: a claim dated
// more than lease.Skew ahead of this copy's first fetch stays counted from that
// fetch once the clock passes its own date, at a later fetch, in the view and
// in the act, so it lapses a day after the first fetch and never holds again.
func TestAReservationAheadWhenFirstFetchedStaysCountedFromThatFetch(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	c5, c6 := t0.Add(23*time.Hour), t0.Add(30*time.Hour)
	bad := w.pushTipAt("iss-5", reservationText("iss-5", c5, c5.Add(lease.Max)), c5)
	w.pushTipAt("iss-6", reservationText("iss-6", c6, c6.Add(lease.Max)), c6)
	q.fetchAt(t0)
	q.fetchAt(t0.Add(23*time.Hour + 10*time.Minute))

	snap, err := View(q.dir, t0.Add(23*time.Hour+10*time.Minute))
	if err != nil || len(snap.Reservations) != 2 {
		t.Fatalf("view = %+v, %v; want the two reservations", snap, err)
	}
	if r := snap.Reservations[0]; r.Record != "iss-5" || r.State != StateLive || !r.EffectiveExpiry.Equal(t0.Add(lease.Max)) {
		t.Fatalf("a claim 23h ahead, ten minutes past its own date = %+v; want live until %s", r, t0.Add(lease.Max))
	}
	q.fetchAt(t0.Add(30*time.Hour + 10*time.Minute))
	snap, err = View(q.dir, t0.Add(30*time.Hour+10*time.Minute))
	if err != nil || len(snap.Reservations) != 2 {
		t.Fatalf("view = %+v, %v; want the two reservations", snap, err)
	}
	for _, r := range snap.Reservations {
		if r.State != StateExpired || !r.EffectiveExpiry.Equal(t0.Add(lease.Max)) {
			t.Fatalf("%s past its own date = %+v; want expired a day after the first fetch, %s", r.Record, r, t0.Add(lease.Max))
		}
	}
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Commit != bad {
		t.Fatalf("Q's reserve past the day and the margin = %+v, %v; want %s taken over", res, err, bad)
	}
}

// TestTheViewCountsATipAheadThatNothingHereKeptFromNow: a tip dated far ahead
// that reached this copy by a plain git fetch, after abcd's last fetch, has no
// first-fetch time kept; the view counts it from its own now, as the act
// does, rather than from that last fetch, so it never shows a reservation the
// act refuses as expired.
func TestTheViewCountsATipAheadThatNothingHereKeptFromNow(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	q.fetchAt(t0)
	ahead := t0.Add(fiveYears)
	w.pushTipAt("iss-5", reservationText("iss-5", ahead, ahead.Add(lease.Max)), ahead)
	q.git("fetch", "-q", "origin", "+"+RefPrefix+"*:"+TrackingPrefix+"*")

	now := t0.Add(lease.Max + time.Hour)
	snap, err := View(q.dir, now)
	if err != nil || len(snap.Reservations) != 1 {
		t.Fatalf("view = %+v, %v; want the one reservation", snap, err)
	}
	if r := snap.Reservations[0]; r.State != StateLive || !r.EffectiveExpiry.Equal(now.Add(lease.Max)) {
		t.Fatalf("a tip ahead that nothing here kept = %+v; want live until %s", r, now.Add(lease.Max))
	}
	w.set(now)
	if _, err := q.reserve("iss-5", "b.go"); !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "until "+stamp(now.Add(lease.Max))) {
		t.Fatalf("Q's reserve = %v; want a back-off until %s", err, stamp(now.Add(lease.Max)))
	}
}

// TestAFirstFetchKeptWhileTheClockRanFastIsKeptAgainByTheNextAct: a first
// fetch kept while this copy's clock ran more than lease.Skew fast is kept
// again as the next act's time once the clock is right, so the tip lapses a
// day after that act rather than a day after the fast time.
func TestAFirstFetchKeptWhileTheClockRanFastIsKeptAgainByTheNextAct(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	ahead := t0.Add(fiveYears)
	bad := w.pushTipAt("iss-5", reservationText("iss-5", ahead, ahead.Add(lease.Max)), ahead)
	q.fetchAt(t0.Add(10 * time.Hour))

	w.set(t0)
	if _, err := q.reserve("iss-5", "b.go"); !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "until "+stamp(t0.Add(lease.Max))) {
		t.Fatalf("Q's reserve with the clock right = %v; want a back-off until %s", err, stamp(t0.Add(lease.Max)))
	}
	snap, err := View(q.dir, t0.Add(lease.Max))
	if err != nil || len(snap.Reservations) != 1 || snap.Reservations[0].State != StateExpired {
		t.Fatalf("view a day after the act = %+v, %v; want expired", snap, err)
	}
	w.set(t0.Add(lease.Max + TakeoverMargin))
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Commit != bad {
		t.Fatalf("Q's reserve a day and the margin after the act = %+v, %v; want %s taken over", res, err, bad)
	}
}
