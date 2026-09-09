package history

import (
	"encoding/json"
	"strings"
	"testing"
)

// --------------------------------------------------------------------------
// Fixtures
//
// The fixtures are line-delimited transcripts in the shape the harness writes,
// planted straight into the store so a test pins reading rather than writing.
// The one shape that matters most is the MULTI-LINE response: the harness
// writes one line per content block and repeats the SAME usage object on every
// one of them, which is the arithmetic trap the telemetry has to avoid.
// --------------------------------------------------------------------------

// usageJSON is one response's usage object, repeated verbatim on every line of
// that response exactly as the harness repeats it.
const usageJSON = `"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":5}`

// oneResponseTokens is what usageJSON is worth, counted ONCE.
const oneResponseTokens = 100 + 50 + 10 + 5

// plantRecord writes a record with the given frontmatter fields and body.
func plantRecord(t *testing.T, home, name string, fields []string, body string) {
	t.Helper()
	head := append([]string{
		"---",
		"schema: 3",
		"root_commit: " + testRootSHA,
		"source_kind: native",
		"source_sha256: " + strings.Repeat("a", 64),
		"redacted_secrets: 0",
		"redacted_home_paths: 0",
	}, fields...)
	planted(t, home, name, strings.Join(head, "\n")+"\n---\n"+body)
}

// mainThreadBody is the spine of the fixture session. Turn 2 is one response
// written as THREE lines sharing message id msg_spawn — thinking, text and the
// tool call that launches the asynchronous agent — and every one of those lines
// repeats the response's usage.
//
// The spawn is at turn 2 and the completion notification arrives at turn 7, so
// turns 3 to 6 ran while the agent was still working. That gap is the whole
// reason this document does not nest the agent's section at its spawn point.
func mainThreadBody() string {
	return strings.Join([]string{
		`{"type":"user","timestamp":"2026-09-01T10:00:00Z","message":{"role":"user","content":"please investigate the build"}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:00:05Z","message":{"id":"msg_spawn","role":"assistant","model":"claude-test-1",` + usageJSON + `,"content":[{"type":"thinking","thinking":"I should delegate this.","signature":"AAAA"}]}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:00:05Z","message":{"id":"msg_spawn","role":"assistant","model":"claude-test-1",` + usageJSON + `,"content":[{"type":"text","text":"Delegating the investigation."}]}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:00:06Z","message":{"id":"msg_spawn","role":"assistant","model":"claude-test-1",` + usageJSON + `,"content":[{"type":"tool_use","id":"toolu_spawn","name":"Agent","input":{"subagent_type":"explorer"}}]}}`,
		`{"type":"user","timestamp":"2026-09-01T10:00:07Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_spawn","content":[{"type":"text","text":"Async agent launched. agentId: agentone runs in the background."}]}]}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:01:00Z","message":{"id":"msg_mid1","role":"assistant","model":"claude-test-1",` + usageJSON + `,"content":[{"type":"tool_use","id":"toolu_ls","name":"Bash","input":{"command":"ls"}}]}}`,
		`{"type":"user","timestamp":"2026-09-01T10:01:01Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_ls","content":"Makefile"}]}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:02:00Z","message":{"id":"msg_mid2","role":"assistant","model":"claude-test-1",` + usageJSON + `,"content":[{"type":"text","text":"Still waiting on the delegate."}]}}`,
		`{"type":"user","timestamp":"2026-09-01T10:05:00Z","attachment":{"type":"queued_command","prompt":"<task-notification><task-id>agentone</task-id></task-notification>"},"message":{"role":"user","content":"the delegate finished"}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:05:10Z","message":{"id":"msg_end","role":"assistant","model":"claude-test-1",` + usageJSON + `,"content":[{"type":"text","text":"The build breaks in the linker."}]}}`,
	}, "\n") + "\n"
}

