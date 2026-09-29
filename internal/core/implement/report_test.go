package implement

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// TestCompareDerivesEachModeFromAFixtureLog pins the comparison against a log
// written the way the run writes it — hand-kept lines with the A/B role labels
// and a quoted number, an agent_end carrying `wall_min` (the key the live run
// writes), context lines, verb-shaped claim lines, a line that is not JSON and
// one missing its timestamp.
func TestCompareDerivesEachModeFromAFixtureLog(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "run-log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	events, bad := ParseLog("run-log.jsonl", data)
	rep := Compare(events, bad)

	if rep.Events != 26 {
		t.Errorf("events = %d, want 26", rep.Events)
	}
	if len(rep.Unparsed) != 2 || rep.Unparsed[0].Line != 7 || rep.Unparsed[1].Line != 8 {
		t.Errorf("unparsed = %+v, want lines 7 and 8", rep.Unparsed)
	}
	want := map[string]ModeTally{
		"single": {Windows: 1, WallMinutes: 60, LanesOpened: 1, LanesLanded: 1, AgentMinutes: 47, LandedPerHour: 1},
		"claim": {Windows: 1, WallMinutes: 60, LanesOpened: 2, LanesLanded: 2, SecondLanesLanded: 1, Collisions: 1,
			Backoffs: 2, BackoffMinutes: 6, AgentMinutes: 42, CeilingWaitMinutes: 10, LandedPerHour: 2},
		"batch":       {Windows: 1, WallMinutes: 60, LanesOpened: 1},
		"split-roles": {Windows: 1, WallMinutes: 30, Refusals: 1},
	}
	var order []string
	for _, m := range rep.Modes {
		order = append(order, m.Mode)
		w, ok := want[m.Mode]
		if !ok {
			t.Errorf("unexpected mode %q", m.Mode)
			continue
		}
		got := m
		got.Mode, got.Sessions = "", nil
		if !reflect.DeepEqual(got, w) {
			t.Errorf("%s:\n got %+v\nwant %+v", m.Mode, got, w)
		}
	}
	if len(order) != 4 || order[0] != "single" || order[1] != "claim" || order[2] != "batch" || order[3] != "split-roles" {
		t.Errorf("mode order = %v", order)
	}
	claim := rep.Modes[1]
	var b SessionTally
	for _, s := range claim.Sessions {
		if s.Session == "B1" {
			b = s
		}
	}
	if b.Role != "second" || b.LanesOpened != 1 || b.LanesLanded != 1 || b.BackoffMinutes != 6 || b.Collisions != 1 || b.AgentMinutes != 30 {
		t.Errorf("claim window, session B1 = %+v", b)
	}
	var a SessionTally
	for _, s := range claim.Sessions {
		if s.Session == "A1" {
			a = s
		}
	}
	if a.AgentMinutes != 12 {
		t.Errorf("claim window, session A1 agent minutes = %v, want 12 (its agent_end carries wall_min)", a.AgentMinutes)
	}
	if rep.Leader != "claim" {
		t.Errorf("leader = %q, want claim", rep.Leader)
	}
	// Context is per session across the run: every context line counted, the
	// last used_pct seen (a quoted number read as one).
	wantCtx := []SessionContext{{Session: "A1", Role: "first", Events: 2, LastUsedPct: 31,
		LastAt: time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)}}
	if !reflect.DeepEqual(rep.Context, wantCtx) {
		t.Errorf("context = %+v, want %+v", rep.Context, wantCtx)
	}
}

// TestAgentMinutesReadEveryKeyTheRunWrites: an agent_end's minutes are read
// from `minutes`, `wall_minutes` or `wall_min`, the last being the key the live
// run's hand-kept lines carry.
func TestAgentMinutesReadEveryKeyTheRunWrites(t *testing.T) {
	log := `{"ts":"2026-09-23T06:00:00Z","session":"A","event":"window_mode","mode":"single"}
{"ts":"2026-09-23T06:01:00Z","session":"A","event":"agent_end","minutes":1}
{"ts":"2026-09-23T06:02:00Z","session":"A","event":"agent_end","wall_minutes":10}
{"ts":"2026-09-23T06:03:00Z","session":"A","event":"agent_end","wall_min":100}
`
	events, bad := ParseLog("x.jsonl", []byte(log))
	rep := Compare(events, bad)
	if len(rep.Modes) != 1 || rep.Modes[0].AgentMinutes != 111 {
		t.Fatalf("modes = %+v, want agent minutes 111", rep.Modes)
	}
}

