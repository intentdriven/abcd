package implement

import (
	"encoding/json"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

// The comparison is derived from the log and from nothing else: the run's
// report reads these numbers rather than counting them by hand
// (itd-2609221656373558, criterion 7). Each event belongs to the window open
// when it happened — the last window_mode line at or before it — and each
// window to its mode, so a mode's figures are the sum over every window that ran
// it. Events before the first window_mode fall in a window labelled "unset".
// One exception: a session_open logged at most JoinGrace before the next
// window_mode belongs to that window — a session joining a second before the
// first sets the mode is joining that window (iss-2609240646544930).
//
// What each figure reads, so a hand-written line can be written to be counted:
//
//   - lanes opened: lane_open lines;
//   - lanes landed: lane_close lines whose outcome is "merged" or "landed";
//   - wall clock: per window, the minutes from its window_mode line to the last
//     event in it;
//   - collisions: claim_denied lines — two sessions reaching for one record;
//   - agent minutes: agent_end lines' minutes (or wall_minutes, or wall_min,
//     the key the run's hand-kept lines carry);
//   - backed-off minutes: backoff lines' minutes, the agent time spent on work
//     that was then abandoned (reported beside agent minutes, never folded into
//     them, since the backed-off agent may or may not have logged an agent_end);
//   - ceiling wait: ceiling_wait lines' minutes, per session;
//   - ceiling overruns: ceiling_overrun lines, and their minutes over;
//   - context: per session over the whole run, not per mode, the number of
//     context lines and the last used_pct among them;
//   - evidence: over the whole run, interventions (by kind, with the minutes
//     each went unnoticed), stops (with the minutes before each was noticed) and
//     decisions;
//   - missing fields: per event and field, the lines lacking a field
//     `implement log` requires of that event — lines written by hand, or before
//     the requirement, that the figures above read as absent;
//   - coverage: each of the measured hand-kept events (CoverageEvents) whose
//     last line falls more than CoverageGapAfter before the run's last line, so
//     a figure that stops partway through the run says so.
//
// A session counts as the second when its session_open says role "second" (or
// "B", the label the run's hand-written lines use).

// UnsetMode labels the events logged before any window opened.
const UnsetMode = "unset"

// JoinGrace is how long before a window_mode a session_open may be logged and
// still count in that window.
const JoinGrace = time.Minute

// CoverageGapAfter is how long before the run's last line a measured event's
// last line may fall before the report names its coverage as stopping.
const CoverageGapAfter = 6 * time.Hour

// CoverageEvents are the hand-kept events the report's figures rest on and a
// run writes throughout, so their stopping partway is a gap, not a lull.
var CoverageEvents = []string{EventLaneOpen, EventLaneClose, EventAgentStart, EventAgentEnd, EventGateRun}

// landedOutcomes are the lane_close outcomes that count as landed.
var landedOutcomes = map[string]bool{"merged": true, "landed": true}

// SessionTally is one session's share of a mode.
type SessionTally struct {
	Session            string  `json:"session"`
	Role               string  `json:"role"`
	LanesOpened        int     `json:"lanes_opened"`
	LanesLanded        int     `json:"lanes_landed"`
	AgentMinutes       float64 `json:"agent_minutes"`
	BackoffMinutes     float64 `json:"backoff_minutes"`
	CeilingWaitMinutes float64 `json:"ceiling_wait_minutes"`
	CeilingOverruns    int     `json:"ceiling_overruns"`
	Collisions         int     `json:"collisions"`
}

// ModeTally is one division mode's figures over every window that ran it.
type ModeTally struct {
	Mode               string  `json:"mode"`
	Windows            int     `json:"windows"`
	WallMinutes        float64 `json:"wall_minutes"`
	LanesOpened        int     `json:"lanes_opened"`
	LanesLanded        int     `json:"lanes_landed"`
	SecondLanesLanded  int     `json:"second_session_lanes_landed"`
	Collisions         int     `json:"collisions"`
	Lapses             int     `json:"claims_lapsed"`
	Backoffs           int     `json:"backoffs"`
	BackoffMinutes     float64 `json:"backoff_minutes"`
	AgentMinutes       float64 `json:"agent_minutes"`
	CeilingWaitMinutes float64 `json:"ceiling_wait_minutes"`
	// CeilingOverruns are the ceiling_overrun lines, and CeilingOverrunMinutes
	// the minutes over they carry.
	CeilingOverruns       int            `json:"ceiling_overruns"`
	CeilingOverrunMinutes float64        `json:"ceiling_overrun_minutes"`
	Refusals              int            `json:"refusals"`
	LandedPerHour         float64        `json:"lanes_landed_per_hour"`
	Sessions              []SessionTally `json:"sessions"`
}

// SessionContext is one session's context measurement across the whole run:
// how many context lines it logged and the last used_pct among them. It is per
// session rather than per mode because an orchestrator's context is carried
// across windows, not reset by them.
type SessionContext struct {
	Session     string    `json:"session"`
	Role        string    `json:"role"`
	Events      int       `json:"events"`
	LastUsedPct float64   `json:"last_used_pct"`
	LastAt      time.Time `json:"last_at"`
}

// Evidence is the run's evidence events counted over the whole run: what a run
// that needs no person would have to do without.
type Evidence struct {
	Interventions       int            `json:"interventions"`
	InterventionsByKind map[string]int `json:"interventions_by_kind"`
	// DetectedAfterMinutes sums the interventions' detected_after_min: how long
	// the needs went unnoticed.
	DetectedAfterMinutes float64 `json:"detected_after_minutes"`
	Stops                int     `json:"stops"`
	// StopNoticedAfterMinutes sums the stops' noticed_after_min.
	StopNoticedAfterMinutes float64 `json:"stop_noticed_after_minutes"`
	Decisions               int     `json:"decisions"`
}

// FieldGap is a field `implement log` requires of an event, and how many of the
// log's lines of that event lack it.
type FieldGap struct {
	Event string `json:"event"`
	Field string `json:"field"`
	Lines int    `json:"lines"`
	Of    int    `json:"of"`
}

// CoverageGap is a measured event whose lines stop partway through the run.
type CoverageGap struct {
	Event string    `json:"event"`
	Lines int       `json:"lines"`
	Last  time.Time `json:"last"`
	// RunLast is the run's last line, load samples aside.
	RunLast     time.Time `json:"run_last"`
	HoursBefore float64   `json:"hours_before"`
}

// Report is the derived comparison.
type Report struct {
	Events   int         `json:"events"`
	Unparsed []Unparsed  `json:"unparsed"`
	Modes    []ModeTally `json:"modes"`
	// Context is each session's context measurement, by session id.
	Context []SessionContext `json:"context"`
	// Evidence counts the run's interventions, stops and decisions.
	Evidence Evidence `json:"evidence"`
	// MissingFields names, per event and required field, the lines lacking it.
	MissingFields []FieldGap `json:"missing_fields"`
	// Coverage names each measured event whose lines stop partway.
	Coverage []CoverageGap `json:"coverage"`
	// Leader is the mode with the most lanes landed per wall-clock hour, "" when
	// no mode landed a lane or two modes tie. It is a figure, not a verdict: the
	// run's report names which mode it would keep, and says why.
	Leader      string `json:"leader"`
	LeaderBasis string `json:"leader_basis"`
	// Outages are the run's lost connections, each with how long it lasted and
	// what was retried, so a short outage is reported though nobody was told of
	// it at the time.
	Outages []OutageSpan `json:"outages"`
}

// OutageSpan is one outage as the log records it.
type OutageSpan struct {
	Start time.Time `json:"start"`
	// End is the outage_end or outage_give_up line; nil while it is open.
	End *time.Time `json:"end,omitempty"`
	// Outcome is "ended" (a probe proved every service back), "cleared" (closed
	// by hand), "gave_up", or "open" at the log's last line.
	Outcome string `json:"outcome"`
	// Minutes is the line's own figure for a closed outage, and for an open one
	// the minutes to the log's last line.
	Minutes  float64  `json:"minutes"`
	Services []string `json:"services"`
	Kinds    []string `json:"kinds"`
	Retried  []string `json:"retried"`
	Probes   int      `json:"probes"`
}

// Compare derives the comparison from a log. It never fails: a line it cannot
// use was already set aside by the parser, and a field it cannot read counts as
// absent.
func Compare(events []Event, unparsed []Unparsed) Report {
	sorted := append([]Event(nil), events...)
	// Time order, and at one instant the window line first: an event belongs to
	// the last window_mode at or before it, so a session_open written in the same
	// second as the window it opens is that window's, not the one before.
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].TS.Equal(sorted[j].TS) {
			return sorted[i].TS.Before(sorted[j].TS)
		}
		return sorted[i].Event == EventWindowMode && sorted[j].Event != EventWindowMode
	})
	sorted = joinsIntoTheirWindow(sorted)

	roles := map[string]string{}
	for _, e := range sorted {
		if e.Event == EventSessionOpen {
			if r := normaliseRole(e.String("role")); r != "" {
				roles[e.Session] = r
			}
		}
	}

	type window struct {
		mode        string
		start, last time.Time
	}
	tallies := map[string]*ModeTally{}
	sessions := map[string]map[string]*SessionTally{}
	get := func(mode string) *ModeTally {
		if t, ok := tallies[mode]; ok {
			return t
		}
		t := &ModeTally{Mode: mode}
		tallies[mode] = t
		sessions[mode] = map[string]*SessionTally{}
		return t
	}
	sess := func(mode, id string) *SessionTally {
		get(mode)
		if s, ok := sessions[mode][id]; ok {
			return s
		}
		s := &SessionTally{Session: id, Role: roles[id]}
		sessions[mode][id] = s
		return s
	}
	contexts := map[string]*SessionContext{}
	var windows []window
	cur := -1
	closeWindow := func() {
		if cur >= 0 {
			w := windows[cur]
			t := get(w.mode)
			t.Windows++
			t.WallMinutes += w.last.Sub(w.start).Minutes()
		}
	}
	for _, e := range sorted {
		if e.Event == EventWindowMode {
			closeWindow()
			mode := e.String("mode")
			if mode == "" {
				mode = UnsetMode
			}
			windows = append(windows, window{mode: mode, start: e.TS, last: e.TS})
			cur = len(windows) - 1
		} else if cur < 0 {
			windows = append(windows, window{mode: UnsetMode, start: e.TS, last: e.TS})
			cur = 0
		}
		w := &windows[cur]
		if e.TS.After(w.last) {
			w.last = e.TS
		}
		t := get(w.mode)
		s := sess(w.mode, e.Session)
		minutes := func(keys ...string) float64 {
			for _, k := range keys {
				if v, ok := e.Number(k); ok && v > 0 && !math.IsInf(v, 0) {
					return v
				}
			}
			return 0
		}
		switch e.Event {
		case EventLaneOpen:
			t.LanesOpened++
			s.LanesOpened++
		case EventLaneClose:
			if landedOutcomes[e.String("outcome")] {
				t.LanesLanded++
				s.LanesLanded++
				if roles[e.Session] == string(RoleSecond) {
					t.SecondLanesLanded++
				}
			}
		case EventClaimDenied:
			t.Collisions++
			s.Collisions++
		case EventClaimLapsed:
			t.Lapses++
		case EventBackoff:
			m := minutes("minutes")
			t.Backoffs++
			t.BackoffMinutes += m
			s.BackoffMinutes += m
		case EventAgentEnd:
			m := minutes("minutes", "wall_minutes", "wall_min")
			t.AgentMinutes += m
			s.AgentMinutes += m
		case EventCeilingWait:
			m := minutes("minutes")
			t.CeilingWaitMinutes += m
			s.CeilingWaitMinutes += m
		case EventCeilingOverrun:
			t.CeilingOverruns++
			t.CeilingOverrunMinutes += minutes("minutes")
			s.CeilingOverruns++
		case EventRefusal:
			t.Refusals++
		case EventContext:
			c, ok := contexts[e.Session]
			if !ok {
				c = &SessionContext{Session: e.Session, Role: roles[e.Session]}
				contexts[e.Session] = c
			}
			c.Events++
			if v, ok := e.Number("used_pct"); ok && !math.IsInf(v, 0) && !math.IsNaN(v) {
				c.LastUsedPct, c.LastAt = v, e.TS
			}
		}
	}
	closeWindow()

	rep := Report{Events: len(sorted), Unparsed: unparsed, Modes: []ModeTally{}, Context: []SessionContext{},
		LeaderBasis: "lanes landed per wall-clock hour", Evidence: evidenceOf(sorted),
		MissingFields: missingFields(sorted), Coverage: coverageGaps(sorted), Outages: outageSpans(sorted)}
	for _, c := range contexts {
		rep.Context = append(rep.Context, *c)
	}
	sort.Slice(rep.Context, func(i, j int) bool { return rep.Context[i].Session < rep.Context[j].Session })
	if rep.Unparsed == nil {
		rep.Unparsed = []Unparsed{}
	}
	for _, t := range tallies {
		if t.WallMinutes > 0 {
			t.LandedPerHour = round2(float64(t.LanesLanded) / (t.WallMinutes / 60))
		}
		t.WallMinutes = round2(t.WallMinutes)
		t.CeilingOverrunMinutes = round2(t.CeilingOverrunMinutes)
		t.Sessions = []SessionTally{}
		for _, s := range sessions[t.Mode] {
			t.Sessions = append(t.Sessions, *s)
		}
		sort.Slice(t.Sessions, func(i, j int) bool { return t.Sessions[i].Session < t.Sessions[j].Session })
		rep.Modes = append(rep.Modes, *t)
	}
	sort.Slice(rep.Modes, func(i, j int) bool {
		return modeRank(rep.Modes[i].Mode) < modeRank(rep.Modes[j].Mode) ||
			(modeRank(rep.Modes[i].Mode) == modeRank(rep.Modes[j].Mode) && rep.Modes[i].Mode < rep.Modes[j].Mode)
	})

	best, tie := -1.0, false
	for _, t := range rep.Modes {
		if t.LanesLanded == 0 || t.WallMinutes <= 0 {
			continue
		}
		switch {
		case t.LandedPerHour > best:
			best, tie, rep.Leader = t.LandedPerHour, false, t.Mode
		case t.LandedPerHour == best:
			tie = true
		}
	}
	if tie {
		rep.Leader = ""
	}
	return rep
}

