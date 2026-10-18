package autonomy

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SchemaVersion is the record's shape version, written as schema_version.
const SchemaVersion = 1

// MaxWindow is the stale-record cap: a record ends at most seven days after
// it was set, so a forgotten one lapses while a run of several days fits
// (dcn-2610180937181736).
const MaxWindow = 7 * 24 * time.Hour

// MaxStepLen bounds a queued step, in characters, so one step cannot fill the
// record's 64 KiB.
const MaxStepLen = 500

// echoCap bounds how much of an untrusted value a message quotes back.
const echoCap = 64

var (
	// ErrUntilUnreadable is an --until that is neither a clock time nor a
	// timestamp.
	ErrUntilUnreadable = errors.New("autonomy: --until is not a time")
	// ErrUntilPast is an --until already past.
	ErrUntilPast = errors.New("autonomy: --until is already past")
	// ErrUntilBeyondCap is an --until later than seven days on.
	ErrUntilBeyondCap = errors.New("autonomy: --until is past the seven-day cap")
	// ErrStepEmpty is a queued step with no words.
	ErrStepEmpty = errors.New("autonomy: the step is empty")
	// ErrStepTooLong is a queued step past MaxStepLen.
	ErrStepTooLong = errors.New("autonomy: the step is too long")
	// ErrQueueFull is a step the queue has no room for.
	ErrQueueFull = errors.New("autonomy: the queue is full")
)

// Who names a person acting on the record and the working copy they act from
// (its root, as git names it).
type Who struct {
	Name string
	Copy string
}

// Record is the away record. The package comment lists its fields.
type Record struct {
	SchemaVersion int          `json:"schema_version"`
	SetBy         string       `json:"set_by"`
	SetFrom       string       `json:"set_from"`
	Start         time.Time    `json:"start"`
	Until         time.Time    `json:"until"`
	OpenEnded     bool         `json:"open_ended"`
	HoldPushes    bool         `json:"hold_pushes"`
	Queue         []QueuedStep `json:"queue"`
	HandedBack    []HandBack   `json:"handed_back"`
	Stops         []Stop       `json:"stops"`
	Closed        *Closing     `json:"closed,omitempty"`
	// DroppedStops, DroppedHandBacks and DroppedSteps count what was let go
	// to keep the record within MaxFileBytes: the oldest stops (by their
	// counts) and hand-backs as new ones arrive, and, only when a record
	// past its bound is carried or closed, the oldest queued steps.
	DroppedStops     int `json:"dropped_stops,omitempty"`
	DroppedHandBacks int `json:"dropped_handed_back,omitempty"`
	DroppedSteps     int `json:"dropped_queue,omitempty"`
}

// QueuedStep is one step queued for the person's return.
type QueuedStep struct {
	Step string    `json:"step"`
	At   time.Time `json:"at"`
	From string    `json:"from"`
}

// HandBack is one owed item the stop check handed back to the person, by its
// branch and the head it had then.
type HandBack struct {
	Branch string    `json:"branch"`
	Head   string    `json:"head"`
	At     time.Time `json:"at"`
}

// StopKind is why a recorded stop happened.
type StopKind string

const (
	// StopWorkOwed is a stop with work owed.
	StopWorkOwed StopKind = "work-owed"
	// StopNoNextLook is a stop with nothing owed and no next look scheduled.
	StopNoNextLook StopKind = "no-next-look"
	// StopAPIError is a turn the host ended on an API error.
	StopAPIError StopKind = "api-error"
	// StopCheckIncomplete is a stop allowed because the check's time budget
	// ran out before it had read the owed work; Detail says what was not
	// read.
	StopCheckIncomplete StopKind = "check-incomplete"
)

// Stop is one recorded stop. Count, First and Last are kept by AddStop: a stop
// identical to the last one recorded adds to that entry instead of a new one.
type Stop struct {
	Kind StopKind `json:"kind"`
	// ErrorType is the host's error type for StopAPIError (rate_limit, …).
	ErrorType string `json:"error_type,omitempty"`
	// Detail is the next item at the stop, in plain words.
	Detail string `json:"detail,omitempty"`
	// PossiblyForcedByCap marks a stop at the host's continuation cap.
	PossiblyForcedByCap bool      `json:"possibly_forced_by_cap,omitempty"`
	Count               int       `json:"count"`
	First               time.Time `json:"first"`
	Last                time.Time `json:"last"`
}

