package history

import (
	"os"
	"strings"
	"testing"
)

// compositeRecord is a record as the pre-lineage binary wrote it: the parent
// session TRUNCATED to a prefix, concatenated with the agent id, in the one
// session_id field. The body is the harness's own line-delimited transcript,
// which still carries the full session identifier on every line — the only
// place the truncated half survives.
func compositeRecord(stored, fullSession string) string {
	return strings.Join([]string{
		"---",
		"schema: 1",
		"session_id: " + stored,
		"root_commit: " + testRootSHA,
		"captured_at: 2026-01-01T00:00:00Z",
		"source_kind: native",
		"source_sha256: " + strings.Repeat("a", 64),
		"redacted_secrets: 0",
		"redacted_home_paths: 0",
		"---",
		`{"type":"user","sessionId":"` + fullSession + `","cwd":"~/work"}`,
		`{"type":"assistant","sessionId":"` + fullSession + `","cwd":"~/work"}`,
		"",
	}, "\n")
}

// TestMigrateRecoversTheParentSessionFromTheBody is the migration's whole
// point: the stored prefix is lossy, so the full parent session id has to come
// out of the record's own body, and the prefix is only the check that the two
// describe the same session.
func TestMigrateRecoversTheParentSessionFromTheBody(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-5a9221e2--agent-acf07c33.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))

	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Migrated) != 1 || len(res.Refused) != 0 {
		t.Fatalf("want 1 migrated and 0 refused, got %d/%d (%+v)", len(res.Migrated), len(res.Refused), res)
	}
	if res.Migrated[0].SessionID != full {
		t.Errorf("recovered session id = %q, want the full id %q", res.Migrated[0].SessionID, full)
	}
	if res.Migrated[0].AgentID != "acf07c33" {
		t.Errorf("agent id = %q, want acf07c33", res.Migrated[0].AgentID)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"session_id: " + full,
		"agent_id: acf07c33",
		"lineage_source: migrated",
		"source_sha256: " + strings.Repeat("a", 64),
	} {
		if !strings.Contains(string(onDisk), want) {
			t.Errorf("migrated record is missing %q:\n%s", want, onDisk)
		}
	}
	if strings.Contains(string(onDisk), "--agent-") {
		t.Errorf("the composite id survived in the frontmatter:\n%s", onDisk)
	}
}

// TestMigrateReportsWithoutApplying pins the default: the store holds the only
// copy of these records, so a run that was not asked to write must not write.
func TestMigrateReportsWithoutApplying(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-5a9221e2--agent-acf07c33.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if res.Applied {
		t.Error("a report-mode run must not report itself as applied")
	}
	if len(res.Migrated) != 1 {
		t.Fatalf("report mode must still say what it WOULD do, got %+v", res)
	}
	if res.Migrated[0].Wrote {
		t.Error("report mode must not claim a write")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("report mode rewrote the record:\n%s", after)
	}
}

// TestMigrateRefusesABodyThatDisagreesWithThePrefix holds the check that keeps
// a lossy truncation from becoming a guess. A body whose session identifier
// does not begin with the stored prefix is a different session; the record is
// left untouched and reported.
func TestMigrateRefusesABodyThatDisagreesWithThePrefix(t *testing.T) {
	repoRoot, home := setupStore(t)
	path := planted(t, home, "20260101T000000.000000000Z-5a9221e2--agent-acf07c33.md",
		compositeRecord("5a9221e2--agent-acf07c33", "deadbeef-0000-0000-0000-000000000000"))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Migrated) != 0 || len(res.Refused) != 1 {
		t.Fatalf("want 0 migrated and 1 refused, got %d/%d", len(res.Migrated), len(res.Refused))
	}
	if !strings.Contains(res.Refused[0].Refused, "prefix") {
		t.Errorf("the refusal must say the body disagreed with the prefix, got %q", res.Refused[0].Refused)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("a refused record must be left untouched:\n%s", after)
	}
}

// TestMigrateIsARepeatableNoOp: a record that already carries an agent id has
// been migrated, and running again must neither rewrite it nor report it.
func TestMigrateIsARepeatableNoOp(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	planted(t, home, "20260101T000000.000000000Z-5a9221e2--agent-acf07c33.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))

	if _, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true}); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true})
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(res.Migrated) != 0 || len(res.Refused) != 0 {
		t.Errorf("a second run must find nothing to do, got %+v", res)
	}
}

