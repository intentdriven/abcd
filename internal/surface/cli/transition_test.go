package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/update"
)

// sessionStartSandbox gives a session-start run its own HOME, no plugin root
// and the given data dir, so nothing a run writes lands outside the test.
func sessionStartSandbox(t *testing.T, data string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
}

// TestSessionStartDoesNotCompareSetupVersion: the ruling CJ1 replaces the
// setup_version comparison. An update is announced once, by whatever swapped
// the binary, so a repo whose recorded setup_version differs from the running
// binary gets no transition notice at session start.
func TestSessionStartDoesNotCompareSetupVersion(t *testing.T) {
	orig := core.Version
	core.Version = "v9.9.9"
	t.Cleanup(func() { core.Version = orig })
	sessionStartSandbox(t, "")

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"),
		[]byte(`{"meta":{"setup_version":"v1.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _ := runCLIStdinErr(t, `{"cwd":"`+repo+`"}`, "hook", "session-start")
	if s := string(out); strings.Contains(s, "last set up with") || strings.Contains(s, "v1.0.0") {
		t.Fatalf("session start must not report a setup_version transition:\n%s", s)
	}
}

// TestSessionStartShowsAnUnseenUpdateOnce is the ruling CJ1b's single
// exception, end to end through the hook: a swap whose own output was
// discarded is shown by the next session start in the shared wording, and not
// by the one after it.
func TestSessionStartShowsAnUnseenUpdateOnce(t *testing.T) {
	data := t.TempDir()
	cache := filepath.Join(data, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := "release_tag=v0.12.0\nrelease_sha=unknown\nfetched_at=2026-09-30T00:00:00Z\nprevious_tag=v0.11.1\ntransition_unseen=yes\n"
	if err := os.WriteFile(filepath.Join(cache, "binary-meta"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	sessionStartSandbox(t, data)
	repo := t.TempDir()
	want := update.UpdatedLine("v0.11.1", "v0.12.0")

	out, _ := runCLIStdinErr(t, `{"cwd":"`+repo+`"}`, "hook", "session-start")
	if n := strings.Count(string(out), want); n != 1 {
		t.Fatalf("the first session after an unseen swap must show %q once, got %d:\n%s", want, n, out)
	}
	out, _ = runCLIStdinErr(t, `{"cwd":"`+repo+`"}`, "hook", "session-start")
	if strings.Contains(string(out), "updated from") {
		t.Fatalf("the next session must not show the update again:\n%s", out)
	}
}