// Closing is who ended the record by word, from which copy, and when.
type Closing struct {
	By   string    `json:"by"`
	From string    `json:"from"`
	At   time.Time `json:"at"`
}

// Begin is the record `abcd mode autonomous` writes at now: until zero is
// open-ended (seven days at most), and otherwise must be after now and no
// later than seven days on. prev is the record already stored, or nil: an
// unclosed one's queue, hand-backs and stops are still unread, so they are
// carried into the new record rather than dropped.
func Begin(prev *Record, who Who, now, until time.Time, holdPushes bool) (*Record, error) {
	now = now.Truncate(time.Second)
	r := &Record{
		SchemaVersion: SchemaVersion,
		SetBy:         who.Name,
		SetFrom:       who.Copy,
		Start:         now,
		HoldPushes:    holdPushes,
	}
	switch {
	case until.IsZero():
		r.OpenEnded = true
		r.Until = now.Add(MaxWindow)
	case !until.After(now):
		return nil, fmt.Errorf("%w: %s", ErrUntilPast, showTime(until))
	case until.After(now.Add(MaxWindow)):
		return nil, fmt.Errorf("%w: %s is later than seven days from now (%s)", ErrUntilBeyondCap, showTime(until), showTime(now.Add(MaxWindow)))
	default:
		r.Until = until
	}
	if prev != nil && prev.Closed == nil {
		r.Queue = append(r.Queue, prev.Queue...)
		r.HandedBack = append(r.HandedBack, prev.HandedBack...)
		r.Stops = append(r.Stops, prev.Stops...)
		r.DroppedStops, r.DroppedHandBacks, r.DroppedSteps = prev.DroppedStops, prev.DroppedHandBacks, prev.DroppedSteps
	}
	r.Fit()
	return r, nil
}

// The parts' bounds keep a record within MaxFileBytes in use, each in its
// JSON form: the queue refuses a step past queueBudget, and the stops and
// hand-backs let their oldest go past theirs. The rest of the record (who,
// from where, when) is a few hundred bytes.
const (
	queueBudget    = 24 << 10
	stopsBudget    = 24 << 10
	handBackBudget = 8 << 10
	// fitBudget is what Fit leaves a record, under MaxFileBytes with room
	// for the closing Back adds and the newline the store writes.
	fitBudget = MaxFileBytes - 1<<10
)

func jsonLen(v any) int {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return 0
	}
	return len(raw)
}

// oldestStop is the index of the stop seen least recently.
func (r *Record) oldestStop() int {
	o := 0
	for i, s := range r.Stops {
		if s.Last.Before(r.Stops[o].Last) {
			o = i
		}
	}
	return o
}

func (r *Record) dropStop(i int) {
	r.DroppedStops += max(r.Stops[i].Count, 1)
	r.Stops = append(r.Stops[:i:i], r.Stops[i+1:]...)
}

// Fit lets go of what a record past its bound cannot hold, oldest first and
// counted: stops, then hand-backs, then queued steps. A record within its
// bound is unchanged. Begin fits what it carries and Back what it closes, so
// neither ever fails on size; Back prints every line before it fits.
func (r *Record) Fit() {
	for jsonLen(r) > fitBudget {
		switch {
		case len(r.Stops) > 0:
			r.dropStop(r.oldestStop())
		case len(r.HandedBack) > 0:
			r.HandedBack = r.HandedBack[1:]
			r.DroppedHandBacks++
		case len(r.Queue) > 0:
			r.Queue = r.Queue[1:]
			r.DroppedSteps++
		default:
			return
		}
	}
}

// Ends is when the record ends: its stored end, but never later than seven
// days after its start, whatever is stored.
func (r *Record) Ends() time.Time {
	limit := r.Start.Add(MaxWindow)
	if r.Until.IsZero() || r.Until.After(limit) {
		return limit
	}
	return r.Until
}

// Holds reports whether the person is away at now: a record, not closed, and
// now before its end. No record is not away.
func (r *Record) Holds(now time.Time) bool {
	return r != nil && r.Closed == nil && now.Before(r.Ends())
}

