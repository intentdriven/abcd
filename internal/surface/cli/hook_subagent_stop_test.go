package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/history"
)

// subagentPayload is the SubagentStop JSON the harness writes to the verb's
// stdin. The field set is the one the shipped harness actually delivers:
// hook_event_name, stop_hook_active, agent_id, agent_transcript_path, agent_type
// and an optional last_assistant_message. parent_agent_id is NOT in this event —
// it is in the harness's per-agent sidecar, which is why there is an attribution
// ladder at all.
func subagentPayload(t *testing.T, session, cwd, agentID, transcript, agentType string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"hook_event_name":       "SubagentStop",
		"stop_hook_active":      false,
		"session_id":            session,
		"cwd":                   cwd,
		"agent_id":              agentID,
		"agent_transcript_path": transcript,
		"agent_type":            agentType,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// writeAgentTranscript writes a settled JSONL transcript and returns its path.
func writeAgentTranscript(t *testing.T, dir, agentID, body string) string {
	t.Helper()
	p := filepath.Join(dir, "agent-"+agentID+".jsonl")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// runHookAllowingFailure runs a hook verb and returns stdout, stderr and whether
// it exited non-zero. SubagentStop is a BLOCKING event — a non-zero exit stops
// the sub-agent from stopping — so the exit code is the property under test in
// several of these, not an incidental.
func runHookAllowingFailure(stdin string, args ...string) (stdout, stderr string, failed bool) {
	cmd := NewRootCommand()
	var so, se bytes.Buffer
	cmd.SetOut(&so)
	cmd.SetErr(&se)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return so.String(), se.String(), err != nil
}

// TestHookSubagentStopStagesLineage is the milestone of step 3: a sub-agent that
// finishes leaves its transcript staged, carrying the lineage that says which
// session and which agent produced it. Without it the corpus records only
// spines, and 176 of the store's existing records had to overload the session id
// to say even that much.
func TestHookSubagentStopStagesLineage(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := writeAgentTranscript(t, t.TempDir(), "a1", `{"role":"assistant","text":"done"}`+"\n")

	runHook(t, subagentPayload(t, "sess-parent", repo, "a1", tp, "general-purpose"),
		"hook", "subagent-stop")

	staged, err := history.ListStaged(rootSHA)
	if err != nil {
		t.Fatalf("ListStaged: %v", err)
	}
	if len(staged) != 1 {
		t.Fatalf("want 1 staged transcript, got %d", len(staged))
	}
	s := staged[0]
	if s.SessionID != "sess-parent" {
		t.Errorf("session id = %q, want the SPAWNING session untruncated", s.SessionID)
	}
	if s.AgentID != "a1" {
		t.Errorf("agent id = %q, want a1", s.AgentID)
	}
	if s.AgentType != "general-purpose" {
		t.Errorf("agent type = %q, want general-purpose", s.AgentType)
	}
	if s.LineageSource != "hook" {
		t.Errorf("lineage source = %q, want hook", s.LineageSource)
	}
	// No harness sidecar beside that transcript, so nothing placed the spawn
	// point and the record must say so rather than read as a child of the main
	// thread.
	if s.SpawnAttribution != "unattributed" {
		t.Errorf("spawn attribution = %q, want unattributed with no harness sidecar", s.SpawnAttribution)
	}
	if s.SourcePath != tp {
		t.Errorf("source path = %q, want %q (the drain re-reads it)", s.SourcePath, tp)
	}
}

// TestHookSubagentStopStagesRatherThanCaptures: SubagentStop fires inside a live
// session, where redaction's ~0.7s/MB is felt directly. The hook must stage.
func TestHookSubagentStopStagesRatherThanCaptures(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := writeAgentTranscript(t, t.TempDir(), "a2", `{"a":1}`+"\n")

	runHook(t, subagentPayload(t, "sess-p", repo, "a2", tp, "general-purpose"),
		"hook", "subagent-stop")

	recs, err := history.List(rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("the hook captured %d record(s); it must only stage", len(recs))
	}
}

// TestHookSubagentStopReadsTheHarnessSidecar is the first rung of the
// attribution ladder. The sidecar is derived from the transcript path by
// substituting the extension — never by walking a directory, which would be the
// on-disk-layout dependency this design exists without.
func TestHookSubagentStopReadsTheHarnessSidecar(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	dir := t.TempDir()
	tp := writeAgentTranscript(t, dir, "a3", `{"a":1}`+"\n")
	side := strings.TrimSuffix(tp, ".jsonl") + ".meta.json"
	if err := os.WriteFile(side, []byte(`{"agentType":"ruthless-reviewer","description":"d",`+
		`"toolUseId":"toolu_01","spawnDepth":2,"parentAgentId":"a0","model":"opus"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	runHook(t, subagentPayload(t, "sess-p", repo, "a3", tp, "ruthless-reviewer"),
		"hook", "subagent-stop")

	staged, err := history.ListStaged(rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 {
		t.Fatalf("want 1 staged transcript, got %d", len(staged))
	}
	s := staged[0]
	if s.SpawnAttribution != "sidecar" {
		t.Errorf("spawn attribution = %q, want sidecar", s.SpawnAttribution)
	}
	if s.ParentAgentID != "a0" {
		t.Errorf("parent agent = %q, want a0", s.ParentAgentID)
	}
	if s.SpawnDepth != 2 {
		t.Errorf("spawn depth = %d, want 2", s.SpawnDepth)
	}
	if s.SpawnToolUseID != "toolu_01" {
		t.Errorf("spawn tool use id = %q, want toolu_01", s.SpawnToolUseID)
	}
}

// TestHookSubagentStopResolvesTheRepoThroughTheSession is the review finding
// that would otherwise have lost exactly the implementation-lane agents. A
// sub-agent given its own worktree records that worktree as its cwd, and the
// harness REMOVES the worktree when the agent stops — so by the time
// SubagentStop fires, resolving the payload's cwd the way session-end does
// fails, and the hook would exit 0 having captured nothing. The spawning
// session's id is the fallback: a store that has seen that session knows which
// repo it belongs to.
func TestHookSubagentStopResolvesTheRepoThroughTheSession(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	// The parent session started in the repo, which is what records the tie.
	runSessionStart(startPayload("sess-parent", repo), "hook", "session-start")

	gone := filepath.Join(t.TempDir(), "worktree-that-was-removed")
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	tp := writeAgentTranscript(t, t.TempDir(), "a4", `{"a":1}`+"\n")

	_, errlog := runHook(t, subagentPayload(t, "sess-parent", gone, "a4", tp, "general-purpose"),
		"hook", "subagent-stop")

	staged, err := history.ListStaged(rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 {
		t.Fatalf("a sub-agent whose worktree was removed staged nothing (stderr: %s)", errlog)
	}
	if staged[0].AgentID != "a4" {
		t.Errorf("agent id = %q, want a4", staged[0].AgentID)
	}
	// Pin the MECHANISM, not just the outcome: the cwd rung must have failed
	// and the session rung must have answered, or this test would still pass
	// if the cwd fallback were quietly resolving to some other repository.
	if !strings.Contains(errlog, "resolved via session") {
		t.Errorf("the store was not resolved through the spawning session: %q", errlog)
	}
}

// TestHookSubagentStopMarksAMissingPayloadField: on a harness that does not
// deliver agent_transcript_path there is nothing to stage, and silence is
// exactly what this intent exists to end. The hook records a marker and
// `history staged` says so.
func TestHookSubagentStopMarksAMissingPayloadField(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)

	_, errlog := runHook(t, subagentPayload(t, "sess-p", repo, "a5", "", "general-purpose"),
		"hook", "subagent-stop")
	if !strings.Contains(errlog, "agent_transcript_path") {
		t.Errorf("stderr does not name the missing field: %q", errlog)
	}
	note, ok, err := history.SubagentGap(rootSHA)
	if err != nil {
		t.Fatalf("SubagentGap: %v", err)
	}
	if !ok {
		t.Fatal("no marker recorded for a payload with no agent_transcript_path")
	}
	if note.FirstSeen.IsZero() {
		t.Error("the marker records no first-seen time")
	}

	t.Chdir(repo)
	stdout, _ := runHook(t, "", "history", "staged")
	if !strings.Contains(stdout, "sub-agent") {
		t.Errorf("`history staged` does not report the gap: %q", stdout)
	}
}

// TestHookSubagentStopAlwaysExitsZero is load-bearing, not tidiness.
// SubagentStop is the event whose exit code 2 BLOCKS — it stops the sub-agent
// from finishing — and a Go error returned from this command is what would put
// a non-zero code on the table at all. So every failure here degrades to a
// diagnostic on stderr and exits 0, rather than leaning on "2 is the only code
// that blocks" to stay safe. (The launcher's own `exit 1` when no binary
// resolves is the deliberate exception: non-zero, not 2, and the only warning a
// user gets that transcripts are going uncaptured — see
// TestSubagentStopNeverBootstraps.)
func TestHookSubagentStopAlwaysExitsZero(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	dir := t.TempDir()
	missing := filepath.Join(dir, "not-there.jsonl")
	fifo := filepath.Join(dir, "fifo.jsonl")

	for _, tc := range []struct {
		name  string
		stdin string
	}{
		{"malformed json", "{not json"},
		{"empty payload", "{}"},
		{"no agent transcript path", subagentPayload(t, "s", repo, "a", "", "t")},
		{"no agent id", subagentPayload(t, "s", repo, "", missing, "t")},
		{"unreadable transcript", subagentPayload(t, "s", repo, "a", missing, "t")},
		{"unresolvable repo and unknown session", subagentPayload(t, "s-unknown", dir, "a", fifo, "t")},
		{"empty transcript", subagentPayload(t, "s", repo, "a", writeAgentTranscript(t, dir, "empty", ""), "t")},
	} {
		_, stderr, failed := runHookAllowingFailure(tc.stdin, "hook", "subagent-stop")
		if failed {
			t.Errorf("%s: subagent-stop exited non-zero — it would have blocked the sub-agent from stopping (stderr: %s)",
				tc.name, stderr)
		}
	}
}

// TestHookSubagentStopWaitsForAnUnsettledTranscript is the stage-time half of
// the flush-race mitigation. A transcript whose final line is a severed JSON
// object is still being written; the hook waits a bounded number of attempts,
// then stages what it has and SAYS the copy may be short. Whether the event
// actually fires before the flush is unmeasured — the diagnostic is what makes
// the rate measurable.
func TestHookSubagentStopWaitsForAnUnsettledTranscript(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "agent-a6.jsonl")
	if err := os.WriteFile(tp, []byte(`{"a":1}`+"\n"+`{"b":`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, errlog := runHook(t, subagentPayload(t, "sess-p", repo, "a6", tp, "general-purpose"),
		"hook", "subagent-stop")

	staged, err := history.ListStaged(rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 {
		t.Fatal("an unsettled transcript was dropped rather than staged short")
	}
	if !strings.Contains(errlog, "incomplete") {
		t.Errorf("stderr does not report the unsettled read: %q", errlog)
	}
}

// TestHookSubagentStopConcurrentCompletionsAllStage is the second review
// finding. Many sub-agents finish at once, and the staging lock's 5s timeout was
// tuned for a single SessionEnd — so any wait the hook adds has to sit outside
// it, or a burst serialises and the slowest completions are refused.
func TestHookSubagentStopConcurrentCompletionsAllStage(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	dir := t.TempDir()
	const n = 12
	payloads := make([]string, n)
	for i := range n {
		id := "b" + string(rune('a'+i))
		tp := writeAgentTranscript(t, dir, id, `{"agent":"`+id+`"}`+"\n")
		payloads[i] = subagentPayload(t, "sess-burst", repo, id, tp, "general-purpose")
	}

	var wg sync.WaitGroup
	fails := make([]string, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, stderr, failed := runHookAllowingFailure(payloads[i], "hook", "subagent-stop"); failed {
				fails[i] = stderr
			}
		}()
	}
	wg.Wait()
	for i, f := range fails {
		if f != "" {
			t.Errorf("concurrent completion %d exited non-zero: %s", i, f)
		}
	}
	staged, err := history.ListStaged(rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != n {
		t.Fatalf("want %d staged transcripts from %d simultaneous completions, got %d", n, n, len(staged))
	}
}

// TestSubagentStopThenSessionStartStoresTheRecord is the wired-end-to-end
// property: staging is only half of capture, and a sub-agent transcript that
// never reaches a record is a file in a 0o700 directory, not a corpus entry.
// The hook stages, the next SessionStart drains, and the record carries the
// lineage the whole schema change exists for.
func TestSubagentStopThenSessionStartStoresTheRecord(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	dir := t.TempDir()
	tp := writeAgentTranscript(t, dir, "a7", `{"role":"assistant","text":"branch"}`+"\n")
	side := strings.TrimSuffix(tp, ".jsonl") + ".meta.json"
	if err := os.WriteFile(side, []byte(`{"agentType":"Explore","toolUseId":"toolu_07","spawnDepth":1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	runHook(t, subagentPayload(t, "sess-e2e", repo, "a7", tp, "Explore"), "hook", "subagent-stop")
	runSessionStart(startPayload("sess-next", repo), "hook", "session-start")

	recs, err := history.List(rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("want 1 stored record after the drain, got %d", len(recs))
	}
	r := recs[0]
	if r.SessionID != "sess-e2e" || r.AgentID != "a7" || r.AgentType != "Explore" ||
		r.SpawnDepth != 1 || r.SpawnAttribution != "sidecar" || r.SpawnToolUseID != "toolu_07" ||
		r.LineageSource != "hook" {
		t.Errorf("stored record lost the hook's lineage: %+v", r)
	}
	// A depth-1 sidecar names no parent, and that is information rather than a
	// gap: the main thread spawned it. spawn_attribution is what keeps that
	// distinguishable from a lineage nothing could recover.
	if r.ParentAgentID != "" {
		t.Errorf("parent agent = %q, want empty — a depth-1 agent was spawned by the main thread", r.ParentAgentID)
	}
	staged, err := history.ListStaged(rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 0 {
		t.Errorf("the staged copy survived a successful drain: %+v", staged)
	}
}
