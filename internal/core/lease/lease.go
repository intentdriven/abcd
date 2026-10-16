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
// by a little. A claim dated further ahead of when the reader first saw it
// (Stamp.FirstSeen) is counted from that sighting.
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
	// rather than read from the claim; zero when nothing recorded it. A claim
	// whose own date (ClaimedAt, or WrittenAt when unreadable) is more than
	// Skew after FirstSeen is counted from FirstSeen at every reading. With no
	// FirstSeen, a claim dated more than Skew ahead of now is counted from now.
	FirstSeen time.Time
}

// Verdict is a claim judged at a moment.
type Verdict struct {
	// EffectiveExpiry is when the claim stops holding: a readable claim's
	// expiry clamped to a day after it was claimed (or after now, when it is
	// dated ahead of the reader, or after it was first seen, when it was dated
	// more than Skew ahead of that sighting), or an unreadable one's WrittenAt
	// (or now, or its first sighting, on the same terms) plus the grace.
	EffectiveExpiry time.Time
	// Live is true while now is before EffectiveExpiry.
	Live bool
	// Unreadable carries the stamp's own mark, so a caller rendering the
	// verdict says why it holds.
	Unreadable bool
}

// Judge says whether a claim holds at now. A readable claim's effective expiry
// is min(ExpiresAt, min(from, now) + Max) and an unreadable one is held until
// min(from, now) + grace, where from is:
//
//   - FirstSeen, when the claim's own date (ClaimedAt, or WrittenAt when
//     unreadable) is more than Skew after FirstSeen: the claim was dated ahead
//     when the reader first saw it, and stays counted from that sighting at
//     every later reading, however the reader's clock has moved since;
//   - now, when no FirstSeen is recorded and the claim's own date is more
//     than Skew ahead of now;
//   - otherwise the claim's own date.
//
// So neither a far-ahead clock nor a hand-edited expiry can hold a record past
// a day from its claim or its first sighting, and for a fixed stamp with a
// sighting the expiry never moves later as time passes. A claim dated within
// Skew of its sighting, as two honest clocks may differ, is judged exactly as
// it would be with no sighting recorded.
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

// Ahead reports whether a claim dated at is more than Skew ahead of now (or
// of a first sighting), the claim Judge counts from that sighting instead of
// from its own date.
func Ahead(at, now time.Time) bool { return at.After(now.Add(Skew)) }

// from is the time a claim dated at is counted from at now: its first
// sighting when it was dated more than Skew ahead of that sighting, a test
// made against the recorded sighting and never against now, so its answer
// does not change as time passes; now when no sighting is recorded and it is
// more than Skew ahead of now; else at itself.
func from(at, firstSeen, now time.Time) time.Time {
	switch {
	case !firstSeen.IsZero() && Ahead(at, firstSeen):
		return firstSeen
	case firstSeen.IsZero() && Ahead(at, now):
		return now
	}
	return at
}

// earlier is the earlier of two times.
func earlier(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}
