package implement

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestLogRefusesAnEventMissingTheFieldsTheReportCounts: a lane_close with no
// outcome and an agent_end with no role, model or minutes are refused at log
// time with nothing written, naming the field (iss-2609240646555891); the same
// events with their fields are written.
func TestLogRefusesAnEventMissingTheFieldsTheReportCounts(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	before := len(eventNames(t, r))
	for name, c := range map[string]struct {
		event  string
		fields map[string]string
		want   string
	}{
		"lane_close without outcome": {EventLaneClose, map[string]string{"lane": "l1", "pr": "12"}, "outcome"},
		"lane_close without lane":    {EventLaneClose, map[string]string{"outcome": "merged"}, "lane"},
		"agent_end without minutes":  {EventAgentEnd, map[string]string{"agent": "a1", "role": "implementer", "model": "opus"}, "minutes"},
		"agent_end without role":     {EventAgentEnd, map[string]string{"agent": "a1", "model": "opus", "minutes": "3"}, "role"},
		"agent_end without model":    {EventAgentEnd, map[string]string{"agent": "a1", "role": "implementer", "minutes": "3"}, "model"},
		"agent_end minutes not a number": {EventAgentEnd,
			map[string]string{"agent": "a1", "role": "implementer", "model": "opus", "minutes": "ten"}, "minutes"},
		"agent_start without agent": {EventAgentStart, map[string]string{"role": "implementer", "model": "opus"}, "agent"},
	} {
		_, err := r.Log("alpha", c.event, c.fields)
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v; want a refusal naming %s", name, err, c.want)
		}
	}
	if after := len(eventNames(t, r)); after != before {
		t.Fatalf("refused log calls wrote %d lines", after-before)
	}
	if _, err := r.Log("alpha", EventLaneClose, map[string]string{"lane": "l1", "outcome": "merged"}); err != nil {
		t.Fatalf("a complete lane_close: %v", err)
	}
	if _, err := r.Log("alpha", EventAgentStart, map[string]string{"agent": "a1", "role": "implementer", "model": "opus"}); err != nil {
		t.Fatalf("a complete agent_start: %v", err)
	}
	// Any of the three minute keys the report reads satisfies the field.
	if _, err := r.Log("alpha", EventAgentEnd, map[string]string{"agent": "a1", "role": "implementer", "model": "opus", "wall_min": "4"}); err != nil {
		t.Fatalf("a complete agent_end: %v", err)
	}
}

// TestTheEvidenceEventsAreLoggable: the run's evidence events — intervention,
// stop and decision — are written by the verb with their fields, their closed
// kind and their timestamps checked, and refused without the fields that make
// them evidence.
func TestTheEvidenceEventsAreLoggable(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	good := map[string]map[string]string{
		EventIntervention: {"at": "2026-09-28T09:20:00Z", "kind": "account", "by": "product thinker",
			"what": "switched account", "why": "limit reached", "detected_after_min": "20", "autonomy_gap": "no account failover"},
		EventStop: {"cause": "CI-only failure", "last_productive": "2026-09-28T10:26:12Z", "noticed_after_min": "6",
			"recovery": "reproduced locally"},
		EventDecision: {"at": "2026-09-28T09:45:00Z", "what": "merge drainH first", "alternative": "merge integ9 first",
			"why": "smaller diff"},
	}
	for ev, f := range good {
		if _, err := r.Log("alpha", ev, f); err != nil {
			t.Fatalf("log %s: %v", ev, err)
		}
		got := lastEvent(t, r, ev)
		for k, v := range f {
			if got.String(k) != v {
				t.Errorf("%s field %s = %q, want %q", ev, k, got.String(k), v)
			}
		}
	}
	before := len(eventNames(t, r))
	for name, c := range map[string]struct {
		event  string
		fields map[string]string
		want   string
	}{
		"intervention of an unknown kind": {EventIntervention, map[string]string{"kind": "coffee", "by": "x", "what": "x", "why": "x", "autonomy_gap": "x"}, "kind"},
		"intervention without its gap":    {EventIntervention, map[string]string{"kind": "ruling", "by": "x", "what": "x", "why": "x"}, "autonomy_gap"},
		"intervention with a bad at":      {EventIntervention, map[string]string{"at": "yesterday", "kind": "ruling", "by": "x", "what": "x", "why": "x", "autonomy_gap": "x"}, "at"},
		"intervention detected not a number": {EventIntervention,
			map[string]string{"kind": "ruling", "by": "x", "what": "x", "why": "x", "autonomy_gap": "x", "detected_after_min": "soon"}, "detected_after_min"},
		"stop without a cause":             {EventStop, map[string]string{"recovery": "x"}, "cause"},
		"stop with a bad last_productive":  {EventStop, map[string]string{"cause": "x", "last_productive": "noon"}, "last_productive"},
		"decision without the alternative": {EventDecision, map[string]string{"what": "x", "why": "x"}, "alternative"},
	} {
		_, err := r.Log("alpha", c.event, c.fields)
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v; want a refusal naming %s", name, err, c.want)
		}
	}
	if after := len(eventNames(t, r)); after != before {
		t.Fatalf("refused log calls wrote %d lines", after-before)
	}
}

