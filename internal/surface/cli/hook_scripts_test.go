package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// The manifest's command for each hook is one short line that runs a script
// under hooks/, because the host prints a hook's whole command in front of its
// message whenever the hook blocks or warns (iss-2610041345196368): an inline
// script several thousand characters long buried every refusal under a wall of
// shell. The tests that drive a hook's behaviour run the script's body; the
// tests in this file hold the wrapper itself.

// hookScriptRef matches the one shape a manifest command takes.
var hookScriptRef = regexp.MustCompile(`^sh "\$\{CLAUDE_PLUGIN_ROOT\}/hooks/([a-z-]+\.sh)"$`)

// maxHookCommand is the longest manifest command the host may print in front of
// a hook's message. The wrapper is under fifty characters.
const maxHookCommand = 64

// repoHooksDir is the committed hooks/ directory, located from this file.
func repoHooksDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to locate the test source file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "hooks"))
}

// resolveHookCommand returns the shell a manifest command runs: the body of the
// hooks/ script it names. A command that names no script is returned unchanged,
// so a test reading a hand-written fixture manifest still works.
func resolveHookCommand(t *testing.T, command string) string {
	t.Helper()
	m := hookScriptRef.FindStringSubmatch(command)
	if m == nil {
		return command
	}
	body, err := os.ReadFile(filepath.Join(repoHooksDir(t), m[1]))
	if err != nil {
		t.Fatalf("the manifest runs hooks/%s, which cannot be read: %v", m[1], err)
	}
	return string(body)
}

// copyHookScripts copies the committed hooks/*.sh into a test plugin root, as
// an install lays them beside the manifest. 0o644: the wrapper runs each script
// with sh, so a payload that loses the execute bit still runs it.
func copyHookScripts(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(repoHooksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sh") || e.Name() == "bootstrap.sh" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoHooksDir(t), e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "hooks", e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// manifestCommands decodes the committed manifest into event -> commands.
func manifestCommands(t *testing.T) map[string][]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoHooksDir(t), "hooks.json"))
	if err != nil {
		t.Fatalf("reading hooks/hooks.json: %v", err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("hooks/hooks.json does not parse: %v", err)
	}
	out := map[string][]string{}
	for event, groups := range doc.Hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				out[event] = append(out[event], h.Command)
			}
		}
	}
	return out
}

// TestHookCommandsAreShortWrappers holds the fix: every command the manifest
// declares is the short wrapper, short enough that the host's error line shows
// the hook's message rather than the hook's source, and the script it runs
// ships beside the manifest.
func TestHookCommandsAreShortWrappers(t *testing.T) {
	cmds := manifestCommands(t)
	if len(cmds) == 0 {
		t.Fatal("the manifest declares no hook commands")
	}
	for event, list := range cmds {
		for _, c := range list {
			if len(c) > maxHookCommand {
				t.Errorf("%s: the command is %d characters; the host prints it in front of every block or warning, so it must stay at most %d (move the shell into a hooks/ script)", event, len(c), maxHookCommand)
			}
			m := hookScriptRef.FindStringSubmatch(c)
			if m == nil {
				t.Errorf("%s: the command %q is not the wrapper that runs a hooks/ script", event, c)
				continue
			}
			if _, err := os.Stat(filepath.Join(repoHooksDir(t), m[1])); err != nil {
				t.Errorf("%s: the command runs hooks/%s, which is not committed: %v", event, m[1], err)
			}
		}
	}
}

// TestHookWrapperRunsTheScript drives the real wrapper end to end: from a plugin
// root holding the committed scripts and a stub binary that refuses, the guard's
// PreToolUse command exits 2 with the binary's message and nothing else.
func TestHookWrapperRunsTheScript(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	cmds := manifestCommands(t)["PreToolUse"]
	if len(cmds) != 1 {
		t.Fatalf("want one PreToolUse command, got %d", len(cmds))
	}
	root := fakePluginRoot(t, `echo "refused: one line" >&2; exit 2`)
	copyHookScripts(t, root)
	stderr, code := runShim(t, cmds[0], root, "")
	if code != 2 || strings.TrimSpace(stderr) != "refused: one line" {
		t.Fatalf("the wrapper must pass the guard's refusal through unchanged: code=%d stderr=%q", code, stderr)
	}
}
