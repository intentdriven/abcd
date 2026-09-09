package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// subAgentStage is the StageMeta a SubagentStop hook produces once its sidecar
// rung has answered.
func subAgentStage(sessionID, agentID string) StageMeta {
	return StageMeta{
		Lineage: CaptureMeta{
			SessionID:        sessionID,
			Kind:             "native",
			AgentID:          agentID,
			AgentType:        "general-purpose",
			SpawnDepth:       1,
			SpawnToolUseID:   "toolu_" + agentID,
			LineageSource:    "hook",
			SpawnAttribution: "sidecar",
		},
	}
}

// stagingDir is the staging directory for the test store.
func stagingDir(home string) string {
	return filepath.Join(home, ".abcd", "history", testRootSHA, "staging")
}

// TestStageWritesLineageSidecar is the point of step 2. A staged file used to be
// identified by parsing its filename, which is why the filename had to encode
// the session; encoding an agent there too would rebuild the composite-identifier
// defect adr-2609090636172016 removed, one directory earlier. So the lineage
// goes in a sidecar and the filename goes back to being just a name.
func TestStageWritesLineageSidecar(t *testing.T) {
	_, home := setupStore(t)

	res, err := Stage(testRootSHA, subAgentStage("sess-parent", "agent-abc"), []byte("hello\n"))
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}
	side := strings.TrimSuffix(res.Staged.Path, stagedSuffix) + stageSidecarSuffix
	data, err := os.ReadFile(side)
	if err != nil {
		t.Fatalf("no staging sidecar beside the staged file: %v", err)
	}
	fi, err := os.Stat(side)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("sidecar mode = %o, want 600", perm)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("sidecar does not parse as JSON: %v", err)
	}
	for k, want := range map[string]any{
		"schema":            float64(1),
		"session_id":        "sess-parent",
		"agent_id":          "agent-abc",
		"agent_type":        "general-purpose",
		"spawn_depth":       float64(1),
		"spawn_tool_use_id": "toolu_agent-abc",
		"lineage_source":    "hook",
		"spawn_attribution": "sidecar",
	} {
		if got[k] != want {
			t.Errorf("sidecar[%q] = %v, want %v", k, got[k], want)
		}
	}
	if _, ok := got["staged_at"]; !ok {
		t.Error("sidecar carries no staged_at")
	}
	// The session must NOT be recoverable from the filename of a sub-agent
	// stage: that is the encoding this sidecar exists to stop.
	if strings.Contains(filepath.Base(res.Staged.Path), "sess-parent") {
		t.Errorf("staged filename %q encodes the session id; lineage belongs in the sidecar",
			filepath.Base(res.Staged.Path))
	}
	_ = home
}

// TestListStagedReadsSidecarLineage: the sidecar is only useful if the listing
// reads it. Nothing else can — the drain has no other source for the lineage it
// hands Capture.
func TestListStagedReadsSidecarLineage(t *testing.T) {
	_, _ = setupStore(t)
	if _, err := Stage(testRootSHA, subAgentStage("sess-p", "agent-1"), []byte("body\n")); err != nil {
		t.Fatalf("Stage failed: %v", err)
	}
	staged, err := ListStaged(testRootSHA)
	if err != nil {
		t.Fatalf("ListStaged: %v", err)
	}
	if len(staged) != 1 {
		t.Fatalf("want 1 staged entry, got %d", len(staged))
	}
	s := staged[0]
	if s.SessionID != "sess-p" || s.AgentID != "agent-1" || s.AgentType != "general-purpose" ||
		s.SpawnDepth != 1 || s.SpawnAttribution != "sidecar" || s.LineageSource != "hook" {
		t.Errorf("staged entry lost its lineage: %+v", s)
	}
}

