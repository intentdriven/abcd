// Package lease is the one liveness judgement every claim in abcd is read
// through: the in-run claim a session takes before opening its lane
// (internal/core/implement) and the reservation a person pushes to the shared
// remote (internal/core/claim). Both are leases, a claim that holds until a
// time its writer chose, within bounds, and lapses on its own when the writer
// is gone; both meet claims nobody can parse. The judgement of the two is one
// function, so a claim the status shows as live is a claim the verbs refuse
// to take, whichever kind it is.
//
// The package is a leaf: it reads no file, runs no command and knows neither
// kind of claim.
package lease

import "time"

// Default is how long a claim holds when the caller names no lease: long
// enough for a lane's build-and-gate cycle, short enough that a session that
// died mid-lane frees its record within the same working window.
const Default = 2 * time.Hour

// Lease bounds. A lease shorter than a minute lapses under its own gate run; a
// lease longer than a day outlives any window a run keeps. Max is also the
// ceiling Judge holds every readable claim to, whatever expiry it states.
const (
	Min = time.Minute
	Max = 24 * time.Hour
)

// Skew is how far ahead of the reader's clock a claim may be dated and still
// be judged by its own dates: the allowance for two honest clocks that differ
// by a little. A claim dated further ahead is counted from when the reader
// first saw it (Stamp.FirstSeen).
const Skew = 5 * time.Minute

// Stamp is what a claim says about its own lifetime, as read.
type Stamp struct {
	// ClaimedAt and ExpiresAt are the claim's own times. They are read only
	// when the claim is readable.
	ClaimedAt time.Time
	ExpiresAt time.Time
	// Unreadable marks a claim nobody can parse: who holds it and until when
	// cannot be said, so it is held from WrittenAt for the caller's grace.
	Unreadable bool
	// WrittenAt is when the claim was last written, as the store records it
	// rather than as the claim says: a claim file's modification time, a
	// reservation commit's committer date.
	WrittenAt time.Time
	// FirstSeen is when the reader first saw this claim, kept by the reader
	// rather than read from the claim. It is read only when the claim's own
	// date (ClaimedAt, or WrittenAt when unreadable) is more than Skew ahead
	// of now; zero then counts from now.
	FirstSeen time.Time
}

// Verdict is a claim judged at a moment.
type Verdict struct {
	// EffectiveExpiry is when the claim stops holding: a readable claim's
	// expiry clamped to a day after it was claimed (or after now, when it is
	// dated a little ahead of the reader, or after it was first seen, when it
	// is dated more than Skew ahead), or an unreadable one's WrittenAt (or
	// now, or its first sighting, on the same terms) plus the grace.
	EffectiveExpiry time.Time
	// Live is true while now is before EffectiveExpiry.
	Live bool
	// Unreadable carries the stamp's own mark, so a caller rendering the
	// verdict says why it holds.
	Unreadable bool
}

// Judge says whether a claim holds at now. A readable claim's effective expiry
// is min(ExpiresAt, min(from, now) + Max) and an unreadable one is held until
// min(from, now) + grace, where from is the claim's own date (ClaimedAt, or
// WrittenAt when unreadable) unless that date is more than Skew ahead of now,
// when from is FirstSeen (now when FirstSeen is zero).
//
// So neither a far-ahead clock nor a hand-edited expiry can hold a record past
// a day from its claim: a claim dated a little ahead of the reader, within
// the Skew two honest clocks may differ by, is clamped from the reader's now,
// and one dated further ahead is counted from when the reader first saw it,
// so it lapses a day after that sighting rather than holding for a day from
// every reading.
func Judge(s Stamp, grace time.Duration, now time.Time) Verdict {
	if s.Unreadable {
		exp := earlier(from(s.WrittenAt, s.FirstSeen, now), now).Add(grace)
		return Verdict{EffectiveExpiry: exp, Live: now.Before(exp), Unreadable: true}
	}
	exp := s.ExpiresAt
	if ceiling := earlier(from(s.ClaimedAt, s.FirstSeen, now), now).Add(Max); ceiling.Before(exp) {
		exp = ceiling
	}
	return Verdict{EffectiveExpiry: exp, Live: now.Before(exp)}
}

// Ahead reports whether a claim dated at is more than Skew ahead of now, the
// claim Judge counts from its first sighting instead of from its own date.
func Ahead(at, now time.Time) bool { return at.After(now.Add(Skew)) }

// from is the time a claim dated at is counted from at now: at itself, or
// when it is more than Skew ahead, its first sighting (now when there is
// none).
func from(at, firstSeen, now time.Time) time.Time {
	if !Ahead(at, now) {
		return at
	}
	if firstSeen.IsZero() {
		return now
	}
	return firstSeen
}

// earlier is the earlier of two times.
func earlier(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}
