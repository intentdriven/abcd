package claim

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The trailer keys a reservation commit carries. They are prefixed and
// hyphenated because git's trailer grammar admits no underscore, and the
// prefix keeps them apart from every other trailer gate.
const (
	keyReservation = "Abcd-Reservation"
	keyHolder      = "Abcd-Holder"
	keyCopy        = "Abcd-Copy"
	keyFootprint   = "Abcd-Footprint"
	keyClaimedAt   = "Abcd-Claimed-At"
	keyExpiresAt   = "Abcd-Expires-At"
	keyReleasedAt  = "Abcd-Released-At"
	keyAssistedBy  = "Assisted-by"
)

// message is one reservation commit's content: the record, who holds it from
// which working copy, the files the work will change and the lease. A
// tombstone, the reservation released, carries ReleasedAt, with ExpiresAt
// equal to it.
type message struct {
	Record     string
	Holder     string
	Copy       string
	Footprint  []string
	ClaimedAt  time.Time
	ExpiresAt  time.Time
	ReleasedAt time.Time
	// AssistedBy is the `Assisted-by:` value, abcd's composed label: commit-tree
	// runs no prepare-commit-msg hook, so the writer names it.
	AssistedBy string
}

// released reports whether the message is a tombstone.
func (m message) released() bool { return !m.ReleasedAt.IsZero() }

// stamp renders a time the way every reservation carries one: RFC 3339, UTC,
// to the second.
func stamp(t time.Time) string { return t.UTC().Truncate(time.Second).Format(time.RFC3339) }

// text is the commit message: a subject, a blank line, then one trailer per
// line. Every value is written here, on standard input to commit-tree, and
// never becomes a git argument, a refspec or a shell word.
func (m message) text() string {
	var b strings.Builder
	if m.released() {
		b.WriteString("abcd reservation released: " + m.Record + "\n\n")
	} else {
		b.WriteString("abcd reservation: " + m.Record + "\n\n")
	}
	line := func(k, v string) { b.WriteString(k + ": " + v + "\n") }
	line(keyReservation, m.Record)
	line(keyHolder, m.Holder)
	line(keyCopy, m.Copy)
	for _, p := range m.Footprint {
		line(keyFootprint, p)
	}
	line(keyClaimedAt, stamp(m.ClaimedAt))
	line(keyExpiresAt, stamp(m.ExpiresAt))
	if m.released() {
		line(keyReleasedAt, stamp(m.ReleasedAt))
	}
	if m.AssistedBy != "" {
		line(keyAssistedBy, m.AssistedBy)
	}
	return b.String()
}

// tip is a reservation branch's head commit as read from this checkout's
// object store: either a readable message or the reason it cannot be read.
// Every value in it is data a writer chose, never instruction text.
type tip struct {
	Commit string
	// WrittenAt is the commit's committer date: the stamp an unreadable tip's
	// hold is counted from.
	WrittenAt time.Time
	// Parent reports whether the commit has a parent (a create has none).
	Parent bool
	message
	// Unreadable is the reason the tip cannot be read, "" when it can.
	Unreadable string
}

// The reasons a tip is unreadable. Each is fixed text: nothing a writer chose
// is echoed into one.
const (
	reasonObjectTooBig  = "its commit object passes the 8192-byte bound"
	reasonMessageTooBig = "its message passes the 4096-byte cap"
	reasonNotEmptyTree  = "its tree is not the empty tree"
	reasonNoObject      = "its commit object does not parse"
	reasonRecord        = "it does not name this record exactly once"
	reasonHolder        = "it does not name one GitHub username as its holder"
	reasonCopy          = "it does not name one working copy"
	reasonTime          = "a time it carries is missing, repeated or not RFC 3339"
	reasonFootprint     = "a footprint it names is not a repository-relative path"
)

// parseTip reads one commit object as the tip of record's reservation branch.
// emptyTree is the repository's empty tree. obj is the object as `git
// cat-file commit` prints it, already bounded by the caller.
func parseTip(record, commit string, obj []byte, emptyTree string) tip {
	t := tip{Commit: commit}
	head, msg, ok := bytes.Cut(obj, []byte("\n\n"))
	if !ok {
		head, msg = obj, nil
	}
	tree := ""
	for _, l := range strings.Split(string(head), "\n") {
		k, v, _ := strings.Cut(l, " ")
		switch k {
		case "tree":
			if tree == "" {
				tree = v
			}
		case "parent":
			t.Parent = true
		case "committer":
			t.WrittenAt = committerTime(v)
		}
	}
	switch {
	case tree == "":
		t.Unreadable = reasonNoObject
		return t
	case len(msg) > MaxMessageBytes:
		t.Unreadable = reasonMessageTooBig
		return t
	case tree != emptyTree:
		t.Unreadable = reasonNotEmptyTree
		return t
	}
	t.message, t.Unreadable = parseMessage(record, string(msg))
	return t
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

// parseMessage reads a reservation message's trailers, from its final
// paragraph, by their known keys; an unknown key is ignored. It returns the
// reason the message is unreadable, "" when it reads.
func parseMessage(record, text string) (message, string) {
	paras := strings.Split(strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n\n")
	last := paras[len(paras)-1]
	vals := map[string][]string{}
	for _, l := range strings.Split(last, "\n") {
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		for _, known := range []string{keyReservation, keyHolder, keyCopy, keyFootprint, keyClaimedAt, keyExpiresAt, keyReleasedAt, keyAssistedBy} {
			if strings.EqualFold(k, known) {
				vals[known] = append(vals[known], strings.TrimSpace(v))
			}
		}
	}
	var m message
	one := func(k string) (string, bool) {
		if len(vals[k]) != 1 {
			return "", false
		}
		return vals[k][0], true
	}
	id, ok := one(keyReservation)
	if !ok || !recordid.CitedIDRe.MatchString(id) || recordid.CanonCitedID(id) != record {
		return m, reasonRecord
	}
	m.Record = record
	if m.Holder, ok = one(keyHolder); !ok || !validUsername(m.Holder) {
		return message{}, reasonHolder
	}
	if m.Copy, ok = one(keyCopy); !ok || !validCopyKey(m.Copy) {
		return message{}, reasonCopy
	}
	var err error
	for _, f := range []struct {
		key      string
		into     *time.Time
		optional bool
	}{{keyClaimedAt, &m.ClaimedAt, false}, {keyExpiresAt, &m.ExpiresAt, false}, {keyReleasedAt, &m.ReleasedAt, true}} {
		v, ok := one(f.key)
		if !ok {
			if f.optional && len(vals[f.key]) == 0 {
				continue
			}
			return message{}, reasonTime
		}
		if *f.into, err = time.Parse(time.RFC3339, v); err != nil {
			return message{}, reasonTime
		}
		*f.into = f.into.UTC()
	}
	for _, p := range vals[keyFootprint] {
		if !fsutil.ValidRelPath(p) {
			return message{}, reasonFootprint
		}
		m.Footprint = append(m.Footprint, p)
	}
	if v, ok := one(keyAssistedBy); ok {
		m.AssistedBy = v
	}
	return m, ""
}
