package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// otherRootSHA is a second repository's store key: a transcript that resolves
// to it is not this destination's to store.
const otherRootSHA = "cccccccccccccccccccccccccccccccccccccccc"

// fakeRepos substitutes a table for the cwd -> repository detection pass, so
// these tests need no git repositories and no filesystem outside t.TempDir.
// The behaviour under test is the placement POLICY — session before file, one
// owner or none — not the detection primitive, which has its own tests.
//
// The DESTINATION's own root belongs in the table too: Ingest proves the pair
// (root, key) names one repository (iss-2609091911060345), so a table that maps
// only the transcripts' recorded directories describes a destination Ingest
// refuses before it reads a source.
func fakeRepos(t *testing.T, table map[string]string) {
	t.Helper()
	prior := resolveRootSHA
	resolveRootSHA = func(cwd string) (string, bool) {
		sha, ok := table[cwd]
		return sha, ok
	}
	t.Cleanup(func() { resolveRootSHA = prior })
}

// transcriptFile writes a line-delimited transcript into dir. agentID empty
// makes it a main-thread transcript.
func transcriptFile(t *testing.T, dir, name, sessionID, agentID, cwd string, extraCwd ...string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	agent := ""
	if agentID != "" {
		agent = `"agentId":"` + agentID + `",`
	}
	lines := []string{
		`{"type":"user",` + agent + `"sessionId":"` + sessionID + `","cwd":"` + cwd + `"}`,
		`{"type":"assistant",` + agent + `"sessionId":"` + sessionID + `","cwd":"` + cwd + `"}`,
	}
	for _, c := range extraCwd {
		lines = append(lines, `{"type":"user",`+agent+`"sessionId":"`+sessionID+`","cwd":"`+c+`"}`)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestIngestRefusesWithoutAnExplicitDestination is the seam itself. The
// destination cannot be defaulted, because a defaulted destination is a
// transcript redacted under whichever repository the operator happened to be
// standing in.
func TestIngestRefusesWithoutAnExplicitDestination(t *testing.T) {
	setupStore(t)
	_, err := Ingest(Destination{RootSHA: testRootSHA}, []string{t.TempDir()}, IngestOptions{})
	if err == nil {
		t.Fatal("Ingest must refuse a destination with no repository root")
	}
	if !strings.Contains(err.Error(), "working directory") {
		t.Errorf("the refusal must say why the working directory is not the authority, got %q", err)
	}
}

// TestIngestStoresOnlyWhatTheDestinationOwns: a transcript whose recorded cwd
// resolves to another repository is skipped and NAMED, never stored here.
func TestIngestStoresOnlyWhatTheDestinationOwns(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA, "/mine": testRootSHA, "/theirs": otherRootSHA})
	transcriptFile(t, filepath.Join(src, "proj-a"), "s1.jsonl", "sess-mine", "", "/mine")
	transcriptFile(t, filepath.Join(src, "proj-b"), "s2.jsonl", "sess-theirs", "", "/theirs")

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Captured) != 1 || res.Captured[0].SessionID != "sess-mine" {
		t.Fatalf("want only the destination's own transcript captured, got %+v", res.Captured)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Reason != SkipOwnedElsewhere {
		t.Fatalf("want one owned-elsewhere skip, got %+v", res.Skipped)
	}
	if res.Skipped[0].RootSHA != otherRootSHA {
		t.Errorf("the skip must name the owning repository by SHA, got %q", res.Skipped[0].RootSHA)
	}
	records, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Errorf("the store must hold exactly the one owned transcript, got %d", len(records))
	}
}

