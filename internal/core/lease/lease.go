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
}

// Verdict is a claim judged at a moment.
type Verdict struct {
	// EffectiveExpiry is when the claim stops holding: a readable claim's
	// expiry clamped to a day after it was claimed (or after now, when it is
	// dated ahead of the reader), or an unreadable one's WrittenAt (or now)
	// plus the grace.
	EffectiveExpiry time.Time
	// Live is true while now is before EffectiveExpiry.
	Live bool
	// Unreadable carries the stamp's own mark, so a caller rendering the
	// verdict says why it holds.
	Unreadable bool
}

// Judge says whether a claim holds at now. A readable claim's effective expiry
// is min(ExpiresAt, min(ClaimedAt, now) + Max), so neither a far-ahead clock
// nor a hand-edited expiry can hold a record past a day from its claim, nor
// past a day from the reader's own now when the claim is dated ahead of it;
// an unreadable one is held until min(WrittenAt, now) + grace.
//
// A claim dated ahead of the reader is clamped at every reading, so it still
// holds while the writer's clock stays ahead: the clamp bounds what one
// reading may promise, not how long a clock may run fast.
func Judge(s Stamp, grace time.Duration, now time.Time) Verdict {
	if s.Unreadable {
		exp := earlier(s.WrittenAt, now).Add(grace)
		return Verdict{EffectiveExpiry: exp, Live: now.Before(exp), Unreadable: true}
	}
	exp := s.ExpiresAt
	if ceiling := earlier(s.ClaimedAt, now).Add(Max); ceiling.Before(exp) {
		exp = ceiling
	}
	return Verdict{EffectiveExpiry: exp, Live: now.Before(exp)}
}

// earlier is the earlier of two times.
func earlier(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}