// joinsIntoTheirWindow moves each session_open logged at most JoinGrace before
// the next window_mode to just after that window_mode, keeping every other
// event where it is. events are in time order.
func joinsIntoTheirWindow(events []Event) []Event {
	after := map[int][]int{} // window_mode index -> the joins moved behind it
	moved := map[int]bool{}
	for i, e := range events {
		if e.Event != EventSessionOpen {
			continue
		}
		for j := i + 1; j < len(events); j++ {
			if events[j].Event != EventWindowMode {
				continue
			}
			if gap := events[j].TS.Sub(e.TS); gap > 0 && gap <= JoinGrace {
				after[j] = append(after[j], i)
				moved[i] = true
			}
			break
		}
	}
	if len(moved) == 0 {
		return events
	}
	out := make([]Event, 0, len(events))
	for i, e := range events {
		if moved[i] {
			continue
		}
		out = append(out, e)
		for _, k := range after[i] {
			out = append(out, events[k])
		}
	}
	return out
}

// evidenceOf counts the evidence events.
func evidenceOf(events []Event) Evidence {
	ev := Evidence{InterventionsByKind: map[string]int{}}
	num := func(e Event, key string) float64 {
		if v, ok := e.Number(key); ok && v > 0 && !math.IsInf(v, 0) {
			return v
		}
		return 0
	}
	for _, e := range events {
		switch e.Event {
		case EventIntervention:
			ev.Interventions++
			kind := e.String("kind")
			if kind == "" {
				kind = "unstated"
			}
			ev.InterventionsByKind[kind]++
			ev.DetectedAfterMinutes += num(e, "detected_after_min")
		case EventStop:
			ev.Stops++
			ev.StopNoticedAfterMinutes += num(e, "noticed_after_min")
		case EventDecision:
			ev.Decisions++
		}
	}
	ev.DetectedAfterMinutes = round2(ev.DetectedAfterMinutes)
	ev.StopNoticedAfterMinutes = round2(ev.StopNoticedAfterMinutes)
	return ev
}

