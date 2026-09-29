package cli

import (
	"strings"
	"testing"
)

// TestImplementCountsDeclaredAgentsAndLogsTheEvidence: through the CLI, an
// agent_start past the session's ceiling is refused at exit 2, check reports the
// agents alive, the evidence events and a ceiling overrun are logged, and the
// report renders the overruns, the evidence and a missing field.
func TestImplementCountsDeclaredAgentsAndLogsTheEvidence(t *testing.T) {
	implementRepo(t)
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first", "--ceiling", "1", "--json")
	mustImplement(t, "implement", "mode", "claim", "--session", "alpha", "--window", "1", "--json")
	mustImplement(t, "implement", "log", "agent_start", "--session", "alpha",
		"--field", "agent=a1", "--field", "role=implementer", "--field", "model=opus", "--json")
	if msg := refusalEnvelope(t, 2, "implement", "log", "agent_start", "--session", "alpha",
		"--field", "agent=a2", "--field", "role=reviewer", "--field", "model=fable", "--json"); !strings.Contains(msg, "ceiling") {
		t.Fatalf("an agent past the ceiling = %q; want the ceiling named", msg)
	}
	if out := mustImplement(t, "implement", "check", "review", "--session", "alpha"); !strings.Contains(out, "1 agent(s) alive of its ceiling 1") {
		t.Fatalf("check does not report the agents alive:\n%s", out)
	}
	if out := mustImplement(t, "implement", "check", "review", "--session", "alpha", "--json"); !strings.Contains(out, `"agents_alive": 1`) {
		t.Fatalf("check --json does not report the agents alive:\n%s", out)
	}
	if msg := refusalEnvelope(t, 2, "implement", "log", "lane_close", "--session", "alpha", "--field", "lane=l1", "--json"); !strings.Contains(msg, "outcome") {
		t.Fatalf("a lane_close without an outcome = %q; want the field named", msg)
	}
	mustImplement(t, "implement", "log", "ceiling_overrun", "--session", "alpha", "--field", "alive=2", "--field", "ceiling=1",
		"--field", "lane=cut", "--field", "minutes=3", "--json")
	mustImplement(t, "implement", "log", "intervention", "--session", "alpha", "--field", "kind=ruling", "--field", "by=product thinker",
		"--field", "what=ruled", "--field", "why=asked", "--field", "autonomy_gap=no ruling channel", "--json")
	mustImplement(t, "implement", "log", "decision", "--session", "alpha", "--field", "what=merge a", "--field", "alternative=merge b",
		"--field", "why=smaller", "--json")
	mustImplement(t, "implement", "log", "stop", "--session", "alpha", "--field", "cause=CI only", "--field", "recovery=reproduced", "--json")

	out := mustImplement(t, "implement", "report")
	for _, want := range []string{"overrun", "evidence: 1 intervention(s)", "1 stop(s)", "1 decision(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("report does not render %q:\n%s", want, out)
		}
	}
	out = mustImplement(t, "implement", "report", "--json")
	for _, want := range []string{`"ceiling_overruns": 1`, `"interventions": 1`, `"missing_fields"`, `"coverage"`} {
		if !strings.Contains(out, want) {
			t.Errorf("report --json does not carry %s:\n%s", want, out)
		}
	}
}
