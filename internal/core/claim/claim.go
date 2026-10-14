// Package claim is the reservation on the shared remote: before a person's
// agent starts on an issue or an intent, it reserves the record as a branch
// nobody merges, `refs/heads/abcd-reserved/<record-id>` on `origin`, so every
// participant — a second person, a second computer, a hosted cloud session —
// sees the reservation before starting the same item (itd-2609091034175565,
// adr-2610031352212697: claims are branches on the shared remote, never a
// hidden ref, because a hosted session's proxy refuses pushes outside
// refs/heads/).
//
// A reservation is a chain of empty-tree commits written with `git
// commit-tree`, whose trailers name the record, the holder (a GitHub
// username), the working copy, the footprint and the lease. Every act is one
// plain fast-forward push, which is already the compare-and-swap: the remote
// takes it only when the tip the writer judged is still the tip.
//
// Every surface a person reads says reservation; inside the code it is a
// claim, the name the in-run claim in internal/core/implement already uses.
// Both are judged by internal/core/lease.
//
// The package owns the message format and its reader, the holder, the
// working-copy key, the fetch, the push protocol and the derived release. It
// never writes to stdout: a front door renders its results and refusals.
package claim

import (
	"errors"
	"fmt"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// Remote is the one remote reservations live on.
	Remote = "origin"
	// RefPrefix is where a reservation branch sits on the remote, and
	// TrackingPrefix where this checkout keeps what it last fetched of them.
	// The prefix is disjoint from the implement loop's lane prefix build/.
	RefPrefix      = "refs/heads/abcd-reserved/"
	TrackingPrefix = "refs/remotes/" + Remote + "/abcd-reserved/"

	// MaxMessageBytes caps a reservation commit's message: the writer refuses
	// a longer one and the reader reads one as unreadable.
	MaxMessageBytes = 4096
	// maxObjectBytes bounds the read of a tip's whole commit object, headers
	// and signature included.
	maxObjectBytes = 8192

	// TakeoverMargin is how long past a reservation's effective expiry (or an
	// unreadable tip's hold) another person waits before taking it over, by
	// the taker's clock: the slack that absorbs small skew between clocks.
	TakeoverMargin = 5 * time.Minute
)

// ErrBackOff marks a refusal whose remedy is to take other work: the record is
// held, and the front door exits 3, as `implement claim` does for contention.
// ErrRefused marks every other refusal and failure, exit 2.
var (
	ErrBackOff = errors.New("the record is reserved")
	ErrRefused = errors.New("the reservation was refused")
)

// refusal is a refusal carrying its class (ErrBackOff or ErrRefused).
type refusal struct {
	msg  string
	kind error
}

func (e *refusal) Error() string        { return e.msg }
func (e *refusal) Is(target error) bool { return target == e.kind }

// refuse is an ErrRefused refusal; every message ends saying what was not done.
func refuse(format string, args ...any) error {
	return &refusal{msg: fmt.Sprintf(format, args...), kind: ErrRefused}
}

// backOff is an ErrBackOff refusal.
func backOff(format string, args ...any) error {
	return &refusal{msg: fmt.Sprintf(format, args...), kind: ErrBackOff}
}

// nothingPushed and noReservation end every refusal: the first for one met
// before the remote was asked, the second for one met reading or writing it.
const (
	nothingPushed = "nothing was pushed"
	noReservation = "no reservation was made"
)

// maxEchoRunes bounds how much of an argument a refusal quotes back.
const maxEchoRunes = 80

// echo renders untrusted text for a refusal: terminal-safe, bounded, quoted.
func echo(s string) string {
	s = termsafe.Sanitize(s)
	if r := []rune(s); len(r) > maxEchoRunes {
		s = string(r[:maxEchoRunes-3]) + "..."
	}
	return fmt.Sprintf("%q", s)
}

// redacted is an error's text with the home directory redacted, as every
// refusal quotes git's or the filesystem's own reason.
func redacted(err error) string { return fsutil.RedactHome(err.Error()) }

// reservableID is the record a reservation names, in canonical form, or a
// refusal. Only an issue or an intent is reservable — the records a build or a
// drain takes — and the argument must already be a cited id in canonical
// grammar, so no ref name is ever formed from anything else and two spellings
// of one record never make two branches.
func reservableID(arg string) (string, error) {
	if !recordid.CitedIDRe.MatchString(arg) {
		return "", refuse("%s is not an issue or intent id (iss-N or itd-N); %s", echo(arg), nothingPushed)
	}
	id := recordid.CanonCitedID(arg)
	if id == "" || (id[:4] != "iss-" && id[:4] != "itd-") {
		return "", refuse("%s is not an issue or intent id (iss-N or itd-N); %s", echo(arg), nothingPushed)
	}
	return id, nil
}

// refName is a reservable id's branch on the remote; trackingName is this
// checkout's copy of it. Both take only reservableID's answer.
func refName(id string) string      { return RefPrefix + id }
func trackingName(id string) string { return TrackingPrefix + id }

// Why a renewal found no reservation of this person's from this copy to renew.
const (
	NotHeldAbsent        = "absent"
	NotHeldUnreadable    = "unreadable"
	NotHeldReleased      = "released"
	NotHeldAnotherPerson = "another_person"
	NotHeldAnotherCopy   = "another_copy"
	NotHeldLapsed        = "lapsed"
)

// NotHeld is the refusal of an act that needs this person's own reservation
// from this copy — a renewal, or a release — when the tip is not one: Why
// names what it is instead, with what the tip says of it, so a caller words
// its own refusal (the land fence) without reading the message. It pushes
// nothing.
type NotHeld struct {
	Record string
	// Why is one of the NotHeld* reasons.
	Why string
	// Holder is the tip's holder, for another person's or another copy's.
	Holder string
	// Expiry is the tip's effective expiry, or an unreadable tip's hold.
	Expiry time.Time
	// ReleasedAt is a tombstone's time; Derived is true instead when the
	// record is finished on origin's default branch.
	ReleasedAt time.Time
	Derived    bool
	// Unreadable is why the tip cannot be read.
	Unreadable string

	msg  string
	kind error
}

func (e *NotHeld) Error() string        { return e.msg }
func (e *NotHeld) Is(target error) bool { return target == e.kind }
