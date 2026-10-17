package lease

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 3, 14, 0, 0, 0, time.UTC)

// TestALiveStampHoldsUntilItsExpiry is the ordinary case on both sides of the
// expiry: live a second before it, lapsed at it.
func TestALiveStampHoldsUntilItsExpiry(t *testing.T) {
	s := Stamp{ClaimedAt: t0, ExpiresAt: t0.Add(Default)}
	v := Judge(s, time.Minute, t0.Add(Default-time.Second))
	if !v.Live || v.Unreadable || !v.EffectiveExpiry.Equal(t0.Add(Default)) {
		t.Fatalf("a second before the expiry = %+v, want live until %s", v, t0.Add(Default))
	}
	if v := Judge(s, time.Minute, t0.Add(Default)); v.Live {
		t.Fatalf("at the expiry = %+v, want lapsed", v)
	}
}

// TestTheEffectiveExpiryIsClampedToADay is C10: a stamp whose expires_at is
// thirty days after its claimed_at holds only until claimed_at plus 24 hours.
func TestTheEffectiveExpiryIsClampedToADay(t *testing.T) {
	s := Stamp{ClaimedAt: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	v := Judge(s, time.Minute, t0.Add(Max-time.Second))
	if !v.Live || !v.EffectiveExpiry.Equal(t0.Add(Max)) {
		t.Fatalf("a second before the clamp = %+v, want live until %s", v, t0.Add(Max))
	}
	if v := Judge(s, time.Minute, t0.Add(Max)); v.Live {
		t.Fatalf("at claimed_at plus a day = %+v, want lapsed although expires_at is weeks away", v)
	}
}

// TestAnUnreadableStampIsHeldForItsGrace holds an unreadable stamp from the
// time it was written for the grace the caller names, and no longer, whatever
// times it might carry.
func TestAnUnreadableStampIsHeldForItsGrace(t *testing.T) {
	s := Stamp{Unreadable: true, WrittenAt: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	v := Judge(s, 2*time.Hour, t0.Add(2*time.Hour-time.Second))
	if !v.Live || !v.Unreadable || !v.EffectiveExpiry.Equal(t0.Add(2*time.Hour)) {
		t.Fatalf("within the grace = %+v, want held until %s", v, t0.Add(2*time.Hour))
	}
	v = Judge(s, 2*time.Hour, t0.Add(2*time.Hour))
	if v.Live || !v.Unreadable {
		t.Fatalf("past the grace = %+v, want lapsed and still unreadable", v)
	}
}

// TestTheBoundsAreTheShippedOnes pins the three bounds the in-run claim
// shipped with.
func TestTheBoundsAreTheShippedOnes(t *testing.T) {
	if Default != 2*time.Hour || Min != time.Minute || Max != 24*time.Hour {
		t.Fatalf("Default, Min, Max = %s, %s, %s; want 2h, 1m, 24h", Default, Min, Max)
	}
}

// TestAFutureDatedStampHoldsNoMoreThanADayFromTheReader: a claim written by a
// clock running ahead (a laptop that thinks it is next year) is clamped from
// the reader's now rather than from its own claimed-at, so at no reading does
// it hold more than a day ahead; an unreadable stamp written ahead is held for
// its grace from the reader's now in the same way (review finding R4 on
// spc-2610031753359849 steps 1-3).
func TestAFutureDatedStampHoldsNoMoreThanADayFromTheReader(t *testing.T) {
	ahead := t0.Add(365 * 24 * time.Hour)
	v := Judge(Stamp{ClaimedAt: ahead, ExpiresAt: ahead.Add(Default)}, time.Minute, t0)
	if !v.Live || !v.EffectiveExpiry.Equal(t0.Add(Max)) {
		t.Fatalf("a claim dated a year ahead = %+v, want live until the reader's now plus a day, %s", v, t0.Add(Max))
	}
	v = Judge(Stamp{Unreadable: true, WrittenAt: ahead}, time.Minute, t0)
	if !v.Live || !v.EffectiveExpiry.Equal(t0.Add(time.Minute)) {
		t.Fatalf("an unreadable stamp written a year ahead = %+v, want held for its grace from now, %s", v, t0.Add(time.Minute))
	}
	// A claim dated in the past is unchanged.
	v = Judge(Stamp{ClaimedAt: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}, time.Minute, t0.Add(time.Hour))
	if !v.EffectiveExpiry.Equal(t0.Add(Max)) {
		t.Fatalf("a past claim = %+v, want clamped to its claimed-at plus a day", v)
	}
}

// TestAStampFarAheadIsCountedFromWhenItWasFirstSeen: a claim dated more than
// Skew ahead of the reader is counted from when the reader first saw it, so
// it stops holding a day after that however far ahead it is dated, and does
// not hold for a day from every reading. An unreadable stamp written ahead is
// held for its grace from the same first sighting.
func TestAStampFarAheadIsCountedFromWhenItWasFirstSeen(t *testing.T) {
	ahead := t0.Add(5 * 365 * 24 * time.Hour)
	s := Stamp{ClaimedAt: ahead, ExpiresAt: ahead.Add(Max), FirstSeen: t0}
	if v := Judge(s, time.Minute, t0.Add(Max-time.Second)); !v.Live || !v.EffectiveExpiry.Equal(t0.Add(Max)) {
		t.Fatalf("a claim five years ahead, a second before a day from its first sighting = %+v; want live until %s", v, t0.Add(Max))
	}
	for _, at := range []time.Duration{Max, Max + Skew, 30 * 24 * time.Hour} {
		if v := Judge(s, time.Minute, t0.Add(at)); v.Live || !v.EffectiveExpiry.Equal(t0.Add(Max)) {
			t.Fatalf("a claim five years ahead, %s after its first sighting = %+v; want lapsed at %s", at, v, t0.Add(Max))
		}
	}
	u := Stamp{Unreadable: true, WrittenAt: ahead, FirstSeen: t0}
	if v := Judge(u, 2*time.Hour, t0.Add(2*time.Hour-time.Second)); !v.Live || !v.EffectiveExpiry.Equal(t0.Add(2*time.Hour)) {
		t.Fatalf("an unreadable stamp written five years ahead, within its grace = %+v; want held until %s", v, t0.Add(2*time.Hour))
	}
	if v := Judge(u, 2*time.Hour, t0.Add(10*time.Hour)); v.Live || !v.EffectiveExpiry.Equal(t0.Add(2*time.Hour)) {
		t.Fatalf("an unreadable stamp written five years ahead, past its grace from its first sighting = %+v; want lapsed at %s", v, t0.Add(2*time.Hour))
	}
	// A first sighting dated after the reading is no later than the reading.
	if v := Judge(Stamp{ClaimedAt: ahead, ExpiresAt: ahead.Add(Max), FirstSeen: ahead}, time.Minute, t0); !v.EffectiveExpiry.Equal(t0.Add(Max)) {
		t.Fatalf("a first sighting ahead of the reader = %+v; want counted from the reader's now, %s", v, t0.Add(Max))
	}
}

// TestAStampWithinTheSkewIsJudgedByItsOwnDates: a claim dated up to Skew
// ahead of the reader, the small difference two honest clocks have, is judged
// exactly as before, whether its first sighting (here, now) is recorded or
// not.
func TestAStampWithinTheSkewIsJudgedByItsOwnDates(t *testing.T) {
	for _, seen := range []time.Time{{}, t0} {
		for _, ahead := range []time.Duration{0, time.Minute, Skew} {
			c := t0.Add(ahead)
			v := Judge(Stamp{ClaimedAt: c, ExpiresAt: c.Add(Default), FirstSeen: seen}, time.Minute, t0)
			if !v.Live || !v.EffectiveExpiry.Equal(c.Add(Default)) {
				t.Fatalf("a claim %s ahead, first seen %s = %+v; want live until its own expiry %s", ahead, seen, v, c.Add(Default))
			}
			v = Judge(Stamp{ClaimedAt: c, ExpiresAt: c.Add(30 * 24 * time.Hour), FirstSeen: seen}, time.Minute, t0)
			if !v.EffectiveExpiry.Equal(t0.Add(Max)) {
				t.Fatalf("a claim %s ahead with a far expiry, first seen %s = %+v; want clamped to the reader's now plus a day, %s", ahead, seen, v, t0.Add(Max))
			}
			v = Judge(Stamp{Unreadable: true, WrittenAt: c, FirstSeen: seen}, 2*time.Hour, t0)
			if !v.EffectiveExpiry.Equal(t0.Add(2 * time.Hour)) {
				t.Fatalf("an unreadable stamp written %s ahead, first seen %s = %+v; want held for its grace from the reader's now, %s", ahead, seen, v, t0.Add(2*time.Hour))
			}
		}
	}
}

// TestAStampAheadWhenFirstSeenStaysCountedFromThatSighting: a claim dated more
// than Skew ahead of its first sighting is counted from that sighting at every
// later reading, including once the reader's clock has passed the claim's own
// date, so it never comes back to life.
func TestAStampAheadWhenFirstSeenStaysCountedFromThatSighting(t *testing.T) {
	for _, tc := range []struct {
		ahead, at time.Duration
		live      bool
	}{
		{23 * time.Hour, 23*time.Hour + 10*time.Minute, true},
		{23 * time.Hour, Max, false},
		{30 * time.Hour, Max + 10*time.Minute, false},
		{30 * time.Hour, 30*time.Hour + 10*time.Minute, false},
		{30 * time.Hour, 60 * time.Hour, false},
	} {
		c := t0.Add(tc.ahead)
		v := Judge(Stamp{ClaimedAt: c, ExpiresAt: c.Add(Max), FirstSeen: t0}, time.Minute, t0.Add(tc.at))
		if v.Live != tc.live || !v.EffectiveExpiry.Equal(t0.Add(Max)) {
			t.Fatalf("a claim dated %s ahead, first seen at %s, judged %s later = %+v; want live=%v, expiry %s",
				tc.ahead, t0, tc.at, v, tc.live, t0.Add(Max))
		}
		u := Judge(Stamp{Unreadable: true, WrittenAt: c, FirstSeen: t0}, 2*time.Hour, t0.Add(tc.at))
		if u.Live || !u.EffectiveExpiry.Equal(t0.Add(2*time.Hour)) {
			t.Fatalf("an unreadable stamp written %s ahead, first seen at %s, judged %s later = %+v; want lapsed at %s",
				tc.ahead, t0, tc.at, u, t0.Add(2*time.Hour))
		}
	}
}

// TestAStampsExpiryNeverRisesAsTimePasses: for a fixed stamp with a recorded
// first sighting, the effective expiry never moves later as the reader's clock
// moves on: from the sighting for a claim dated more than Skew ahead of it,
// and from the claim's own date for one dated within Skew of it (before that
// date the reader's now is the clamp, as it always was).
func TestAStampsExpiryNeverRisesAsTimePasses(t *testing.T) {
	aheads := []time.Duration{-time.Hour, 0, time.Minute, Skew, Skew + time.Second, time.Hour,
		23 * time.Hour, Max, 30 * time.Hour, 5 * 365 * 24 * time.Hour}
	leases := []time.Duration{Min, Default, Max, 30 * 24 * time.Hour}
	for _, ahead := range aheads {
		c := t0.Add(ahead)
		start := t0
		if !Ahead(c, t0) && c.After(t0) {
			start = c
		}
		stamps := []Stamp{{Unreadable: true, WrittenAt: c, FirstSeen: t0}}
		for _, l := range leases {
			stamps = append(stamps, Stamp{ClaimedAt: c, ExpiresAt: c.Add(l), FirstSeen: t0})
		}
		for _, s := range stamps {
			var last time.Time
			for now := start; now.Before(t0.Add(72 * time.Hour)); now = now.Add(7 * time.Minute) {
				exp := Judge(s, 2*time.Hour, now).EffectiveExpiry
				if !last.IsZero() && exp.After(last) {
					t.Fatalf("stamp %+v: expiry rose from %s to %s at %s", s, last, exp, now)
				}
				last = exp
			}
		}
	}
}

// TestARecordedSightingChangesNothingForAClaimWithinTheSkewOfIt: a claim dated
// no more than Skew after its first sighting is judged exactly as one with no
// sighting recorded, at every reading from the sighting on, so counting from
// sightings never shortens a claim from a clock that differs by a little.
func TestARecordedSightingChangesNothingForAClaimWithinTheSkewOfIt(t *testing.T) {
	for _, ahead := range []time.Duration{-time.Hour, 0, time.Second, time.Minute, Skew} {
		c := t0.Add(ahead)
		for _, s := range []Stamp{
			{ClaimedAt: c, ExpiresAt: c.Add(Max)},
			{ClaimedAt: c, ExpiresAt: c.Add(Default)},
			{Unreadable: true, WrittenAt: c},
		} {
			seen := s
			seen.FirstSeen = t0
			for now := t0; now.Before(t0.Add(30 * time.Hour)); now = now.Add(time.Minute) {
				if a, b := Judge(seen, 2*time.Hour, now), Judge(s, 2*time.Hour, now); a != b {
					t.Fatalf("a claim dated %s after its sighting, judged at %s: %+v with the sighting, %+v without; want the same", ahead, now, a, b)
				}
			}
		}
	}
}

// TestTheSkewIsFiveMinutes pins the allowance for clocks that differ.
func TestTheSkewIsFiveMinutes(t *testing.T) {
	if Skew != 5*time.Minute {
		t.Fatalf("Skew = %s; want 5m", Skew)
	}
}
