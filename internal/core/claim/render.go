package claim

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// The record render's words for a record with no reservation to show: none in
// a view abcd fetched, or unknown in one nothing here fetched, which is never
// read as free.
const (
	LineNone    = "none"
	LineUnknown = "unknown"
)

// Whose copy a reservation is from, when it is the reader's own.
const (
	LineThisCopy    = "this copy"
	LineAnotherCopy = "another copy"
)

// Line is one record's reservation as the record render shows it, read from
// this checkout's disk alone: its state (a view state, or LineNone or
// LineUnknown), its holder and expiry, whose copy it is when it is the
// reader's own, the view's age, and the sentence the text form prints.
type Line struct {
	State           string `json:"state"`
	Holder          string `json:"holder,omitempty"`
	Copy            string `json:"copy,omitempty"`
	EffectiveExpiry string `json:"effective_expiry,omitempty"`
	ReleasedBy      string `json:"released_by,omitempty"`
	Unreadable      string `json:"unreadable,omitempty"`
	FetchedAt       string `json:"fetched_at,omitempty"`
	AgeSeconds      *int64 `json:"age_seconds,omitempty"`
	Text            string `json:"text"`
}

// LineFor is id's reservation as last fetched, for the record render: no
// fetch and no push. ok is false when id is not reservable or the checkout has
// no remote named origin, where there are no reservations to show. A view no
// abcd fetch stamped, holding nothing for id, is LineUnknown: a render never
// says a record is free on a view it cannot date.
func LineFor(root, id string, now time.Time, o Options) (l Line, ok bool) {
	id, err := reservableID(id)
	if err != nil {
		return Line{}, false
	}
	if out, err := localGit(root, 64<<10, "remote"); err != nil || !slices.Contains(strings.Fields(out), Remote) {
		return Line{}, false
	}
	now = now.UTC().Truncate(time.Second)
	snap, err := View(root, now)
	if err != nil {
		return Line{State: LineUnknown, Text: "unknown, the reservations here could not be read: " + err.Error()}, true
	}
	if !snap.FetchedAt.IsZero() {
		l.FetchedAt = stamp(snap.FetchedAt)
		age := int64(now.Sub(snap.FetchedAt).Seconds())
		l.AgeSeconds = &age
	}
	r, found := snap.Lookup(id)
	switch {
	case !found && snap.FetchedAt.IsZero():
		l.State, l.Text = LineUnknown, "unknown, nothing has been fetched here; `abcd peers` fetches"
		return l, true
	case !found:
		l.State, l.Text = LineNone, LineNone+", "+snap.AsOf(now)
		return l, true
	}
	l.State, l.Holder, l.ReleasedBy, l.Unreadable = r.State, r.Holder, r.ReleasedBy, r.Unreadable
	l.EffectiveExpiry = stamp(r.EffectiveExpiry)
	switch {
	case r.Copy == CopyThis:
		l.Copy = LineThisCopy
	case r.Holder != "" && sameHolder(r.Holder, readerHolder(root, o)):
		l.Copy = LineAnotherCopy
	}
	who := r.Holder
	switch l.Copy {
	case LineThisCopy:
		who += " (this copy)"
	case LineAnotherCopy:
		who += " (another copy of yours)"
	}
	var text string
	switch {
	case r.State == StateUnreadable:
		text = fmt.Sprintf("unreadable (%s), held until %s", r.Unreadable, l.EffectiveExpiry)
		if r.HeldFromFetch {
			text += ", counted from when this copy first fetched it"
		}
	case r.ReleasedBy == ReleasedByDefaultBranch:
		text = "released: the record is finished on origin's default branch"
	case r.State == StateReleased:
		text = fmt.Sprintf("released by %s at %s", who, stamp(r.ClaimedAt))
	default:
		text = fmt.Sprintf("%s, reserved by %s until %s", r.State, who, l.EffectiveExpiry)
	}
	l.Text = text + ", " + snap.AsOf(now)
	return l, true
}

// Lookup is id's reservation in the view, when it holds one.
func (s Snapshot) Lookup(id string) (Reservation, bool) {
	for _, r := range s.Reservations {
		if r.Record == id {
			return r, true
		}
	}
	return Reservation{}, false
}

// HeldBy is how a render names a reservation another copy holds on id at now
// (a live one, or an unreadable one within its hold), with the view's age; ""
// when nothing holds it but this copy. It is the board's held-by text.
func (s Snapshot) HeldBy(id string, now time.Time) string {
	r, ok := s.Lookup(id)
	if !ok || r.Copy == CopyThis {
		return ""
	}
	switch {
	case r.State == StateLive:
		return fmt.Sprintf("reserved by %s until %s, %s", r.Holder, stamp(r.EffectiveExpiry), s.AsOf(now))
	case r.State == StateUnreadable && now.Before(r.EffectiveExpiry):
		return fmt.Sprintf("an unreadable reservation holds it until %s, %s", stamp(r.EffectiveExpiry), s.AsOf(now))
	}
	return ""
}

// AsOf dates the view for a render: how long ago abcd last fetched the
// reservations here, or that the age is unknown when nothing here stamped a
// fetch (a view a plain git fetch left).
func (s Snapshot) AsOf(now time.Time) string {
	if s.FetchedAt.IsZero() {
		return "as of a fetch of unknown age"
	}
	d := now.Sub(s.FetchedAt)
	switch {
	case d < time.Minute:
		return "as of the last fetch under a minute ago"
	case d < 2*time.Hour:
		return "as of the last fetch " + count(int(d/time.Minute), "minute") + " ago"
	case d < 48*time.Hour:
		return "as of the last fetch " + count(int(d/time.Hour), "hour") + " ago"
	}
	return "as of the last fetch " + count(int(d/(24*time.Hour)), "day") + " ago"
}

// count is n noun, plural unless n is one.
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// readerHolder is the GitHub username the person reading resolves to, read as
// a reservation verb would read it; "" when it cannot be told.
func readerHolder(root string, o Options) string {
	email, err := personalConfig(root, "user.email")
	if err != nil {
		return ""
	}
	u, err := resolveHolder(root, o.home(), email)
	if err != nil {
		return ""
	}
	return u
}

// ReaderHolder is the GitHub username the person reading resolves to, read as
// a reservation verb reads it, with no network act; "" when it cannot be
// told. A render that tells this person's other copy apart from another
// person reads it.
func ReaderHolder(root string, o Options) string { return readerHolder(root, o) }

// SameHolder reports whether two usernames name one person, compared without
// regard to case, as the forge compares them.
func SameHolder(a, b string) bool { return a != "" && sameHolder(a, b) }
