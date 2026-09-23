package implement

import (
	"math"
	"sort"
	"time"
)

// The comparison is derived from the log and from nothing else: the run's
// report reads these numbers rather than counting them by hand
// (itd-2609221656373558, criterion 7). Each event belongs to the window open
// when it happened — the last window_mode line at or before it — and each
// window to its mode, so a mode's figures are the sum over every window that ran
// it. Events before the first window_mode fall in a window labelled "unset".
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
//   - context: per session over the whole run, not per mode, the number of
//     context lines and the last used_pct among them.
//
// A session counts as the second when its session_open says role "second" (or
// "B", the label the run's hand-written lines use).

// UnsetMode labels the events logged before any window opened.
const UnsetMode = "unset"

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
	Collisions         int     `json:"collisions"`
}

// ModeTally is one division mode's figures over every window that ran it.
type ModeTally struct {
	Mode               string         `json:"mode"`
	Windows            int            `json:"windows"`
	WallMinutes        float64        `json:"wall_minutes"`
	LanesOpened        int            `json:"lanes_opened"`
	LanesLanded        int            `json:"lanes_landed"`
	SecondLanesLanded  int            `json:"second_session_lanes_landed"`
	Collisions         int            `json:"collisions"`
	Lapses             int            `json:"claims_lapsed"`
	Backoffs           int            `json:"backoffs"`
	BackoffMinutes     float64        `json:"backoff_minutes"`
	AgentMinutes       float64        `json:"agent_minutes"`
	CeilingWaitMinutes float64        `json:"ceiling_wait_minutes"`
	Refusals           int            `json:"refusals"`
	LandedPerHour      float64        `json:"lanes_landed_per_hour"`
	Sessions           []SessionTally `json:"sessions"`
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

// Report is the derived comparison.
type Report struct {
	Events   int         `json:"events"`
	Unparsed []Unparsed  `json:"unparsed"`
	Modes    []ModeTally `json:"modes"`
	// Context is each session's context measurement, by session id.
	Context []SessionContext `json:"context"`
	// Leader is the mode with the most lanes landed per wall-clock hour, "" when
	// no mode landed a lane or two modes tie. It is a figure, not a verdict: the
	// run's report names which mode it would keep, and says why.
	Leader      string `json:"leader"`
	LeaderBasis string `json:"leader_basis"`
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
		w.last = e.TS
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
		LeaderBasis: "lanes landed per wall-clock hour"}
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