// Lapsed reports whether the record's end has passed while it is still
// unclosed: what is in it has not been read by `abcd mode back`.
func (r *Record) Lapsed(now time.Time) bool {
	return r != nil && r.Closed == nil && !now.Before(r.Ends())
}

// Close ends the record by word: who ended it, from which copy, and when.
func (r *Record) Close(who Who, now time.Time) {
	r.Closed = &Closing{By: who.Name, From: who.Copy, At: now.Truncate(time.Second)}
}

// AddStep queues a step for the person. Its whitespace is folded to single
// spaces, so it is one line; an empty step, or one past MaxStepLen
// characters, is refused and nothing is kept.
func (r *Record) AddStep(step, from string, now time.Time) error {
	step = strings.Join(strings.Fields(step), " ")
	if step == "" {
		return ErrStepEmpty
	}
	if n := utf8.RuneCountInString(step); n > MaxStepLen {
		return fmt.Errorf("%w: %d characters, at most %d", ErrStepTooLong, n, MaxStepLen)
	}
	q := append(r.Queue[:len(r.Queue):len(r.Queue)], QueuedStep{Step: step, At: now.Truncate(time.Second), From: from})
	if jsonLen(q) > queueBudget {
		return fmt.Errorf("%w: %d steps are queued, as many as the record holds; nothing is dropped, and `abcd mode back` prints them", ErrQueueFull, len(r.Queue))
	}
	r.Queue = q
	return nil
}

// AddStop records a stop at now. A stop identical to one already recorded,
// anywhere in the list (same kind, error type, detail, which names the item,
// and cap label), adds to that entry's count and last time instead, so two
// copies blocked on different items, alternating, keep two entries. Past
// stopsBudget the stop seen least recently goes, counted in DroppedStops.
func (r *Record) AddStop(s Stop, now time.Time) {
	now = now.Truncate(time.Second)
	for i := range r.Stops {
		if sameStop(r.Stops[i], s) {
			r.Stops[i].Count++
			r.Stops[i].Last = now
			return
		}
	}
	s.Count, s.First, s.Last = 1, now, now
	r.Stops = append(r.Stops, s)
	for len(r.Stops) > 1 && jsonLen(r.Stops) > stopsBudget {
		r.dropStop(r.oldestStop())
	}
}

func sameStop(a, b Stop) bool {
	return a.Kind == b.Kind && a.ErrorType == b.ErrorType && a.Detail == b.Detail && a.PossiblyForcedByCap == b.PossiblyForcedByCap
}

// AddHandBack records an item handed back, once per branch and head; it
// reports whether this call added it.
func (r *Record) AddHandBack(branch, head string, now time.Time) bool {
	for _, h := range r.HandedBack {
		if h.Branch == branch && h.Head == head {
			return false
		}
	}
	r.HandedBack = append(r.HandedBack, HandBack{Branch: branch, Head: head, At: now.Truncate(time.Second)})
	for len(r.HandedBack) > 1 && jsonLen(r.HandedBack) > handBackBudget {
		r.HandedBack = r.HandedBack[1:]
		r.DroppedHandBacks++
	}
	return true
}

// Terms are the record's plain statement, one line each: until when, set by
// whom and from which copy; that pushes wait, when the person said so; that
// the mode grants no rights, the person's standing instruction deciding; and
// how it ends.
func (r *Record) Terms() []string {
	until := "until " + showTime(r.Ends())
	if r.OpenEnded {
		until += " (open-ended, seven days at most)"
	}
	lines := []string{fmt.Sprintf("Away %s, set by %s from %s.", until, showWho(r.SetBy), showCopy(r.SetFrom))}
	if r.HoldPushes {
		lines = append(lines, "The person said pushes wait for their return (--hold-pushes): finish the local work, then queue the push with `abcd mode queue \"<step>\"`.")
	}
	lines = append(lines,
		"The away mode grants no rights: the person's own standing instruction decides what is committed, pushed, and opened or merged as a pull request.",
		"It ends at its time or with `abcd mode back`, run only on the person's word.")
	return lines
}

