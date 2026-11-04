package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The flag, the preconditions and the consent of `abcd drain --drive`
// (spc-2610230813348813 step 3).

// driveRoutes enables the claude runner on the machine.
const driveRoutes = `{"runner":{"claude":{}}}`

// driveWorld is a repository with its drain rule, a home with the claude runner
// enabled and one walled directory, a stand-in harness on PATH, and an
// environment holding no session marker and no credential.
type driveWorld struct {
	repo, home, profile string
	terminalCalls       int
	terminal            loop.TerminalOptions
}

func newDriveWorld(t *testing.T) *driveWorld {
	t.Helper()
	w := &driveWorld{repo: drainRuleRepo(t, drainrule.ProposalFrontmatter()), home: t.TempDir()}
	t.Setenv("HOME", w.home)
	for _, k := range driveSessionMarkers {
		t.Setenv(k, "")
	}
	for _, k := range runner.CredentialVars() {
		t.Setenv(k, "")
	}
	w.machineConfig(t, driveRoutes)
	w.profile = abcdhome.Path(w.home, "profiles", "walled")
	if err := os.MkdirAll(w.profile, 0o700); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return w
}

func (w *driveWorld) machineConfig(t *testing.T, body string) {
	t.Helper()
	path := abcdhome.Path(w.home, "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// passWalls proves every wall and stands a recorder in for the loop.
func (w *driveWorld) passWalls(t *testing.T) {
	t.Helper()
	walls, term := driveWalls, driveTerminal
	t.Cleanup(func() { driveWalls, driveTerminal = walls, term })
	driveWalls = driveWallChecks{
		static: func(context.Context, runner.Profile) ([]string, error) { return nil, nil },
		live:   func(context.Context, runner.Profile) error { return nil },
	}
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		w.terminalCalls++
		w.terminal = o
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
}

// refused runs args and returns the refusal, failing unless it is exit 2.
func refusedDrive(t *testing.T, args ...string) string {
	t.Helper()
	var out, errb strings.Builder
	code := Run(args, &out, &errb)
	if code != 2 {
		t.Fatalf("%v exits %d, want 2:\n%s%s", args, code, out.String(), errb.String())
	}
	return out.String() + errb.String()
}

// noDrainBegun fails when anything was written to the drain's state.
func noDrainBegun(t *testing.T, repo string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(loop.DrainStateRel))); err == nil {
		t.Fatal("a refused drive begins no drain")
	}
}

func TestADriveFromInsideAnAgentsSessionRefuses(t *testing.T) {
	for _, marker := range []string{"CLAUDECODE", "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_ENTRYPOINT"} {
		t.Run(marker, func(t *testing.T) {
			w := newDriveWorld(t)
			w.passWalls(t)
			t.Setenv(marker, "1")
			text := refusedDrive(t, "drain", "--drive", "--yes")
			if !strings.Contains(text, "agent's own session") {
				t.Fatalf("the refusal says why: %s", text)
			}
			noDrainBegun(t, w.repo)
			if w.terminalCalls != 0 {
				t.Fatal("nothing runs")
			}
		})
	}
}

func TestACredentialInTheEnvironmentRefusesNamingTheBudgetIntent(t *testing.T) {
	for _, name := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN"} {
		t.Run(name, func(t *testing.T) {
			w := newDriveWorld(t)
			w.passWalls(t)
			t.Setenv(name, "a-value-never-shown")
			text := refusedDrive(t, "drain", "--drive", "--yes")
			for _, want := range []string{name, runner.BudgetIntent} {
				if !strings.Contains(text, want) {
					t.Errorf("the refusal %q does not name %q", text, want)
				}
			}
			if strings.Contains(text, "a-value-never-shown") {
				t.Fatal("a refusal names the variable, never its value")
			}
			noDrainBegun(t, w.repo)
		})
	}
}

func TestARunnerNotEnabledRefuses(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.machineConfig(t, `{}`)
	if text := refusedDrive(t, "drain", "--drive", "--yes"); !strings.Contains(text, "runner.claude") {
		t.Fatalf("the refusal names the setting: %s", text)
	}
	noDrainBegun(t, w.repo)
}

func TestAHarnessNotAdmittedRefuses(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	t.Setenv("PATH", t.TempDir())
	if text := refusedDrive(t, "drain", "--drive", "--yes"); !strings.Contains(text, "claude") || !strings.Contains(text, "PATH") {
		t.Fatalf("the refusal names the harness: %s", text)
	}
	noDrainBegun(t, w.repo)
}

