package implement

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The run log is one file per UTC day, one JSON object per line, append-only.
// Every line carries `ts` (RFC 3339, UTC), `session` and `event`, then the
// event's own fields. A line reaches the file in one O_APPEND write
// (fsutil.AppendLineIn), so two sessions writing at once each land whole lines.
//
// The vocabulary is the run's measurement ruling of 2026-09-22 plus the one
// event this package adds (claim_released). Lines the run writes by hand with
// `printf >>` stay readable here, because the reader is keyed on the same three
// fields and ignores nothing it does not understand.

// Event names. The first group is written only by this package's verbs, so the
// log cannot claim a join, a window or a claim that the run state does not hold.
const (
	EventSessionOpen   = "session_open"
	EventSessionClose  = "session_close"
	EventWindowMode    = "window_mode"
	EventClaim         = "claim"
	EventClaimDenied   = "claim_denied"
	EventClaimLapsed   = "claim_lapsed"
	EventClaimReleased = "claim_released"

	EventBackoff     = "backoff"
	EventLaneOpen    = "lane_open"
	EventLaneClose   = "lane_close"
	EventAgentStart  = "agent_start"
	EventAgentEnd    = "agent_end"
	EventCeilingWait = "ceiling_wait"
	EventGateRun     = "gate_run"
	EventReview      = "review"
	EventFallback    = "fallback"
	EventStop        = "stop"
	EventRefusal     = "refusal"
	EventPR          = "pr"
	EventCapture     = "capture"
	// EventContext is an orchestrator's context measurement: used_pct (the share
	// of its context window in use), role and note. The run measures it because
	// it is also an experiment in keeping a session alive for days.
	EventContext = "context"

	// EventLoad is the load check's warning (itd-2609231434459890), written by
	// `implement load` alone and only when it warns inside a live run. The run's
	// hand-kept load samples share the name and carry no `triggers`, which is how
	// a reader tells the two apart.
	EventLoad = "load"
)

// verbOwnedEvents are written by join, leave, mode, claim, release and load
// alone.
var verbOwnedEvents = []string{
	EventSessionOpen, EventSessionClose, EventWindowMode,
	EventClaim, EventClaimDenied, EventClaimLapsed, EventClaimReleased,
	EventLoad,
}

// loggableEvents are the events `implement log` writes on a session's word.
var loggableEvents = []string{
	EventBackoff, EventLaneOpen, EventLaneClose, EventAgentStart, EventAgentEnd,
	EventCeilingWait, EventGateRun, EventReview, EventFallback, EventStop,
	EventRefusal, EventPR, EventCapture, EventContext,
}

// LoggableEvents returns the events `implement log` accepts, for help text and
// shell completion.
func LoggableEvents() []string { return slices.Clone(loggableEvents) }

// Limits on one hand-logged event, so a runaway caller cannot turn the log into
// something no reader can hold.
const (
	maxFields     = 32
	maxValueBytes = 1024
	maxLineBytes  = 16 << 10
	// maxLogBytes caps one day's file on read.
	maxLogBytes = 64 << 20
)

// reservedFields are the three every line carries; a field cannot restate them.
var reservedFields = []string{"ts", "session", "event"}