// TestStageIdempotencyIsPerSessionAndAgent widens the idempotency key. Keyed on
// the session alone, a session's second sub-agent completion would REPLACE the
// first one's staged transcript and the first would be lost silently — the
// precise failure staging exists to end.
func TestStageIdempotencyIsPerSessionAndAgent(t *testing.T) {
	_, home := setupStore(t)
	for _, a := range []string{"agent-1", "agent-2"} {
		if _, err := Stage(testRootSHA, subAgentStage("sess-x", a), []byte("body of "+a+"\n")); err != nil {
			t.Fatalf("Stage(%s): %v", a, err)
		}
	}
	if _, err := Stage(testRootSHA, mainStage("sess-x"), []byte("the spine\n")); err != nil {
		t.Fatalf("Stage(main): %v", err)
	}
	if names := stagedNames(t, home); len(names) != 3 {
		t.Fatalf("want 3 staged files (two agents and the main thread), got %d: %v", len(names), names)
	}
	// Identical bytes for the same (session, agent) is still a no-op.
	res, err := Stage(testRootSHA, subAgentStage("sess-x", "agent-1"), []byte("body of agent-1\n"))
	if err != nil {
		t.Fatalf("re-Stage: %v", err)
	}
	if res.Wrote {
		t.Error("re-staging identical bytes for the same (session, agent) wrote again")
	}
	// Different bytes for the same (session, agent) replace in place.
	res, err = Stage(testRootSHA, subAgentStage("sess-x", "agent-1"), []byte("body of agent-1, longer\n"))
	if err != nil {
		t.Fatalf("re-Stage longer: %v", err)
	}
	if !res.Replaced {
		t.Error("longer bytes for the same (session, agent) did not replace the staged copy")
	}
	if names := stagedNames(t, home); len(names) != 3 {
		t.Fatalf("replacement changed the staged file count: %v", names)
	}
}