// Lines are what `abcd mode back` prints: one plain line per queued step, per
// handed-back item and per recorded stop, the cap label and count included.
func (r *Record) Lines() []string {
	var lines []string
	for _, q := range r.Queue {
		lines = append(lines, "queued: "+termsafe.CleanProseLine(q.Step, MaxStepLen*4))
	}
	for _, h := range r.HandedBack {
		lines = append(lines, fmt.Sprintf("handed back: %s at %s", clean(h.Branch), shortHead(h.Head)))
	}
	for _, s := range r.Stops {
		lines = append(lines, stopLine(s))
	}
	const kept = "were let go to keep the record within 64 KiB"
	if r.DroppedSteps > 0 {
		lines = append(lines, fmt.Sprintf("%s %s", count(r.DroppedSteps, "earlier queued step", "earlier queued steps"), kept))
	}
	if r.DroppedHandBacks > 0 {
		lines = append(lines, fmt.Sprintf("%s %s", count(r.DroppedHandBacks, "earlier item handed back", "earlier items handed back"), kept))
	}
	if r.DroppedStops > 0 {
		lines = append(lines, fmt.Sprintf("%s %s", count(r.DroppedStops, "earlier stop", "earlier stops"), kept))
	}
	return lines
}

func stopLine(s Stop) string {
	var b strings.Builder
	switch s.Kind {
	case StopWorkOwed:
		b.WriteString("stopped with work owed")
	case StopNoNextLook:
		b.WriteString("stopped with no next look")
	case StopCheckIncomplete:
		b.WriteString("allowed to stop, the check out of time")
	case StopAPIError:
		b.WriteString("stopped on an API error")
		if s.ErrorType != "" {
			b.WriteString(" (" + clean(s.ErrorType) + ")")
		}
	default:
		b.WriteString("stopped (" + clean(string(s.Kind)) + ")")
	}
	if s.PossiblyForcedByCap {
		b.WriteString(", possibly forced by the cap")
	}
	if s.Detail != "" {
		b.WriteString(": " + termsafe.CleanProseLine(s.Detail, 256))
	}
	if s.Count > 1 {
		fmt.Fprintf(&b, " (%d times, first %s, last %s)", s.Count, showTime(s.First), showTime(s.Last))
	} else {
		b.WriteString(" (" + showTime(s.Last) + ")")
	}
	return b.String()
}

func clean(s string) string { return termsafe.CleanProseLine(s, echoCap*2) }

func shortHead(h string) string {
	if len(h) > 12 {
		h = h[:12]
	}
	return clean(h)
}

// showTime renders a time for the person, in the machine's local time.
func showTime(t time.Time) string { return t.Local().Format("2006-01-02 15:04 MST") }

func showWho(name string) string {
	if strings.TrimSpace(name) == "" {
		return "someone git does not name"
	}
	return clean(name)
}

func showCopy(p string) string {
	if p == "" {
		return "an unnamed working copy"
	}
	return clean(fsutil.DisplayPath(p))
}

// clockRe is a clock time, H:MM or HH:MM.
var clockRe = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)

// localLayouts are the timestamp forms read in now's location; RFC 3339,
// which names its own offset, is tried first.
var localLayouts = []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02 15:04"}

// ParseUntil reads --until against now, in now's location (the machine's
// local time at the front door): a clock time is its next occurrence after
// now; a timestamp is RFC 3339 or a date and time read in local time. One
// that is neither is ErrUntilUnreadable, and one not after now ErrUntilPast.
func ParseUntil(s string, now time.Time) (time.Time, error) {
	raw := s
	s = strings.TrimSpace(s)
	loc := now.Location()
	if m := clockRe.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		if h > 23 || mi > 59 {
			return time.Time{}, unreadable(raw)
		}
		y, mo, d := now.Date()
		t := time.Date(y, mo, d, h, mi, 0, 0, loc)
		if !t.After(now) {
			t = time.Date(y, mo, d+1, h, mi, 0, 0, loc)
		}
		return t, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		for _, layout := range localLayouts {
			if t, err = time.ParseInLocation(layout, s, loc); err == nil {
				break
			}
		}
	}
	if err != nil {
		return time.Time{}, unreadable(raw)
	}
	if !t.After(now) {
		return time.Time{}, fmt.Errorf("%w: %s", ErrUntilPast, showTime(t))
	}
	return t, nil
}

func unreadable(raw string) error {
	return fmt.Errorf("%w: %q is neither a clock time (09:00) nor a timestamp (2026-10-19T09:00, or RFC 3339)",
		ErrUntilUnreadable, termsafe.CleanProseLine(raw, echoCap))
}
