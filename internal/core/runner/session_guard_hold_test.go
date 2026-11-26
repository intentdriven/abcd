package runner

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSessionGuardHoldsAKilledGuard: the session's PreToolUse command holds the
// tool call (exit 2) when the guard is stopped by a signal, as the plugin's
// hook script does, and passes the guard's own 0, 1 and 2 through unchanged.
func TestSessionGuardHoldsAKilledGuard(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"allowed", "exit 0", 0},
		{"faulted", "exit 1", 1},
		{"refused", "exit 2", 2},
		{"killed", "kill -9 $$", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "abcd")
			if err := os.WriteFile(bin, []byte("#!/bin/sh\ncat >/dev/null\n"+c.body+"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("/bin/sh", "-c", preToolUseCommand(t, Session{Bin: bin, Hooks: SessionHooks()}))
			cmd.Stdin = strings.NewReader(`{"tool_name":"Bash"}`)
			err := cmd.Run()
			got := 0
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				got = ee.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("exit %d, want %d", got, c.want)
			}
		})
	}
}

// preToolUseCommand is the one command the session's settings wire to PreToolUse.
func preToolUseCommand(t *testing.T, s Session) string {
	t.Helper()
	var v struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}
	if err := json.Unmarshal(s.Settings(), &v); err != nil {
		t.Fatal(err)
	}
	g := v.Hooks["PreToolUse"]
	if len(g) != 1 || len(g[0].Hooks) != 1 {
		t.Fatalf("PreToolUse is wired as %+v; want one command", g)
	}
	return g[0].Hooks[0].Command
}
