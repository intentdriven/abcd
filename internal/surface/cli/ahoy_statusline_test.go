package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// harnessSettingsFixture points CLAUDE_CONFIG_DIR at a temp directory holding
// one settings.json, the host harness's user-level settings as `ahoy` reads
// them (spc-70). It returns the file's path.
func harnessSettingsFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestAhoyBareRendersTheStatusLineSignal pins the one line the bare board
// gained: the harness's status-line state, from the same detection pass.
func TestAhoyBareRendersTheStatusLineSignal(t *testing.T) {
	hermeticRepo(t)
	out := runCLI(t, "ahoy")
	if !strings.Contains(string(out), "  statusline:  no-harness\n") {
		t.Errorf("bare render lacks the statusline line:\n%s", out)
	}
	harnessSettingsFixture(t, `{"statusLine": {"type": "command", "command": "bash /tmp/prev.sh"}}`)
	out = runCLI(t, "ahoy")
	if !strings.Contains(string(out), "  statusline:  foreign\n") {
		t.Errorf("bare render does not report a foreign status line:\n%s", out)
	}
}

// TestAhoyInstallYesReportsTheSkippedStatusLineOffer: --yes leaves the offer
// and the text render says so, with its reason and the way to answer it.
func TestAhoyInstallYesReportsTheSkippedStatusLineOffer(t *testing.T) {
	hermeticRepo(t)
	settings := harnessSettingsFixture(t, `{"padding": 1}`)
	before, _ := os.ReadFile(settings)

	out := runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	var res struct {
		Status          string   `json:"status"`
		OptionalSkipped []string `json:"optional_skipped"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if !contains(res.OptionalSkipped, "statusline.offered") {
		t.Errorf("optional_skipped = %v, want statusline.offered", res.OptionalSkipped)
	}
	if after, _ := os.ReadFile(settings); string(after) != string(before) {
		t.Error("--yes rewrote the harness settings")
	}

	text := runCLI(t, "ahoy", "install", "--yes")
	s := string(text)
	if !strings.Contains(s, "optional, not covered by --yes: ") || !strings.Contains(s, "statusline.offered") {
		t.Errorf("text render does not name the skipped offer:\n%s", s)
	}
	if !strings.Contains(s, "the status line rewrites a setting of the host harness") {
		t.Errorf("text render does not say why the offer needs an answer:\n%s", s)
	}
}

// TestAhoyInstallPipedAnswersWireTheStatusLine drives the offer through the
// documented `yes |` form: every category approval, the offer, and the element
// prompts are answered off a real pipe, and the two files land.
func TestAhoyInstallPipedAnswersWireTheStatusLine(t *testing.T) {
	hermeticRepo(t)
	settings := harnessSettingsFixture(t, `{"statusLine": {"type": "command", "command": "bash /tmp/prev.sh"}}`)
	answers := strings.Repeat("y\n", 24)
	out, errOut, err := runCLIPipedStdinSplit(t, answers, "ahoy", "install",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	if err != nil {
		t.Fatalf("install: %v\n%s\n%s", err, out, errOut)
	}
	var res struct {
		Writes []string `json:"writes"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if !contains(res.Writes, "~/.abcd/statusline.json") {
		t.Errorf("writes = %v, want the user-level setting", res.Writes)
	}
	raw, _ := os.ReadFile(settings)
	if !strings.Contains(string(raw), "' statusline\"") {
		t.Errorf("harness not pointed at abcd:\n%s", raw)
	}
	// The element prompts were asked, each keyed by its element, on stderr.
	if !strings.Contains(string(errOut), "statusline.branch (on/off) [on]:") {
		t.Errorf("element prompt missing from the transcript:\n%s", errOut)
	}

	// Uninstall hands the previous command back and says so.
	text := runCLI(t, "ahoy", "uninstall")
	if !strings.Contains(string(text), "  status line: restored the previous status command\n") {
		t.Errorf("uninstall render lacks the status-line line:\n%s", text)
	}
	raw, _ = os.ReadFile(settings)
	if !strings.Contains(string(raw), `"command": "bash /tmp/prev.sh"`) {
		t.Errorf("previous command not restored:\n%s", raw)
	}
}

// TestAhoyInstallSanitizesRefusalNotes: a refusal note names things the user
// wrote — a harness path, a status command — and the text render masks a
// terminal escape in it the way every sibling renderer does. The harness
// directory here carries an escape, so the status-line refusal that names it
// arrives at the render with the escape in it.
func TestAhoyInstallSanitizesRefusalNotes(t *testing.T) {
	hermeticRepo(t)
	dir := filepath.Join(t.TempDir(), "cfg\x1b[31m")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"),
		[]byte(`{"statusLine": {"type": "static", "text": "hi"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	answers := strings.Repeat("y\n", 24)
	out, errOut, err := runCLIPipedStdinSplit(t, answers, "ahoy", "install",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false")
	if err != nil {
		t.Fatalf("install: %v\n%s\n%s", err, out, errOut)
	}
	s := string(out)
	if !strings.Contains(s, "  note: ") || !strings.Contains(s, "status line") {
		t.Fatalf("precondition: the render carries no status-line refusal note:\n%s", s)
	}
	if strings.Contains(s, "\x1b") {
		t.Errorf("the install render printed a note with a raw terminal escape:\n%q", s)
	}
}
