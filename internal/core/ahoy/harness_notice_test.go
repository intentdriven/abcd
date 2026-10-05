package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHarnessNotice is the session-start line: silent with nothing to say,
// one finding named with its remedy, several counted with a pointer to the
// board, and a dangling abcd status line — which detection reports through its
// own repair gap — named here too. It reads only; the settings file is
// untouched.
func TestHarnessNotice(t *testing.T) {
	t.Run("silent when nothing calls abcd", func(t *testing.T) {
		setupHermetic(t)
		homeHarnessSettings(t, `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "echo hi"}]}]}}`)
		if n := HarnessNotice(); n != "" {
			t.Errorf("notice = %q, want silence", n)
		}
	})
	t.Run("silent with no harness", func(t *testing.T) {
		setupHermetic(t)
		if n := HarnessNotice(); n != "" {
			t.Errorf("notice = %q, want silence", n)
		}
	})
	t.Run("one stray hook", func(t *testing.T) {
		home, _ := setupHermetic(t)
		stale := filepath.Join(home, "ABCDevelopment", "abcd", "bin", "abcd-darwin-arm64")
		p := homeHarnessSettings(t, `{"hooks": {"SubagentStop": [{"hooks": [{"type": "command", "command": "`+stale+` hook subagent-stop"}]}]}}`)
		before, _ := os.ReadFile(p)
		n := HarnessNotice()
		for _, want := range []string{"~/.claude/settings.json", "SubagentStop", "~/ABCDevelopment/abcd/bin/abcd-darwin-arm64 hook subagent-stop", "remove this entry", "`abcd ahoy`"} {
			if !strings.Contains(n, want) {
				t.Errorf("notice lacks %q: %q", want, n)
			}
		}
		if strings.Contains(n, home) || strings.Contains(n, "\n") {
			t.Errorf("notice is not one home-free line: %q", n)
		}
		if after, _ := os.ReadFile(p); string(after) != string(before) {
			t.Error("the notice wrote the settings file")
		}
	})
	t.Run("a dangling status line", func(t *testing.T) {
		setupHermetic(t)
		homeHarnessSettings(t, `{"statusLine": {"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}}`)
		n := HarnessNotice()
		for _, want := range []string{"status line", "gone", "abcd ahoy install"} {
			if !strings.Contains(n, want) {
				t.Errorf("notice lacks %q: %q", want, n)
			}
		}
	})
	t.Run("an untrusted status line", func(t *testing.T) {
		_, pluginRoot := setupHermetic(t)
		homeHarnessSettings(t, `{"statusLine": {"type": "command", "command": "'`+filepath.Join(pluginRoot, "abcd")+`' statusline"}}`)
		n := HarnessNotice()
		for _, want := range []string{"status line", "fails the trust checks", "path-entry", "abcd ahoy install"} {
			if !strings.Contains(n, want) {
				t.Errorf("notice lacks %q: %q", want, n)
			}
		}
	})
	t.Run("several findings", func(t *testing.T) {
		setupHermetic(t)
		homeHarnessSettings(t, `{
  "hooks": {"SessionEnd": [{"hooks": [{"type": "command", "command": "abcd hook session-end"}]}]},
  "statusLine": {"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}
}`)
		n := HarnessNotice()
		for _, want := range []string{"2 abcd entries", "hooks.SessionEnd", "statusLine", "`abcd ahoy`"} {
			if !strings.Contains(n, want) {
				t.Errorf("notice lacks %q: %q", want, n)
			}
		}
	})
}