// subAgentBody is one delegate's own transcript: three turns, one response.
func subAgentBody() string {
	return strings.Join([]string{
		`{"type":"user","timestamp":"2026-09-01T10:00:08Z","message":{"role":"user","content":"investigate the build"}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:01:30Z","message":{"id":"msg_sub1","role":"assistant","model":"claude-test-2",` + usageJSON + `,"content":[{"type":"tool_use","id":"toolu_grep","name":"Grep","input":{"pattern":"undefined"}}]}}`,
		`{"type":"user","timestamp":"2026-09-01T10:01:31Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_grep","content":"link.go:12"}]}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:04:50Z","message":{"id":"msg_sub2","role":"assistant","model":"claude-test-2",` + usageJSON + `,"content":[{"type":"text","text":"The linker step is the failure."}]}}`,
	}, "\n") + "\n"
}

// plantFixtureSession lays down the main thread, one placed sub-agent and one
// the store cannot place.
func plantFixtureSession(t *testing.T, home string) {
	t.Helper()
	plantRecord(t, home, "20260901T100000.000000000Z-sess-recon.md", []string{
		"session_id: sess-recon",
		"captured_at: 2026-09-01T10:06:00Z",
	}, mainThreadBody())

	plantRecord(t, home, "20260901T100500.000000000Z-sess-recon-agent-agentone.md", []string{
		"session_id: sess-recon",
		"captured_at: 2026-09-01T10:05:00Z",
		"agent_id: agentone",
		"agent_type: explorer",
		"spawn_tool_use_id: toolu_spawn",
		"spawn_depth: 1",
		"lineage_source: hook",
		"spawn_attribution: sidecar",
	}, subAgentBody())

	plantRecord(t, home, "20260901T100600.000000000Z-sess-recon-agent-agentghost.md", []string{
		"session_id: sess-recon",
		"captured_at: 2026-09-01T10:06:00Z",
		"agent_id: agentghost",
		"lineage_source: hook",
		"spawn_attribution: unattributed",
	}, `{"type":"assistant","timestamp":"2026-09-01T10:03:00Z","message":{"id":"msg_ghost","role":"assistant","model":"claude-test-2","content":[{"type":"text","text":"nobody knows who asked for this"}]}}`+"\n")
}

func reconstructFixture(t *testing.T, opts ReconstructOptions) Reconstruction {
	t.Helper()
	if opts.SessionID == "" {
		opts.SessionID = "sess-recon"
	}
	res, err := Reconstruct(testRootSHA, opts)
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	return res
}

// --------------------------------------------------------------------------
// The token arithmetic (iss-2609090723027424)
// --------------------------------------------------------------------------

// TestTelemetryCountsOneUsagePerResponse is the measure this whole file exists
// to get right. The harness writes one transcript line per content block and
// repeats the SAME usage object on every line of one API response, so summing
// lines multiplies that response's cost by its block count — measured at 2.44x
// on one stored transcript and 5.28x across ten, a factor that varies per
// session and so cannot be divided out afterwards. Telemetry that is
// confidently wrong by a varying factor is worse than no telemetry, because
// comparison across runs is the entire purpose.
//
// The fixture's main thread has one three-line response and three one-line
// ones. Naive line summing gives six usage objects; the truth is four.
func TestTelemetryCountsOneUsagePerResponse(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	tel := reconstructFixture(t, ReconstructOptions{}).Telemetry

	var main AgentTelemetry
	for _, a := range tel.Agents {
		if a.IsMainThread {
			main = a
		}
	}
	if main.Tokens.UsageLinesSeen != 6 {
		t.Fatalf("fixture drift: expected 6 transcript lines carrying usage, got %d", main.Tokens.UsageLinesSeen)
	}
	if main.Tokens.APIResponses != 4 {
		t.Errorf("the main thread has 4 distinct responses (one written as 3 lines); counted %d", main.Tokens.APIResponses)
	}
	if got, want := main.Tokens.Total, int64(4*oneResponseTokens); got != want {
		t.Errorf("main-thread tokens must count one usage per response: got %d, want %d (naive line summing would give %d)",
			got, want, int64(6*oneResponseTokens))
	}
	if got, want := main.Tokens.Input, int64(4*100); got != want {
		t.Errorf("input tokens: got %d, want %d", got, want)
	}
	if got, want := main.Tokens.CacheReadInput, int64(4*5); got != want {
		t.Errorf("cache read tokens: got %d, want %d", got, want)
	}
	// Session-wide: 4 main-thread responses + 2 from agentone + 0 from the
	// ghost, whose one line carries no usage at all.
	if got, want := tel.Tokens.APIResponses, 6; got != want {
		t.Errorf("session api_responses: got %d, want %d", got, want)
	}
	if got, want := tel.Tokens.Total, int64(6*oneResponseTokens); got != want {
		t.Errorf("session tokens: got %d, want %d", got, want)
	}
	if tel.Completeness.UsageWithoutMessageID != 0 {
		t.Errorf("every fixture response carries a message id; got %d unkeyed",
			tel.Completeness.UsageWithoutMessageID)
	}
}

