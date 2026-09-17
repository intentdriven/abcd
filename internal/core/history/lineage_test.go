package history

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// planted writes a record file straight into the store, bypassing Capture, so a
// test can pin how a record written by an EARLIER binary reads back.
func planted(t *testing.T, home, name, content string) string {
	t.Helper()
	p := filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records", name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestSchemaOneRecordReadsAsMainThread is the both-shapes guarantee. Every
// record in the store was written under schema 1, which has no lineage fields
// at all; a reader that admitted only the current schema would lose the corpus.
// A schema-1 record must parse as a main-thread record with empty lineage, and
// a record written now must stamp the current version.
func TestSchemaOneRecordReadsAsMainThread(t *testing.T) {
	repoRoot, home := setupStore(t)

	planted(t, home, "20250101T000000.000000000Z-sess-old.md", strings.Join([]string{
		"---",
		"schema: 1",
		"session_id: sess-old",
		"root_commit: " + testRootSHA,
		"captured_at: 2025-01-01T00:00:00Z",
		"source_kind: native",
		"source_sha256: " + strings.Repeat("0", 64),
		"redacted_secrets: 0",
		"redacted_home_paths: 0",
		"---",
		"user: an older binary wrote this",
		"",
	}, "\n"))

	rec, body, err := Read(repoRoot, testRootSHA, "sess-old")
	if err != nil {
		t.Fatalf("a schema-1 record must still be readable: %v", err)
	}
	if rec.AgentID != "" || rec.ParentAgentID != "" || rec.AgentType != "" ||
		rec.SpawnDepth != 0 || rec.SpawnToolUseID != "" || rec.LineageSource != "" {
		t.Errorf("a schema-1 record must read as main-thread with empty lineage, got %+v", rec)
	}
	if !strings.Contains(string(body), "an older binary wrote this") {
		t.Errorf("schema-1 body lost: %q", body)
	}

	res, err := Capture(repoRoot, testRootSHA, []byte("user: new\n"), CaptureMeta{SessionID: "sess-new", Kind: "native"})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "schema: 3") {
		t.Errorf("a record written now must stamp the current schema, got:\n%s", onDisk)
	}
}

// TestLineageRoundTripsThroughTheRecord pins the six fields end to end: what
// Capture is handed is what marshalRecord writes and what parseRecord reads
// back. Without this a field can be added to the struct, ignored by the writer,
// and still look right in the CaptureResult the caller already holds.
func TestLineageRoundTripsThroughTheRecord(t *testing.T) {
	repoRoot, _ := setupStore(t)

	meta := CaptureMeta{
		SessionID:        "sess-lineage",
		Kind:             "native",
		AgentID:          "agent-7",
		ParentAgentID:    "agent-1",
		AgentType:        "ruthless-reviewer",
		SpawnDepth:       2,
		SpawnToolUseID:   "toolu_abc123",
		LineageSource:    "hook",
		SpawnAttribution: "sidecar",
	}
	if _, err := Capture(repoRoot, testRootSHA, []byte("assistant: reviewed\n"), meta); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	recs, err := List(repoRoot, testRootSHA)
	if err != nil || len(recs) != 1 {
		t.Fatalf("List = (%d records, %v)", len(recs), err)
	}
	got := recs[0]
	if got.SessionID != "sess-lineage" {
		t.Errorf("session_id = %q; a sub-agent record carries the FULL spawning session id", got.SessionID)
	}
	if got.AgentID != "agent-7" || got.ParentAgentID != "agent-1" || got.AgentType != "ruthless-reviewer" ||
		got.SpawnDepth != 2 || got.SpawnToolUseID != "toolu_abc123" || got.LineageSource != "hook" {
		t.Errorf("lineage did not round-trip through the record: %+v", got)
	}
}