// TestLegacyStagedFileWithoutSidecarStillDrains: there are real sidecar-less
// .raw files on disk, written by the binary before this change. An upgrade that
// stranded them would leave unredacted transcript text sitting at 0o700 with
// nothing that would ever collect it.
func TestLegacyStagedFileWithoutSidecarStillDrains(t *testing.T) {
	repoRoot, home := setupStore(t)
	sdir := stagingDir(home)
	if err := os.MkdirAll(sdir, 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(sdir, "20250101T000000.000000000Z-sess-legacy"+stagedSuffix)
	if err := os.WriteFile(legacy, []byte("an older binary staged this\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Failed) != 0 {
		t.Fatalf("legacy staged file failed to drain: %+v", res.Failed)
	}
	if len(res.Captured) != 1 || res.Captured[0].SessionID != "sess-legacy" {
		t.Fatalf("want the legacy file stored under sess-legacy, got %+v", res.Captured)
	}
	if res.Captured[0].AgentID != "" {
		t.Errorf("a sidecar-less staged file must drain as a main-thread transcript, got agent %q",
			res.Captured[0].AgentID)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("the legacy staged file survived a successful drain")
	}
}

// TestDrainTakesMainThreadFirst: listStaged is chronological and a session's
// main thread stages LAST, because it ends last. A bounded pass would therefore
// drain the branches and leave the spine — the one record that makes the rest
// legible.
func TestDrainTakesMainThreadFirst(t *testing.T) {
	repoRoot, _ := setupStore(t)
	for _, a := range []string{"agent-1", "agent-2"} {
		if _, err := Stage(testRootSHA, subAgentStage("sess-order", a), []byte("branch "+a+"\n")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Stage(testRootSHA, mainStage("sess-order"), []byte("the spine\n")); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("want 1 captured under a budget of 1, got %d", len(res.Captured))
	}
	if res.Captured[0].AgentID != "" {
		t.Errorf("the truncated pass stored a branch (agent %q) and left the spine",
			res.Captured[0].AgentID)
	}
	if res.Remaining != 2 {
		t.Errorf("Remaining = %d, want 2", res.Remaining)
	}
}

// TestDrainByteBudgetBoundsThePass: redaction cost tracks bytes, not files, so a
// count-only budget bounds the wrong thing. Three equal transcripts under a byte
// budget that fits one must stop after one, with MaxEntries unset.
func TestDrainByteBudgetBoundsThePass(t *testing.T) {
	repoRoot, _ := setupStore(t)
	body := strings.Repeat("x", 1000) + "\n"
	for _, id := range []string{"sess-b1", "sess-b2", "sess-b3"} {
		if _, err := Stage(testRootSHA, mainStage(id), []byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{MaxBytes: 1500})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("want 1 captured under a 1500-byte budget of 1001-byte entries, got %d", len(res.Captured))
	}
	if res.Remaining != 2 {
		t.Errorf("Remaining = %d, want 2", res.Remaining)
	}
}

// TestDrainByteBudgetAlwaysMakesProgress: an entry larger than the whole byte
// budget must still drain, or the raw unredacted file it names would be skipped
// by every pass forever — the failure the budget exists to bound, inverted.
func TestDrainByteBudgetAlwaysMakesProgress(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if _, err := Stage(testRootSHA, mainStage("sess-huge"), []byte(strings.Repeat("y", 5000)+"\n")); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{MaxBytes: 10})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("an over-budget entry was skipped forever instead of drained: %+v", res)
	}
}

// TestDrainCarriesSidecarLineageIntoTheRecord closes the loop: the lineage the
// hook staged must reach the stored record, or the sidecar is decoration.
func TestDrainCarriesSidecarLineageIntoTheRecord(t *testing.T) {
	repoRoot, home := setupStore(t)
	if _, err := Stage(testRootSHA, subAgentStage("sess-l", "agent-z"), []byte("branch body\n")); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("want 1 captured, got %+v", res)
	}
	r := res.Captured[0]
	if r.SessionID != "sess-l" || r.AgentID != "agent-z" || r.AgentType != "general-purpose" ||
		r.SpawnDepth != 1 || r.SpawnAttribution != "sidecar" || r.LineageSource != "hook" {
		t.Errorf("record lost the staged lineage: %+v", r)
	}
	// Both files go when the transcript is stored: a sidecar left behind names a
	// staged file that no longer exists.
	entries, err := os.ReadDir(stagingDir(home))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), stageSidecarSuffix) {
			t.Errorf("sidecar %s survived the drain of its staged file", e.Name())
		}
	}
}

