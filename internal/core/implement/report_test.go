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
