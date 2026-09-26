package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
)

// questionCall builds the host's PreToolUse payload for a call to its question
// tool: the tool name and a question, nothing the gate reads beyond the name
// and the session's directory.
func questionCall(t *testing.T, cwd string) string {
	t.Helper()
	payload := map[string]any{
		"session_id":      "s1",
		"cwd":             cwd,
		"hook_event_name": "PreToolUse",
		"tool_name":       questionTools[0],
		"tool_input": map[string]any{"questions": []any{map[string]any{
			"question": "Ship it?", "header": "Ship", "multiSelect": false,
			"options": []any{map[string]any{"label": "Yes", "description": "ship"}, map[string]any{"label": "No", "description": "hold"}},
		}}},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func questionMarker(root string) string {
	return filepath.Join(root, filepath.FromSlash(mode.QuestionOpenRelPath))
}

// TestGuardRefusesAQuestionWhileManaged is criterion 2: with the mode reading
// managed, a question to the human through the host's question tool is refused
// with the host's blocking status, and the refusal names the two settings and
// the verb that sets them. Nothing is marked open.
func TestGuardRefusesAQuestionWhileManaged(t *testing.T) {
	root := managedCheckout(t)
	stdout, stderr, code := runGuard(questionCall(t, root), "guard", "hook")

	if code != 2 {
		t.Fatalf("a question while managed must exit 2; got %d (stderr %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("the refusal belongs on stderr alone; stdout = %q", stdout)
	}
	for _, want := range []string{
		"abcd mode product-thinker", "abcd mode facilitator",
		"the product thinker", "the technical facilitator",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the refusal must name %q; stderr = %q", want, stderr)
		}
	}
	if strings.Count(strings.TrimRight(stderr, "\n"), "\n") != 0 {
		t.Errorf("the refusal is one line; stderr = %q", stderr)
	}
	if _, err := os.Lstat(questionMarker(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused question was marked open: %v", err)
	}
}

// TestGuardAdmitsAQuestionOnceAddressed: with the mode naming whom the agent is
// asking, the question runs silently and is marked open for the reset.
func TestGuardAdmitsAQuestionOnceAddressed(t *testing.T) {
	for _, st := range []mode.State{mode.ProductThinker, mode.Facilitator} {
		t.Run(string(st), func(t *testing.T) {
			root := managedCheckout(t)
			if err := mode.SetAt(root, st); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := runGuard(questionCall(t, root), "guard", "hook")
			if code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("an addressed question must be admitted silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			data, err := os.ReadFile(questionMarker(root))
			if err != nil {
				t.Fatalf("the admitted question was not marked open: %v", err)
			}
			if strings.TrimSpace(string(data)) != string(st) {
				t.Errorf("marker = %q, want %q", data, st)
			}
		})
	}
}

// TestGuardLeavesQuestionsAloneWhereUnmanaged: a repository abcd does not
// manage has no badge and nowhere to set the mode, so its questions are never
// gated, and the gate creates nothing there.
func TestGuardLeavesQuestionsAloneWhereUnmanaged(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	t.Chdir(repo)

	stdout, stderr, code := runGuard(questionCall(t, repo), "guard", "hook")
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("an unmanaged question must pass silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the gate created .abcd/ in an unmanaged repository: %v", err)
	}
}

// TestGuardQuestionGateFailsOpenLoud: a mode store the gate cannot read is not
// a decision, so the question runs and the gate says so on the loud,
// non-blocking status — the guard's own fail-open-loud contract.
func TestGuardQuestionGateFailsOpenLoud(t *testing.T) {
	root := managedCheckout(t)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(mode.FileRelPath)), []byte("sideways\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runGuard(questionCall(t, root), "guard", "hook")
	if code != 1 {
		t.Fatalf("an unreadable mode store must fail open loud (exit 1); got %d (stderr %q)", code, stderr)
	}
	if !strings.Contains(stderr, "NOT CHECKED") {
		t.Errorf("the fail-open must say so; stderr = %q", stderr)
	}
}

// TestGuardQuestionGateFailsOpenWhereTheModeCannotBeSet
// (iss-2609260100382261): a refusal whose remedy cannot run refuses forever.
// With the tier present but not writable the mode reads managed, and `abcd
// mode` would fail on permission denied, so the gate does not refuse: it lets
// the question run on the loud status, names why, and leaves nothing behind.
func TestGuardQuestionGateFailsOpenWhereTheModeCannotBeSet(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a read-only directory does not refuse root")
	}
	root := managedCheckout(t)
	tier := filepath.Join(root, filepath.FromSlash(mode.TierRelPath))
	if err := os.Chmod(tier, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tier, 0o700) })

	if _, _, err := runSplit(t, "", "mode", "facilitator"); err == nil {
		t.Fatal("precondition: `abcd mode facilitator` succeeded on a read-only tier")
	}
	_, stderr, code := runGuard(questionCall(t, root), "guard", "hook")
	if code != 1 {
		t.Fatalf("a question whose remedy cannot run must fail open loud (exit 1); got %d (stderr %q)", code, stderr)
	}
	for _, want := range []string{"NOT CHECKED", "UNGATED", "cannot be set"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the fail-open must say %q; stderr = %q", want, stderr)
		}
	}
	if strings.Contains(stderr, questionRefusal) {
		t.Errorf("the gate still named a remedy that cannot run; stderr = %q", stderr)
	}
	ents, err := os.ReadDir(tier)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 0 {
		t.Errorf("the gate left residue in the tier: %v", ents)
	}
}