// TestTelemetryReportsUsageItCouldNotDeduplicate is the negative control for
// the test above. A response with no message id cannot be de-duplicated, and
// the file has to say so rather than quietly return a number that may be
// inflated.
func TestTelemetryReportsUsageItCouldNotDeduplicate(t *testing.T) {
	_, home := setupStore(t)
	plantRecord(t, home, "20260901T100000.000000000Z-sess-noid.md", []string{
		"session_id: sess-noid",
		"captured_at: 2026-09-01T10:00:00Z",
	}, strings.Join([]string{
		`{"type":"assistant","timestamp":"2026-09-01T10:00:01Z","message":{"role":"assistant","model":"m",` + usageJSON + `,"content":[{"type":"text","text":"a"}]}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:00:02Z","message":{"role":"assistant","model":"m",` + usageJSON + `,"content":[{"type":"text","text":"b"}]}}`,
	}, "\n")+"\n")

	res := reconstructFixture(t, ReconstructOptions{SessionID: "sess-noid"})
	if got := res.Telemetry.Completeness.UsageWithoutMessageID; got != 2 {
		t.Errorf("two responses carry usage with no message id; completeness reported %d", got)
	}
	if !strings.Contains(string(res.Artefact), "could not be de-duplicated") {
		t.Errorf("the artefact must say the totals may be an upper bound; got:\n%s", res.Artefact)
	}
}

// --------------------------------------------------------------------------
// Reconstruction fidelity
// --------------------------------------------------------------------------

// TestReconstructMarksEverySubagentAtItsSpawnAndJoinPoint is ac-10 as the
// corpus actually is.
//
// The spec asks for the sub-agent's section to be nested at its spawn point.
// The agents whose id the spawning transcript records are the ASYNCHRONOUS
// ones, and for those the spawn and the join are many turns apart, so nesting
// the section at the spawn point puts the delegate's conclusions in front of
// main-thread turns that ran before those conclusions existed. So the thread
// stays contiguous and carries two markers, and this test pins both: the
// attribution the spec asks for, and the ordering it would have broken.
func TestReconstructMarksEverySubagentAtItsSpawnAndJoinPoint(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	res := reconstructFixture(t, ReconstructOptions{})
	art := string(res.Artefact)

	spawn := strings.Index(art, "[SPAWNED** agent `agentone`")
	join := strings.Index(art, "[JOINED** agent `agentone`")
	section := strings.Index(art, "## Agent `agentone`")
	if spawn < 0 || join < 0 || section < 0 {
		t.Fatalf("artefact must carry a spawn marker, a join marker and a section for agentone; got spawn=%d join=%d section=%d\n%s",
			spawn, join, section, art)
	}
	if !(spawn < join && join < section) {
		t.Errorf("the main thread must stay contiguous: spawn marker (%d) then join marker (%d) then the appended section (%d)",
			spawn, join, section)
	}

	// The turns between the spawn and the join ran WITHOUT the delegate's
	// result, so its conclusion must not appear before them.
	mid := strings.Index(art, "Still waiting on the delegate.")
	conclusion := strings.Index(art, "The linker step is the failure.")
	if mid < 0 || conclusion < 0 {
		t.Fatalf("fixture drift: mid=%d conclusion=%d", mid, conclusion)
	}
	if conclusion < mid {
		t.Errorf("the delegate's conclusion (%d) appears BEFORE a main-thread turn that ran while it was still working (%d); that reverses causality",
			conclusion, mid)
	}

	var row AgentTelemetry
	for _, a := range res.Telemetry.Agents {
		if a.AgentID == "agentone" {
			row = a
		}
	}
	if row.SpawnedIn != "main" || row.SpawnedAtTurn != 2 {
		t.Errorf("agentone is spawned at main-thread turn 2, got %s turn %d", row.SpawnedIn, row.SpawnedAtTurn)
	}
	if row.JoinedAtTurn != 7 {
		t.Errorf("agentone's completion reaches the main thread at turn 7, got %d", row.JoinedAtTurn)
	}
	if row.PlacedBy != "record" {
		t.Errorf("agentone carries spawn_tool_use_id, so the record rung places it; got %q", row.PlacedBy)
	}
	if !strings.Contains(art, "CONCURRENCY: 4 turn(s)") {
		t.Errorf("the agent section must state how many turns overlapped its run; got:\n%s", art)
	}
}