// TestIngestPlacesTheSessionBeforeTheFile is iss-2609090723023943. A sub-agent
// handed its own worktree records that worktree as its cwd and the harness
// removes it on stop, so the directory is gone by ingest time — while the
// parent transcript, one file away, resolves cleanly. Resolving per file
// orphans exactly those agents.
func TestIngestPlacesTheSessionBeforeTheFile(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	// Only the parent's directory resolves; the worktree is gone.
	fakeRepos(t, map[string]string{repoRoot: testRootSHA, "/mine": testRootSHA})
	proj := filepath.Join(src, "proj-a")
	transcriptFile(t, proj, "sess-w.jsonl", "sess-w", "", "/mine")
	transcriptFile(t, filepath.Join(proj, "subagents"), "agent-a1.jsonl", "sess-w", "a1", "/gone-worktree")

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Orphans) != 0 {
		t.Fatalf("a sub-agent whose SESSION resolves is not an orphan, got %+v", res.Orphans)
	}
	if len(res.Captured) != 2 {
		t.Fatalf("want the session and its sub-agent captured, got %+v", res.Captured)
	}
	var sub *Ingested
	for i := range res.Captured {
		if res.Captured[i].AgentID == "a1" {
			sub = &res.Captured[i]
		}
	}
	if sub == nil {
		t.Fatal("the worktree-isolated sub-agent was not captured")
	}
	if sub.Via != "session-cwd" {
		t.Errorf("the sub-agent must be placed by its session, got via=%q", sub.Via)
	}
	if sub.SessionID != "sess-w" {
		t.Errorf("a sub-agent record carries the FULL spawning session id, got %q", sub.SessionID)
	}
}

// TestIngestPlacesASessionFromTheStoreWhenNoDirectorySurvives: the second
// session-level rung. Every directory the session recorded is gone, but this
// machine already noted which store the session belongs to.
func TestIngestPlacesASessionFromTheStoreWhenNoDirectorySurvives(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA})
	if err := NoteSessionRepo(repoRoot, testRootSHA, "sess-noted"); err != nil {
		t.Fatal(err)
	}
	transcriptFile(t, filepath.Join(src, "proj-a"), "agent-a1.jsonl", "sess-noted", "a1", "/gone")

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Captured) != 1 || res.Captured[0].Via != "store" {
		t.Fatalf("want the session placed from the store, got captured=%+v orphans=%+v", res.Captured, res.Orphans)
	}
}

// TestIngestIgnoresAndReportsOrphans: a transcript whose repository cannot be
// found is listed with the project name as given and its recorded directory
// home-redacted, and NOTHING is written.
func TestIngestIgnoresAndReportsOrphans(t *testing.T) {
	repoRoot, home := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA})
	transcriptFile(t, filepath.Join(src, "some-project"), "s1.jsonl", "sess-orphan", "", home+"/gone")

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Captured) != 0 {
		t.Fatalf("an orphan must not be stored, got %+v", res.Captured)
	}
	if len(res.Orphans) != 1 {
		t.Fatalf("want one reported orphan, got %+v", res.Orphans)
	}
	if res.Orphans[0].Project != "some-project" {
		t.Errorf("the orphan must be reported under its project directory name as given, got %q", res.Orphans[0].Project)
	}
	if strings.Contains(res.Orphans[0].Cwd, home) {
		t.Errorf("the reported working directory must be home-redacted, got %q", res.Orphans[0].Cwd)
	}
	records, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("the store must be untouched by an ignored orphan, got %d records", len(records))
	}
}

// TestIngestAdoptsOnlyProjectsNamedByTheDestination: adoption is opt-in and by
// name, and the adoption is stamped on the record rather than only reported.
func TestIngestAdoptsOnlyProjectsNamedByTheDestination(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA})
	transcriptFile(t, filepath.Join(src, "claimed"), "s1.jsonl", "sess-claimed", "", "/gone")
	transcriptFile(t, filepath.Join(src, "unclaimed"), "s2.jsonl", "sess-unclaimed", "", "/gone")

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src},
		IngestOptions{Adopt: []string{"claimed"}})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Captured) != 1 || res.Captured[0].SessionID != "sess-claimed" {
		t.Fatalf("want only the claimed project adopted, got %+v", res.Captured)
	}
	if len(res.Orphans) != 1 || res.Orphans[0].Project != "unclaimed" {
		t.Fatalf("the unclaimed project stays an orphan, got %+v", res.Orphans)
	}
	onDisk, err := os.ReadFile(res.Captured[0].RecordPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"lineage_source: ingest", "adopted_project: claimed"} {
		if !strings.Contains(string(onDisk), want) {
			t.Errorf("an adopted record must carry %q:\n%s", want, onDisk)
		}
	}
}

