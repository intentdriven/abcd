package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// standInHarness is a claude on PATH whose login and live launch answer as the
// body says.
func standInHarness(t *testing.T, loggedIn bool, stream string) {
	t.Helper()
	state := "false"
	if loggedIn {
		state = "true"
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "stream"), []byte(stream), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nif [ \"$1\" = auth ]; then echo '{\"loggedIn\":" + state + "}'; exit 0; fi\n" + wallHomeLine + "cat '" + filepath.Join(bin, "stream") + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// wallHomeLine has a stand-in report the shell's home the proof asks about.
const wallHomeLine = `printf '{"type":"user","message":{"content":[{"type":"tool_result","content":"abcd-home:%s"}]}}\n' "$ABCD_STAGE_HOME"` + "\n" +
	`printf '{"type":"user","message":{"content":[{"type":"tool_result","content":"abcd-env:PATH"}]}}\n'` + "\n"

const (
	wallInit    = `{"type":"system","subtype":"init","session_id":"s","model":"m","tools":["Read","Bash","mcp__abcd__approve"],"apiKeySource":"none","mcp_servers":[{"name":"abcd"}],"plugins":[]}` + "\n"
	wallRefused = `{"type":"system","subtype":"hook_response","hook_event":"PreToolUse","stdout":"Blocked by the abcd guard (git-push-force)"}` + "\n"
	wallResult  = `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s"}` + "\n"
)

// recordTerminal stands a recorder in for the loop.
func (w *driveWorld) recordTerminal(t *testing.T) {
	t.Helper()
	term := driveTerminal
	t.Cleanup(func() { driveTerminal = term })
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		w.terminalCalls++
		w.terminal = o
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
}

// TestADriveWritesTheSettingsProvesTheWallsAndThenStarts: every wall holding,
// the walled directory's settings name this binary and the loop starts.
func TestADriveWritesTheSettingsProvesTheWallsAndThenStarts(t *testing.T) {
	w := newDriveWorld(t)
	w.recordTerminal(t)
	standInHarness(t, true, wallInit+wallRefused+wallResult)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminalCalls != 1 {
		t.Fatalf("the loop starts once the walls hold: %d", w.terminalCalls)
	}
	raw, err := os.ReadFile(filepath.Join(w.profile, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), runner.SelfBinary()) || !strings.Contains(string(raw), "guard hook") {
		t.Fatalf("the settings name this binary:\n%s", raw)
	}
	if w.terminal.BeforeStage == nil {
		t.Fatal("the loop compares the settings before each stage")
	}
	if err := w.terminal.BeforeStage(); err != nil {
		t.Fatalf("what abcd wrote holds: %v", err)
	}
	if err := os.WriteFile(filepath.Join(w.profile, "settings.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := w.terminal.BeforeStage(); err == nil {
		t.Fatal("a difference stops the drive")
	}
}

// TestAWallThatDoesNotHoldRefusesNamingIt: each refused before the loop starts.
func TestAWallThatDoesNotHoldRefusesNamingIt(t *testing.T) {
	for _, c := range []struct {
		name     string
		loggedIn bool
		stream   string
		wall     string
	}{
		{"login", false, wallInit + wallRefused + wallResult, runner.WallLogin},
		{"guard", true, wallInit + wallResult, runner.WallGuard},
		{"tools", true, strings.Replace(wallInit, `"Read",`, `"Read","WebFetch",`, 1) + wallRefused + wallResult, runner.WallTools},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newDriveWorld(t)
			w.recordTerminal(t)
			standInHarness(t, c.loggedIn, c.stream)
			text := refusedDrive(t, "drain", "--drive", "--yes")
			if !strings.Contains(text, c.wall) {
				t.Fatalf("the refusal names the wall %q: %s", c.wall, text)
			}
			if w.terminalCalls != 0 {
				t.Fatal("the loop does not start")
			}
			noDrainBegun(t, w.repo)
		})
	}
}

// TestASettingsFileAPersonEditedIsReplacedAndSaid.
func TestASettingsFileAPersonEditedIsReplacedAndSaid(t *testing.T) {
	w := newDriveWorld(t)
	w.recordTerminal(t)
	standInHarness(t, true, wallInit+wallRefused+wallResult)
	if err := os.WriteFile(filepath.Join(w.profile, "settings.json"), []byte(`{"model":"mine"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !strings.Contains(errb.String(), "replaced") {
		t.Fatalf("the replacement is said:\n%s", errb.String())
	}
	raw, _ := os.ReadFile(filepath.Join(w.profile, "settings.json"))
	if strings.Contains(string(raw), "mine") {
		t.Fatalf("the edit is gone:\n%s", raw)
	}
}

// TestAStagesFileToolsWriteOnlyWhereTheStageWorks: on a walled launch the guard
// holds a file tool to the stage's working tree and run directories, and to
// nothing protected inside them. No other session is judged this way.
func TestAStagesFileToolsWriteOnlyWhereTheStageWorks(t *testing.T) {
	tree, run := t.TempDir(), t.TempDir()
	stage := tree + string(os.PathListSeparator) + run
	payload := func(path string) string {
		raw, _ := json.Marshal(map[string]any{"tool_name": "Write", "cwd": tree, "tool_input": map[string]string{"file_path": path}})
		return string(raw)
	}
	home := t.TempDir()
	for _, c := range []struct {
		name, path string
		denied     bool
	}{
		{"its tree", filepath.Join(tree, "a.go"), false},
		{"a run directory", filepath.Join(run, "receipt.json"), false},
		{"the person's machine configuration", abcdhome.Path(home, "config.json"), true},
		{"a walled directory", abcdhome.Path(home, "profiles", "walled", "settings.json"), true},
		{"a start-up file", filepath.Join(home, ".zshrc"), true},
		{"out through a parent", filepath.Join(tree, "..", "elsewhere"), true},
		{"the tree's git directory", filepath.Join(tree, ".git", "config"), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(runner.StageWritesEnv, stage)
			out, _ := runHook(t, payload(c.path), "guard", "hook")
			if got := strings.Contains(out, `"deny"`); got != c.denied {
				t.Fatalf("denied = %v, want %v: %s", got, c.denied, out)
			}
		})
	}
	t.Setenv(runner.StageWritesEnv, "")
	if out, _ := runHook(t, payload(filepath.Join(home, ".zshrc")), "guard", "hook"); strings.Contains(out, "writes only") {
		t.Fatalf("another session is not judged this way: %s", out)
	}
}
