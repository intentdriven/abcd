package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
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
	for _, k := range runner.HeldCredentials() {
		t.Setenv(k, "")
	}
	w.machineConfig(t, driveRoutes)
	for _, kv := range [][2]string{{"user.name", "Pat Example"}, {"user.email", "pat@example.invalid"}} {
		cmd := exec.Command("git", "-C", w.repo, "config", kv[0], kv[1])
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
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
		static:   func(context.Context, runner.Profile) ([]string, error) { return nil, nil },
		live:     func(context.Context, runner.Profile) error { return nil },
		settings: func(runner.Profile) error { return nil },
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
	// git stays reachable, and no claude beside it.
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH")
	}
	bin := t.TempDir()
	if err := os.Symlink(git, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
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
	if text := refusedDrive(t, "drain", "--drive", "--yes"); !strings.Contains(text, abcdhome.Display("profiles")+"/") {
		t.Fatalf("the refusal names the directory to make: %s", text)
	}
	for _, n := range []string{"one", "two"} {
		if err := os.MkdirAll(abcdhome.Path(w.home, "profiles", n), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	text := refusedDrive(t, "drain", "--drive", "--yes")
	for _, want := range []string{"one", "two", "--profile <name>"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal %q does not name %q", text, want)
		}
	}
	noDrainBegun(t, w.repo)
}

// twoProfiles is a world holding the directories alpha and beta, and no other.
func (w *driveWorld) twoProfiles(t *testing.T) {
	t.Helper()
	if err := os.Remove(w.profile); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(abcdhome.Path(w.home, "profiles", n), 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDriveProfileRunsOnTheDirectoryItNames: the checks and the loop, a run's record
// and its probe, are each on the named directory.
func TestDriveProfileRunsOnTheDirectoryItNames(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.twoProfiles(t)
	var walled []string
	driveWalls.static = func(_ context.Context, p runner.Profile) ([]string, error) {
		walled = append(walled, p.Dir)
		return nil, nil
	}
	driveWalls.live = func(_ context.Context, p runner.Profile) error {
		walled = append(walled, p.Dir)
		return nil
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--profile", "beta"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	beta := abcdhome.Path(w.home, "profiles", "beta")
	if len(walled) != 2 || walled[0] != beta || walled[1] != beta {
		t.Fatalf("the checks ran on %q, want %q", walled, beta)
	}
	if w.terminalCalls != 1 || w.terminal.Options.Profile != "beta" {
		t.Fatalf("the loop starts on beta: %d call(s), profile %q", w.terminalCalls, w.terminal.Options.Profile)
	}
}

// recordConfigDir has the stand-in harness keep the directory each launch was
// pointed at, and reports the file that holds them.
func recordConfigDir(t *testing.T) string {
	t.Helper()
	log := filepath.Join(t.TempDir(), "config-dirs")
	bin := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' \"$CLAUDE_CONFIG_DIR\" >> '" + log + "'\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// TestAWaitOnANamedProfileProbesAndLaunchesOnThatDirectory: what a limit's wait starts
// with is all on the named directory, whether the wait is the live proof's or the
// run's, and nothing moves to its sibling.
func TestAWaitOnANamedProfileProbesAndLaunchesOnThatDirectory(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.twoProfiles(t)
	proofs, _ := limitedProof(t, 2, func(now time.Time) time.Time { return now.Add(3 * time.Hour) })
	var proved []string
	limited := driveWalls.live
	driveWalls.live = func(ctx context.Context, p runner.Profile) error {
		proved = append(proved, p.Dir)
		return limited(ctx, p)
	}
	log := recordConfigDir(t)
	at := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--until", at, "--profile", "beta"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	beta := abcdhome.Path(w.home, "profiles", "beta")
	if *proofs != 3 {
		t.Fatalf("the proof runs until the limit lifts: %d", *proofs)
	}
	for _, dir := range proved {
		if dir != beta {
			t.Fatalf("a proof after the wait ran on %q, want %q", dir, beta)
		}
	}
	if w.terminal.Probe == nil {
		t.Fatal("under a bound the run probes a limit")
	}
	for _, probe := range []func(context.Context, string, string) (bool, *runner.Failure){
		w.terminal.Probe, w.terminal.Runners.Config.ProbeLimit, w.terminal.Options.Runners.ProbeLimit,
	} {
		probe(context.Background(), loop.RoleImplementer, "")
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("the probes launched: %v", err)
	}
	dirs := strings.Fields(string(raw))
	if len(dirs) != 3 {
		t.Fatalf("three probes, three launches: %q", dirs)
	}
	for _, dir := range dirs {
		if dir != beta {
			t.Fatalf("a launch after the wait ran on %q, want %q", dir, beta)
		}
	}
}

// TestAProfileThatFailsItsChecksIsRefusedBeforeAnythingIsReserved: each refusal comes
// before the consent, and repeats nothing typed.
func TestAProfileThatFailsItsChecksIsRefusedBeforeAnythingIsReserved(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.twoProfiles(t)
	folder := abcdhome.Path(w.home, "profiles")
	if err := os.Symlink(t.TempDir(), filepath.Join(folder, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(folder, "alpha"), 0o770); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, want string }{
		{"absent-one", "no profile directory"},
		{"../profiles/beta", "plain"},
		{"linked", "is a link"},
		{"alpha", "chmod go-w"},
	} {
		text := refusedDrive(t, "drain", "--drive", "--yes", "--profile", c.name)
		for _, want := range []string{c.want, "nothing started, nothing reserved"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: the refusal %q does not say %q", c.name, text, want)
			}
		}
		noDrainBegun(t, w.repo)
		if w.terminalCalls != 0 {
			t.Fatalf("%s: nothing runs", c.name)
		}
	}
}

// TestAWorldOfOneDirectoryIsAsItWas: no flag behaves as it always did, and the
// directory there may be named too.
func TestAWorldOfOneDirectoryIsAsItWas(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	for _, args := range [][]string{{"drain", "--drive", "--yes"}, {"drain", "--drive", "--yes", "--profile", "walled"}} {
		var out, errb strings.Builder
		if code := Run(args, &out, &errb); code != 0 {
			t.Fatalf("%v: exit %d:\n%s%s", args, code, out.String(), errb.String())
		}
		if w.terminal.Options.Profile != "walled" {
			t.Fatalf("%v: the run records the directory it runs on: %q", args, w.terminal.Options.Profile)
		}
	}
}

// TestAResumeCommandNamesTheProfileThatWasNamed: the lines a drive ends on name what
// to type, and with several directories that takes the flag.
func TestAResumeCommandNamesTheProfileThatWasNamed(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.twoProfiles(t)
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.TerminalStopped, Run: "run-1", Issue: "iss-1", Lane: "lane-1", Resume: loop.TerminalResumeCommand}, nil
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--profile", "beta"}, &out, &errb); code != 3 {
		t.Fatalf("a stop exits 3, got %d:\n%s%s", code, out.String(), errb.String())
	}
	if want := "to resume: abcd drain --drive --profile beta"; !strings.Contains(out.String(), want) {
		t.Fatalf("the end does not say %q:\n%s", want, out.String())
	}
}

// TestADriveOnAnotherProfileThanALaneStartedOnIsRefused: nothing crosses from one
// subscription to the other unasked, and the refusal says both what to type to go on
// and where else the other profile may run.
func TestADriveOnAnotherProfileThanALaneStartedOnIsRefused(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.twoProfiles(t)
	was := driveLanesElsewhere
	t.Cleanup(func() { driveLanesElsewhere = was })
	driveLanesElsewhere = func(_, profile string) ([]loop.LaneOnProfile, error) {
		if profile == "alpha" {
			return nil, nil
		}
		return []loop.LaneOnProfile{{RunID: "run-1", Issue: "iss-1", Profile: "alpha"}}, nil
	}
	text := refusedDrive(t, "drain", "--drive", "--yes", "--profile", "beta")
	for _, want := range []string{"run-1", "iss-1", "alpha", "beta", "--profile alpha", "another checkout", "nothing started, nothing reserved"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal %q does not name %q", text, want)
		}
	}
	if w.terminalCalls != 0 {
		t.Fatal("nothing runs")
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--profile", "alpha"}, &out, &errb); code != 0 {
		t.Fatalf("the profile it started on goes on: exit %d:\n%s%s", code, out.String(), errb.String())
	}
}

// TestTheConsentNamesTheProfile: the question says which subscription it would spend.
func TestTheConsentNamesTheProfile(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.twoProfiles(t)
	withTerminal(t)
	out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive", "--profile", "beta")
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	for _, want := range []string{"profile:  beta", abcdhome.Display("profiles", "beta"), "separate login and settings"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the consent does not name %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "alpha") {
		t.Errorf("the consent names the other one:\n%s", out)
	}
}

// TestAWallNotProvedRefusesNamingIt: with the production check, the drive does
// not start until a wall has been proved.
func TestAWallNotProvedRefusesNamingIt(t *testing.T) {
	w := newDriveWorld(t)
	text := refusedDrive(t, "drain", "--drive", "--yes")
	if !strings.Contains(text, "check") {
		t.Fatalf("the refusal names the check: %s", text)
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
	for _, want := range []string{"cap:", "8 lane", "agent run", "runner: claude", "subscription", abcdhome.Display("profiles", "walled"), "Start?"} {
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
		{"drain", "--profile", "walled"},
		{"drain", "--drive", "--profile", ""},
	} {
		refusedDrive(t, args...)
	}
}

// TestADriveEndHaltedSaysWhatHolds: the end words carry the refusal's own reason, so a
// lifted stop, a signal's mark and a held read each say what is true of them.
func TestADriveEndHaltedSaysWhatHolds(t *testing.T) {
	const why = "run-1 was halted by a stop; the stop has lifted; run the take-over"
	var b bytes.Buffer
	res := loop.TerminalResult{Ended: loop.TerminalHalted, Why: why}
	renderDriveEnd(&b, res)
	if !strings.Contains(b.String(), why) {
		t.Fatalf("the end: %q", b.String())
	}
	if strings.Contains(driveEndWords(res.Ended), "elsewhere") {
		t.Fatalf("the words claim what may not be so: %q", driveEndWords(res.Ended))
	}
}