// TestSubagentRecordFilenameNamesTheAgent pins the readable-convenience half of
// the schema change: a sub-agent's record file says which agent it holds, so an
// operator listing the directory can tell the spine from the branches. Nothing
// parses this back — listRecords reads frontmatter — but a directory of files
// named only for their session is unreadable once a session has a dozen.
func TestSubagentRecordFilenameNamesTheAgent(t *testing.T) {
	repoRoot, _ := setupStore(t)

	main, err := Capture(repoRoot, testRootSHA, []byte("user: spine\n"),
		CaptureMeta{SessionID: "sess-fn", Kind: "native"})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := Capture(repoRoot, testRootSHA, []byte("user: branch\n"),
		CaptureMeta{SessionID: "sess-fn", Kind: "native", AgentID: "agent-fn",
			LineageSource: "hook", SpawnAttribution: "unattributed"})
	if err != nil {
		t.Fatal(err)
	}
	if b := filepath.Base(main.Record.Path); !strings.HasSuffix(b, "-sess-fn.md") {
		t.Errorf("main-thread record filename = %q, want <stamp>-sess-fn.md", b)
	}
	if b := filepath.Base(sub.Record.Path); !strings.HasSuffix(b, "-sess-fn-agent-agent-fn.md") {
		t.Errorf("sub-agent record filename = %q, want <stamp>-sess-fn-agent-agent-fn.md", b)
	}
}

// TestLineageFieldsAreRedactedWithTheBody is the redaction-bypass guard. Every
// lineage field is externally supplied — an agent type comes off a harness
// payload — and frontmatter has never been scanned, so a field written straight
// into it would be a hole beside the gate. The scalars go through the SAME
// two-stage pass as the body, so what lands on disk is redacted.
func TestLineageFieldsAreRedactedWithTheBody(t *testing.T) {
	base := t.TempDir()
	user := "zzlineageuser42"
	home := filepath.Join(base, user)
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records"), 0o755); err != nil {
		t.Fatal(err)
	}

	repoRoot := t.TempDir()
	res, err := Capture(repoRoot, testRootSHA, []byte("assistant: done\n"), CaptureMeta{
		SessionID:        "sess-redactfm",
		Kind:             "native",
		AgentID:          "agent-redactfm",
		AgentType:        "reviewer-of " + home + "/notes",
		LineageSource:    "hook",
		SpawnAttribution: "unattributed",
	})
	if err != nil {
		t.Fatalf("Capture refused: %v", err)
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(onDisk), user) {
		t.Errorf("the caller's home survived in the frontmatter:\n%s", onDisk)
	}
	rec, _, err := Read(repoRoot, testRootSHA, "agent-redactfm")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if strings.Contains(rec.AgentType, user) {
		t.Errorf("agent_type read back unredacted: %q", rec.AgentType)
	}
	if !strings.Contains(rec.AgentType, "reviewer-of ") {
		t.Errorf("agent_type lost its non-identifying content: %q", rec.AgentType)
	}
	if !strings.Contains(string(onDisk), "assistant: done") {
		t.Errorf("the body was lost when the scalars were split back off:\n%s", onDisk)
	}
}

