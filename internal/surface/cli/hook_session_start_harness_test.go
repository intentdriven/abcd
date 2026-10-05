package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHookSessionStartNamesAnAbcdHookInTheHarnessSettings is the session-start
// half of iss-2610050556323779: a hook in the person's harness settings that
// runs abcd — here the stale local build whose SubagentStop entry recreated
// ~/.abcd — is named at session start with its remedy, on stderr, with the
// counted pointer on stdout and exit 0. The notice reads only: the settings
// file is byte-identical afterwards.
func TestHookSessionStartNamesAnAbcdHookInTheHarnessSettings(t *testing.T) {
	repo, home := gitRepoNoStore(t)
	noAmbientPluginRoot(t)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(dir, "settings.json")
	stale := filepath.Join(home, "ABCDevelopment", "abcd", "bin", "abcd-darwin-arm64")
	body := `{"hooks": {"SubagentStop": [{"hooks": [{"type": "command", "command": "` + stale + ` hook subagent-stop"}]}]}}`
	if err := os.WriteFile(settings, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runSessionStart(startPayload("s1", repo), "hook", "session-start")

	if code != 0 {
		t.Errorf("a notice is not a hook failure; got exit %d (stderr %q)", code, stderr)
	}
	for _, want := range []string{"~/.claude/settings.json", "SubagentStop", "abcd-darwin-arm64 hook subagent-stop", "remove this entry", "`abcd ahoy`"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr lacks %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, home) {
		t.Errorf("the notice carries the home path:\n%s", stderr)
	}
	if !strings.Contains(stdout, "1 session-start notice(s)") || strings.Contains(stdout, "SubagentStop") {
		t.Errorf("stdout must carry the constant count only:\n%s", stdout)
	}
	if after, _ := os.ReadFile(settings); string(after) != body {
		t.Errorf("session start edited the harness settings:\n%s", after)
	}
}