// fieldKeyRe is a field name: lower snake case, as the run's events use.
var fieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// logFileRe is the name of one day's log.
var logFileRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}\.jsonl$`)

// Event is one line of the run log as it was read: the three fixed fields
// parsed, and every field (the fixed three included) kept as raw JSON so a
// reader can take what it understands and leave the rest.
type Event struct {
	TS      time.Time                  `json:"ts"`
	Session string                     `json:"session"`
	Event   string                     `json:"event"`
	Fields  map[string]json.RawMessage `json:"-"`
}

// String returns a field as a string: a JSON string's value, a number's or a
// boolean's literal text, "" when absent or null.
func (e Event) String(key string) string {
	raw, ok := e.Fields[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	t := string(bytes.TrimSpace(raw))
	if t == "null" {
		return ""
	}
	return t
}

// Number returns a field as a number, accepting a JSON number or a string that
// holds one (a hand-written line may quote either), and ok=false otherwise.
func (e Event) Number(key string) (float64, bool) {
	raw, ok := e.Fields[key]
	if !ok {
		return 0, false
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return f, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v, true
		}
	}
	return 0, false
}

// encodeLine renders one log line: ts, session and event first, then the fields
// in key order, so a line reads the same way whoever wrote it.
func encodeLine(ts time.Time, session, event string, fields map[string]any) ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	write := func(k string, v any) error {
		if b.Len() > 1 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		vb, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("field %s: %w", k, err)
		}
		b.Write(kb)
		b.WriteByte(':')
		b.Write(vb)
		return nil
	}
	if err := write("ts", ts.UTC().Format(time.RFC3339)); err != nil {
		return nil, err
	}
	if err := write("session", session); err != nil {
		return nil, err
	}
	if err := write("event", event); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := write(k, fields[k]); err != nil {
			return nil, err
		}
	}
	b.WriteByte('}')
	if b.Len() > maxLineBytes {
		return nil, refusal("the event is %d bytes, over the %d a log line may carry", b.Len(), maxLineBytes)
	}
	return b.Bytes(), nil
}

// append writes one event line to the day's log. It is the one writer every
// verb in this package goes through.
func (r *Run) append(session, event string, fields map[string]any) (time.Time, error) {
	ts := r.now()
	line, err := encodeLine(ts, session, event, fields)
	if err != nil {
		return ts, err
	}
	root, err := r.root()
	if err != nil {
		return ts, fmt.Errorf("cannot open the run state: %w", err)
	}
	defer root.Close()
	if err := fsutil.AppendLineIn(root, logFileName(ts), line, fileMode); err != nil {
		return ts, fmt.Errorf("cannot append to the run log: %w", err)
	}
	return ts, nil
}

// logFileName is the day file an event at ts belongs in.
func logFileName(ts time.Time) string { return ts.UTC().Format(time.DateOnly) + ".jsonl" }

// Log appends one event on a joined session's word: the run's hand-kept events
// (lane_open, agent_end, backoff, …) through the same single-write append the
// verbs use. The event must be one of LoggableEvents — the verb-owned events are
// refused, because a hand-written claim the claims directory does not hold is a
// log that lies — and fields are key=value pairs under the limits above. A value
// that parses as an integer, a decimal or a boolean is written as a JSON number
// or boolean when it round-trips to the same text; anything else as a string.
func (r *Run) Log(session, event string, fields map[string]string) (Event, error) {
	if slices.Contains(verbOwnedEvents, event) {
		return Event{}, refusal("%s is written by the implement verbs themselves, never by hand", event)
	}
	if !slices.Contains(loggableEvents, event) {
		return Event{}, refusal("unknown event %q (one of: %v)", event, loggableEvents)
	}
	if len(fields) > maxFields {
		return Event{}, refusal("%d fields, over the %d an event may carry", len(fields), maxFields)
	}
	typed := make(map[string]any, len(fields))
	for k, v := range fields {
		if slices.Contains(reservedFields, k) {
			return Event{}, refusal("field %q is set by the log itself", k)
		}
		if !fieldKeyRe.MatchString(k) {
			return Event{}, refusal("field name %q is not lower snake case", k)
		}
		if len(v) > maxValueBytes {
			return Event{}, refusal("field %s is %d bytes, over %d", k, len(v), maxValueBytes)
		}
		typed[k] = typedValue(v)
	}
	var out Event
	err := r.withLock(func() error {
		if _, err := r.requireSession(session); err != nil {
			return err
		}
		ts, err := r.append(session, event, typed)
		out = Event{TS: ts, Session: session, Event: event}
		return err
	})
	return out, err
}

// typedValue reads a hand-given value as the JSON type it spells, and only when
// writing that value back gives the same text: `0123456` (a short sha), `-0`,
// `+5`, `1.50` and `1e3` stay the strings they were, because a number would
// lose what the caller wrote. The report's readers accept a quoted number, so a
// figure kept as a string is still counted.
func typedValue(v string) any {
	switch v {
	case "true":
		return true
	case "false":
		return false
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil && strconv.FormatInt(i, 10) == v {
		return i
	}
	// NaN and the infinities have no JSON spelling, and a negative zero reads
	// back as zero.
	if f, err := strconv.ParseFloat(v, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) &&
		!(f == 0 && math.Signbit(f)) && strconv.FormatFloat(f, 'f', -1, 64) == v {
		return f
	}
	return v
}

// Unparsed is one log line the reader could not use, with why.
type Unparsed struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

// ReadLog reads every day's log in the run directory, in date order. A missing
// directory is an empty log. Lines that are not a JSON object carrying ts,
// session and event are returned as Unparsed rather than failing the read: the
// log is hand-written in part, and a comparison that stopped at the first odd
// line would report nothing.
func (r *Run) ReadLog() ([]Event, []Unparsed, error) {
	if !r.exists {
		return nil, nil, nil
	}
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	root, err := r.root()
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	var events []Event
	var bad []Unparsed
	for _, e := range entries {
		if e.IsDir() || !logFileRe.MatchString(e.Name()) {
			continue
		}
		data, err := fsutil.ReadGuardedInRoot(root, e.Name(), maxLogBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot read the run log %s: %w", e.Name(), err)
		}
		ev, b := ParseLog(e.Name(), data)
		events = append(events, ev...)
		bad = append(bad, b...)
	}
	return events, bad, nil
}

// ParseLog parses one log file's bytes. name labels Unparsed entries.
func ParseLog(name string, data []byte) ([]Event, []Unparsed) {
	var events []Event
	var bad []Unparsed
	for i, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		ev, reason := parseLine(line)
		if reason != "" {
			bad = append(bad, Unparsed{File: name, Line: i + 1, Reason: reason})
			continue
		}
		events = append(events, ev)
	}
	return events, bad
}

// parseLine parses one line, or says why it cannot.
func parseLine(line []byte) (Event, string) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return Event{}, "not a JSON object"
	}
	ev := Event{Fields: fields}
	ev.Session = ev.String("session")
	ev.Event = ev.String("event")
	tsText := ev.String("ts")
	if ev.Event == "" || ev.Session == "" || tsText == "" {
		return Event{}, "missing ts, session or event"
	}
	ts, err := time.Parse(time.RFC3339, tsText)
	if err != nil {
		return Event{}, "ts is not RFC 3339"
	}
	ev.TS = ts.UTC()
	return ev, ""
}