func TestAJudgeRoutedToOpencodeRefuses(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.machineConfig(t, `{"roles":{"drain-judge":{"runner":"opencode"}},"runner":{"claude":{},"opencode":{}}}`)
	if text := refusedDrive(t, "drain", "--drive", "--yes"); !strings.Contains(text, "opencode") {
		t.Fatalf("the refusal names opencode: %s", text)
	}
	noDrainBegun(t, w.repo)
}

func TestARolePinnedToTheHostRefuses(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	if err := os.WriteFile(filepath.Join(w.repo, ".abcd", "config.json"), []byte(`{"roles":{"implementer":{"runner":"host"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	text := refusedDrive(t, "drain", "--drive", "--yes")
	if !strings.Contains(text, "implementer") || !strings.Contains(text, "host") {
		t.Fatalf("the refusal names the role: %s", text)
	}
	noDrainBegun(t, w.repo)
}

func TestNoWalledDirectoryOrSeveralRefuses(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	if err := os.Remove(w.profile); err != nil {
		t.Fatal(err)
	}
	if text := refusedDrive(t, "drain", "--drive", "--yes"); !strings.Contains(text, "~/.abcd.noindex/profiles/") {
		t.Fatalf("the refusal names the directory to make: %s", text)
	}
	for _, n := range []string{"one", "two"} {
		if err := os.MkdirAll(abcdhome.Path(w.home, "profiles", n), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if text := refusedDrive(t, "drain", "--drive", "--yes"); !strings.Contains(text, runner.ProfilesIntent) {
		t.Fatalf("the refusal names the profiles intent: %s", text)
	}
	noDrainBegun(t, w.repo)
}

// TestAWallNotProvedRefusesNamingIt: with the production check, the drive does
// not start until a wall has been proved.
func TestAWallNotProvedRefusesNamingIt(t *testing.T) {
	w := newDriveWorld(t)
	text := refusedDrive(t, "drain", "--drive", "--yes")
	if !strings.Contains(text, "wall") {
		t.Fatalf("the refusal names the wall: %s", text)
	}
	noDrainBegun(t, w.repo)
}

func TestADriveWithNoTerminalAndNoYesRefuses(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	text := refusedDrive(t, "drain", "--drive")
	if !strings.Contains(text, "--yes") {
		t.Fatalf("the refusal names the scripted yes: %s", text)
	}
	if w.terminalCalls != 0 {
		t.Fatal("nothing runs without a yes")
	}
	noDrainBegun(t, w.repo)
}

// withTerminal makes the drive read its stdin as a terminal's.
func withTerminal(t *testing.T) {
	t.Helper()
	was := connectIsTerminal
	t.Cleanup(func() { connectIsTerminal = was })
	connectIsTerminal = func(any) bool { return true }
}

func TestTheConsentNamesTheCapTheEstimateTheRunnerAndTheRoute(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive")
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	for _, want := range []string{"cap:", "8 lane", "agent run", "runner: claude", "subscription", "~/.abcd.noindex/profiles/walled", "Start?"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the consent does not name %q:\n%s", want, out)
		}
	}
	if w.terminalCalls != 0 {
		t.Fatal("a no starts nothing")
	}
	noDrainBegun(t, w.repo)
}

func TestAYesAtTheTerminalStartsTheLoopOnTheWalledRoute(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	if out, err := runCLIPipedStdin(t, "y\n", "drain", "--drive", "--max", "3"); err != nil {
		t.Fatalf("drive: %v\n%s", err, out)
	}
	if w.terminalCalls != 1 {
		t.Fatalf("the loop starts once, got %d", w.terminalCalls)
	}
	if w.terminal.Drain.Max != 3 {
		t.Fatalf("the cap reaches the drain: %d", w.terminal.Drain.Max)
	}
	if got := w.terminal.Runners.Config.RouteFor(loop.RoleImplementer).Runner; got != runner.Claude {
		t.Fatalf("every role is on the walled launch: %q", got)
	}
}

func TestDashYesIsTheScriptedYesAndTheDefaultCapIsEight(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminalCalls != 1 || w.terminal.Drain.Max != 8 {
		t.Fatalf("the cap defaults to eight: %d calls, max %d", w.terminalCalls, w.terminal.Drain.Max)
	}
	if strings.Contains(out.String(), "Start?") {
		t.Fatalf("the scripted yes asks nothing:\n%s", out.String())
	}
}

func TestTheDrivesFlagsRefuseWhatTheyCannotTake(t *testing.T) {
	newDriveWorld(t)
	for _, args := range [][]string{
		{"drain", "--yes"},
		{"drain", "--drive", "--dry-run"},
		{"drain", "--drive", "--judgement", "x.json"},
	} {
		refusedDrive(t, args...)
	}
}