// TestBlockingSpanInAgentTypeRefusesTheWrite is the fail-closed tail of the
// same guarantee. A scalar whose redaction merely changed it is stored changed;
// a scalar carrying a span that SURVIVES redaction refuses the whole write,
// exactly as a surviving span in the body does. The pattern is built to survive
// on purpose: it matches both the raw token and the token's masked fingerprint.
func TestBlockingSpanInAgentTypeRefusesTheWrite(t *testing.T) {
	repoRoot, home := setupStore(t)
	cfgDir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"patterns":{"sticky":{"regex":"ACM[A-Za-z0-9*]{13}Z9","kind":"token","label":"sticky token","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	token := "ACME" + strings.Repeat("Q", 12) + "Z9"

	res, err := Capture(repoRoot, testRootSHA, []byte("assistant: hi\n"), CaptureMeta{
		SessionID:        "sess-stickyfm",
		Kind:             "native",
		AgentID:          "agent-stickyfm",
		AgentType:        token,
		LineageSource:    "hook",
		SpawnAttribution: "unattributed",
	})
	var rerr *RedactionResidualError
	if !errors.As(err, &rerr) {
		t.Fatalf("Capture = (wrote=%v, err=%v); a surviving blocking span in agent_type must refuse the write",
			res.Wrote, err)
	}
	entries, err := os.ReadDir(filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("a refused capture wrote a record anyway: %s", e.Name())
		}
	}
}

// TestTwoSubagentsWithIdenticalBytesBothStore bounds the idempotency key from
// the other side. Two sub-agents of ONE session can produce byte-identical
// transcripts — two reviewers handed the same file and answering "no findings"
// is enough — and without the agent id in the key the second collapses into the
// first's record and is lost while Capture reports success.
func TestTwoSubagentsWithIdenticalBytesBothStore(t *testing.T) {
	repoRoot, _ := setupStore(t)
	raw := []byte("assistant: no findings\n")

	first, err := Capture(repoRoot, testRootSHA, raw,
		CaptureMeta{SessionID: "sess-twins", Kind: "native", AgentID: "agent-a",
			LineageSource: "hook", SpawnAttribution: "unattributed"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := Capture(repoRoot, testRootSHA, raw,
		CaptureMeta{SessionID: "sess-twins", Kind: "native", AgentID: "agent-b",
			LineageSource: "hook", SpawnAttribution: "unattributed"})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !second.Wrote {
		t.Fatal("a second sub-agent with identical bytes must get its own record, not collapse into the first")
	}
	if second.Record.Path == first.Record.Path {
		t.Fatal("two sub-agents must not share a record path")
	}
	recs, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records for two sub-agents, got %d", len(recs))
	}
}

// TestSubagentCaptureIdempotentOnSourceSHA is the other half of the pair: the
// widened key must not be so wide that re-draining the same staged transcript
// writes a duplicate.
func TestSubagentCaptureIdempotentOnSourceSHA(t *testing.T) {
	repoRoot, _ := setupStore(t)
	raw := []byte("assistant: no findings\n")
	meta := CaptureMeta{SessionID: "sess-idemsub", Kind: "native", AgentID: "agent-idem",
		LineageSource: "hook", SpawnAttribution: "unattributed"}

	first, err := Capture(repoRoot, testRootSHA, raw, meta)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := Capture(repoRoot, testRootSHA, raw, meta)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.Wrote {
		t.Error("re-capturing one sub-agent's identical source must be a no-op")
	}
	if second.Record.Path != first.Record.Path {
		t.Errorf("idempotent capture returned %q, want %q", second.Record.Path, first.Record.Path)
	}
}

// TestReadResolvesFilenameThenAgentThenSession pins the three-step resolution.
// The session step prefers the MAIN THREAD record even when a sub-agent's is
// newer: a reader who asks for a session by its id is asking for its spine.
func TestReadResolvesFilenameThenAgentThenSession(t *testing.T) {
	repoRoot, _ := setupStore(t)

	mainRes, err := Capture(repoRoot, testRootSHA, []byte("user: the spine\n"),
		CaptureMeta{SessionID: "sess-res", Kind: "native"})
	if err != nil {
		t.Fatal(err)
	}
	subRes, err := Capture(repoRoot, testRootSHA, []byte("assistant: a branch\n"),
		CaptureMeta{SessionID: "sess-res", Kind: "native", AgentID: "agent-res",
			LineageSource: "hook", SpawnAttribution: "unattributed"})
	if err != nil {
		t.Fatal(err)
	}

	if rec, _, err := Read(repoRoot, testRootSHA, filepath.Base(subRes.Record.Path)); err != nil ||
		rec.AgentID != "agent-res" {
		t.Errorf("step 1 (record filename) resolved to %+v (%v)", rec, err)
	}
	if rec, _, err := Read(repoRoot, testRootSHA, "agent-res"); err != nil || rec.AgentID != "agent-res" {
		t.Errorf("step 2 (agent id) resolved to %+v (%v)", rec, err)
	}
	rec, _, err := Read(repoRoot, testRootSHA, "sess-res")
	if err != nil {
		t.Fatalf("step 3 (session id): %v", err)
	}
	if rec.AgentID != "" || rec.Path != mainRes.Record.Path {
		t.Errorf("step 3 must prefer the main-thread record even though the sub-agent's is newer; got %+v", rec)
	}
}

// TestListForSessionReturnsMainThreadAndEverySubagent is ac-3's read side: one
// session identifier, the whole session. Main thread first, because it is what
// makes the rest legible.
func TestListForSessionReturnsMainThreadAndEverySubagent(t *testing.T) {
	repoRoot, _ := setupStore(t)

	if _, err := Capture(repoRoot, testRootSHA, []byte("user: spine\n"),
		CaptureMeta{SessionID: "sess-all", Kind: "native"}); err != nil {
		t.Fatal(err)
	}
	for _, agent := range []string{"agent-1", "agent-2"} {
		if _, err := Capture(repoRoot, testRootSHA, []byte("assistant: "+agent+"\n"), CaptureMeta{
			SessionID: "sess-all", Kind: "native", AgentID: agent, AgentType: "reviewer",
			SpawnDepth: 1, LineageSource: "hook", SpawnAttribution: "sidecar",
		}); err != nil {
			t.Fatal(err)
		}
	}
	// A record for a different session must not be swept in.
	if _, err := Capture(repoRoot, testRootSHA, []byte("user: elsewhere\n"),
		CaptureMeta{SessionID: "sess-other", Kind: "native"}); err != nil {
		t.Fatal(err)
	}

	got, err := ListForSession(repoRoot, testRootSHA, "sess-all")
	if err != nil {
		t.Fatalf("ListForSession: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 records for sess-all, got %d", len(got))
	}
	if got[0].AgentID != "" {
		t.Errorf("main thread must come first, got %q", got[0].AgentID)
	}
	seen := map[string]bool{}
	for _, r := range got {
		if r.SessionID != "sess-all" {
			t.Errorf("a record from %q leaked into the session listing", r.SessionID)
		}
		seen[r.AgentID] = true
	}
	for _, want := range []string{"", "agent-1", "agent-2"} {
		if !seen[want] {
			t.Errorf("agent %q missing from the session listing", want)
		}
	}
}

// TestCaptureRejectsAMalformedLineageScalar holds the boundary. The frontmatter
// is one scalar per line and the agent id is embedded in a record filename, so
// a newline or a path separator in an externally supplied field is a structural
// injection, not a cosmetic problem.
func TestCaptureRejectsAMalformedLineageScalar(t *testing.T) {
	repoRoot, _ := setupStore(t)
	base := CaptureMeta{SessionID: "sess-bad", Kind: "native"}

	cases := []struct {
		name string
		mut  func(*CaptureMeta)
	}{
		{"path-traversal agent id", func(m *CaptureMeta) { m.AgentID = "../evil" }},
		{"path-traversal parent agent id", func(m *CaptureMeta) { m.ParentAgentID = "../evil" }},
		{"newline in agent type", func(m *CaptureMeta) { m.AgentType = "ok\nsession_id: hijacked" }},
		{"carriage return in tool use id", func(m *CaptureMeta) { m.SpawnToolUseID = "ok\rmore" }},
		{"unknown lineage source", func(m *CaptureMeta) { m.LineageSource = "guessed" }},
		{"negative spawn depth", func(m *CaptureMeta) { m.SpawnDepth = -1 }},
		{"spawn attribution on the main thread", func(m *CaptureMeta) { m.SpawnAttribution = "sidecar" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta := base
			c.mut(&meta)
			if _, err := Capture(repoRoot, testRootSHA, []byte("x\n"), meta); err == nil {
				t.Errorf("expected rejection for %s", c.name)
			}
		})
	}
}

// TestRecordFilenameIsStableForTheSameInputs guards the one thing migration and
// every held path depend on: the filename scheme is not re-derived per call.
func TestRecordFilenameIsStableForTheSameInputs(t *testing.T) {
	at := time.Date(2026, 9, 9, 6, 24, 22, 51000000, time.UTC)
	if got, want := recordFilename(at, "sess", ""), "20260909T062422.051000000Z-sess.md"; got != want {
		t.Errorf("recordFilename(main) = %q, want %q", got, want)
	}
	if got, want := recordFilename(at, "sess", "ag1"), "20260909T062422.051000000Z-sess-agent-ag1.md"; got != want {
		t.Errorf("recordFilename(sub) = %q, want %q", got, want)
	}
}

// bodyOf reads a record file straight off disk and returns its body.
func bodyOf(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := parseRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// TestASecondStopSupersedesTheFirstRecord holds the unit of the store: it is
// one (session, agent), not one transcript. The harness fires its stop event on
// EVERY stop, so an agent resumed with a follow-up message stops again carrying
// a longer transcript that begins with the one already stored. The source sha
// differs, so idempotency cannot see it — and without supersession every reader
// of the set gets the same agent twice.
func TestASecondStopSupersedesTheFirstRecord(t *testing.T) {
	repoRoot, _ := setupStore(t)
	meta := CaptureMeta{
		SessionID: "sess-twostop", Kind: "native", AgentID: "agent-twostop",
		SpawnDepth: 1, SpawnAttribution: "sidecar", LineageSource: "hook",
	}
	firstStop := []byte(`{"role":"user","text":"first prompt"}` + "\n")
	secondStop := append(append([]byte{}, firstStop...),
		[]byte(`{"role":"user","text":"follow-up prompt"}`+"\n")...)

	first, err := Capture(repoRoot, testRootSHA, firstStop, meta)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Capture(repoRoot, testRootSHA, secondStop, meta)
	if err != nil {
		t.Fatalf("the second stop must be storable: %v", err)
	}
	if !second.Wrote {
		t.Fatal("a longer second stop must be stored, not discarded as a duplicate")
	}
	if second.Superseded == nil || second.Superseded.Path != first.Record.Path {
		t.Errorf("the second stop must report what it superseded, got %+v", second.Superseded)
	}
	if _, err := os.Stat(first.Record.Path); !os.IsNotExist(err) {
		t.Errorf("the superseded record is still on disk (%v); the agent now appears twice", err)
	}
	recs, err := ListForSession(repoRoot, testRootSHA, "sess-twostop")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("one agent that stopped twice must leave ONE live record, got %d", len(recs))
	}
	if !strings.Contains(bodyOf(t, recs[0].Path), "follow-up prompt") {
		t.Error("the live record must hold the longer transcript")
	}
}

// TestAShorterRearrivalDoesNotWriteASecondRecord is the other direction: a
// re-drain of an older, truncated staged copy after the fuller one is already
// stored must be a no-op, not a second record.
func TestAShorterRearrivalDoesNotWriteASecondRecord(t *testing.T) {
	repoRoot, _ := setupStore(t)
	meta := CaptureMeta{
		SessionID: "sess-short", Kind: "native", AgentID: "agent-short",
		SpawnDepth: 1, SpawnAttribution: "sidecar", LineageSource: "hook",
	}
	short := []byte(`{"role":"user","text":"first prompt"}` + "\n")
	long := append(append([]byte{}, short...), []byte(`{"role":"user","text":"more"}`+"\n")...)

	full, err := Capture(repoRoot, testRootSHA, long, meta)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Capture(repoRoot, testRootSHA, short, meta)
	if err != nil {
		t.Fatal(err)
	}
	if again.Wrote {
		t.Error("a truncated re-arrival must not write a second record")
	}
	if again.Record.Path != full.Record.Path {
		t.Errorf("the no-op must return the stored record, got %q want %q", again.Record.Path, full.Record.Path)
	}
	recs, err := ListForSession(repoRoot, testRootSHA, "sess-short")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 live record, got %d", len(recs))
	}
}

// TestTheMainThreadSpineIsSupersededToo pins the identical rule on the main
// thread. Sessions restart and end more than once — the store this was designed
// against already carried several sessions with more than one main-thread
// record — and a double-counted spine makes the branches unreadable.
func TestTheMainThreadSpineIsSupersededToo(t *testing.T) {
	repoRoot, _ := setupStore(t)
	meta := CaptureMeta{SessionID: "sess-spine", Kind: "native"}
	firstEnd := []byte(`{"role":"user","text":"turn one"}` + "\n")
	secondEnd := append(append([]byte{}, firstEnd...), []byte(`{"role":"user","text":"turn two"}`+"\n")...)

	if _, err := Capture(repoRoot, testRootSHA, firstEnd, meta); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(repoRoot, testRootSHA, secondEnd, meta); err != nil {
		t.Fatal(err)
	}
	recs, err := ListForSession(repoRoot, testRootSHA, "sess-spine")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("one session that ended twice must leave ONE main-thread record, got %d", len(recs))
	}
	if !strings.Contains(bodyOf(t, recs[0].Path), "turn two") {
		t.Error("the live spine must hold the longer transcript")
	}
}