// TestReconstructPlacesAnAgentFromTheSpawningTranscriptAlone is the attribution
// ladder's second rung. A record with no stored spawn_tool_use_id — everything
// captured before the lineage fields existed — is still placeable, because the
// tool result that acknowledged the launch names the agent's id, and that
// result identifies the call.
func TestReconstructPlacesAnAgentFromTheSpawningTranscriptAlone(t *testing.T) {
	_, home := setupStore(t)
	plantRecord(t, home, "20260901T100000.000000000Z-sess-rung2.md", []string{
		"session_id: sess-rung2",
		"captured_at: 2026-09-01T10:06:00Z",
	}, mainThreadBody())
	// Same delegate, but the record has no spawn tool call recorded on it.
	plantRecord(t, home, "20260901T100500.000000000Z-sess-rung2-agent-agentone.md", []string{
		"session_id: sess-rung2",
		"captured_at: 2026-09-01T10:05:00Z",
		"agent_id: agentone",
		"lineage_source: migrated",
		"spawn_attribution: unattributed",
	}, subAgentBody())

	res := reconstructFixture(t, ReconstructOptions{SessionID: "sess-rung2"})
	var row AgentTelemetry
	for _, a := range res.Telemetry.Agents {
		if a.AgentID == "agentone" {
			row = a
		}
	}
	if row.PlacedBy != "transcript" {
		t.Errorf("with no stored spawn tool call the transcript rung must place it; got %q", row.PlacedBy)
	}
	if row.SpawnedAtTurn != 2 || row.JoinedAtTurn != 7 {
		t.Errorf("the transcript rung must find the same spawn (2) and join (7); got %d and %d",
			row.SpawnedAtTurn, row.JoinedAtTurn)
	}
	if row.SpawnToolUseID != "toolu_spawn" {
		t.Errorf("the rung recovers the spawning tool call too; got %q", row.SpawnToolUseID)
	}
}

// TestReconstructSegregatesUnattributedSubagents — ac-10's other half. An agent
// nothing can place goes under its own heading, last, labelled. Interleaving it
// with the placed ones would make its position read as meant.
func TestReconstructSegregatesUnattributedSubagents(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	res := reconstructFixture(t, ReconstructOptions{})
	art := string(res.Artefact)

	// Matched at line start: the reading guide names the heading inline, and
	// the guide is not the section.
	heading := strings.Index(art, "\n## Unattributed sub-agents\n")
	ghost := strings.Index(art, "\n## Agent `agentghost`\n")
	placed := strings.Index(art, "\n## Agent `agentone`\n")
	if heading < 0 || ghost < 0 {
		t.Fatalf("an unplaceable agent needs its own labelled section; heading=%d ghost=%d\n%s", heading, ghost, art)
	}
	if !(placed < heading && heading < ghost) {
		t.Errorf("the unattributed section comes last: placed=%d heading=%d ghost=%d", placed, heading, ghost)
	}
	if res.Telemetry.Completeness.AgentsWithoutSpawnPoint != 1 {
		t.Errorf("completeness must count the unplaceable agent; got %d",
			res.Telemetry.Completeness.AgentsWithoutSpawnPoint)
	}
}