// TestDrainRereadsALongerSource is the flush-race mitigation. If SubagentStop
// fires before the harness has finished writing the transcript, the staged copy
// is a strict prefix of what is on disk by the time the drain runs. The source
// replaces it ONLY on that prefix relation, so a recycled path can never
// substitute a different transcript.
func TestDrainRereadsALongerSource(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := filepath.Join(t.TempDir(), "agent.jsonl")
	if err := os.WriteFile(src, []byte("line one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta := subAgentStage("sess-r", "agent-r")
	meta.SourcePath = src
	if _, err := Stage(testRootSHA, meta, []byte("line one\n")); err != nil {
		t.Fatal(err)
	}
	// The harness finishes writing between the stage and the drain.
	if err := os.WriteFile(src, []byte("line one\nline two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("want 1 captured, got %+v", res)
	}
	// The catch is counted, not silent: the rate is what the flush-race
	// measurement reports, and a swap nobody counts cannot be measured.
	if res.Extended != 1 {
		t.Errorf("Extended = %d, want 1 — a completed source went uncounted", res.Extended)
	}
	_, body, err := Read(testRootSHA, "agent-r")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(string(body), "line two") {
		t.Errorf("the drain did not re-read the completed source; body = %q", body)
	}
}

// TestDrainIgnoresADivergentSource is the other half of the prefix rule: a
// source path the harness recycled for a DIFFERENT transcript must never
// overwrite what was staged.
func TestDrainIgnoresADivergentSource(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := filepath.Join(t.TempDir(), "agent.jsonl")
	if err := os.WriteFile(src, []byte("staged bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta := subAgentStage("sess-d", "agent-d")
	meta.SourcePath = src
	if _, err := Stage(testRootSHA, meta, []byte("staged bytes\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("a completely different, much longer transcript\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dr, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if dr.Extended != 0 {
		t.Errorf("Extended = %d, want 0 — a divergent source was counted as a caught truncation", dr.Extended)
	}
	_, body, err := Read(testRootSHA, "agent-d")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if strings.Contains(string(body), "completely different") {
		t.Error("a recycled source path overwrote the staged transcript; the prefix check did not hold")
	}
}

// TestUnreadableSidecarIsReportedNotMisattributed: a sidecar that is present but
// corrupt cannot be ignored in favour of the filename. The filename key of a
// sub-agent stage is the AGENT id, so parsing it as a session would file the
// transcript under a session that does not exist. It is reported and left.
func TestUnreadableSidecarIsReportedNotMisattributed(t *testing.T) {
	repoRoot, home := setupStore(t)
	res, err := Stage(testRootSHA, subAgentStage("sess-c", "agent-c"), []byte("body\n"))
	if err != nil {
		t.Fatal(err)
	}
	side := strings.TrimSuffix(res.Staged.Path, stagedSuffix) + stageSidecarSuffix
	if err := os.WriteFile(side, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	dr, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(dr.Captured) != 0 {
		t.Errorf("a transcript with an unreadable sidecar was stored anyway: %+v", dr.Captured)
	}
	if len(dr.Failed) != 1 {
		t.Fatalf("want 1 reported failure, got %+v", dr.Failed)
	}
	if _, err := os.Stat(res.Staged.Path); err != nil {
		t.Errorf("the staged file was removed despite the failure: %v", err)
	}
	_ = home
}

// TestConcurrentSubAgentStagesAllLand: many sub-agents can finish at once, and
// every one of their transcripts must be staged. The staging lock's 5s timeout
// was tuned for a single SessionEnd, so anything a stage waits on has to sit
// OUTSIDE the lock or a burst serialises on it and the slowest completions are
// refused with a contention error.
func TestConcurrentSubAgentStagesAllLand(t *testing.T) {
	_, home := setupStore(t)
	const n = 16
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := "agent-" + string(rune('a'+i))
			_, errs[i] = Stage(testRootSHA, subAgentStage("sess-burst", id),
				[]byte("branch "+id+"\n"))
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent stage %d failed: %v", i, err)
		}
	}
	if names := stagedNames(t, home); len(names) != n {
		t.Fatalf("want %d staged files from %d concurrent completions, got %d", n, n, len(names))
	}
	staged, err := ListStaged(testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, s := range staged {
		if s.Err != "" {
			t.Errorf("staged entry %s reports %s", s.AgentID, s.Err)
		}
		seen[s.AgentID] = true
	}
	if len(seen) != n {
		t.Errorf("want %d distinct agents staged, got %d", n, len(seen))
	}
}

// TestTranscriptSettled is the stage-time half of the flush-race mitigation: the
// predicate a caller waits on. A JSONL transcript whose last line is a partial
// object is still being written.
func TestTranscriptSettled(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want bool
	}{
		{"complete line", "{\"a\":1}\n", true},
		{"complete line without a trailing newline", "{\"a\":1}", true},
		{"truncated final object", "{\"a\":1}\n{\"b\":", false},
		{"trailing blank lines are ignored", "{\"a\":1}\n\n\n", true},
		{"empty", "", false},
		{"whitespace only", "   \n", false},
	} {
		if got := TranscriptSettled([]byte(tc.raw)); got != tc.want {
			t.Errorf("%s: TranscriptSettled = %v, want %v", tc.name, got, tc.want)
		}
	}
}
