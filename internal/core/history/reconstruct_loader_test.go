package history

import "testing"

// middleTurnsRetained reports how many of a thread's turns outside the spine
// window (the first spineHeadTurns and last spineTailTurns) still hold decoded
// blocks or source lines.
func middleTurnsRetained(th *thread) int {
	n := 0
	for i, tn := range th.turns {
		if i < spineHeadTurns || i >= len(th.turns)-spineTailTurns {
			continue
		}
		if len(tn.blocks) > 0 || len(tn.raw) > 0 {
			n++
		}
	}
	return n
}

// TestSpineModeReachesTheLoader pins iss-2609091155497399: the mode is a
// loader input, not only a renderer one. In spine mode a delegate that hosts
// no other agent is reduced to its head and tail turns as it is parsed, so a
// session of verbose delegates is never resident in full just to be elided;
// the main thread, and a delegate whose turns place a nested agent, stay
// whole, because placement reads them. No thread keeps its record body once it
// has been parsed, in either mode.
func TestSpineModeReachesTheLoader(t *testing.T) {
	_, home := setupStore(t)
	plantRecord(t, home, "20260901T100000.000000000Z-sess-nest.md", []string{
		"session_id: sess-nest", "captured_at: 2026-09-01T10:06:00Z",
	}, mainThreadBody())
	plantRecord(t, home, "20260901T100500.000000000Z-sess-nest-agent-agenthost.md", []string{
		"session_id: sess-nest", "captured_at: 2026-09-01T10:05:00Z",
		"agent_id: agenthost", "spawn_depth: 1", "lineage_source: hook", "spawn_attribution: sidecar",
	}, subAgentBody())
	plantRecord(t, home, "20260901T100600.000000000Z-sess-nest-agent-agentleaf.md", []string{
		"session_id: sess-nest", "captured_at: 2026-09-01T10:06:00Z",
		"agent_id: agentleaf", "parent_agent_id: agenthost", "spawn_depth: 2",
		"lineage_source: hook", "spawn_attribution: sidecar",
	}, subAgentBody())
	records, err := ListForSession("", testRootSHA, "sess-nest")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []ReconstructMode{ModeFull, ModeSpine} {
		threads, dropped := loadThreads(records, mode)
		if len(dropped) != 0 || len(threads) != 3 {
			t.Fatalf("%s: %d threads, %d dropped; fixture drift", mode, len(threads), len(dropped))
		}
		for _, th := range threads {
			if th.bodyLen == 0 || len(th.turns) != th.turnCount.Total {
				t.Fatalf("%s %s: parsed %d turns of %d, body %d bytes", mode, th.label(), len(th.turns), th.turnCount.Total, th.bodyLen)
			}
			retained := middleTurnsRetained(th)
			reduce := mode == ModeSpine && th.label() == "agentleaf"
			switch {
			case reduce && retained != 0:
				t.Errorf("spine: leaf delegate %s kept %d middle turn(s) decoded", th.label(), retained)
			case !reduce && retained == 0:
				t.Errorf("%s: %s lost its middle turns, which placement or the render reads", mode, th.label())
			}
		}
	}
}