// TestPromptHookResetsTheModeAfterAQuestion is criterion 4: the next human
// message after an admitted question resets the mode to managed, clears the
// marker and says so in one stderr line; a message with no question open
// changes nothing and says nothing about the mode.
func TestPromptHookResetsTheModeAfterAQuestion(t *testing.T) {
	root := managedCheckout(t)
	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if _, _, code := runGuard(questionCall(t, root), "guard", "hook"); code != 0 {
		t.Fatalf("precondition: the addressed question was not admitted (exit %d)", code)
	}

	prompt := `{"session_id":"s-reset","hook_event_name":"UserPromptSubmit","prompt":"yes, ship it","cwd":"` + root + `"}`
	_, stderr, err := runSplit(t, prompt, "hook", "prompt-router")
	if err != nil {
		t.Fatalf("prompt-router: %v\n%s", err, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.Managed {
		t.Errorf("mode after the answer = %q, want managed", got)
	}
	if _, err := os.Lstat(questionMarker(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the marker survived the answer: %v", err)
	}
	if n := modeResetLines(stderr); n != 1 {
		t.Errorf("want exactly one stderr line saying the mode was reset, got %d:\n%s", n, stderr)
	}

	// A human who sets the hat by hand keeps it across their next message.
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	_, stderr, err = runSplit(t, prompt, "hook", "prompt-router")
	if err != nil {
		t.Fatalf("prompt-router: %v\n%s", err, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.Facilitator {
		t.Errorf("with no question open the mode moved to %q", got)
	}
	if n := modeResetLines(stderr); n != 0 {
		t.Errorf("no question was open, yet the hook reported a reset:\n%s", stderr)
	}
}

// TestPromptHookLeavesAHandSetModeOverAnUnremovableMarker
// (iss-2609260100393814): with a directory planted at the marker's path, every
// message would otherwise reset the hand-set mode and repeat the same error.
// Each message says so in one line, and the mode stays where the human put it.
func TestPromptHookLeavesAHandSetModeOverAnUnremovableMarker(t *testing.T) {
	root := managedCheckout(t)
	if err := os.MkdirAll(filepath.Join(questionMarker(root), "planted"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	prompt := `{"session_id":"s-planted","hook_event_name":"UserPromptSubmit","prompt":"carry on","cwd":"` + root + `"}`
	for i := range 2 {
		_, stderr, err := runSplit(t, prompt, "hook", "prompt-router")
		if err != nil {
			t.Fatalf("message %d: prompt-router: %v\n%s", i+1, err, stderr)
		}
		if got, _ := mode.ReadAt(root); got != mode.Facilitator {
			t.Fatalf("message %d: the hand-set mode moved to %q", i+1, got)
		}
		if n := modeResetLines(stderr); n != 1 {
			t.Errorf("message %d: want one loud line about the marker, got %d:\n%s", i+1, n, stderr)
		}
		if strings.Contains(stderr, "reset to managed") {
			t.Errorf("message %d: the hook claims a reset it must not make:\n%s", i+1, stderr)
		}
	}
}

func modeResetLines(stderr string) int {
	n := 0
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, "abcd mode:") {
			n++
		}
	}
	return n
}

// TestModeVerbSetsTheLine is criterion 3: the `mode` verb is the setter, as it
// was, and the line shows the state it set on its next render.
func TestModeVerbSetsTheLine(t *testing.T) {
	root := managedCheckout(t)
	for _, tc := range []struct{ word, label string }{
		{"product-thinker", "waiting on the product thinker"},
		{"facilitator", "waiting on the technical facilitator"},
		{"managed", "abcd-managed"},
	} {
		if _, stderr, err := runSplit(t, "", "mode", tc.word); err != nil {
			t.Fatalf("mode %s: %v\n%s", tc.word, err, stderr)
		}
		stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
		if err != nil {
			t.Fatalf("statusline: %v\n%s", err, stderr)
		}
		if !strings.Contains(stdout, " "+tc.label+" ") {
			t.Errorf("after `mode %s` the line reads %q, want the badge %q", tc.word, stdout, tc.label)
		}
	}
}