// TestMigrateKeepsTheFilenameAndTheBody pins the two things the migration must
// not touch: the filename a reader may already hold, and the redacted body.
func TestMigrateKeepsTheFilenameAndTheBody(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	const name = "20260101T000000.000000000Z-5a9221e2--agent-acf07c33.md"
	path := planted(t, home, name, compositeRecord("5a9221e2--agent-acf07c33", full))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bodyBefore := strings.SplitN(string(before), "\n---\n", 2)[1]

	if _, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the record was renamed or removed: %v", err)
	}
	bodyAfter := strings.SplitN(string(after), "\n---\n", 2)[1]
	if bodyBefore != bodyAfter {
		t.Errorf("the body changed:\nbefore:\n%s\nafter:\n%s", bodyBefore, bodyAfter)
	}
}

// TestMigrateParsesTheWorkflowShapedComposite is the shape the spec does not
// describe. The store also holds `<prefix>--wf_<id>--agent-<agent>`, written
// for an agent launched inside a workflow: 71 of this machine's 176 composite
// records carry it. Splitting on the FIRST "--" would read `wf_...` as part of
// the session prefix and refuse every one of them.
func TestMigrateParsesTheWorkflowShapedComposite(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "0b80a953-a8a4-409e-ac1e-1238f37bbe35"
	planted(t, home, "20260101T000000.000000000Z-wf.md",
		compositeRecord("0b80a953--wf_3fce0699-ef2--agent-a11f6a0d", full))

	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Migrated) != 1 {
		t.Fatalf("want the workflow-shaped composite migrated, got %+v", res)
	}
	got := res.Migrated[0]
	if got.SessionID != full || got.AgentID != "a11f6a0d" {
		t.Errorf("workflow composite parsed as session=%q agent=%q, want %q / a11f6a0d", got.SessionID, got.AgentID, full)
	}
	if got.ExtraSegment != "wf_3fce0699-ef2" {
		t.Errorf("the segment between the session and the agent must be reported, got %q", got.ExtraSegment)
	}
}

// TestMigrateSaysLineageIsUnknownWithoutASidecar: with nothing to enrich from,
// the record must say so through the attribution rung rather than leave an
// empty parent that reads as "the main thread spawned it".
func TestMigrateSaysLineageIsUnknownWithoutASidecar(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-a.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))

	if _, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "spawn_attribution: unattributed") {
		t.Errorf("a record with no recoverable lineage must be stamped unattributed:\n%s", onDisk)
	}
	if strings.Contains(string(onDisk), "agent_type:") || strings.Contains(string(onDisk), "spawn_depth:") {
		t.Errorf("nothing was recoverable, so nothing may be claimed:\n%s", onDisk)
	}
}

// TestMigrateEnrichesFromTheHarnessSidecar is the correction to the spec, which
// says agent_type and spawn_depth "were never held" anywhere. They were: the
// harness's per-agent metadata file is still on disk for the great majority of
// these records, keyed by agent id. Where the lookup answers, the migrated
// record carries the type, the depth, the spawning tool call and — at depth two
// and deeper — the parent agent.
func TestMigrateEnrichesFromTheHarnessSidecar(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-a.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))

	var asked LineageRef
	res, err := Migrate(testRootSHA, MigrateOptions{
		RepoRoot: repoRoot,
		Apply:    true,
		Lineage: func(ref LineageRef) (HarnessLineage, bool) {
			asked = ref
			return HarnessLineage{
				AgentType:      "reviewer",
				ParentAgentID:  "b0b0b0b0",
				SpawnDepth:     2,
				SpawnToolUseID: "toolu_01AdZjRn",
			}, true
		},
	})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if asked.SessionID != full || asked.AgentID != "acf07c33" {
		t.Errorf("the lookup was asked for %+v, want the recovered session and the agent id", asked)
	}
	if len(res.Migrated) != 1 || res.Migrated[0].SpawnAttribution != "sidecar" {
		t.Fatalf("an answered lookup must be recorded as the sidecar rung, got %+v", res)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"agent_type: reviewer",
		"parent_agent_id: b0b0b0b0",
		"spawn_depth: 2",
		"spawn_tool_use_id: toolu_01AdZjRn",
		"spawn_attribution: sidecar",
		"lineage_source: migrated",
	} {
		if !strings.Contains(string(onDisk), want) {
			t.Errorf("enriched record is missing %q:\n%s", want, onDisk)
		}
	}
}

