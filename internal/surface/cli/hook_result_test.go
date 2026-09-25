package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hook_result_test.go — the staging hooks carry a machine-readable result
// (iss-2608261550596333). session-end and subagent-stop exit 0 on every path by
// design, so a programmatic caller could tell a staged transcript from a lost
// one only by string-matching human stderr; an adaptor that did so shipped a
// silent permanent-loss bug. Under --json each hook writes one JSON line to
// stdout naming whether the transcript was captured, and still exits 0. Without
// --json stdout stays empty, the host-facing shape.

func decodeHookResult(t *testing.T, stdout string) hookStageResult {
	t.Helper()
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("want exactly one JSON line on stdout, got %d:\n%s", len(lines), stdout)
	}
	var r hookStageResult
	if err := json.Unmarshal([]byte(lines[0]), &r); err != nil {
		t.Fatalf("stdout is not a JSON result: %v\n%s", err, stdout)
	}
	return r
}

func TestSessionEndJSONResultReportsStagedAndNotCaptured(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	if err := os.WriteFile(tp, []byte(`{"role":"user","text":"hello"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := endPayload(t, "sess-json", repo, tp)

	stdout, _ := runHook(t, in, "hook", "session-end", "--json")
	r := decodeHookResult(t, stdout)
	if !r.Captured || r.Outcome != "staged" || r.SessionID != "sess-json" || r.Bytes == 0 {
		t.Fatalf("first stage: got %+v, want captured staged sess-json with a byte count", r)
	}

	stdout, _ = runHook(t, in, "hook", "session-end", "--json")
	if r := decodeHookResult(t, stdout); !r.Captured || r.Outcome != "already_staged" {
		t.Fatalf("repeat stage: got %+v, want captured already_staged", r)
	}

	// A failure is still exit 0 (runHook fails the test otherwise), and says so.
	stdout, stderr := runHook(t, endPayload(t, "sess-lost", repo, ""), "hook", "session-end", "--json")
	r = decodeHookResult(t, stdout)
	if r.Captured || r.Outcome != "not_captured" || r.Reason == "" {
		t.Fatalf("missing transcript_path: got %+v, want not_captured with a reason", r)
	}
	if !strings.Contains(stderr, "capturing nothing") {
		t.Fatalf("the human stderr line is kept beside the result: %q", stderr)
	}

	// Without --json, stdout stays empty.
	if stdout, _ := runHook(t, in, "hook", "session-end"); stdout != "" {
		t.Fatalf("session-end without --json wrote stdout: %q", stdout)
	}
}

func TestSubagentStopJSONResultReportsStagedAndNotCaptured(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	tp := writeAgentTranscript(t, t.TempDir(), "aj1", `{"role":"assistant","text":"done"}`+"\n")

	stdout, _ := runHook(t, subagentPayload(t, "sess-p", repo, "aj1", tp, "general-purpose"),
		"hook", "subagent-stop", "--json")
	r := decodeHookResult(t, stdout)
	if !r.Captured || r.Outcome != "staged" || r.AgentID != "aj1" {
		t.Fatalf("sub-agent stage: got %+v, want captured staged aj1", r)
	}

	stdout, _ = runHook(t, subagentPayload(t, "sess-p", repo, "aj2", "", "general-purpose"),
		"hook", "subagent-stop", "--json")
	if r := decodeHookResult(t, stdout); r.Captured || r.Outcome != "not_captured" || r.Reason == "" {
		t.Fatalf("missing transcript path: got %+v, want not_captured with a reason", r)
	}
}
