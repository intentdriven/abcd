package reservesweep

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// Kind is what one commit on a reservation branch is, as the sweep reads it.
// Anything the sweep cannot classify is KindUnknown, and a branch carrying
// one is never spent: the sweep fails closed.
type Kind string

const (
	// KindUnknown is a commit the sweep cannot classify: an unparseable
	// object, a tree that is not the empty tree, a message the claim
	// package's reader refuses (a time that does not read, a hand-back kind
	// outside claim.HandBackKinds among them), or a subject or a trailer the
	// sweep does not know.
	KindUnknown Kind = "unknown"
	// KindReservation is a reservation, a renewal or a take-over: it carries
	// no release time.
	KindReservation Kind = "reservation"
	// KindPlainTombstone is a release that hands nothing back.
	KindPlainTombstone Kind = "plain_tombstone"
	// KindHandBack is a release that hands the record back to a person, so
	// other accounts' drains do not take it again.
	KindHandBack Kind = "hand_back"
)

// commitInfo is one commit on a reservation branch as the sweep read it.
type commitInfo struct {
	Commit string
	Kind   Kind
	// Why says what made the commit unknown; "" otherwise.
	Why string
	// CommittedAt is the committer date.
	CommittedAt time.Time
	// ClaimedAt and ReleasedAt are the commit's own trailers, zero when it
	// does not carry them (ReleasedAt on a reservation).
	ClaimedAt  time.Time
	ReleasedAt time.Time
	// Parent is the first parent, "" for a root commit.
	Parent string
}

// classify reads one commit object (as `git cat-file commit` prints it) on
// record's reservation branch: its kind, times and first parent. emptyTree is
// the repository's empty tree.
func classify(record, commit string, obj []byte, emptyTree string) commitInfo {
	c := commitInfo{Commit: commit, Kind: KindUnknown}
	headers, msg, ok := bytes.Cut(obj, []byte("\n\n"))
	if !ok {
		c.Why = "its commit object does not parse"
		return c
	}
	tree, parents := "", 0
	for _, l := range strings.Split(string(headers), "\n") {
		k, v, _ := strings.Cut(l, " ")
		switch k {
		case "tree":
			if tree == "" {
				tree = v
			}
		case "parent":
			parents++
			if c.Parent == "" {
				c.Parent = v
			}
		case "committer":
			c.CommittedAt = committerTime(v)
		}
	}
	switch {
	case tree == "" || c.CommittedAt.IsZero():
		c.Why = "its commit object does not parse"
		return c
	case parents > 1:
		// abcd writes only one-parent commits here, and the walk reads the
		// first-parent line alone, so a second parent could hold a hand-back
		// the walk never reads.
		c.Why = "it has more than one parent"
		return c
	case tree != emptyTree:
		c.Why = "its tree is not the empty tree"
		return c
	}
	kind, why := classifyMessage(record, string(msg), &c)
	c.Kind, c.Why = kind, why
	return c
}

// classifyMessage reads a reservation commit's message through the claim
// package's reader (claim.ReadReservationMessage), the one the drain reads a
// reservation branch with, and keeps the sweep's stricter, fail-closed
// reading on top of it: the message must name record, carry its kind's
// subject and nothing besides, and carry no trailer the reader does not know,
// since a key the drain ignores today may mark a release that keeps the
// record from other accounts' drains tomorrow. A hand-back is a tombstone
// carrying one of claim.HandBackKinds with its record blob; the reader refuses
// any other hand-back, and a refusal is KindUnknown. The rules the sweep keeps
// over a hand-back are dcn-2610161806433529 (a plain release keeps a hand-back
// that still stands) and dcn-2610161826443524 (a pending hand-back is
// published later over the issue's next plain tombstone), so judge keeps a
// branch carrying one anywhere. It fills c's times and returns the kind, or
// KindUnknown with why.
func classifyMessage(record, text string, c *commitInfo) (Kind, string) {
	m, err := claim.ReadReservationMessage(text)
	switch {
	case err != nil:
		return KindUnknown, err.Error()
	case m.Record != record:
		return KindUnknown, "it does not name this record exactly once"
	case len(m.Other) > 0:
		return KindUnknown, "it carries a trailer the sweep does not know"
	case m.Released() && m.Lead != "abcd reservation released: "+record:
		return KindUnknown, "its subject is not a release's"
	case !m.Released() && m.Lead != "abcd reservation: "+record:
		return KindUnknown, "its subject is not a reservation's"
	}
	c.ClaimedAt, c.ReleasedAt = m.ClaimedAt, m.ReleasedAt
	switch {
	case m.HandedBack != "" && slices.Contains(claim.HandBackKinds, m.HandedBack):
		return KindHandBack, ""
	case m.HandedBack != "" || m.RecordBlob != "":
		return KindUnknown, "its hand-back is not one known kind"
	case m.Released():
		return KindPlainTombstone, ""
	}
	return KindReservation, ""
}

// committerTime is the time in a committer header's value, "<name> <<email>>
// <unix> <zone>"; the zero time when it does not parse.
func committerTime(v string) time.Time {
	f := strings.Fields(v)
	if len(f) < 2 {
		return time.Time{}
	}
	sec, err := strconv.ParseInt(f[len(f)-2], 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}
