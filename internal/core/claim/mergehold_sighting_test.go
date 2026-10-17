package claim

import (
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// The merge hold counts a release reservation dated more than lease.Skew
// ahead from when this copy first saw it, whatever fetch brought it: the
// judgement keeps that first sighting itself, so a caller whose own act
// fetched the reservations (the land fence's renewal) is judged as `abcd land`
// is. Where no sighting can be kept, no local tier or one that cannot be
// written, such a release is not counted as a hold at all: it is returned as
// dated ahead and not counted, for the caller to name.

// releaseText is a readable release reservation message for subject, held by
// quinn from another copy, claimed at claimed and expiring at expires.
func releaseText(subject string, claimed, expires time.Time) string {
	return "abcd reservation: " + subject + "\n\nAbcd-Reservation: " + subject + "\nAbcd-Holder: quinn\n" +
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n" +
		"Abcd-Claimed-At: " + stamp(claimed) + "\nAbcd-Expires-At: " + stamp(expires) + "\n"
}

// TestTheMergeHoldKeepsTheFirstSightingOfAReleaseDatedAhead: a release dated
// five years ahead, readable or not, reaches this copy through a fetch that
// keeps no sighting (a reservation act's own, as the land fence's renewal
// makes). The merge hold keeps the first sighting when it judges, so the hold
// ends a day (or the default hold) after it and never slides with the clock.
func TestTheMergeHoldKeepsTheFirstSightingOfAReleaseDatedAhead(t *testing.T) {
	ahead := t0.Add(fiveYears)
	for _, c := range []struct {
		name string
		msg  string
		hold time.Duration
	}{
		{"readable", releaseText("rel-v0.16.0", ahead, ahead.Add(lease.Max)), lease.Max},
		{"unreadable", "abcd reservation: rel-v0.16.0\n\nnot a reservation\n", lease.Default},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			w.pushTipAt("rel-v0.16.0", c.msg, ahead)
			if _, err := fetch(p.dir, t0, noReservation, reservations); err != nil {
				t.Fatal(err)
			}
			for _, at := range []time.Duration{0, c.hold / 2, c.hold - time.Second} {
				m, err := MergeHolds(p.dir, t0.Add(at), p.opts())
				if err != nil || !m.Held() || !m.Holds[0].EffectiveExpiry.Equal(t0.Add(c.hold)) {
					t.Fatalf("at +%s: %+v, %v; want held until %s", at, m, err, stamp(t0.Add(c.hold)))
				}
			}
			if m, err := MergeHolds(p.dir, t0.Add(c.hold), p.opts()); err != nil || m.Held() {
				t.Fatalf("at +%s: held %v (%q), %v; want the hold ended %s after it was first seen", c.hold, m.Held(), m.Reason(), err, c.hold)
			}
		})
	}
}