// TestCeilingOverrunIsALoggableEvent: going over the ceiling has its own event,
// carrying the agents alive, the ceiling, the lane and the minutes over
// (iss-2609240646549900), and each is required.
func TestCeilingOverrunIsALoggableEvent(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	f := map[string]string{"alive": "5", "ceiling": "4", "lane": "cut", "minutes": "1"}
	if _, err := r.Log("alpha", EventCeilingOverrun, f); err != nil {
		t.Fatalf("log ceiling_overrun: %v", err)
	}
	for _, k := range []string{"alive", "ceiling", "lane", "minutes"} {
		short := map[string]string{}
		for kk, v := range f {
			if kk != k {
				short[kk] = v
			}
		}
		if _, err := r.Log("alpha", EventCeilingOverrun, short); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), k) {
			t.Errorf("ceiling_overrun without %s: %v; want a refusal naming it", k, err)
		}
	}
}

// TestAgentStartIsRefusedAtTheCeiling: the verb counts the agents a session
// has declared alive — its agent_start lines since it joined whose agent has no
// agent_end after — and refuses an agent_start that would take the count past
// the session's ceiling, logging the refusal (iss-2609240646542516). A session
// that stated no ceiling is not counted against one.
func TestAgentStartIsRefusedAtTheCeiling(t *testing.T) {
	r, c := newRun(t)
	if _, err := r.Join("alpha", RoleFirst, "", "", 2); err != nil {
		t.Fatal(err)
	}
	start := func(agent string) error {
		c.advance(time.Minute)
		_, err := r.Log("alpha", EventAgentStart, map[string]string{"agent": agent, "role": "implementer", "model": "opus"})
		return err
	}
	end := func(agent string) error {
		c.advance(time.Minute)
		_, err := r.Log("alpha", EventAgentEnd, map[string]string{"agent": agent, "role": "implementer", "model": "opus", "minutes": "1"})
		return err
	}
	if err := start("a1"); err != nil {
		t.Fatal(err)
	}
	if err := start("a2"); err != nil {
		t.Fatal(err)
	}
	v, err := r.Check("alpha", StageReview, nil)
	if err != nil || v.AgentsAlive != 2 || v.Ceiling != 2 {
		t.Fatalf("check at the ceiling = %+v, %v; want 2 alive of 2", v, err)
	}
	err = start("a3")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "ceiling") {
		t.Fatalf("a third agent under a ceiling of 2 = %v; want a refusal", err)
	}
	ref := lastEvent(t, r, EventRefusal)
	if ref.String("condition") != "agent_ceiling" || ref.String("agent") != "a3" || ref.String("alive") != "2" || ref.String("ceiling") != "2" {
		t.Fatalf("refusal line = %+v", ref.Fields)
	}
	if got := lastEvent(t, r, EventAgentStart); got.String("agent") != "a2" {
		t.Fatalf("the refused agent_start was written: %+v", got.Fields)
	}
	// Restarting an agent already alive does not count it twice.
	if err := start("a2"); err != nil {
		t.Fatalf("restating a live agent: %v", err)
	}
	if err := end("a1"); err != nil {
		t.Fatal(err)
	}
	if err := start("a3"); err != nil {
		t.Fatalf("a slot freed by an agent_end: %v", err)
	}

	// No ceiling stated: nothing to count against.
	join(t, r, "beta", RoleSecond)
	for i := 0; i < 5; i++ {
		if _, err := r.Log("beta", EventAgentStart, map[string]string{"agent": "b" + string(rune('0'+i)), "role": "reviewer", "model": "fable"}); err != nil {
			t.Fatalf("beta agent %d: %v", i, err)
		}
	}
}

// TestAgentsAliveCountsOnlySinceTheSessionJoined: a session that left and joined
// again starts from none alive — an agent its earlier self never ended does not
// hold a slot for ever.
func TestAgentsAliveCountsOnlySinceTheSessionJoined(t *testing.T) {
	r, c := newRun(t)
	if _, err := r.Join("alpha", RoleFirst, "", "", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Log("alpha", EventAgentStart, map[string]string{"agent": "old", "role": "implementer", "model": "opus"}); err != nil {
		t.Fatal(err)
	}
	c.advance(time.Minute)
	if _, err := r.Leave("alpha", "rotation"); err != nil {
		t.Fatal(err)
	}
	c.advance(time.Minute)
	if _, err := r.Join("alpha", RoleFirst, "", "", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Log("alpha", EventAgentStart, map[string]string{"agent": "new", "role": "implementer", "model": "opus"}); err != nil {
		t.Fatalf("an agent after a rejoin: %v", err)
	}
}
