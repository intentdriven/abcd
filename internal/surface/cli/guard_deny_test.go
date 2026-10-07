package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
)

// The guard hook's answer to the host (iss-2610070637562567). A block is the
// host's PreToolUse deny: exit 0 and exactly one JSON object on stdout whose
// hookSpecificOutput carries the decision and its reason, because the host
// shows that reason to the person without echoing the hook's command and hands
// it to the agent. With exit 0 the host discards stderr, so a deny writes
// nothing there. An allow prints nothing on stdout.

// hookOutputKeys decodes a hook's stdout as exactly one JSON object holding
// hookSpecificOutput and nothing else, and returns that inner object's members.
func hookOutputKeys(t *testing.T, stdout string) map[string]string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(stdout))
	var doc map[string]map[string]string
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("stdout is not one hook output object: %v\nstdout = %q", err, stdout)
	}
	if dec.More() {
		t.Fatalf("stdout carries more than one JSON value: %q", stdout)
	}
	if len(doc) != 1 || doc["hookSpecificOutput"] == nil {
		t.Fatalf("stdout must hold hookSpecificOutput alone: %q", stdout)
	}
	inner := doc["hookSpecificOutput"]
	if inner["hookEventName"] != "PreToolUse" {
		t.Fatalf("hookEventName = %q, want PreToolUse; stdout = %q", inner["hookEventName"], stdout)
	}
	return inner
}

// encodeHookOutput is the byte-exact form a hook output object takes: one
// line, the event name first, HTML characters unescaped, a closing newline.
func encodeHookOutput(t *testing.T, inner map[string]string) string {
	t.Helper()
	type decision struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision,omitempty"`
		PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
		AdditionalContext        string `json:"additionalContext,omitempty"`
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(map[string]decision{"hookSpecificOutput": {
		HookEventName:            inner["hookEventName"],
		PermissionDecision:       inner["permissionDecision"],
		PermissionDecisionReason: inner["permissionDecisionReason"],
		AdditionalContext:        inner["additionalContext"],
	}})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// mustDeny asserts the hook answered with the host's deny — exit 0, an empty
// stderr, and stdout exactly one deny object byte for byte — and returns its
// reason.
func mustDeny(t *testing.T, stdout, stderr string, code int) string {
	t.Helper()
	if code != 0 {
		t.Fatalf("a block is the host's deny, which exits 0; got exit %d (stdout %q stderr %q)", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("the host discards stderr on exit 0, so a deny writes nothing there; stderr = %q", stderr)
	}
	inner := hookOutputKeys(t, stdout)
	keys := make([]string, 0, len(inner))
	for k := range inner {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"hookEventName", "permissionDecision", "permissionDecisionReason"}) {
		t.Fatalf("a deny carries exactly the event, the decision and the reason; keys = %v", keys)
	}
	if inner["permissionDecision"] != "deny" {
		t.Fatalf("permissionDecision = %q, want deny", inner["permissionDecision"])
	}
	reason := inner["permissionDecisionReason"]
	if strings.TrimSpace(reason) == "" {
		t.Fatalf("a deny must give its reason; stdout = %q", stdout)
	}
	if want := encodeHookOutput(t, inner); stdout != want {
		t.Fatalf("the deny is not in its one exact form:\n got %q\nwant %q", stdout, want)
	}
	return reason
}

// mustAdmitWithNote asserts the hook let the call run and told the agent
// something: exit 0, an empty stderr, and stdout exactly one object carrying
// additionalContext and NO permission decision, which would bypass the host's
// own permission flow. It returns the note.
func mustAdmitWithNote(t *testing.T, stdout, stderr string, code int) string {
	t.Helper()
	if code != 0 {
		t.Fatalf("an admitted call exits 0; got exit %d (stdout %q stderr %q)", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("an admitted call with a note writes nothing on stderr; stderr = %q", stderr)
	}
	inner := hookOutputKeys(t, stdout)
	if _, set := inner["permissionDecision"]; set {
		t.Fatalf("an admit must never set permissionDecision; stdout = %q", stdout)
	}
	if len(inner) != 2 || strings.TrimSpace(inner["additionalContext"]) == "" {
		t.Fatalf("an admit with a note carries the event and additionalContext alone; stdout = %q", stdout)
	}
	if want := encodeHookOutput(t, inner); stdout != want {
		t.Fatalf("the note is not in its one exact form:\n got %q\nwant %q", stdout, want)
	}
	return inner["additionalContext"]
}

// isDeny reports whether stdout carries a permission decision at all, for the
// checks that a non-decision never blocks.
func isDeny(stdout string) bool { return strings.Contains(stdout, `"permissionDecision"`) }

// TestGuardHookFoldsDiagnosticsIntoTheDenyReason: a diagnostic the hook wrote
// before it blocked — here the notice that the repo's .abcd/guard.json did not
// load — reached a human beside the old exit-2 block on stderr. A deny exits 0,
// where the host discards stderr, so the notice travels in the deny's reason,
// before the block it preceded, and nothing is written on stderr.
func TestGuardHookFoldsDiagnosticsIntoTheDenyReason(t *testing.T) {
	dir := guardRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", `git commit --no-verify -m "wip"`, dir), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	notice := strings.Index(reason, "DROPPED")
	block := strings.Index(reason, "Blocked by the abcd guard")
	if notice < 0 || block < 0 {
		t.Fatalf("the reason must carry the drop notice and the block; reason = %q", reason)
	}
	if notice > block {
		t.Errorf("the notice must come before the block it preceded; reason = %q", reason)
	}
	for _, l := range strings.Split(reason, "\n") {
		if l == "" {
			t.Errorf("the reason's lines are joined without blank lines; reason = %q", reason)
		}
	}
}

// TestGuardHookAllowPrintsNothingOnStdout: the host reads a hook's stdout as
// its decision, so every answer that is not a block or a note leaves stdout
// empty, whatever it says on stderr and whatever status it exits with.
func TestGuardHookAllowPrintsNothingOnStdout(t *testing.T) {
	for name, tc := range map[string]struct {
		command string
		broken  bool
		code    int
	}{
		"silent allow":                {"git status --porcelain", false, 0},
		"warn":                        {"git clean -fd", false, 1},
		"allow beside a dropped repo": {"ls -la", true, 1},
		"fail open":                   {"", false, 1},
	} {
		t.Run(name, func(t *testing.T) {
			dir := guardRepo(t)
			if tc.broken {
				if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte("{not json"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			stdout, stderr, code := runGuard(preToolUse(t, "Bash", tc.command, dir), "guard", "hook")
			if code != tc.code {
				t.Fatalf("exit %d, want %d (stderr %q)", code, tc.code, stderr)
			}
			if stdout != "" {
				t.Errorf("an allow prints nothing on stdout; stdout = %q", stdout)
			}
		})
	}
	t.Run("well-built question", func(t *testing.T) {
		root := managedCheckout(t)
		setMode(t, root, mode.ProductThinker)
		stdout, _, code := runGuard(askPayload(t, root, wellBuilt()), "guard", "hook")
		if code != 0 || stdout != "" {
			t.Errorf("an admitted question prints nothing on stdout; exit %d stdout %q", code, stdout)
		}
	})
}
