package reservesweep

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

// Kind is what one commit on a reservation branch is, as the sweep reads it.
// Anything the sweep cannot classify is KindUnknown, and a branch carrying
// one is never spent: the sweep fails closed.
type Kind string

const (
	// KindUnknown is a commit the sweep cannot classify: an unparseable
	// object, a tree that is not the empty tree, a subject or a trailer it
	// does not know, a time that does not read.
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

// The trailer keys a reservation commit carries. The sweep reads them itself
// rather than through internal/core/claim, whose message reader is
// unexported and, on the default branch today, ignores a key it does not
// know, so a hand-back tombstone would read there as a plain one.
//
// TODO(feat/drain-many-accounts): once the drain work lands, read the kind
// from internal/core/claim (its message's hand-back kind) instead of this
// classifier, keeping the fail-closed answer for any kind it does not name.
// The rules this classifier keeps are dcn-2610161806433529 (a plain release
// keeps a hand-back that still stands) and dcn-2610161826443524 (a pending
// hand-back is published later over the issue's next plain tombstone).
const (
	keyReservation = "Abcd-Reservation"
	keyHolder      = "Abcd-Holder"
	keyCopy        = "Abcd-Copy"
	keyFootprint   = "Abcd-Footprint"
	keyClaimedAt   = "Abcd-Claimed-At"
	keyExpiresAt   = "Abcd-Expires-At"
	keyReleasedAt  = "Abcd-Released-At"
	keyAssistedBy  = "Assisted-by"
	// The hand-back's two keys (feat/drain-many-accounts): its kind and the
	// blob of the record it was handed back over. Either one marks the
	// tombstone as a hand-back whatever its value says.
	keyHandedBack = "Abcd-Handed-Back"
	keyRecordBlob = "Abcd-Record"
)

// plainKeys are the keys a reservation or a plain tombstone may carry; every
// other key, the hand-back's apart, makes the commit unknown.
var plainKeys = []string{keyReservation, keyHolder, keyCopy, keyFootprint, keyClaimedAt, keyExpiresAt, keyReleasedAt, keyAssistedBy}

// handBackKeys are the keys that mark a hand-back.
var handBackKeys = []string{keyHandedBack, keyRecordBlob}

// maxMessageBytes is the reservation message cap internal/core/claim keeps
// (claim.MaxMessageBytes); a longer message is unknown here, as it is
// unreadable there.
const maxMessageBytes = 4096

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
	tree := ""
	for _, l := range strings.Split(string(headers), "\n") {
		k, v, _ := strings.Cut(l, " ")
		switch k {
		case "tree":
			if tree == "" {
				tree = v
			}
		case "parent":
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
	case tree != emptyTree:
		c.Why = "its tree is not the empty tree"
		return c
	case len(msg) > maxMessageBytes:
		c.Why = "its message passes the 4096-byte cap"
		return c
	}
	kind, why := classifyMessage(record, string(msg), &c)
	c.Kind, c.Why = kind, why
	return c
}

// classifyMessage reads a reservation commit's message: exactly a subject and
// one trailer paragraph, every trailer a key the sweep knows. It fills c's
// times and returns the kind, or KindUnknown with why.
func classifyMessage(record, text string, c *commitInfo) (Kind, string) {
	text = strings.TrimRight(text, "\n")
	paras := strings.Split(text, "\n\n")
	if len(paras) != 2 || strings.Contains(paras[0], "\n") {
		return KindUnknown, "its message is not one subject and one trailer paragraph"
	}
	vals := map[string][]string{}
	for _, l := range strings.Split(paras[1], "\n") {
		k, v, ok := strings.Cut(l, ":")
		known := ""
		if ok {
			k = strings.TrimSpace(k)
			for _, want := range append(append([]string(nil), plainKeys...), handBackKeys...) {
				if strings.EqualFold(k, want) {
					known = want
				}
			}
		}
		if known == "" {
			return KindUnknown, "it carries a trailer the sweep does not know"
		}
		vals[known] = append(vals[known], strings.TrimSpace(v))
	}
	one := func(k string) (string, bool) {
		if len(vals[k]) != 1 {
			return "", false
		}
		return vals[k][0], true
	}
	if id, ok := one(keyReservation); !ok || id != record {
		return KindUnknown, "it does not name this record exactly once"
	}
	for _, k := range []string{keyHolder, keyCopy, keyExpiresAt} {
		if v, ok := one(k); !ok || v == "" {
			return KindUnknown, "it does not carry one " + k
		}
	}
	at, ok := one(keyClaimedAt)
	if !ok {
		return KindUnknown, "it does not carry one " + keyClaimedAt
	}
	var err error
	if c.ClaimedAt, err = time.Parse(time.RFC3339, at); err != nil {
		return KindUnknown, "a time it carries is not RFC 3339"
	}
	c.ClaimedAt = c.ClaimedAt.UTC()
	if _, err := time.Parse(time.RFC3339, vals[keyExpiresAt][0]); err != nil {
		return KindUnknown, "a time it carries is not RFC 3339"
	}
	released := len(vals[keyReleasedAt]) > 0
	handBack := len(vals[keyHandedBack]) > 0 || len(vals[keyRecordBlob]) > 0
	switch {
	case released:
		rel, ok := one(keyReleasedAt)
		if !ok {
			return KindUnknown, "it carries more than one " + keyReleasedAt
		}
		if c.ReleasedAt, err = time.Parse(time.RFC3339, rel); err != nil {
			return KindUnknown, "a time it carries is not RFC 3339"
		}
		c.ReleasedAt = c.ReleasedAt.UTC()
		if paras[0] != "abcd reservation released: "+record {
			return KindUnknown, "its subject is not a release's"
		}
	case paras[0] != "abcd reservation: "+record:
		return KindUnknown, "its subject is not a reservation's"
	case handBack:
		return KindUnknown, "it carries a hand-back but is no release"
	}
	switch {
	case handBack:
		return KindHandBack, ""
	case released:
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
