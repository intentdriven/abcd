package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/gittest"
)

// secondRepo builds a SECOND git repository with its store bootstrapped under
// the HOME already in force, so a test can hold two destinations at once.
// sessionEndRepo cannot be called twice for this: it repoints HOME, which would
// strand the first repository's store.
func secondRepo(t *testing.T) (repo, rootSHA string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo = t.TempDir()
	env := append(gittest.Env(t),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
	)
	for _, args := range [][]string{
		{"init", "-q"},
		// A different root message, so the two fixtures cannot share a root SHA
		// under gittest's fixed identity and dates.
		{"commit", "-q", "--allow-empty", "-m", "second root"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	cmd.Dir = repo
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-list: %v", err)
	}
	rootSHA = strings.TrimSpace(string(out))
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "transcripts", rootSHA, "records"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo, rootSHA
}

// runRecovery drives the root command with an optional stdin and returns both
// streams and whether it exited non-zero.
func runRecovery(stdin string, args ...string) (stdout, stderr string, err error) {
	cmd := NewRootCommand()
	var so, se bytes.Buffer
	cmd.SetOut(&so)
	cmd.SetErr(&se)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	// Execute BEFORE the buffers are read: a return statement evaluates its
	// operands left to right, so reading them in the same statement would
	// snapshot two empty strings.
	err = cmd.Execute()
	return so.String(), se.String(), err
}

// plantComposite writes a pre-lineage record straight into a store.
func plantComposite(t *testing.T, rootSHA, stored, fullSession string) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Join([]string{
		"---",
		"schema: 1",
		"session_id: " + stored,
		"root_commit: " + rootSHA,
		"captured_at: 2026-01-01T00:00:00Z",
		"source_kind: native",
		"source_sha256: " + strings.Repeat("a", 64),
		"redacted_secrets: 0",
		"redacted_home_paths: 0",
		"---",
		`{"type":"user","sessionId":"` + fullSession + `"}`,
		"",
	}, "\n")
	dir := filepath.Join(home, ".abcd", "transcripts", rootSHA, "records")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "20260101T000000.000000000Z-c.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// writeAgentMeta writes the harness's per-agent metadata file into dir.
func writeAgentMeta(t *testing.T, dir, agentID, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent-"+agentID+".meta.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestHistoryMigrateReportsByDefaultAndWritesOnlyOnApply is the front door's
// half of the migration's safety property.
func TestHistoryMigrateReportsByDefaultAndWritesOnlyOnApply(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := plantComposite(t, rootSHA, "5a9221e2--agent-acf07c33", full)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	out, _, runErr := runRecovery("", "history", "migrate")
	if runErr != nil {
		t.Fatalf("history migrate: %v\n%s", runErr, out)
	}
	if !strings.Contains(out, "report only") {
		t.Errorf("the default run must say it wrote nothing, got:\n%s", out)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("the default run rewrote the record:\n%s", after)
	}

	if out, _, runErr = runRecovery("", "history", "migrate", "--apply"); runErr != nil {
		t.Fatalf("history migrate --apply exited non-zero: %s", out)
	}
	applied, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(applied), "session_id: "+full) {
		t.Errorf("--apply did not write the recovered session id:\n%s", applied)
	}
}

// TestHistoryMigrateRecoversLineageFromADeclaredSidecarRoot wires the front
// door's own contribution: core knows no on-disk layout, so the type and depth
// only reach a migrated record if this verb finds the harness's file.
func TestHistoryMigrateRecoversLineageFromADeclaredSidecarRoot(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	path := plantComposite(t, rootSHA, "5a9221e2--agent-acf07c33", "5a9221e2-fa77-4be5-84d3-779199c449d7")
	// Nested two directories deep, to pin that the index finds the file by NAME
	// rather than by a layout it assumes.
	roots := t.TempDir()
	writeAgentMeta(t, filepath.Join(roots, "a-project", "a-session", "subagents"), "acf07c33",
		`{"agentType":"reviewer","spawnDepth":2,"toolUseId":"toolu_x","parentAgentId":"b0b0"}`)

	out, _, runErr := runRecovery("", "history", "migrate", "--apply", "--sidecar-root", roots)
	if runErr != nil {
		t.Fatalf("history migrate: %v\n%s", runErr, out)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"agent_type: reviewer", "spawn_depth: 2", "parent_agent_id: b0b0", "spawn_attribution: sidecar"} {
		if !strings.Contains(string(onDisk), want) {
			t.Errorf("the migrated record is missing %q:\n%s", want, onDisk)
		}
	}
}

// TestHistoryIngestWritesIntoTheNamedRepositoryNotTheWorkingDirectory is the
// seam at the surface. Standing in one repository and naming another must store
// the transcripts in the one that was NAMED — an operator recovering a backlog
// is not standing in the repository the transcripts belong to.
func TestHistoryIngestWritesIntoTheNamedRepositoryNotTheWorkingDirectory(t *testing.T) {
	destRepo, destSHA := sessionEndRepo(t)
	// A second repository, and a working directory inside it.
	otherRepo, otherSHA := secondRepo(t)
	if otherSHA == destSHA {
		t.Fatal("the two fixtures share a root commit, so this test cannot tell their stores apart")
	}
	t.Chdir(otherRepo)

	src := t.TempDir()
	proj := filepath.Join(src, "a-project")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","sessionId":"sess-named","cwd":"` + destRepo + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(proj, "s1.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, runErr := runRecovery("", "history", "ingest", "--into", destRepo, src)
	if runErr != nil {
		t.Fatalf("history ingest: %v\n%s", runErr, out)
	}
	if !strings.Contains(out, "into ") {
		t.Errorf("the run must say which repository it wrote into, got:\n%s", out)
	}
	dest, err := history.List(destRepo, destSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(dest) != 1 || dest[0].SessionID != "sess-named" {
		t.Errorf("the named destination must hold the transcript, got %+v", dest)
	}
	other, err := history.List(otherRepo, otherSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Errorf("the working directory's repository must hold nothing, got %+v", other)
	}
}

// TestHistoryIngestPromptsForOrphansOnlyUnderThatPolicy: the prompt lives in
// the front door, and it is off unless the destination's own configuration
// turns it on.
func TestHistoryIngestPromptsForOrphansOnlyUnderThatPolicy(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	src := t.TempDir()
	proj := filepath.Join(src, "a-project")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","sessionId":"sess-orphan","cwd":"/no/such/directory/here"}` + "\n"
	if err := os.WriteFile(filepath.Join(proj, "s1.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	// Default policy: ignored and reported, and stdin is never read.
	out, _, runErr := runRecovery("y\n", "history", "ingest", "--into", repo, src)
	if runErr != nil {
		t.Fatalf("history ingest: %v\n%s", runErr, out)
	}
	if !strings.Contains(out, "orphan") {
		t.Errorf("an orphan must be reported, got:\n%s", out)
	}
	records, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("the default policy must store nothing, got %+v", records)
	}

	// Under `prompt`, the front door asks and an explicit yes adopts.
	cfgDir := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "history.json"),
		[]byte(`{"schema_version":1,"on_orphan":"prompt"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, runErr := runRecovery("y\n", "history", "ingest", "--into", repo, src)
	if runErr != nil {
		t.Fatalf("history ingest: %v\n%s", runErr, out)
	}
	if !strings.Contains(errOut, "adopt") {
		t.Errorf("the prompt must be asked out of band, got stderr:\n%s", errOut)
	}
	records, err = history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].AdoptedProject != "a-project" {
		t.Errorf("an answered prompt must adopt and stamp the project, got %+v", records)
	}
}

// TestHistoryIngestRefusesWithNoSource: with no operand and nothing declared,
// there is nothing to ingest and the verb says where to declare it.
func TestHistoryIngestRefusesWithNoSource(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	t.Chdir(repo)
	_, _, runErr := runRecovery("", "history", "ingest", "--into", repo)
	if runErr == nil {
		t.Fatal("history ingest with no source must fail")
	}
	if !strings.Contains(runErr.Error(), history.ConfigRelPath) {
		t.Errorf("the refusal must name where roots are declared, got: %v", runErr)
	}
}

// TestHistoryIngestRefusesWithoutAnExplicitDestination: the surface has no
// default destination either. Standing in a repository is not the same as
// naming one, and the difference is which repository's redaction configuration
// applies to somebody's transcripts.
func TestHistoryIngestRefusesWithoutAnExplicitDestination(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	t.Chdir(repo)
	_, _, runErr := runRecovery("", "history", "ingest", t.TempDir())
	if runErr == nil {
		t.Fatal("history ingest without --into must fail rather than assume the working directory")
	}
	if !strings.Contains(runErr.Error(), "--into") {
		t.Errorf("the refusal must name the flag that answers it, got: %v", runErr)
	}
}
