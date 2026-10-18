package claim

import (
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// Renewal is one reservation RenewDue renewed, or failed to: Until is the
// renewed lease's end, and Err why it could not be renewed.
type Renewal struct {
	Record string
	Until  time.Time
	Err    error
}

// RenewDue renews this person's own live reservations from this copy that
// are half through their lease, and nothing else: the renewal a holder
// waiting at the load gate gets, so a reservation made by hand does not lapse
// while its preflight waits its turn (iss-2610170926295770). It reads the
// reservations as last fetched (View), which asks the network nothing, and
// each renewal is a renew-only, live-only act (Request.Renew, Request.Live):
// it never reserves a record anew, never takes one over and never revives one
// that has lapsed. A renewal keeps the tip's own term (renewalTerm). A
// release subject is left alone: its hold is a release cut's, kept by the
// cut. The error is a view that could not be read; each renewal's own
// failure is in its Renewal.
func RenewDue(root string, o Options) ([]Renewal, error) {
	now := o.now()
	snap, err := View(root, now)
	if err != nil {
		return nil, err
	}
	var out []Renewal
	for _, r := range snap.Reservations {
		if r.Copy != CopyThis || r.State != StateLive || isRelease(r.Record) || !halfThrough(r, now) {
			continue
		}
		res, err := Reserve(root, Request{Record: r.Record, Renew: true, Live: true, Lease: renewalTerm(r)}, o)
		out = append(out, Renewal{Record: r.Record, Until: res.ExpiresAt, Err: err})
	}
	return out, nil
}

// halfThrough reports whether r is half through its lease at now, counted from
// its claimed-at to its effective expiry: the point a renewal is due.
func halfThrough(r Reservation, now time.Time) bool {
	return !now.Before(r.ClaimedAt.Add(r.EffectiveExpiry.Sub(r.ClaimedAt) / 2))
}

// renewalTerm is the lease a renewal of r pushes: r's own term, never shorter
// than the default nor longer than a day, so a renewal never cuts a longer
// term a fence gave a pull request waiting on a person.
func renewalTerm(r Reservation) time.Duration {
	return min(max(lease.Default, r.EffectiveExpiry.Sub(r.ClaimedAt)), lease.Max)
}
