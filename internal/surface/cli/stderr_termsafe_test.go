package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stderr_termsafe_test.go — the stderr prints that do not pass through cli.Run
// mask terminal-display attack runes at the print site (iss-2609260221565656).
// Run masks the one refusal line it prints for every verb
// (error_termsafe_surface_test.go), but a hook writes its diagnostics itself and
// returns nil or a message-less exit code, so Run never sees them. Each of those
// prints masks its own line, and these tests reach each one with the hostile
// text in the place a host, a payload or a committed file would put it.

// hostilePath is a transcript path a host payload could carry: ESC opening a
// colour sequence and an RLO override, the two runes the review reproduced.
const hostilePath = "/nonexistent/\x1b[31mRED‮.jsonl"

func TestSessionEndMasksAttackRunesOnStderr(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	stdout, stderr := runHook(t, endPayload(t, "sess-hostile", repo, hostilePath), "hook", "session-end", "--json")
	if !strings.Contains(stderr, "RED") {
		t.Fatalf("stderr no longer echoes the transcript path, so this test proves nothing:\n%q", stderr)
	}
	assertNoAttackRunes(t, "session-end stderr", stderr)
	r := decodeHookResult(t, stdout)
	assertNoAttackRunes(t, "session-end --json reason", r.Reason)
	if !strings.Contains(stderr, r.Reason) {
		t.Fatalf("the stderr line and the --json reason must say the same thing:\nstderr %q\nreason %q", stderr, r.Reason)
	}
}

func TestSubagentStopMasksAttackRunesOnStderr(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	stdout, stderr := runHook(t, subagentPayload(t, "sess-hostile", repo, "ah1", hostilePath, "general-purpose"),
		"hook", "subagent-stop", "--json")
	if !strings.Contains(stderr, "RED") {
		t.Fatalf("stderr no longer echoes the transcript path, so this test proves nothing:\n%q", stderr)
	}
	assertNoAttackRunes(t, "subagent-stop stderr", stderr)
	r := decodeHookResult(t, stdout)
	assertNoAttackRunes(t, "subagent-stop --json reason", r.Reason)
}

// A committed .abcd/guard.json is repository text a pull request can set. An
// unknown top-level key carrying ESC and RLO makes the strict decoder's
// unknown-field error name the key, raw, on every Go toolchain (a type error's
// wording names a map key on some toolchains and not others); the hook
// announces the dropped repo layer on stderr and keeps the bundled hazards armed.
func TestGuardHookMasksAttackRunesInADroppedRepoLayer(t *testing.T) {
	dir := guardRepo(t)
	hostile := `{"schema_version": 1, "x\u001b[31mRED‮": 5}`
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(hostile), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runGuard(preToolUse(t, "Bash", "ls -la", dir), "guard", "hook")
	if code == 0 || code == 2 {
		t.Fatalf("a dropped repo layer is loud and non-blocking (exit 1); got %d, stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "RED") {
		t.Fatalf("the drop notice no longer echoes the decoder's error, so this test proves nothing:\n%q", stderr)
	}
	assertNoAttackRunes(t, "guard hook stderr", stderr)
}

// failOpen and the other hook diagnostics format through one helper; the
// helper is the print site, so it is pinned directly for the values no current
// payload can deliver raw (a read error, a parser error) — the next error
// message to embed its input must not be the one that reaches the terminal.
func TestDiagnosticLineMasksAttackRunes(t *testing.T) {
	var b strings.Builder
	msg := diagnosticLine(&b, "abcd guard: NOT CHECKED — %v; %s", jsonErr(t), hostileOperand)
	assertNoAttackRunes(t, "diagnostic line", b.String())
	assertNoAttackRunes(t, "returned message", msg)
	if strings.Count(b.String(), "\n") != 1 || !strings.HasSuffix(b.String(), "\n") {
		t.Fatalf("a diagnostic is exactly one line, its own terminator last:\n%q", b.String())
	}
	if !strings.HasPrefix(b.String(), msg) {
		t.Fatalf("the returned message is the line printed:\nline %q\nmsg  %q", b.String(), msg)
	}
}

// jsonErr is an error whose text carries a raw ESC, the shape a decoder error
// naming a map key takes.
func jsonErr(t *testing.T) error {
	t.Helper()
	var v map[string]int
	err := json.Unmarshal([]byte(`{"k\u001b[31m": "s"}`), &v)
	if err == nil {
		t.Fatal("want a type error")
	}
	return err
}