// TestMigrateRedactsTheLineageItLearns closes the bypass the schema opened.
// Everything a lookup returns is externally supplied and lands in frontmatter,
// which is never scanned on the read path — so it goes through the same
// sanitise-then-verify pass as a captured body.
func TestMigrateRedactsTheLineageItLearns(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-a.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))

	if _, err := Migrate(testRootSHA, MigrateOptions{
		RepoRoot: repoRoot,
		Apply:    true,
		Lineage: func(LineageRef) (HarnessLineage, bool) {
			return HarnessLineage{AgentType: "agent in " + home + "/secrets", SpawnDepth: 1}, true
		},
	}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(onDisk), home) {
		t.Errorf("a lineage scalar carried the caller's home into frontmatter unredacted:\n%s", onDisk)
	}
}

// TestMigrateLeavesMainThreadRecordsAlone: a record whose session id is not a
// composite is not this verb's business.
func TestMigrateLeavesMainThreadRecordsAlone(t *testing.T) {
	repoRoot, home := setupStore(t)
	path := planted(t, home, "20260101T000000.000000000Z-plain.md",
		compositeRecord("5a9221e2-fa77-4be5-84d3-779199c449d7", "5a9221e2-fa77-4be5-84d3-779199c449d7"))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Migrated) != 0 || len(res.Refused) != 0 {
		t.Errorf("a main-thread record must be untouched, got %+v", res)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("a main-thread record was rewritten")
	}
}

// --------------------------------------------------------------------------
// Lineage framing (security review, finding 2)
// --------------------------------------------------------------------------

// TestMigrateValidatesTheLineageBeforeItFramesIt. Everything a lineage lookup
// returns is externally supplied, and migrate feeds it to frameLineage, which
// writes one scalar per line and splits the block back off by position. A
// scalar carrying line breaks therefore does not corrupt the split — it
// RE-AIMS it: the lookup below places its own frame marker at the index the
// splitter expects, so the split SUCCEEDS and writes attacker-chosen values
// into agent_type, spawn_tool_use_id, lineage_source and spawn_attribution
// while the real ones are discarded and the run reports success. Capture
// validates before it frames; migrate must too.
func TestMigrateValidatesTheLineageBeforeItFramesIt(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-a.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Migrate(testRootSHA, MigrateOptions{
		RepoRoot: repoRoot,
		Apply:    true,
		Lineage: func(LineageRef) (HarnessLineage, bool) {
			return HarnessLineage{
				// One scalar, five lines: it fills agent_type and every
				// scalar after it, and the tool-use id then supplies the
				// frame marker at exactly the offset unframeLineage checks.
				AgentType:      "forgedtype\nforgedtool\nhook\ntranscript\nforgedproject",
				SpawnToolUseID: lineageFrameEnd,
				SpawnDepth:     1,
			}, true
		},
	})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Migrated) != 0 || len(res.Refused) != 1 {
		t.Fatalf("a lineage scalar carrying line breaks must be refused, not migrated; got %d migrated / %d refused (%+v)",
			len(res.Migrated), len(res.Refused), res)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("a refused migration must leave the record untouched; it now reads:\n%s", after)
	}
	for _, forged := range []string{"lineage_source: hook", "spawn_attribution: transcript", "forgedtool", "forgedproject"} {
		if strings.Contains(string(after), forged) {
			t.Errorf("the record carries the forged value %q:\n%s", forged, after)
		}
	}
}

// TestFrameLineageRefusesAScalarWithALineBreak puts the guarantee in the layer
// that holds it. The front door sanitises these scalars today, so the exploit
// above is unreachable through the CLI — but the invariant store.go states is
// the STORE's, and a core primitive that silently accepts a scalar it will
// then mis-split is one caller away from being wrong again.
func TestFrameLineageRefusesAScalarWithALineBreak(t *testing.T) {
	for _, tc := range []struct {
		name string
		meta CaptureMeta
	}{
		{"newline in agent type", CaptureMeta{AgentID: "a", AgentType: "x\ny"}},
		{"carriage return in the tool use id", CaptureMeta{AgentID: "a", SpawnToolUseID: "x\ry"}},
		{"newline in the agent id", CaptureMeta{AgentID: "a\n" + lineageFrameEnd}},
		{"newline in the adopted project", CaptureMeta{AgentID: "a", AdoptedProject: "x\ny"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := frameLineage(tc.meta, nil); err == nil {
				t.Errorf("frameLineage accepted a scalar containing a line break; the frame it writes is one scalar per line, so the split it feeds is re-aimable by the value")
			}
		})
	}
	if _, err := frameLineage(CaptureMeta{AgentID: "a", AgentType: "reviewer"}, []byte("body\n")); err != nil {
		t.Errorf("a clean lineage must still frame: %v", err)
	}
}
