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
	"strings"
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
	// EventCeilingOverrun is a session going over its agent ceiling: alive (the
	// agents alive), ceiling, lane and minutes (how long it was over). The verb
	// refuses an agent_start past the ceiling, so an overrun is what the host
	// did anyway — a fork, an agent started outside the log — and says so.
	EventCeilingOverrun = "ceiling_overrun"

	// The evidence events: what an autonomous run records so a later run can be
	// built to need no person. An intervention is a person acting on the run
	// (kind, by, what, why, autonomy_gap — what abcd or the host would need so
	// no person is needed — and optionally at and detected_after_min); a stop is
	// a stall (cause, and optionally last_productive, noticed_after_min and
	// recovery); a decision is a judgement call a person would normally make
	// (what, alternative, why, and optionally at).
	EventIntervention = "intervention"
	EventDecision     = "decision"

	// EventLoad is the load check's warning (itd-2609231434459890), written by
	// `implement load` alone and only when it warns inside a live run. The run's
	// hand-kept load samples share the name and carry no `triggers`, which is how
	// a reader tells the two apart.
	EventLoad = "load"
)

// verbOwnedEvents are written by join, leave, mode, claim, release, load and
// the outage verbs alone: a hand-written outage_end would end an outage the
// run state still holds.
var verbOwnedEvents = []string{
	EventSessionOpen, EventSessionClose, EventWindowMode,
	EventClaim, EventClaimDenied, EventClaimLapsed, EventClaimReleased,
	EventLoad,
	EventOutageStart, EventOutageProbe, EventOutageEnd, EventOutageGiveUp,
}

// loggableEvents are the events `implement log` writes on a session's word.
var loggableEvents = []string{
	EventBackoff, EventLaneOpen, EventLaneClose, EventAgentStart, EventAgentEnd,
	EventCeilingWait, EventGateRun, EventReview, EventFallback, EventStop,
	EventRefusal, EventPR, EventCapture, EventContext, EventCeilingOverrun,
	EventIntervention, EventDecision,
}

// InterventionKinds is the closed vocabulary of an intervention's kind.
var InterventionKinds = []string{
	"session_open", "account", "ruling", "restart", "close_session", "file_restore", "permission", "other",
}

// fieldKind is what a checked field must hold.
type fieldKind int

const (
	fieldText   fieldKind = iota // any non-empty value
	fieldNumber                  // a non-negative number
	fieldTime                    // an RFC 3339 timestamp
	fieldKindOf                  // one of InterventionKinds
)

// fieldRule is one field an event is checked for. A rule with alternatives is
// met by any one of them (agent_end's minutes under the keys the report reads).
type fieldRule struct {
	names    []string
	kind     fieldKind
	optional bool
}

// eventFields are the fields `implement log` requires, or checks when given, per
// event: the ones the report counts, so a line it would read as absent is
// refused when it is written rather than found missing afterwards
// (iss-2609240646555891). An event not listed takes any fields.
var eventFields = map[string][]fieldRule{
	EventLaneClose:  {{names: []string{"lane"}}, {names: []string{"outcome"}}},
	EventAgentStart: {{names: []string{"agent"}}},
	EventAgentEnd: {{names: []string{"agent"}}, {names: []string{"role"}}, {names: []string{"model"}},
		{names: []string{"minutes", "wall_minutes", "wall_min"}, kind: fieldNumber}},
	EventCeilingOverrun: {{names: []string{"alive"}, kind: fieldNumber}, {names: []string{"ceiling"}, kind: fieldNumber},
		{names: []string{"lane"}}, {names: []string{"minutes"}, kind: fieldNumber}},
	EventIntervention: {{names: []string{"kind"}, kind: fieldKindOf}, {names: []string{"by"}}, {names: []string{"what"}},
		{names: []string{"why"}}, {names: []string{"autonomy_gap"}},
		{names: []string{"at"}, kind: fieldTime, optional: true},
		{names: []string{"detected_after_min"}, kind: fieldNumber, optional: true}},
	EventStop: {{names: []string{"cause"}},
		{names: []string{"last_productive"}, kind: fieldTime, optional: true},
		{names: []string{"noticed_after_min"}, kind: fieldNumber, optional: true}},
	EventDecision: {{names: []string{"what"}}, {names: []string{"alternative"}}, {names: []string{"why"}},
		{names: []string{"at"}, kind: fieldTime, optional: true}},
	// The outage events are verb-owned: these rules hold the verbs' own lines
	// (checkOutageFields) and let the report name a hand-appended line that
	// lacks what it reads.
	EventOutageStart: {{names: []string{"service"}}, {names: []string{"kind"}}, {names: []string{"lane"}}, {names: []string{"what"}}},
	EventOutageProbe: {{names: []string{"ok"}}, {names: []string{"failures"}, kind: fieldNumber}},
	EventOutageEnd: {{names: []string{"minutes"}, kind: fieldNumber}, {names: []string{"services"}}, {names: []string{"kinds"}},
		{names: []string{"retried"}}, {names: []string{"started_at"}, kind: fieldTime}},
	EventOutageGiveUp: {{names: []string{"minutes"}, kind: fieldNumber}, {names: []string{"services"}}, {names: []string{"kinds"}},
		{names: []string{"retried"}}, {names: []string{"started_at"}, kind: fieldTime}, {names: []string{"hourly_since"}, kind: fieldTime}},
}