// TestDivergentTranscriptsForOneAgentBothStore bounds supersession. Two bodies
// where neither is a prefix of the other are not two snapshots of one run, and
// collapsing them would discard bytes abcd holds the only copy of. They stay
// side by side; only the prefix relation supersedes.
func TestDivergentTranscriptsForOneAgentBothStore(t *testing.T) {
	repoRoot, _ := setupStore(t)
	meta := CaptureMeta{
		SessionID: "sess-diverge", Kind: "native", AgentID: "agent-diverge",
		SpawnDepth: 1, SpawnAttribution: "sidecar", LineageSource: "hook",
	}
	if _, err := Capture(repoRoot, testRootSHA, []byte("one thing\n"), meta); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(repoRoot, testRootSHA, []byte("another thing entirely\n"), meta); err != nil {
		t.Fatal(err)
	}
	recs, err := ListForSession(repoRoot, testRootSHA, "sess-diverge")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("divergent bodies must both survive, got %d record(s)", len(recs))
	}
}

// TestUnknownSpawnIsDistinguishableFromNoParent closes the two-empties defect.
// An empty parent_agent_id used to mean "the main thread spawned it", but a
// hook-sourced record with no harness sidecar has it empty too — and an empty
// spawn_depth with it. A reader could not tell a genuine depth-1 child of the
// main thread from a record whose lineage was simply never recovered.
// spawn_attribution names the rung of the ladder that answered, so the two
// records read differently.
func TestUnknownSpawnIsDistinguishableFromNoParent(t *testing.T) {
	repoRoot, _ := setupStore(t)

	if _, err := Capture(repoRoot, testRootSHA, []byte("assistant: known\n"), CaptureMeta{
		SessionID: "sess-attrib", Kind: "native", AgentID: "agent-known",
		SpawnDepth: 1, SpawnAttribution: "sidecar", LineageSource: "hook",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(repoRoot, testRootSHA, []byte("assistant: unknown\n"), CaptureMeta{
		SessionID: "sess-attrib", Kind: "native", AgentID: "agent-unknown",
		SpawnAttribution: "unattributed", LineageSource: "hook",
	}); err != nil {
		t.Fatal(err)
	}

	known, _, err := Read(repoRoot, testRootSHA, "agent-known")
	if err != nil {
		t.Fatal(err)
	}
	unknown, _, err := Read(repoRoot, testRootSHA, "agent-unknown")
	if err != nil {
		t.Fatal(err)
	}
	if known.ParentAgentID != "" || unknown.ParentAgentID != "" {
		t.Fatal("both fixtures are meant to carry an empty parent_agent_id")
	}
	if known.SpawnAttribution != "sidecar" {
		t.Errorf("a child of the main thread must record that its spawn WAS attributed, got %q",
			known.SpawnAttribution)
	}
	if unknown.SpawnAttribution != "unattributed" {
		t.Errorf("an unrecovered spawn must say so, got %q", unknown.SpawnAttribution)
	}
	if known.SpawnAttribution == unknown.SpawnAttribution {
		t.Error("no parent and unknown parent must not read identically")
	}
}

// TestSubagentCaptureRequiresASpawnAttribution makes the ambiguity structurally
// impossible for anything written from now on: a sub-agent record cannot be
// written without saying which rung placed it, and a record that claims nothing
// placed it cannot also name a parent.
func TestSubagentCaptureRequiresASpawnAttribution(t *testing.T) {
	repoRoot, _ := setupStore(t)
	cases := []struct {
		name string
		meta CaptureMeta
	}{
		{"sub-agent with no attribution", CaptureMeta{
			SessionID: "sess-attreq", Kind: "native", AgentID: "agent-x", LineageSource: "hook"}},
		{"unknown attribution rung", CaptureMeta{
			SessionID: "sess-attreq", Kind: "native", AgentID: "agent-x",
			SpawnAttribution: "guessed", LineageSource: "hook"}},
		{"unattributed but names a parent", CaptureMeta{
			SessionID: "sess-attreq", Kind: "native", AgentID: "agent-x",
			ParentAgentID: "agent-p", SpawnAttribution: "unattributed", LineageSource: "hook"}},
		{"unattributed but carries a depth", CaptureMeta{
			SessionID: "sess-attreq", Kind: "native", AgentID: "agent-x",
			SpawnDepth: 2, SpawnAttribution: "unattributed", LineageSource: "hook"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Capture(repoRoot, testRootSHA, []byte("x\n"), c.meta); err == nil {
				t.Errorf("expected rejection for %s", c.name)
			}
		})
	}
}