// missingFields counts, per event and required field, the lines lacking it. A
// field is present when any of its accepted names carries a non-empty value.
func missingFields(events []Event) []FieldGap {
	type key struct{ event, field string }
	gaps := map[key]int{}
	of := map[string]int{}
	for _, e := range events {
		rules := eventFields[e.Event]
		if len(rules) == 0 {
			continue
		}
		of[e.Event]++
		for _, r := range rules {
			if r.optional {
				continue
			}
			has := false
			for _, n := range r.names {
				if e.String(n) != "" {
					has = true
					break
				}
			}
			if !has {
				gaps[key{e.Event, r.names[0]}]++
			}
		}
	}
	out := []FieldGap{}
	for k, n := range gaps {
		out = append(out, FieldGap{Event: k.event, Field: k.field, Lines: n, Of: of[k.event]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Event != out[j].Event {
			return out[i].Event < out[j].Event
		}
		return out[i].Field < out[j].Field
	})
	return out
}

// outageSpans reads the outages from the log: an outage_start opens one, each
// outage_probe while it is open counts, and an outage_end or outage_give_up
// closes it with the figures it carries. A closing line with no open outage
// (a log read from a later day than the start) opens and closes one from its
// started_at; the clear that follows a give-up closes nothing new. events are
// in time order.
func outageSpans(events []Event) []OutageSpan {
	out := []OutageSpan{}
	open := -1
	var last time.Time
	for _, e := range events {
		if e.TS.After(last) {
			last = e.TS
		}
		switch e.Event {
		case EventOutageStart:
			if open >= 0 {
				continue
			}
			out = append(out, OutageSpan{Start: e.TS, Outcome: "open",
				Services: listField(e, "service"), Kinds: listField(e, "kind"), Retried: listField(e, "what")})
			open = len(out) - 1
		case EventOutageProbe:
			if open >= 0 {
				out[open].Probes++
			}
		case EventOutageEnd, EventOutageGiveUp:
			start, _ := time.Parse(time.RFC3339, e.String("started_at"))
			if open < 0 {
				// A closer for the outage just closed (a clear after a give-up,
				// or a probe that ended it again because the record's removal
				// failed) is the same outage, not a new one.
				if n := len(out); n > 0 && out[n-1].End != nil && out[n-1].Start.Equal(start) {
					continue
				}
				if start.IsZero() {
					start = e.TS
				}
				out = append(out, OutageSpan{Start: start})
				open = len(out) - 1
			}
			s := &out[open]
			end := e.TS
			s.End = &end
			switch {
			case e.Event == EventOutageGiveUp:
				s.Outcome = "gave_up"
			case e.String("how") == "cleared":
				s.Outcome = "cleared"
			default:
				s.Outcome = "ended"
			}
			if m, ok := e.Number("minutes"); ok && m >= 0 && !math.IsInf(m, 0) {
				s.Minutes = round2(m)
			} else {
				s.Minutes = round2(end.Sub(s.Start).Minutes())
			}
			for key, dst := range map[string]*[]string{"services": &s.Services, "kinds": &s.Kinds, "retried": &s.Retried} {
				if v := listField(e, key); len(v) > 0 {
					*dst = v
				}
			}
			if n, ok := e.Number("probes"); ok && n >= 0 {
				s.Probes = int(n)
			}
			open = -1
		}
	}
	if open >= 0 {
		out[open].Minutes = round2(last.Sub(out[open].Start).Minutes())
	}
	for i := range out {
		for _, l := range []*[]string{&out[i].Services, &out[i].Kinds, &out[i].Retried} {
			if *l == nil {
				*l = []string{}
			}
		}
	}
	return out
}

// listField reads a field holding a JSON list of strings, or one string (a
// comma-separated list when written by hand), as a list.
func listField(e Event, key string) []string {
	raw, ok := e.Fields[key]
	if !ok {
		return nil
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	s := e.String(key)
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// coverageGaps names each measured event whose last line falls more than
// CoverageGapAfter before the run's last line, load samples aside (the load
// check writes them whatever the run is doing). events are in time order.
func coverageGaps(events []Event) []CoverageGap {
	var runLast time.Time
	last := map[string]time.Time{}
	lines := map[string]int{}
	for _, e := range events {
		if e.Event == EventLoad {
			continue
		}
		if e.TS.After(runLast) {
			runLast = e.TS
		}
		if slices.Contains(CoverageEvents, e.Event) {
			lines[e.Event]++
			if e.TS.After(last[e.Event]) {
				last[e.Event] = e.TS
			}
		}
	}
	out := []CoverageGap{}
	for _, ev := range CoverageEvents {
		t, ok := last[ev]
		if !ok || runLast.Sub(t) <= CoverageGapAfter {
			continue
		}
		out = append(out, CoverageGap{Event: ev, Lines: lines[ev], Last: t, RunLast: runLast,
			HoursBefore: round2(runLast.Sub(t).Hours())})
	}
	return out
}

// Compare derives the comparison over the run's whole log.
func (r *Run) Compare() (Report, error) {
	events, bad, err := r.ReadLog()
	if err != nil {
		return Report{}, err
	}
	return Compare(events, bad), nil
}

// normaliseRole maps the role labels a session_open line may carry onto the two
// roles: the vocabulary Join writes, and the A/B labels of the run's hand-kept
// lines.
func normaliseRole(s string) string {
	switch s {
	case string(RoleFirst), "A":
		return string(RoleFirst)
	case string(RoleSecond), "B":
		return string(RoleSecond)
	}
	return s
}

// modeRank orders the modes the way the run tries them, then anything else.
func modeRank(m string) int {
	for i, k := range Modes() {
		if string(k) == m {
			return i
		}
	}
	if m == UnsetMode {
		return -1
	}
	return len(Modes())
}

// round2 rounds to two decimals, so a JSON consumer sees 12.5 rather than
// 12.499999999.
func round2(f float64) float64 { return math.Round(f*100) / 100 }