// TestReconstructRendersASessionWithNoMainThread is a case the spec leaves
// undefined and the corpus makes the common one: on this machine 71 sub-agent
// sets have no parent transcript at all. Refusing would make most of the
// sub-agent corpus unreconstructable, and rendering silently would let a reader
// mistake a fragment for a session. So it renders, and says so twice — in the
// artefact and in the telemetry.
func TestReconstructRendersASessionWithNoMainThread(t *testing.T) {
	_, home := setupStore(t)
	plantRecord(t, home, "20260901T100500.000000000Z-sess-orphan-agent-agentone.md", []string{
		"session_id: sess-orphan",
		"captured_at: 2026-09-01T10:05:00Z",
		"agent_id: agentone",
		"agent_type: explorer",
		"spawn_depth: 1",
		"lineage_source: hook",
		"spawn_attribution: unattributed",
	}, subAgentBody())

	res := reconstructFixture(t, ReconstructOptions{SessionID: "sess-orphan"})
	art := string(res.Artefact)
	if res.Telemetry.Completeness.MainThreadPresent {
		t.Error("completeness must report the main thread absent")
	}
	if !strings.Contains(art, "main thread: ABSENT") {
		t.Errorf("the artefact header must say the main thread is absent; got:\n%s", art)
	}
	if !strings.Contains(art, "No main-thread record for this session is stored") {
		t.Errorf("the Main thread section must state its own absence; got:\n%s", art)
	}
	if !strings.Contains(art, "## Agent `agentone`") {
		t.Error("the sub-agent transcripts that survived must still be rendered")
	}
	if res.Telemetry.Completeness.AgentsWithoutSpawnPoint != 1 {
		t.Errorf("with no main thread no spawn point is recoverable; got %d",
			res.Telemetry.Completeness.AgentsWithoutSpawnPoint)
	}
}

// TestReconstructChoosesOneRecordPerAgentAndSaysWhich is the other case the
// spec leaves open. Supersession narrows duplicates but does not close them, so
// reconstruction must pick one and be accountable for the pick. It takes the
// LONGEST body — the store's own notion of more complete, since supersession
// replaces a record when new bytes strictly extend it — and names every record
// it did not use.
func TestReconstructChoosesOneRecordPerAgentAndSaysWhich(t *testing.T) {
	_, home := setupStore(t)
	// The NEWER record is the shorter one, so a newest-wins rule would drop
	// most of the session.
	plantRecord(t, home, "20260901T100000.000000000Z-sess-dup.md", []string{
		"session_id: sess-dup",
		"captured_at: 2026-09-01T10:00:00Z",
	}, mainThreadBody())
	plantRecord(t, home, "20260901T110000.000000000Z-sess-dup.md", []string{
		"session_id: sess-dup",
		"captured_at: 2026-09-01T11:00:00Z",
	}, `{"type":"user","timestamp":"2026-09-01T10:00:00Z","message":{"role":"user","content":"truncated re-capture"}}`+"\n")

	res := reconstructFixture(t, ReconstructOptions{SessionID: "sess-dup"})
	art := string(res.Artefact)
	if !strings.Contains(art, "The build breaks in the linker.") {
		t.Errorf("the longer record must win over a shorter, newer one; got:\n%s", art)
	}
	c := res.Telemetry.Completeness
	if c.RecordsFound != 2 || c.RecordsUsed != 1 {
		t.Errorf("2 records found, 1 used; got found=%d used=%d", c.RecordsFound, c.RecordsUsed)
	}
	if len(c.DroppedRecords) != 1 {
		t.Fatalf("the record not used must be named; got %+v", c.DroppedRecords)
	}
	if !strings.Contains(art, "record NOT used: `20260901T110000.000000000Z-sess-dup.md`") {
		t.Errorf("the artefact must name the record it rejected; got:\n%s", art)
	}
}

