package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plantForReconstruct writes one record straight into the store under the HOME
// already in force, so a surface test exercises rendering rather than capture.
func plantForReconstruct(t *testing.T, rootSHA, name string, fields []string, body string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	head := append([]string{
		"---",
		"schema: 3",
		"root_commit: " + rootSHA,
		"source_kind: native",
		"source_sha256: " + strings.Repeat("b", 64),
		"redacted_secrets: 0",
		"redacted_home_paths: 0",
	}, fields...)
	p := filepath.Join(home, ".abcd", "history", rootSHA, "transcripts", name)
	if err := os.WriteFile(p, []byte(strings.Join(head, "\n")+"\n---\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// plantReconstructSession lays a main thread and one delegate into the store.
func plantReconstructSession(t *testing.T, rootSHA string) {
	t.Helper()
	usage := `"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}`
	plantForReconstruct(t, rootSHA, "20260901T100000.000000000Z-sess-cli.md", []string{
		"session_id: sess-cli",
		"captured_at: 2026-09-01T10:06:00Z",
	}, strings.Join([]string{
		`{"type":"user","timestamp":"2026-09-01T10:00:00Z","message":{"role":"user","content":"do the thing"}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:00:05Z","message":{"id":"msg_1","role":"assistant","model":"claude-test-1",` + usage + `,"content":[{"type":"tool_use","id":"toolu_spawn","name":"Agent","input":{"subagent_type":"explorer"}}]}}`,
		`{"type":"assistant","timestamp":"2026-09-01T10:00:05Z","message":{"id":"msg_1","role":"assistant","model":"claude-test-1",` + usage + `,"content":[{"type":"text","text":"delegating"}]}}`,
		`{"type":"user","timestamp":"2026-09-01T10:05:00Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_spawn","content":"agentId: agentcli done"}]}}`,
	}, "\n")+"\n")
	plantForReconstruct(t, rootSHA, "20260901T100500.000000000Z-sess-cli-agent-agentcli.md", []string{
		"session_id: sess-cli",
		"captured_at: 2026-09-01T10:05:00Z",
		"agent_id: agentcli",
		"agent_type: explorer",
		"spawn_tool_use_id: toolu_spawn",
		"spawn_depth: 1",
		"lineage_source: hook",
		"spawn_attribution: sidecar",
	}, `{"type":"assistant","timestamp":"2026-09-01T10:02:00Z","message":{"id":"msg_s","role":"assistant","model":"claude-test-2",`+usage+`,"content":[{"type":"text","text":"the delegate answer"}]}}`+"\n")
}

// TestReconstructWritesBothFiles is the verb's contract: ONE artefact and ONE
// telemetry file, both landing in --out, both named after the session. A
// telemetry file describing an artefact that was never written would measure
// nothing, so the pair is what the verb promises.
func TestReconstructWritesBothFiles(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	plantReconstructSession(t, rootSHA)

	out := t.TempDir()
	stdout, _, err := runRecovery("", "history", "reconstruct", "sess-cli", "--out", out)
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}

	artefact, err := os.ReadFile(filepath.Join(out, "sess-cli.md"))
	if err != nil {
		t.Fatalf("the artefact must be written as <session>.md: %v", err)
	}
	telRaw, err := os.ReadFile(filepath.Join(out, "sess-cli.telemetry.json"))
	if err != nil {
		t.Fatalf("the telemetry must be written as <session>.telemetry.json: %v", err)
	}
	if !strings.Contains(string(artefact), "## Main thread") ||
		!strings.Contains(string(artefact), "## Agent `agentcli`") {
		t.Errorf("the artefact must carry the main thread and every sub-agent:\n%s", artefact)
	}
	var tel map[string]any
	if err := json.Unmarshal(telRaw, &tel); err != nil {
		t.Fatalf("the telemetry file must be valid JSON: %v", err)
	}
	for _, key := range []string{"tokens", "tool_calls", "models", "agent_types", "agents", "completeness", "wall_clock_seconds", "turns"} {
		if _, ok := tel[key]; !ok {
			t.Errorf("telemetry is missing %q; keys are %v", key, telemetryKeys(tel))
		}
	}
	if !strings.Contains(stdout, "wrote:") || !strings.Contains(stdout, "sess-cli.md") {
		t.Errorf("the run must report what it wrote; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "artefact:") {
		t.Errorf("the run must report the artefact's size, since that is what a caller cannot see from a filename; got:\n%s", stdout)
	}
}

// TestReconstructToStdoutWritesNoFiles. `--out -` is the pipe form; it must not
// leave anything on disk.
func TestReconstructToStdoutWritesNoFiles(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	plantReconstructSession(t, rootSHA)

	before, _ := os.ReadDir(repo)
	stdout, _, err := runRecovery("", "history", "reconstruct", "sess-cli", "--out", "-")
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}
	after, _ := os.ReadDir(repo)
	if len(after) != len(before) {
		t.Errorf("--out - must write no files; %d entries became %d", len(before), len(after))
	}
	if !strings.Contains(stdout, "# Session sess-cli") {
		t.Errorf("the artefact must reach stdout; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "sess-cli.telemetry.json") || !strings.Contains(stdout, `"schema_version"`) {
		t.Errorf("the telemetry must reach stdout alongside it; got:\n%s", stdout)
	}
}

// TestReconstructJSONEnvelopeCarriesTheTelemetry. `--json` is the machine door,
// and the measures are the reason the verb exists, so they must be in the
// envelope rather than only in a file the caller has to go and read.
func TestReconstructJSONEnvelopeCarriesTheTelemetry(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	plantReconstructSession(t, rootSHA)

	out := t.TempDir()
	stdout, _, err := runRecovery("", "--json", "history", "reconstruct", "sess-cli", "--out", out)
	if err != nil {
		t.Fatalf("reconstruct --json: %v", err)
	}
	var env struct {
		ArtefactName  string   `json:"artefact_name"`
		TelemetryName string   `json:"telemetry_name"`
		ArtefactBytes int      `json:"artefact_bytes"`
		Written       []string `json:"written"`
		Telemetry     struct {
			SessionID string         `json:"session_id"`
			ToolCalls map[string]int `json:"tool_calls"`
			Tokens    struct {
				Total        int64 `json:"total"`
				APIResponses int   `json:"api_responses"`
			} `json:"tokens"`
		} `json:"telemetry"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("--json must emit one JSON envelope: %v\n%s", err, stdout)
	}
	if env.ArtefactName != "sess-cli.md" || env.TelemetryName != "sess-cli.telemetry.json" {
		t.Errorf("envelope names wrong: %+v", env)
	}
	if env.ArtefactBytes == 0 || len(env.Written) != 2 {
		t.Errorf("the envelope must report the size and both written files: %+v", env)
	}
	if env.Telemetry.Tokens.APIResponses != 2 {
		t.Errorf("two responses (one written as two lines) plus the delegate's: got %d",
			env.Telemetry.Tokens.APIResponses)
	}
	if env.Telemetry.ToolCalls["Agent"] != 1 {
		t.Errorf("the envelope must carry the tool histogram: %+v", env.Telemetry.ToolCalls)
	}
	// The envelope is a success envelope the CLI's error scrub never sees, so
	// the written paths must already be home-redacted.
	home, _ := os.UserHomeDir()
	if strings.Contains(stdout, home) {
		t.Errorf("the envelope must not carry the absolute home root; got:\n%s", stdout)
	}
}

// TestReconstructRefusesAnAbsentOutDirectory. The verb writes a document whose
// value is being findable afterwards, so a mistyped --out is a refusal rather
// than a directory quietly created somewhere nobody will look.
func TestReconstructRefusesAnAbsentOutDirectory(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	plantReconstructSession(t, rootSHA)

	_, _, err := runRecovery("", "history", "reconstruct", "sess-cli", "--out", filepath.Join(t.TempDir(), "nope"))
	if err == nil || !strings.Contains(err.Error(), "not an existing directory") {
		t.Errorf("want a refusal naming the missing directory, got %v", err)
	}
}

// TestReconstructRefusesAnUnknownSession. A session with no records is the
// caller asking about something that is not there.
func TestReconstructRefusesAnUnknownSession(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	t.Chdir(repo)

	_, _, err := runRecovery("", "history", "reconstruct", "sess-missing", "--out", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "no records for session") {
		t.Errorf("want a refusal naming the session, got %v", err)
	}
}

func telemetryKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