// TestCompareOfAnEmptyLogNamesNoLeader: nothing landed, nothing to lead.
func TestCompareOfAnEmptyLogNamesNoLeader(t *testing.T) {
	rep := Compare(nil, nil)
	if rep.Leader != "" || len(rep.Modes) != 0 || rep.Unparsed == nil || rep.Context == nil {
		t.Fatalf("empty report = %+v", rep)
	}
}

// TestReadLogReadsEveryDay: a run that crosses midnight UTC is two files, and
// the comparison reads both; a file that is not a day's log is ignored.
func TestReadLogReadsEveryDay(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	c.advance(24 * time.Hour)
	if _, err := r.SetMode("alpha", ModeClaim, 2); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Dir, "notes.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	evs, bad, err := r.ReadLog()
	if err != nil || len(bad) != 0 || len(evs) != 2 {
		t.Fatalf("ReadLog = %d events, %v, %v", len(evs), bad, err)
	}
	if evs[0].Event != EventSessionOpen || evs[1].Event != EventWindowMode {
		t.Fatalf("events = %v, %v", evs[0].Event, evs[1].Event)
	}
}

// TestAJoinJustBeforeTheWindowItOpensIsThatWindows: a session_open logged a
// second before the first session sets the window's mode belongs to that
// window, not the one before (iss-2609240646544930), so its lanes and the
// session itself are reported under one mode.
func TestAJoinJustBeforeTheWindowItOpensIsThatWindows(t *testing.T) {
	log := `{"ts":"2026-09-23T06:00:00Z","session":"A","event":"window_mode","mode":"single"}
{"ts":"2026-09-23T06:10:00Z","session":"A","event":"lane_open","lane":"l1"}
{"ts":"2026-09-23T07:00:00Z","session":"B","event":"session_open","role":"second"}
{"ts":"2026-09-23T07:00:01Z","session":"A","event":"window_mode","mode":"claim","window":2}
{"ts":"2026-09-23T07:30:00Z","session":"B","event":"lane_open","lane":"l2"}
{"ts":"2026-09-23T07:40:00Z","session":"B","event":"lane_close","lane":"l2","outcome":"merged"}
`
	events, bad := ParseLog("x.jsonl", []byte(log))
	rep := Compare(events, bad)
	for _, m := range rep.Modes {
		for _, s := range m.Sessions {
			if s.Session == "B" && m.Mode != "claim" {
				t.Errorf("session B is reported under %s; its join and its lane are the claim window's", m.Mode)
			}
		}
		if m.Mode == "single" && m.WallMinutes != 10 {
			t.Errorf("single window wall minutes = %v, want 10 (the join is not its last event)", m.WallMinutes)
		}
	}
	// A join well before the next window stays in the window it happened in.
	log2 := `{"ts":"2026-09-23T06:00:00Z","session":"A","event":"window_mode","mode":"single"}
{"ts":"2026-09-23T06:30:00Z","session":"B","event":"session_open","role":"second"}
{"ts":"2026-09-23T07:00:00Z","session":"A","event":"window_mode","mode":"claim","window":2}
`
	events, bad = ParseLog("y.jsonl", []byte(log2))
	rep = Compare(events, bad)
	if rep.Modes[0].Mode != "single" || len(rep.Modes[0].Sessions) != 2 {
		t.Errorf("a join half an hour before the next window moved: %+v", rep.Modes)
	}
}

// TestCeilingOverrunsAreCounted: each ceiling_overrun line is counted in its
// mode and for its session, beside the ceiling waits (iss-2609240646549900).
func TestCeilingOverrunsAreCounted(t *testing.T) {
	log := `{"ts":"2026-09-23T06:00:00Z","session":"A","event":"window_mode","mode":"single"}
{"ts":"2026-09-23T06:10:00Z","session":"A","event":"ceiling_overrun","alive":5,"ceiling":4,"lane":"cut","minutes":1}
{"ts":"2026-09-23T06:20:00Z","session":"A","event":"ceiling_overrun","alive":6,"ceiling":5,"lane":"l2","minutes":"12"}
`
	events, bad := ParseLog("x.jsonl", []byte(log))
	rep := Compare(events, bad)
	if len(rep.Modes) != 1 || rep.Modes[0].CeilingOverruns != 2 || rep.Modes[0].CeilingOverrunMinutes != 13 {
		t.Fatalf("modes = %+v; want 2 overruns, 13 minutes over", rep.Modes)
	}
	if s := rep.Modes[0].Sessions[0]; s.CeilingOverruns != 2 {
		t.Fatalf("session tally = %+v", s)
	}
}