// TestIngestIsIdempotent: ingesting the same material twice adds nothing.
func TestIngestIsIdempotent(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA, "/mine": testRootSHA})
	transcriptFile(t, filepath.Join(src, "proj-a"), "s1.jsonl", "sess-idem", "", "/mine")
	dest := Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}

	if _, err := Ingest(dest, []string{src}, IngestOptions{}); err != nil {
		t.Fatalf("first Ingest: %v", err)
	}
	first, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Ingest(dest, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("second Ingest: %v", err)
	}
	if len(res.Captured) != 1 || res.Captured[0].Wrote {
		t.Errorf("a second ingest of the same bytes must report the file and write nothing, got %+v", res.Captured)
	}
	second, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != len(first) {
		t.Errorf("a second ingest added records: %d -> %d", len(first), len(second))
	}
}

// TestIngestRefusesAnAmbiguousOwner: a transcript recorded in two repositories
// is never split between stores.
func TestIngestRefusesAnAmbiguousOwner(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA, "/mine": testRootSHA, "/theirs": otherRootSHA})
	transcriptFile(t, filepath.Join(src, "proj-a"), "s1.jsonl", "sess-two", "", "/mine", "/theirs")

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Captured) != 0 {
		t.Fatalf("an ambiguously owned transcript must not be stored, got %+v", res.Captured)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Reason != SkipAmbiguousOwner {
		t.Fatalf("want one ambiguous-owner skip, got %+v", res.Skipped)
	}
}

// TestIngestRedactsUnderTheDestinationsOwnConfiguration is the reason the
// destination is an operand. The repository the transcript RAN in is not the
// one whose scanner configuration applies; the repository it is stored in is.
func TestIngestRedactsUnderTheDestinationsOwnConfiguration(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if err := os.MkdirAll(filepath.Join(repoRoot, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The DESTINATION declares a detector for a token shape only it knows about.
	cfg := `{"patterns":{"house_token":{"regex":"HOUSE-[0-9]{6}","kind":"house_token","label":"house token","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(repoRoot, ".abcd", "config", "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA, "/mine": testRootSHA})
	proj := filepath.Join(src, "proj-a")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","sessionId":"sess-cfg","cwd":"/mine","text":"HOUSE-123456"}` + "\n"
	if err := os.WriteFile(filepath.Join(proj, "s1.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("want the transcript captured, got %+v", res)
	}
	onDisk, err := os.ReadFile(res.Captured[0].RecordPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(onDisk), "HOUSE-123456") {
		t.Errorf("the destination's own detector did not govern its own store:\n%s", onDisk)
	}
}

// TestIngestEnrichesASubAgentFromTheLineageRung: ingest holds the transcript
// path, so the ladder's first rung can answer for it where migrate's cannot.
func TestIngestEnrichesASubAgentFromTheLineageRung(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA, "/mine": testRootSHA})
	proj := filepath.Join(src, "proj-a")
	transcriptFile(t, proj, "sess-e.jsonl", "sess-e", "", "/mine")
	agentPath := transcriptFile(t, proj, "agent-a1.jsonl", "sess-e", "a1", "/mine")

	var sawPath string
	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src},
		IngestOptions{Lineage: func(ref LineageRef) (HarnessLineage, bool) {
			if ref.AgentID != "a1" {
				return HarnessLineage{}, false
			}
			sawPath = ref.SourcePath
			return HarnessLineage{AgentType: "reviewer", SpawnDepth: 1, SpawnToolUseID: "toolu_x"}, true
		}})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if sawPath != agentPath {
		t.Errorf("the lookup must be handed the transcript path it can derive a sidecar from, got %q", sawPath)
	}
	var rec string
	for _, c := range res.Captured {
		if c.AgentID == "a1" {
			rec = c.RecordPath
		}
	}
	if rec == "" {
		t.Fatalf("the sub-agent was not captured: %+v", res)
	}
	onDisk, err := os.ReadFile(rec)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"agent_type: reviewer", "spawn_depth: 1", "spawn_attribution: sidecar", "lineage_source: ingest"} {
		if !strings.Contains(string(onDisk), want) {
			t.Errorf("the ingested sub-agent record is missing %q:\n%s", want, onDisk)
		}
	}
}

