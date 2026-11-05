package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// approveLine is the approver's judgement of one shell line.
func approveLine(f *guard.Files, line string) guard.Approval {
	raw, _ := json.Marshal(map[string]string{"command": line})
	a, _ := guard.ApprovePrompt(guard.Loaded{Registry: guard.Defaults()}, f, "Bash", raw)
	return a
}

// TestAStagesGuardFailsClosed: on a walled launch, whatever the guard cannot
// judge is denied, not let through with a warning; another session keeps its
// warning.
func TestAStagesGuardFailsClosed(t *testing.T) {
	tree := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Symlink(outside, filepath.Join(tree, "dangling")); err != nil {
		t.Fatal(err)
	}
	write := func(path string) string {
		raw, _ := json.Marshal(map[string]any{"tool_name": "Write", "cwd": tree, "tool_input": map[string]string{"file_path": path}})
		return string(raw)
	}
	for name, stdin := range map[string]string{
		"an oversized payload": strings.Repeat(" ", maxHookStdinBytes+2),
		"unreadable JSON":      `{"tool_name":"Write",`,
		"no file named":        write(""),
		"a relative path":      write("a.go"),
		"a tilde":              write("~/a.go"),
		"a dangling link":      write(filepath.Join(tree, "dangling")),
		"a shell with no text": `{"tool_name":"Bash","cwd":"` + tree + `","tool_input":{"command":"   "}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(runner.StageWritesEnv, tree)
			out, _, _ := runHookAllowingFailure(stdin, "guard", "hook")
			if !strings.Contains(out, `"deny"`) {
				t.Fatalf("not denied: %q", out)
			}
		})
	}
	t.Setenv(runner.StageWritesEnv, "")
	if out, _, _ := runHookAllowingFailure(strings.Repeat(" ", maxHookStdinBytes+2), "guard", "hook"); strings.Contains(out, `"deny"`) {
		t.Fatalf("another session keeps its warning: %q", out)
	}
}

// TestAStagesRulesAreTheBases: under a walled launch the rules a hook injects are
// those of the lane's base, not what the lane wrote.
func TestAStagesRulesAreTheBases(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/rules.json", `{"schema_version":1,"disabled":false,"domains":{"BASEONE":{"recall":["word"],"rules":["the base's"]}}}`)
	repo.Commit("the base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Write(".abcd/rules.json", `{"schema_version":1,"disabled":false,"domains":{"LANEONE":{"recall":["word"],"rules":["the lane's"]}}}`)
	t.Setenv(runner.StageBaseEnv, base)
	rs, err := rulesForHook(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rs.Domains["LANEONE"]; ok {
		t.Fatal("a rule the lane wrote reached the hook")
	}
	t.Setenv(runner.StageBaseEnv, "")
	if rs, err = rulesForHook(repo.Root()); err != nil {
		t.Fatal(err)
	}
	if _, ok := rs.Domains["LANEONE"]; !ok {
		t.Fatal("another session reads the working tree's")
	}
}

// TestAStagesHookReadsTheStagesHome: the hook judges a shell line as the stage's
// shell runs it.
func TestAStagesHookReadsTheStagesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	profile := abcdhome.Path(home, "profiles", "walled")
	t.Setenv(runner.StageHomeEnv, abcdhome.Path(home, "stage-homes", "walled"))
	t.Setenv(runner.StageProtectEnv, profile)
	f := stageFilesFromEnv(guard.NewFiles(t.TempDir()))
	if a := approveLine(f, "claude --version"); a.Allow {
		t.Fatal("the stage's reading refuses a further agent")
	}
}

// TestAPreflightRunsInTheStagesEnvironment.
func TestAPreflightRunsInTheStagesEnvironment(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "env")
	was := drivePreflightProgram
	t.Cleanup(func() { drivePreflightProgram = was })
	drivePreflightProgram = func() (string, []string, error) {
		return "/bin/sh", []string{"-c", "/usr/bin/env > '" + out + "'"}, nil
	}
	w := newDriveWorld(t)
	t.Setenv("GH_TOKEN", "a-value-never-passed")
	t.Setenv("SSH_AUTH_SOCK", "a-value-never-passed")
	p, err := runner.DriveProfile(w.home)
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	if err := drivePreflight(os.Stderr, p)(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "a-value-never-passed") {
		t.Fatalf("something outside the list reached the preflight:\n%s", raw)
	}
	for _, want := range []string{"HOME=" + p.ShellHome + "\n", abcdhome.PersonHomeEnv + "=" + w.home + "\n"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the environment lacks %q:\n%s", want, raw)
		}
	}
}

// TestALaneThatChangesItsGatesIsRefused: no flag opens it; a person's own session
// handles such a change.
func TestALaneThatChangesItsGatesIsRefused(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	if err := driveGateCheck()(repo.Root(), base); err != nil {
		t.Fatalf("nothing changed: %v", err)
	}
	repo.Write("a.go", "package a\n\n// more\n")
	repo.Commit("code")
	if err := driveGateCheck()(repo.Root(), base); err != nil {
		t.Fatalf("code is no gate: %v", err)
	}
	for _, path := range []string{"Makefile", "scripts/check.sh"} {
		base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
		repo.Write(path, "x\n")
		repo.Commit(path)
		var refused *loop.PreflightRefused
		if err := driveGateCheck()(repo.Root(), base); !errors.As(err, &refused) || !strings.Contains(refused.Reason, path) || !strings.Contains(refused.Reason, "a person's own session") {
			t.Fatalf("%s: %v", path, err)
		}
	}
	var refused *loop.PreflightRefused
	if err := driveGateCheck()(repo.Root(), "not-a-commit"); !errors.As(err, &refused) {
		t.Fatalf("a diff that cannot be read refuses: %v", err)
	}
}

// TestAMergedDefaultBranchIsNotTheLanesChange: read from what the lane merged, a
// Makefile the default branch changed hands nothing back.
func TestAMergedDefaultBranchIsNotTheLanesChange(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("base")
	repo.Git("checkout", "-q", "-b", "lane")
	repo.Write("b.go", "package a\n")
	repo.Commit("the lane's work")
	repo.Git("checkout", "-q", "-")
	repo.Write("Makefile", "x\n")
	repo.Commit("the default branch's")
	merged := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Git("checkout", "-q", "lane")
	repo.Git("merge", "-q", "--no-edit", merged)
	if err := driveGateCheck()(repo.Root(), merged); err != nil {
		t.Fatalf("the lane did not change the Makefile: %v", err)
	}
}

// TestThereIsNoFlagThatOpensAGate: a drive has no way to open a gate file.
func TestThereIsNoFlagThatOpensAGate(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	if text := refusedDrive(t, "drain", "--drive", "--yes", "--trust-lane-gates"); !strings.Contains(text, "unknown flag") {
		t.Fatalf("the flag is gone: %s", text)
	}
}

// TestAPreflightLeavesNoProcessBehind: on success and on failure.
func TestAPreflightLeavesNoProcessBehind(t *testing.T) {
	w := newDriveWorld(t)
	p, err := runner.DriveProfile(w.home)
	if err != nil {
		t.Fatal(err)
	}
	p.Repo = w.repo
	if _, err := runner.WriteWallSettings(p, runner.SelfBinary()); err != nil {
		t.Fatal(err)
	}
	was := drivePreflightProgram
	t.Cleanup(func() { drivePreflightProgram = was })
	for name, exit := range map[string]string{"success": "0", "failure": "1"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			left := filepath.Join(dir, "left")
			drivePreflightProgram = func() (string, []string, error) {
				return "/bin/sh", []string{"-c", "/bin/sleep 60 >/dev/null 2>&1 & echo $! > '" + left + "'; exit " + exit}, nil
			}
			_ = drivePreflight(os.Stderr, p)(context.Background(), dir)
			pid := waitPidFile(t, left)
			if !pidGone(pid, 5*time.Second) {
				t.Fatalf("process %d outlived the preflight", pid)
			}
		})
	}
}

// TestANoAtTheConsentWritesNothing: an n stops nothing and touches nothing.
func TestANoAtTheConsentWritesNothing(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	wrote := false
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		wrote = true
		return nil, nil
	}
	if out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if wrote {
		t.Fatal("a no wrote the settings")
	}
}

// TestTheEndNamesWhatStoppedItAndWhatItPassedOver.
func TestTheEndNamesWhatStoppedItAndWhatItPassedOver(t *testing.T) {
	var b strings.Builder
	renderDriveEnd(&b, loop.TerminalResult{Ended: loop.TerminalWall, Why: "abcd's binary changed while the drive ran",
		PassedOver: []loop.Excluded{{ID: "iss-7", Check: loop.CheckNeedsPerson, Reason: "it needs a person's own session"}}})
	for _, want := range []string{"abcd's binary changed while the drive ran", "iss-7", "a person's own session"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the end does not name %q:\n%s", want, b.String())
		}
	}
	for _, never := range []string{"wall"} {
		if strings.Contains(b.String(), never) {
			t.Errorf("the end says %q:\n%s", never, b.String())
		}
	}
}

// TestTheGateReadsThePersonsHomeUnderAStagesHome: the hold lands in the person's
// folder, and the stage's home gains nothing.
func TestTheGateReadsThePersonsHomeUnderAStagesHome(t *testing.T) {
	snap := machine(3, 3, 3)
	waitFixture(t, snap)
	person, _ := os.UserHomeDir()
	stage := t.TempDir()
	t.Setenv(abcdhome.PersonHomeEnv, person)
	t.Setenv(abcdhome.StageHomeEnv, stage)
	t.Setenv("HOME", stage)
	if code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m"); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if _, err := os.Stat(abcdhome.Path(person, implement.WaitHoldFileName)); err != nil {
		t.Fatalf("the hold is in the person's folder: %v", err)
	}
	if _, err := os.Stat(abcdhome.Path(stage)); err == nil {
		t.Fatal("the stage's home gained abcd's folder")
	}
}

// TestASecondDriveOnACheckoutIsRefused: exit 3, before any proof is spent.
func TestASecondDriveOnACheckoutIsRefused(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	proved := false
	walls := driveWalls
	driveWalls.live = func(ctx context.Context, p runner.Profile) error {
		proved = true
		return walls.live(ctx, p)
	}
	lock := filepath.Join(w.repo, filepath.FromSlash(driveLockRel))
	if err := os.MkdirAll(filepath.Dir(lock), 0o700); err != nil {
		t.Fatal(err)
	}
	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(lock, time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if code != 3 || proved || w.terminalCalls != 0 {
		t.Fatalf("exit %d, proved %v, %d runs:\n%s%s", code, proved, w.terminalCalls, out.String(), errb.String())
	}
}

// TestADriveTakesThePaceTheOthersTake: --pace, --sub-agents and --fix-rounds reach
// the moves.
func TestADriveTakesThePaceTheOthersTake(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--pace", "30/5", "--sub-agents", "1", "--fix-rounds", "2"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	o := w.terminal.Options
	if o.Pace == nil || *o.Pace != "30/5" || o.SubAgents == nil || *o.SubAgents != "1" || o.FixRounds == nil || *o.FixRounds != "2" {
		t.Fatalf("the flags were dropped: %+v", o)
	}
}

// TestADrivesJSONIsJSON: the consent goes to standard error under --json.
func TestADrivesJSONIsJSON(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--json"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !json.Valid([]byte(out.String())) {
		t.Fatalf("standard output is not JSON:\n%s", out.String())
	}
	if !strings.Contains(errb.String(), "cap:") {
		t.Fatalf("the consent is on standard error:\n%s", errb.String())
	}
}

// TestAConflictingCapIsRefusedBeforeTheProof: nothing is spent on a --max the
// drain in progress would refuse.
func TestAConflictingCapIsRefusedBeforeTheProof(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	proved := false
	driveWalls.live = func(context.Context, runner.Profile) error {
		proved = true
		return nil
	}
	state := filepath.Join(w.repo, filepath.FromSlash(loop.DrainStateRel))
	if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(loop.DrainState{SchemaVersion: loop.DrainSchemaVersion, Max: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	text := refusedDrive(t, "drain", "--drive", "--yes", "--max", "5")
	if proved || !strings.Contains(text, "--max") {
		t.Fatalf("proved %v: %s", proved, text)
	}
}

// TestHowADriveEndsIsInItsExit: a script can tell a clean finish from a drive that
// stopped short; the run and lane are named even when a move refused.
func TestHowADriveEndsIsInItsExit(t *testing.T) {
	for ended, want := range map[string]int{
		loop.DrainStoppedCap:   0,
		loop.DrainStoppedEmpty: 0,
		loop.TerminalStopped:   3,
		loop.TerminalWall:      2,
		loop.TerminalOutage:    3,
		loop.TerminalLimited:   3,
		loop.TerminalWaited:    3,
		loop.TerminalHalted:    3,
		loop.TerminalKilled:    driveKilledExit,
	} {
		t.Run(ended, func(t *testing.T) {
			w := newDriveWorld(t)
			w.passWalls(t)
			driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
				return loop.TerminalResult{Ended: ended}, nil
			}
			var out, errb strings.Builder
			if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != want {
				t.Fatalf("exit %d, want %d:\n%s%s", code, want, out.String(), errb.String())
			}
		})
	}
	w := newDriveWorld(t)
	w.passWalls(t)
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.TerminalRefused, Run: "run-1", Lane: "lane-1", Issue: "iss-1", Resume: loop.TerminalResumeCommand}, errors.New("a move refused")
	}
	var out, errb strings.Builder
	Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	if text := out.String() + errb.String(); !strings.Contains(text, "run-1") || !strings.Contains(text, loop.TerminalResumeCommand) {
		t.Fatalf("a refused end still names what it left:\n%s", text)
	}
}

// TestThePublicTextSaysWhatAProfileIs: a profile is a separate login and settings
// on one account, and a stage's shell reads what the account can.
func TestThePublicTextSaysWhatAProfileIs(t *testing.T) {
	here, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(here, "..", "..", "..")
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	consent, err := runCLIPipedStdin(t, "n\n", "drain", "--drive")
	if err != nil {
		t.Fatalf("%v\n%s", err, consent)
	}
	help, herr := runCLIPipedStdin(t, "", "drain", "--help")
	if herr != nil {
		t.Fatal(herr)
	}
	page, err := os.ReadFile(filepath.Join(root, "commands", "drain.md"))
	if err != nil {
		t.Fatal(err)
	}
	brief, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "04-surfaces", "35-drain.md"))
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"consent": string(consent), "help": string(help), "page": string(page), "brief": string(brief)} {
		// This world's own directory is called walled; the consent names it.
		text = strings.ReplaceAll(text, abcdhome.Display("profiles", "walled"), "")
		flat := strings.Join(strings.Fields(text), " ")
		for _, want := range []string{"separate login and settings", "reads what this account can"} {
			if !strings.Contains(flat, want) {
				t.Errorf("the %s does not say %q", name, want)
			}
		}
		for _, never := range []string{"sandboxed", "isolated", "walled", "proves the wall", "carries only the variables"} {
			if strings.Contains(flat, never) {
				t.Errorf("the %s says %q", name, never)
			}
		}
	}
}

// TestTheHooksStandAsideOnAStage: the away mode, the stop hook, the checklist and
// recall do not steer an agent of the drive's. Each first shows it speaks to any
// other session.
func TestTheHooksStandAsideOnAStage(t *testing.T) {
	t.Run("away", func(t *testing.T) {
		root := awayCheckout(t)
		setAway(t, "--until", "09:00")
		start := func() string {
			so, _, err := runSplit(t, awayStartPayload("s-start", root, "startup"), "hook", "session-start")
			if err != nil {
				t.Fatal(err)
			}
			return so
		}
		if !strings.Contains(start(), "## AWAY") {
			t.Fatal("another session is told it is away")
		}
		t.Setenv(runner.StageWritesEnv, root)
		if so := start(); strings.Contains(so, "## AWAY") {
			t.Fatalf("the away block reached a stage:\n%s", so)
		}
	})
	t.Run("stop", func(t *testing.T) {
		root := awayCheckout(t)
		pinStopClock(t, stopTestNow, time.UTC)
		owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
		fakeOwed(t, &owed)
		setAwayAt(t, root, stopTestNow, false)
		out, _, _ := runStop(t, stopHookPayload(t, "s1", root, false, "Done.", "*/10 * * * *"))
		if out.Decision != "block" {
			t.Fatal("another session's stop is blocked")
		}
		t.Setenv(runner.StageWritesEnv, root)
		if _, so, _ := runStop(t, stopHookPayload(t, "s2", root, false, "Done.", "*/10 * * * *")); so != "" {
			t.Fatalf("the stop hook steered a stage: %q", so)
		}
	})
	t.Run("recall", func(t *testing.T) {
		t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
		_, repo := memoryFixture(t)
		id := addRaceNote(t, "session-one")
		rs, err := rules.Load(repo)
		if err != nil {
			t.Fatal(err)
		}
		recalled := func() []string {
			var w strings.Builder
			return recallMemory(&w, rs, repo, "why are the tests failing?", nil, rules.InjectionBudgetBytes, promptRoot(rootLookup{}, repo), rules.BracketFresh).Recalled
		}
		if got := recalled(); !slices.Contains(got, id) {
			t.Fatalf("another session recalls its notes: %v", got)
		}
		t.Setenv(runner.StageWritesEnv, repo)
		if got := recalled(); len(got) != 0 {
			t.Fatalf("recall reached a stage: %v", got)
		}
	})
	t.Run("checklist", func(t *testing.T) {
		root := checklistCheckout(t)
		mustChecklist(t, "add", "Fix the parser's error path")
		if sessionStartChecklist(root, 4000) == "" {
			t.Fatal("another session is shown its checklist")
		}
		t.Setenv(runner.StageWritesEnv, root)
		if got := sessionStartChecklist(root, 4000); got != "" {
			t.Fatalf("the checklist reached a stage: %q", got)
		}
	})
}

// TestAGateChangeIsReadExactly: odd names and renames are read as they are, the
// configured hooks count, and the path is quoted as data.
func TestAGateChangeIsReadExactly(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("scripts/old.sh", "x\n")
	repo.Git("config", "core.hooksPath", "tools/hooks")
	repo.Commit("base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Git("mv", "scripts/old.sh", "elsewhere.sh")
	repo.Commit("rename out of the gate")
	var refused *loop.PreflightRefused
	if err := driveGateCheck()(repo.Root(), base); !errors.As(err, &refused) {
		t.Fatalf("a rename out of a gate path is a change to it: %v", err)
	}
	base = strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Write("tools/hooks/commit-msg", "x\n")
	repo.Commit("the configured hooks")
	if err := driveGateCheck()(repo.Root(), base); !errors.As(err, &refused) {
		t.Fatalf("the configured hooks are gates: %v", err)
	}
	if !strings.Contains(refused.Reason, strconv.Quote("tools/hooks/commit-msg")) {
		t.Fatalf("the path is quoted: %s", refused.Reason)
	}
	if !strings.Contains(refused.Reason, "a person's own session") {
		t.Fatalf("the reason names who handles it: %s", refused.Reason)
	}
}

// TestAGateIssueIsPassedOverForAPersonsOwnSession.
func TestAGateIssueIsPassedOverForAPersonsOwnSession(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Git("config", "core.hooksPath", "tools/hooks")
	pass := drivePassOver(repo.Root())
	for _, found := range []string{"Makefile", "scripts/check.sh:12", "tools/hooks/pre-push", ".githooks/commit-msg"} {
		if why := pass("iss-1", found); !strings.Contains(why, "a person's own session") {
			t.Errorf("%s: %q", found, why)
		}
	}
	if why := pass("iss-1", "internal/a.go"); why != "" {
		t.Errorf("code is opened: %q", why)
	}
}

// TestTheConsentSaysALaneOnAGateIsHandedBack.
func TestTheConsentSaysALaneOnAGateIsHandedBack(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "a person's own session") {
		t.Fatalf("the consent does not say who handles it:\n%s", out.String())
	}
}

// TestASecondDriveTouchesNothing: the lock comes before anything is written.
func TestASecondDriveTouchesNothing(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	wrote := false
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		wrote = true
		return nil, nil
	}
	lock := filepath.Join(w.repo, filepath.FromSlash(driveLockRel))
	if err := os.MkdirAll(filepath.Dir(lock), 0o700); err != nil {
		t.Fatal(err)
	}
	held, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(lock, time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if code != 3 || wrote {
		t.Fatalf("exit %d, wrote %v", code, wrote)
	}
}

// TestADriveThatHeldAnIssueBackIsNoCleanFinish.
func TestADriveThatHeldAnIssueBackIsNoCleanFinish(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty, HeldBack: []string{"iss-7"}}, nil
	}
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	if code != 3 || !strings.Contains(out.String(), "iss-7") {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
}