// TestTheReportNamesMissingFieldsAndCoverageThatStops: a lane_close with no
// outcome and an agent_end with no minutes are counted by the field they miss,
// and an event kind whose lines stop hours before the run's last line is named
// (iss-2609240646555891) — every line still parses, so nothing else would say.
func TestTheReportNamesMissingFieldsAndCoverageThatStops(t *testing.T) {
	log := `{"ts":"2026-09-23T06:00:00Z","session":"A","event":"window_mode","mode":"single"}
{"ts":"2026-09-23T06:05:00Z","session":"A","event":"agent_start","agent":"a1","role":"implementer","model":"opus"}
{"ts":"2026-09-23T07:00:00Z","session":"A","event":"agent_end","agent":"a1","role":"implementer","model":"opus","minutes":55}
{"ts":"2026-09-23T08:00:00Z","session":"A","event":"lane_close","lane":"l1","pr":12,"merge":"abc1234"}
{"ts":"2026-09-23T09:00:00Z","session":"A","event":"agent_end","lane":"l2"}
{"ts":"2026-09-24T09:00:00Z","session":"A","event":"lane_close","lane":"l3","outcome":"merged"}
{"ts":"2026-09-24T09:05:00Z","session":"A","event":"load","load1":3}
`
	events, bad := ParseLog("x.jsonl", []byte(log))
	rep := Compare(events, bad)
	missing := map[string]int{}
	for _, m := range rep.MissingFields {
		missing[m.Event+"."+m.Field] = m.Lines
	}
	for k, want := range map[string]int{"lane_close.outcome": 1, "agent_end.minutes": 1, "agent_end.role": 1, "agent_end.model": 1, "agent_end.agent": 1} {
		if missing[k] != want {
			t.Errorf("missing %s = %d, want %d (all: %v)", k, missing[k], want, missing)
		}
	}
	if _, ok := missing["lane_close.lane"]; ok {
		t.Errorf("a field every line carries is reported missing: %v", missing)
	}
	stops := map[string]CoverageGap{}
	for _, g := range rep.Coverage {
		stops[g.Event] = g
	}
	g, ok := stops[EventAgentEnd]
	if !ok || !g.Last.Equal(time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)) || g.HoursBefore != 24 {
		t.Errorf("agent_end coverage = %+v, %v; want its lines to stop 24h before the run's last", g, ok)
	}
	if _, ok := stops[EventAgentStart]; !ok {
		t.Errorf("agent_start's coverage stops too: %+v", rep.Coverage)
	}
	if _, ok := stops[EventLaneClose]; ok {
		t.Errorf("lane_close runs to the end and is named: %+v", rep.Coverage)
	}
}

// TestTheReportCountsTheEvidenceEvents: interventions (by kind, with the
// minutes each went unnoticed), stops and decisions are counted over the run;
// the older stop lines that carry none of the evidence fields still count, and
// nothing fails on them.
func TestTheReportCountsTheEvidenceEvents(t *testing.T) {
	log := `{"ts":"2026-09-23T06:00:00Z","session":"A","event":"stop","condition":"needs a ruling"}
{"ts":"2026-09-28T09:00:00Z","session":"A","event":"intervention","at":"2026-09-24T02:40:00Z","kind":"file_restore","by":"pt","what":"x","why":"y","autonomy_gap":"z","detected_after_min":"20"}
{"ts":"2026-09-28T09:01:00Z","session":"A","event":"intervention","kind":"ruling","by":"pt","what":"x","why":"y","autonomy_gap":"z"}
{"ts":"2026-09-28T09:02:00Z","session":"A","event":"intervention","kind":"ruling","by":"pt","what":"x","why":"y","autonomy_gap":"z","detected_after_min":5}
{"ts":"2026-09-28T09:03:00Z","session":"A","event":"stop","cause":"CI-only","last_productive":"2026-09-28T08:00:00Z","noticed_after_min":"6","recovery":"r"}
{"ts":"2026-09-28T09:04:00Z","session":"A","event":"decision","what":"x","alternative":"y","why":"z"}
`
	events, bad := ParseLog("x.jsonl", []byte(log))
	rep := Compare(events, bad)
	ev := rep.Evidence
	if ev.Interventions != 3 || ev.InterventionsByKind["ruling"] != 2 || ev.InterventionsByKind["file_restore"] != 1 ||
		ev.DetectedAfterMinutes != 25 || ev.Stops != 2 || ev.StopNoticedAfterMinutes != 6 || ev.Decisions != 1 {
		t.Fatalf("evidence = %+v", ev)
	}
}