// RequiredFields returns the fields `implement log` requires on event, each as
// its accepted names ("minutes|wall_minutes|wall_min"), for help text and the
// report's missing-field count.
func RequiredFields(event string) []string {
	var out []string
	for _, r := range eventFields[event] {
		if !r.optional {
			out = append(out, strings.Join(r.names, "|"))
		}
	}
	return out
}

// checkFields refuses an event whose fields break its rules, naming the field.
func checkFields(event string, fields map[string]string) error {
	for _, r := range eventFields[event] {
		name, v, ok := "", "", false
		for _, n := range r.names {
			if val, has := fields[n]; has {
				name, v, ok = n, val, true
				break
			}
		}
		if !ok {
			if r.optional {
				continue
			}
			return refusal("%s needs the field %s (the report reads it; required: %s)",
				event, strings.Join(r.names, " or "), strings.Join(RequiredFields(event), ", "))
		}
		if strings.TrimSpace(v) == "" {
			return refusal("%s field %s is empty", event, name)
		}
		switch r.kind {
		case fieldNumber:
			if f, err := strconv.ParseFloat(v, 64); err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
				return refusal("%s field %s is %q, not a number of zero or more", event, name, v)
			}
		case fieldTime:
			if _, err := time.Parse(time.RFC3339, v); err != nil {
				return refusal("%s field %s is %q, not an RFC 3339 time (2006-01-02T15:04:05Z)", event, name, v)
			}
		case fieldKindOf:
			if !slices.Contains(InterventionKinds, v) {
				return refusal("%s field %s is %q (one of: %s)", event, name, v, strings.Join(InterventionKinds, ", "))
			}
		}
	}
	return nil
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

// AgentsAlive counts the agents a session has declared alive: the agents named
// by its agent_start lines since it joined, each with no agent_end of the same
// agent after it. It counts what the session wrote and nothing else — abcd runs
// no agent, and a line the session never wrote (a fork, an agent the host
// started outside the log) is invisible here — so it is the declared count the
// ceiling is held against, not a census of processes. Lines naming no agent
// cannot be matched and are not counted.
func (r *Run) AgentsAlive(s Session) (int, error) {
	alive, err := r.aliveAgents(s)
	return len(alive), err
}

// aliveAgents is AgentsAlive's set: the agent names alive, by the log.
func (r *Run) aliveAgents(s Session) (map[string]bool, error) {
	events, _, err := r.ReadLog()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].TS.Before(events[j].TS) })
	alive := map[string]bool{}
	for _, e := range events {
		if e.Session != s.Session || e.TS.Before(s.JoinedAt) {
			continue
		}
		name := e.String("agent")
		if name == "" {
			continue
		}
		switch e.Event {
		case EventAgentStart:
			alive[name] = true
		case EventAgentEnd:
			delete(alive, name)
		}
	}
	return alive, nil
}

// agentCeiling refuses, and logs the refusal of, an agent_start that would take
// the session past its ceiling. An agent already alive, restated, is not a new
// one. The caller holds the lock.
func (r *Run) agentCeiling(s Session, agent string) error {
	alive, err := r.aliveAgents(s)
	if err != nil {
		return err
	}
	if len(alive) < s.Ceiling || alive[agent] {
		return nil
	}
	return r.refuseLogged(s.Session, "agent_ceiling",
		map[string]any{"agent": agent, "alive": len(alive), "ceiling": s.Ceiling},
		fmt.Sprintf("session %s has %d agent(s) alive of its ceiling %d; log the agent_end of one before starting %s",
			s.Session, len(alive), s.Ceiling, agent))
}

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
	if event == EventBackoff {
		if err := backoffFields(fields); err != nil {
			return Event{}, err
		}
	}
	if err := checkFields(event, fields); err != nil {
		return Event{}, err
	}
	var out Event
	err := r.withLock(session, func() error {
		s, err := r.requireSession(session)
		if err != nil {
			return err
		}
		if event == EventAgentStart && s.Ceiling > 0 {
			if err := r.agentCeiling(s, fields["agent"]); err != nil {
				return err
			}
		}
		ts, err := r.append(session, event, typed)
		out = Event{TS: ts, Session: session, Event: event}
		return err
	})
	return out, err
}

// backoffFields holds a hand-logged backoff to what criterion 6 of
// itd-2609221656373558 says the log names: the reason the session backed off and
// the minutes it spent, a number no smaller than zero. A backoff missing either
// counts in the comparison as a backoff that cost nothing for no reason, so it
// is refused rather than written.
func backoffFields(fields map[string]string) error {
	if strings.TrimSpace(fields["reason"]) == "" {
		return refusal("a backoff names its reason (--field reason=<why>)")
	}
	m, err := strconv.ParseFloat(fields["minutes"], 64)
	if err != nil || math.IsNaN(m) || math.IsInf(m, 0) || m < 0 {
		return refusal("a backoff names the minutes it spent as a number no smaller than zero (--field minutes=<n>), not %q", fields["minutes"])
	}
	return nil
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