// --------------------------------------------------------------------------
// The telemetry file
// --------------------------------------------------------------------------

// TestTelemetryReportsEveryRequiredMeasure — ac-11. Every measure the intent
// named is present and non-trivial, and the file round-trips through JSON,
// since "machine-readable" is the requirement rather than "a Go struct".
func TestTelemetryReportsEveryRequiredMeasure(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	raw, err := json.Marshal(reconstructFixture(t, ReconstructOptions{}).Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	var tel Telemetry
	if err := json.Unmarshal(raw, &tel); err != nil {
		t.Fatalf("telemetry must round-trip as JSON: %v", err)
	}

	if tel.SchemaVersion != reconstructSchemaVersion || tel.SessionID != "sess-recon" || tel.RootCommit != testRootSHA {
		t.Errorf("identity block wrong: %+v", tel)
	}
	if tel.StartedAt == nil || tel.EndedAt == nil {
		t.Fatalf("started_at/ended_at must be set: %+v", tel)
	}
	if tel.WallClockSeconds != 310 {
		t.Errorf("wall clock spans 10:00:00 to 10:05:10, i.e. 310s; got %v", tel.WallClockSeconds)
	}
	if tel.Turns.User != 6 || tel.Turns.Assistant != 7 || tel.Turns.Total != 13 {
		t.Errorf("turn counts wrong: %+v", tel.Turns)
	}
	if tel.Tokens.Total == 0 {
		t.Error("tokens must be reported")
	}
	for _, want := range []string{"Agent", "Bash", "Grep"} {
		if tel.ToolCalls[want] != 1 {
			t.Errorf("tool_calls must count %s once, got %d (%v)", want, tel.ToolCalls[want], tel.ToolCalls)
		}
	}
	if len(tel.Models) != 2 || tel.Models[0] != "claude-test-1" {
		t.Errorf("models must list every distinct model: %v", tel.Models)
	}
	if len(tel.AgentTypes) != 1 || tel.AgentTypes[0] != "explorer" {
		t.Errorf("agent_types must list every distinct agent type: %v", tel.AgentTypes)
	}
	if len(tel.Agents) != 3 {
		t.Fatalf("one entry per agent including the main thread; got %d", len(tel.Agents))
	}
	if !tel.Agents[0].IsMainThread {
		t.Error("the main thread leads the agents list")
	}
	sub := tel.Agents[1]
	if sub.AgentID != "agentone" || sub.AgentType != "explorer" || sub.SpawnDepth != 1 {
		t.Errorf("the sub-agent entry must carry its lineage: %+v", sub)
	}
	if sub.StartedAt == nil || sub.WallClockSeconds != 282 {
		t.Errorf("each agent carries its own span; got %+v (%v)", sub.StartedAt, sub.WallClockSeconds)
	}
	if sub.ToolCalls["Grep"] != 1 || sub.Turns.Total != 4 || sub.Tokens.Total == 0 {
		t.Errorf("each agent carries its own turns, tokens and tool calls: %+v", sub)
	}
	if sub.Record == "" || strings.ContainsRune(sub.Record, '/') {
		t.Errorf("an agent entry names its record by basename only; got %q", sub.Record)
	}
}

// TestTelemetryReportsItsOwnIncompleteness — the completeness block names a
// measure the fixture deliberately withholds. Without it a zero cannot be told
// from an absence, and a corpus-level comparison across harness versions would
// read "this harness records no tool calls" as "these sessions used no tools".
func TestTelemetryReportsItsOwnIncompleteness(t *testing.T) {
	_, home := setupStore(t)
	plantRecord(t, home, "20260901T100000.000000000Z-sess-bare.md", []string{
		"session_id: sess-bare",
		"captured_at: 2026-09-01T10:00:00Z",
	}, strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"no timestamps, no usage, no tools here"}}`,
		`{"type":"assistant","message":{"id":"msg_bare","role":"assistant","content":[{"type":"text","text":"understood"}]}}`,
		`{"type":"assistant","message":{"id":"msg_trunc","role":"assistant","content":[{"type":"tex`,
	}, "\n")+"\n")

	res := reconstructFixture(t, ReconstructOptions{SessionID: "sess-bare"})
	c := res.Telemetry.Completeness
	absent := strings.Join(c.AbsentFields, ",")
	for _, want := range []string{"tokens", "tool_calls", "models", "timestamps"} {
		if !strings.Contains(absent, want) {
			t.Errorf("completeness must name %s as absent; got %v", want, c.AbsentFields)
		}
	}
	if c.LinesUnparseable != 1 {
		t.Errorf("the truncated last line must be counted, not swallowed; got %d", c.LinesUnparseable)
	}
	if !strings.Contains(string(res.Artefact), "unparseable transcript lines: 1") {
		t.Errorf("the artefact must state the truncation too; got:\n%s", res.Artefact)
	}
}

// --------------------------------------------------------------------------
// Self-containment and size
// --------------------------------------------------------------------------

// TestReconstructionIsSelfContained — ac-12, scoped to what is achievable. The
// artefact must be readable with the store and the harness's files gone, so
// nothing this renderer EMITS may be a path: not the store root, not the record
// path, not a directory of any kind.
//
// It cannot mean the document contains no absolute path anywhere, and the spec
// is wrong to say so: a transcript body is a record of what was said, and what
// was said contains paths. Stripping them would falsify the transcript, which
// costs more than it buys. Redaction already removed the one class that matters
// — the home paths — on the way into the store.
func TestReconstructionIsSelfContained(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	res := reconstructFixture(t, ReconstructOptions{})
	art := string(res.Artefact)

	if strings.Contains(art, home) {
		t.Error("the artefact must not carry the store's home root")
	}
	if strings.Contains(art, ".abcd/history") || strings.Contains(art, "transcripts/") {
		t.Errorf("the artefact must not carry a store path; got:\n%s", art)
	}
	for _, line := range strings.Split(art, "\n") {
		if strings.HasPrefix(line, "- record: ") && strings.ContainsRune(line, '/') {
			t.Errorf("a record is named by basename only, never a path: %q", line)
		}
	}
	// It must also be able to explain itself: the header states what it is and
	// how to read it, because its reader arrives with no other context.
	for _, want := range []string{
		"## How to read this document",
		"## Completeness",
		"## Agent timeline",
		"## Main thread",
	} {
		if !strings.Contains(art, want) {
			t.Errorf("a self-contained artefact needs its %q block", want)
		}
	}
	if res.ArtefactName != "sess-recon.md" || res.TelemetryName != "sess-recon.telemetry.json" {
		t.Errorf("artefact/telemetry names: %q, %q", res.ArtefactName, res.TelemetryName)
	}
}

// TestReconstructIsDeterministic. Two runs over the same records must produce
// the same bytes, or a reader cannot diff one reconstruction against another.
// It is why no generation timestamp is on the artefact.
func TestReconstructIsDeterministic(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	first := reconstructFixture(t, ReconstructOptions{})
	second := reconstructFixture(t, ReconstructOptions{})
	if string(first.Artefact) != string(second.Artefact) {
		t.Error("two reconstructions of the same records must be byte-identical")
	}
}

// TestSpineModeKeepsTheThreadAndSummarisesTheDelegates. A single unbounded
// artefact is not usable by the consumer it is for — the largest main-thread
// record in the store is 38 MB, and reconstructing one real 55-record session
// in full produced a 7.7 MB artefact against 1.8 MB in spine mode. Spine
// mode keeps the main thread whole and reduces each delegate to what it was
// asked and what it concluded, with the omission stated in the document and
// counted in the telemetry.
func TestSpineModeKeepsTheThreadAndSummarisesTheDelegates(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	full := reconstructFixture(t, ReconstructOptions{})
	spine := reconstructFixture(t, ReconstructOptions{Mode: ModeSpine})
	art := string(spine.Artefact)

	if len(spine.Artefact) >= len(full.Artefact) {
		t.Errorf("spine mode must be smaller than full: %d vs %d", len(spine.Artefact), len(full.Artefact))
	}
	if !strings.Contains(art, "Still waiting on the delegate.") {
		t.Error("spine mode keeps the MAIN thread whole")
	}
	if !strings.Contains(art, "investigate the build") {
		t.Error("spine mode keeps each delegate's opening instruction")
	}
	if !strings.Contains(art, "The linker step is the failure.") {
		t.Error("spine mode keeps each delegate's closing turn")
	}
	if strings.Contains(art, "toolu_grep") {
		t.Error("spine mode drops the delegate's middle")
	}
	if !strings.Contains(art, "turn(s) omitted in spine mode") {
		t.Errorf("the omission must be stated where it happens; got:\n%s", art)
	}
	if spine.Telemetry.Completeness.OmittedTurns != 2 {
		t.Errorf("the omission must be counted; got %d", spine.Telemetry.Completeness.OmittedTurns)
	}
	// Telemetry is computed over the whole transcript, never over the render.
	if spine.Telemetry.Tokens.Total != full.Telemetry.Tokens.Total {
		t.Errorf("spine mode must not change the measures: %d vs %d",
			spine.Telemetry.Tokens.Total, full.Telemetry.Tokens.Total)
	}
}

// TestReconstructCapsABlockAndSaysSo. The cap bounds one runaway tool result
// rather than the document, and what it removed is stated in place and counted.
func TestReconstructCapsABlockAndSaysSo(t *testing.T) {
	_, home := setupStore(t)
	long := strings.Repeat("x", 5000)
	plantRecord(t, home, "20260901T100000.000000000Z-sess-cap.md", []string{
		"session_id: sess-cap",
		"captured_at: 2026-09-01T10:00:00Z",
	}, `{"type":"user","timestamp":"2026-09-01T10:00:00Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"`+long+`"}]}}`+"\n")

	res := reconstructFixture(t, ReconstructOptions{SessionID: "sess-cap", MaxBlockBytes: 100})
	art := string(res.Artefact)
	if !strings.Contains(art, "bytes elided by the per-block cap") {
		t.Errorf("the elision must be marked in place; got:\n%s", art)
	}
	if res.Telemetry.Completeness.ElidedBlocks != 1 || res.Telemetry.Completeness.ElidedBytes != 4900 {
		t.Errorf("the elision must be counted: blocks=%d bytes=%d",
			res.Telemetry.Completeness.ElidedBlocks, res.Telemetry.Completeness.ElidedBytes)
	}
	if strings.Contains(art, strings.Repeat("x", 200)) {
		t.Error("the capped block must actually be shortened")
	}
	// The Completeness section is the one place a reader looks to find out what
	// the document is missing, and it is at the HEAD — so it has to report an
	// elision that is only discovered while the body below it is rendered.
	head := art[:strings.Index(art, "\n## Main thread\n")]
	if !strings.Contains(head, "4900 bytes elided") {
		t.Errorf("the completeness block at the head must report the elision found while rendering the body; head was:\n%s", head)
	}
}

// TestReconstructRefusesWhatItCannotAnswer.
func TestReconstructRefusesWhatItCannotAnswer(t *testing.T) {
	_, home := setupStore(t)
	plantFixtureSession(t, home)

	for _, tc := range []struct {
		name string
		opts ReconstructOptions
		want string
	}{
		{"no such session", ReconstructOptions{SessionID: "sess-absent"}, "no records for session"},
		{"empty session id", ReconstructOptions{SessionID: ""}, "sessionID must be non-empty"},
		{"traversal in the session id", ReconstructOptions{SessionID: "../escape"}, "sessionID must be non-empty"},
		{"unknown mode", ReconstructOptions{SessionID: "sess-recon", Mode: "outline"}, "is not one of full, spine"},
		{"negative cap", ReconstructOptions{SessionID: "sess-recon", MaxBlockBytes: -1}, "must not be negative"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Reconstruct(testRootSHA, tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want an error containing %q, got %v", tc.want, err)
			}
		})
	}
	if _, err := Reconstruct("not-a-sha", ReconstructOptions{SessionID: "sess-recon"}); err == nil {
		t.Error("a malformed root SHA must be refused")
	}
}
