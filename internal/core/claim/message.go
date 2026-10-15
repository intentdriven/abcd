package claim

import (
	"bytes"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
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

// family is one kind of empty-tree commit this package writes to the shared
// remote and reads back: the trailer keys its messages carry, and the words a
// refusal names its messages by. The trailers are read by the one collector
// (collectTrailers) and judged by the family's own validator; a family never
// reads another's keys.
type family struct {
	// name is the family's noun in a refusal: "reservation" gives "the
	// reservation message" and "the reservation commit".
	name string
	// keys are the trailer keys the family's messages carry, each in its
	// canonical spelling; the collector reads no other.
	keys []string
	// shorten is the remedy a refusal of a message over the cap names.
	shorten string
}

// reservationFamily is the reservation's: a record held by one person from one
// working copy for a lease, validated by validateReservation.
var reservationFamily = &family{
	name:    "reservation",
	keys:    []string{keyReservation, keyHolder, keyCopy, keyFootprint, keyClaimedAt, keyExpiresAt, keyReleasedAt, keyAssistedBy},
	shorten: "name fewer or shorter footprint paths",
}

// commitMessage is one family's commit message, as the empty-tree writer
// (writeEmptyCommit) measures, judges and writes it.
type commitMessage interface {
	// text is the whole message: a subject, a blank line, then one trailer
	// per line, every value written on standard input and never as an
	// argument.
	text() string
	// messageFamily is the family the message belongs to.
	messageFamily() *family
}

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

// messageFamily makes a reservation message a commitMessage.
func (message) messageFamily() *family { return reservationFamily }

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

// head is what every family's reader reads of one commit besides its
// message: the commit, its date and parentage, and why it cannot be read.
// Every value in it is data a writer chose, never instruction text.
type head struct {
	Commit string
	// WrittenAt is the commit's committer date: the stamp an unreadable tip's
	// hold is counted from.
	WrittenAt time.Time
	// Parent reports whether the commit has a parent (a create has none).
	Parent bool
	// Unreadable is the reason the commit cannot be read, "" when it can.
	Unreadable string
	// DatedFromFetch is true when WrittenAt is not the commit's date, which
	// could not be read, but when this copy first fetched the commit.
	DatedFromFetch bool
}

// tip is a reservation branch's head commit as read from this checkout's
// object store: either a readable message or the reason it cannot be read.
type tip struct {
	head
	message
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

// parseTip reads one commit object as a commit of the family parse reads:
// the head, and the message parse makes of the commit's message text, or the
// reason the commit is unreadable (the message then the zero value). emptyTree
// is the repository's empty tree. obj is the object as `git cat-file commit`
// prints it, already bounded by the caller. The bounds and the empty tree are
// every family's; parse is the family's own.
func parseTip[M any](commit string, obj []byte, emptyTree string, parse func(text string) (M, string)) (head, M) {
	h := head{Commit: commit}
	var zero M
	headers, msg, ok := bytes.Cut(obj, []byte("\n\n"))
	if !ok {
		headers, msg = obj, nil
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
			h.Parent = true
		case "committer":
			h.WrittenAt = committerTime(v)
		}
	}
	switch {
	case tree == "":
		h.Unreadable = reasonNoObject
		return h, zero
	case len(msg) > MaxMessageBytes:
		h.Unreadable = reasonMessageTooBig
		return h, zero
	case tree != emptyTree:
		h.Unreadable = reasonNotEmptyTree
		return h, zero
	}
	m, why := parse(string(msg))
	h.Unreadable = why
	return h, m
}

// reservationParser is the parser parseTip reads record's reservation branch
// with.
func reservationParser(record string) func(string) (message, string) {
	return func(text string) (message, string) { return parseMessage(record, text) }
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

// trailers is a message's trailer block as collectTrailers reads it: the
// known keys' values, and what else the message carries, which a validator
// that admits nothing outside its trailers refuses on.
type trailers struct {
	// vals holds each known key's values, in order, under its canonical
	// spelling.
	vals map[string][]string
	// other is the final paragraph's lines that carry no known key, verbatim.
	other []string
	// lead is the message before its final paragraph (the subject, and any
	// paragraph after it), "" when the message is one paragraph.
	lead string
}

// collectTrailers reads a message's trailers, from its final paragraph, by the
// keys given, matched without regard to case; a line carrying any other key,
// or none, is kept in other. It is every family's one trailer reader: a family
// judges what it collects with its own validator.
func collectTrailers(text string, keys []string) trailers {
	paras := strings.Split(strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n\n")
	last := paras[len(paras)-1]
	t := trailers{vals: map[string][]string{}, lead: strings.Join(paras[:len(paras)-1], "\n\n")}
	for _, l := range strings.Split(last, "\n") {
		k, v, ok := strings.Cut(l, ":")
		known := ""
		if ok {
			k = strings.TrimSpace(k)
			for _, want := range keys {
				if strings.EqualFold(k, want) {
					known = want
				}
			}
		}
		if known == "" {
			t.other = append(t.other, l)
			continue
		}
		t.vals[known] = append(t.vals[known], strings.TrimSpace(v))
	}
	return t
}

// one is key's value when the trailers carry exactly one.
func (t trailers) one(key string) (string, bool) {
	if len(t.vals[key]) != 1 {
		return "", false
	}
	return t.vals[key][0], true
}

// parseMessage reads a reservation message: its trailers through the one
// collector, judged by the reservation's validator. It returns the reason the
// message is unreadable, "" when it reads.
func parseMessage(record, text string) (message, string) {
	return validateReservation(record, collectTrailers(text, reservationFamily.keys))
}

// validateReservation is the reservation family's validator: the trailers must
// name record exactly once, one GitHub username as holder, one working copy,
// the lease's times and only repository-relative footprint paths. A key it
// does not know, and any text outside the trailers, is ignored.
func validateReservation(record string, t trailers) (message, string) {
	var m message
	id, ok := t.one(keyReservation)
	if !ok || !recordid.CitedIDRe.MatchString(id) || recordid.CanonCitedID(id) != record {
		return m, reasonRecord
	}
	m.Record = record
	if m.Holder, ok = t.one(keyHolder); !ok || !validUsername(m.Holder) {
		return message{}, reasonHolder
	}
	if m.Copy, ok = t.one(keyCopy); !ok || !validCopyKey(m.Copy) {
		return message{}, reasonCopy
	}
	var err error
	for _, f := range []struct {
		key      string
		into     *time.Time
		optional bool
	}{{keyClaimedAt, &m.ClaimedAt, false}, {keyExpiresAt, &m.ExpiresAt, false}, {keyReleasedAt, &m.ReleasedAt, true}} {
		v, ok := t.one(f.key)
		if !ok {
			if f.optional && len(t.vals[f.key]) == 0 {
				continue
			}
			return message{}, reasonTime
		}
		if *f.into, err = time.Parse(time.RFC3339, v); err != nil {
			return message{}, reasonTime
		}
		*f.into = f.into.UTC()
	}
	for _, p := range t.vals[keyFootprint] {
		if !validFootprint(p) {
			return message{}, reasonFootprint
		}
		m.Footprint = append(m.Footprint, p)
	}
	if v, ok := t.one(keyAssistedBy); ok {
		m.AssistedBy = v
	}
	return m, ""
}

// validFootprint reports whether p is a footprint path a reservation may carry:
// a repository-relative path (fsutil.ValidRelPath) of printing runes only, no
// C1 control, bidi mark or zero-width rune among them. The writer refuses any
// other and the reader reads a tip carrying one as unreadable, so a footprint
// that reaches a refusal or a listing cannot reorder or hide the text around
// it.
func validFootprint(p string) bool {
	if !fsutil.ValidRelPath(p) || termsafe.Sanitize(p) != p {
		return false
	}
	for _, r := range p {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