// TestIngestSkipsAFileThatIsNotOneTranscript: two sessions in one file is not a
// transcript, and a transcript is never split between records.
func TestIngestSkipsAFileThatIsNotOneTranscript(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: testRootSHA, "/mine": testRootSHA})
	proj := filepath.Join(src, "proj-a")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"sessionId":"sess-a","cwd":"/mine"}` + "\n" + `{"sessionId":"sess-b","cwd":"/mine"}` + "\n"
	if err := os.WriteFile(filepath.Join(proj, "mixed.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Captured) != 0 || len(res.Skipped) != 1 || res.Skipped[0].Reason != SkipNoSession {
		t.Fatalf("want one no-session-id skip and nothing captured, got %+v", res)
	}
}

// TestIngestRefusesADestinationWhoseRootAndKeyDisagree is iss-2609091911060345.
//
// The destination is a PAIR: the repository root the redaction scanner is built
// from, and the root-commit key that selects the store the records land in.
// Ingest refused an empty root and shape-checked the key, and then trusted that
// the two named the same repository. A mismatched pair redacts a transcript
// under one repository's configuration and files it into another's corpus —
// exactly the fault the explicit-destination seam exists to prevent. The seam
// must defend its own invariant rather than relying on its one caller deriving
// both halves from a single detection.
func TestIngestRefusesADestinationWhoseRootAndKeyDisagree(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	// The destination root IS a repository — it is simply not the repository the
	// store key names.
	fakeRepos(t, map[string]string{repoRoot: otherRootSHA, "/mine": testRootSHA})
	transcriptFile(t, filepath.Join(src, "proj-a"), "s1.jsonl", "sess-mine", "", "/mine")

	_, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err == nil {
		t.Fatal("Ingest must refuse a destination whose repository root and store key name different repositories")
	}
	for _, want := range []string{otherRootSHA, testRootSHA} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name both halves of the pair it rejected (missing %q): %v", want, err)
		}
	}
	records, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("a refused destination must store nothing, got %d record(s)", len(records))
	}
}

// TestIngestRefusesADestinationRootThatIsNoRepository is the other half of the
// same invariant: a root whose own root commit cannot be resolved is not a
// destination this may reason about. Fail closed — an unresolvable root is not
// evidence that the pair agrees.
func TestIngestRefusesADestinationRootThatIsNoRepository(t *testing.T) {
	repoRoot, _ := setupStore(t)
	src := t.TempDir()
	// Nothing maps repoRoot, so its own root commit does not resolve.
	fakeRepos(t, map[string]string{"/mine": testRootSHA})
	transcriptFile(t, filepath.Join(src, "proj-a"), "s1.jsonl", "sess-mine", "", "/mine")

	_, err := Ingest(Destination{RepoRoot: repoRoot, RootSHA: testRootSHA}, []string{src}, IngestOptions{})
	if err == nil {
		t.Fatal("Ingest must refuse a destination root whose own root commit cannot be resolved")
	}
	if !strings.Contains(err.Error(), "root commit") {
		t.Errorf("the refusal must say what it could not resolve: %v", err)
	}
}
