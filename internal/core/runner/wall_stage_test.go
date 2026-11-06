package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// realGitRead has this test read the git configuration as a drive does.
func realGitRead(t *testing.T) {
	t.Helper()
	was := stageGitRead
	t.Cleanup(func() { stageGitRead = was })
	stageGitRead = readStageGit
}

// TestAStagesHomeIsApartFromTheLogin: nothing a stage's shell does under its home
// lands beside the login.
func TestAStagesHomeIsApartFromTheLogin(t *testing.T) {
	p := wallProfile(t)
	if strings.HasPrefix(p.ShellHome+"/", p.Dir+"/") {
		t.Fatalf("the shell's home %q is inside the walled directory %q", p.ShellHome, p.Dir)
	}
}

// TestAStagesHomeStartsEmptyAtEachLaunch: nothing one stage leaves there reaches
// the next, and a home that is a link is refused rather than followed.
func TestAStagesHomeStartsEmptyAtEachLaunch(t *testing.T) {
	f, c, p := walledFake(t, "ok")
	planted := filepath.Join(p.ShellHome, ".config", "git", "config")
	if err := os.MkdirAll(filepath.Dir(planted), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planted, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Run(context.Background(), shellRequest(f)); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(planted); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("what the last stage left is still there: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.ShellHome, ".gitconfig")); err != nil {
		t.Fatalf("the home keeps its git configuration: %v", err)
	}
	elsewhere := t.TempDir()
	if err := os.RemoveAll(p.ShellHome); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, p.ShellHome); err != nil {
		t.Fatal(err)
	}
	if err := ResetStageHome(p); err == nil {
		t.Fatal("a home that is a link is refused")
	}
}

// TestAChangedBinaryStopsTheDrive: abcd's binary and the harness are compared by
// content, not by name.
func TestAChangedBinaryStopsTheDrive(t *testing.T) {
	p := wallProfile(t)
	bin := filepath.Join(t.TempDir(), "abcd")
	if err := os.WriteFile(bin, []byte("one"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteWallSettings(p, bin); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, bin); err != nil {
		t.Fatalf("what was written holds: %v", err)
	}
	if err := os.WriteFile(bin, []byte("two"), 0o700); err != nil {
		t.Fatal(err)
	}
	var we *WallError
	if err := CheckWallSettings(p, bin); !errors.As(err, &we) || we.Wall != WallSettingsFile {
		t.Fatalf("a changed binary is the settings wall: %v", err)
	}
}

// TestAChangedHarnessStopsTheDrive.
func TestAChangedHarnessStopsTheDrive(t *testing.T) {
	dir := t.TempDir()
	harness := filepath.Join(dir, "claude")
	if err := os.WriteFile(harness, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, wallBin); err != nil {
		t.Fatalf("what was written holds: %v", err)
	}
	if err := os.WriteFile(harness, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, wallBin); err == nil {
		t.Fatal("a changed harness stops the drive")
	}
}

// TestACarriedValueHoldsNoCredential: a proxy's credentials are stripped, and only
// the locale variables a shell reads are carried, by name.
func TestACarriedValueHoldsNoCredential(t *testing.T) {
	f, c, _ := walledFake(t, "ok")
	t.Setenv("HTTPS_PROXY", "http://someone:a-value-never-passed@proxy.example.invalid:8080")
	t.Setenv("http_proxy", "http://a-value-never-passed@proxy.example.invalid:8080")
	t.Setenv("LC_ALL", "C")
	t.Setenv("LC_A_PERSONS_OWN", "a-value-never-passed")
	if _, _, err := c.Run(context.Background(), shellRequest(f)); err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "claude.env"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "a-value-never-passed") {
		t.Fatalf("a credential was carried:\n%s", raw)
	}
	env := strings.Split(string(raw), "\n")
	for _, want := range []string{"HTTPS_PROXY=http://proxy.example.invalid:8080", "http_proxy=http://proxy.example.invalid:8080", "LC_ALL=C"} {
		if !slices.Contains(env, want) {
			t.Errorf("the environment lacks %q:\n%s", want, raw)
		}
	}
}

// TestAWalledLaunchTellsItsGuardWhatItIs: the approver and the hooks read the
// stage's home, the walled directory, its declared paths and its base.
func TestAWalledLaunchTellsItsGuardWhatItIs(t *testing.T) {
	f, c, p := walledFake(t, "ok")
	req := shellRequest(f)
	req.Declared = []string{"docs/page.md"}
	if _, _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}
	argv := strings.Join(f.argv(t, Claude), " ")
	for _, want := range []string{"--stage-home=" + p.ShellHome, "--protect=" + p.Dir, "--declared=docs/page.md"} {
		if !strings.Contains(argv, want) {
			t.Errorf("the approver's arguments lack %q: %s", want, argv)
		}
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "claude.env"))
	if err != nil {
		t.Fatal(err)
	}
	env := strings.Split(string(raw), "\n")
	for _, want := range []string{StageHomeEnv + "=" + p.ShellHome, StageProtectEnv + "=" + p.Dir, StageDeclaredEnv + "=docs/page.md", StageBaseEnv + "=" + req.Base} {
		if !slices.Contains(env, want) {
			t.Errorf("the hooks' environment lacks %q", want)
		}
	}
}

// gitIdentityWorld is a home and a repository with a global configuration.
func gitIdentityWorld(t *testing.T, global string) (Profile, string) {
	t.Helper()
	realGitRead(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		t.Setenv(k, "")
	}
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(global), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(abcdhome.Path(home, "profiles", "walled"), 0o700); err != nil {
		t.Fatal(err)
	}
	p, err := DriveProfile(home, "")
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(home, "work", "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	init := exec.Command("git", "-C", repo, "init", "-q")
	init.Env = gittest.Env(t)
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	p.Repo = repo
	return p, home
}

// TestNoGitIdentityRefusesTheStart: a stage never commits under a name git made
// up from the account and the host.
func TestNoGitIdentityRefusesTheStart(t *testing.T) {
	p, _ := gitIdentityWorld(t, "[core]\n\tpager = cat\n")
	var we *WallError
	if _, err := WriteWallSettings(p, wallBin); !errors.As(err, &we) || we.Wall != WallIdentity {
		t.Fatalf("no identity is the identity wall: %v", err)
	}
}

// TestAGitIdentityBehindAnIncludeIsFound: an include, and one conditional on the
// repository's directory, both count.
func TestAGitIdentityBehindAnIncludeIsFound(t *testing.T) {
	p, home := gitIdentityWorld(t, "")
	inc := filepath.Join(home, "work.inc")
	if err := os.WriteFile(inc, []byte("[user]\n\tname = Pat Example\n\temail = pat@example.invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	real, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	cfg := "[includeIf \"gitdir:" + filepath.Join(real, "work") + "/\"]\n\tpath = " + inc + "\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatalf("the included identity is found: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(p.ShellHome, ".gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "pat@example.invalid") || !strings.Contains(string(raw), "useConfigOnly") {
		t.Fatalf("the home's configuration:\n%s", raw)
	}
}

// TestAPreflightsEnvironmentIsTheStagesAndKeepsTheMachinesGate: nothing beyond the
// list, under the stage's home, with the person's home named for the load gate
// and the Go caches kept.
func TestAPreflightsEnvironmentIsTheStagesAndKeepsTheMachinesGate(t *testing.T) {
	p := wallProfile(t)
	for _, k := range []string{"GH_TOKEN", "SSH_AUTH_SOCK", "AWS_SECRET_ACCESS_KEY"} {
		t.Setenv(k, "a-value-never-passed")
	}
	t.Setenv("GOFLAGS", "-p=1")
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	env := PreflightEnv(p)
	if strings.Contains(strings.Join(env, "\n"), "a-value-never-passed") {
		t.Fatalf("something outside the list reached the preflight: %q", env)
	}
	for _, want := range []string{"HOME=" + p.ShellHome, StageHomeEnv + "=" + p.ShellHome, abcdhome.PersonHomeEnv + "=" + p.PersonHome, "GOFLAGS=-p=1"} {
		if !slices.Contains(env, want) {
			t.Errorf("the preflight's environment lacks %q: %q", want, env)
		}
	}
}

// TestTheProofCatchesACredentialUnderAHarnessPrefix: the harness's own variables
// are expected in a stage's shell, and its credential variables are not.
func TestTheProofCatchesACredentialUnderAHarnessPrefix(t *testing.T) {
	if !stageShellAllowed("CLAUDE_CODE_ENTRYPOINT") {
		t.Fatal("the harness's own variable is expected")
	}
	for _, name := range credentialVars {
		if stageShellAllowed(name) {
			t.Errorf("%s is allowed in a stage's shell", name)
		}
	}
}

// TestAWalledLaunchAndTheProofEachWriteOneLedgerEntry: a stage reaches the ledger
// through the dispatcher and the proof through the same one function, so neither
// goes unrecorded.
func TestAWalledLaunchAndTheProofEachWriteOneLedgerEntry(t *testing.T) {
	f := newFake(t, "wall-ok", Claude)
	c := mustLoad(t, claudeRoute, "")
	p, err := DriveProfile(profileHome(t, "walled"), "")
	if err != nil {
		t.Fatal(err)
	}
	p.Ledger = c.Ledger()
	if _, err := WriteWallSettings(p, selfBinary()); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, c.Walled(p), true)
	if _, err := h.d.Dispatch(context.Background(), shellRequest(f)); err != nil {
		t.Fatal(err)
	}
	if got := ledgerFiles(t, c.Ledger().home); len(got) != 1 || got[0].Role != shellRequest(f).Role {
		t.Fatalf("the stage's one entry: %+v", got)
	}
	if err := ProveWall(context.Background(), p, selfBinary()); err != nil {
		t.Fatal(err)
	}
	got := ledgerFiles(t, c.Ledger().home)
	roles := []string{}
	for _, e := range got {
		roles = append(roles, e.Role)
	}
	if len(got) != 2 || !slices.Contains(roles, "wall-proof") {
		t.Fatalf("the proof's one entry beside it: %q", roles)
	}
}

// TestAProofThatCannotBeRecordedIsNotStarted: as a stage, nothing starts that the
// ledger cannot hold.
func TestAProofThatCannotBeRecordedIsNotStarted(t *testing.T) {
	f := newFake(t, "wall-ok", Claude)
	p := wallProfile(t)
	p.Ledger = newLedger("")
	if err := ProveWall(context.Background(), p, wallBin); err == nil {
		t.Fatal("a proof the ledger cannot hold is refused")
	}
	if f.launched(Claude) {
		t.Fatal("nothing started")
	}
}

// TestEachCheckoutHasAStageHomeOfItsOwn: a second drive elsewhere wipes nothing.
func TestEachCheckoutHasAStageHomeOfItsOwn(t *testing.T) {
	p := wallProfile(t)
	a, b := p.ForCheckout(t.TempDir()), p.ForCheckout(t.TempDir())
	if a.ShellHome == b.ShellHome || a.EnvFile == b.EnvFile {
		t.Fatalf("one home for two checkouts: %q", a.ShellHome)
	}
	if !slices.Contains(a.Env, "CLAUDE_ENV_FILE="+a.EnvFile) || !slices.Contains(a.Env, StageHomeEnv+"="+a.ShellHome) {
		t.Fatalf("the launch names this checkout's: %q", a.Env)
	}
	if _, err := WriteWallSettings(a, wallBin); err != nil {
		t.Fatal(err)
	}
	mine := filepath.Join(a.ShellHome, "kept")
	if err := os.WriteFile(mine, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteWallSettings(b, wallBin); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); err != nil {
		t.Fatalf("the other drive wiped this one's home: %v", err)
	}
}

// TestGoInstallsLandUnderTheStagesHome: what a stage installs never shadows a
// program a later command runs, and a carried proxy list holds no credential.
func TestGoInstallsLandUnderTheStagesHome(t *testing.T) {
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p.EnvFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "export GOBIN="+shellquote.Single(filepath.Join(p.ShellHome, "bin"))) {
		t.Fatalf("the shell file:\n%s", raw)
	}
	env := stageEnv([]string{"GOPROXY=https://someone:a-value-never-passed@proxy.example.invalid,direct"})
	if strings.Contains(strings.Join(env, " "), "a-value-never-passed") || !slices.Contains(env, "GOPROXY=https://proxy.example.invalid,direct") {
		t.Fatalf("carried: %q", env)
	}
}

// TestProgramsAreResolvedOnce: what a program name means is fixed at the start, so
// one placed earlier on PATH later is not what runs.
func TestProgramsAreResolvedOnce(t *testing.T) {
	early, late := t.TempDir(), t.TempDir()
	for _, name := range []string{"git", "make"} {
		if err := os.WriteFile(filepath.Join(late, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", early+string(os.PathListSeparator)+late)
	PinPrograms("git", "make", "gh")
	if err := os.WriteFile(filepath.Join(early, "git"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := exec.LookPath("git")
	if err != nil || got != filepath.Join(late, "git") {
		t.Fatalf("git resolves to %q %v", got, err)
	}
}

// TestALaunchLeavesNoProcessBehind: whatever came of the harness, nothing it
// started in its group outlives the launch.
func TestALaunchLeavesNoProcessBehind(t *testing.T) {
	f, c, _ := walledFake(t, "leave-child")
	_, _, _ = c.Run(context.Background(), shellRequest(f))
	raw, err := os.ReadFile(filepath.Join(f.log, "left.pid"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return
		}
	}
	t.Fatalf("process %d outlived the launch", pid)
}

// TestHookedGitHoldsNoCredential: a hook the driver's git runs gets what a stage
// carries, and git's own settings stay.
func TestHookedGitHoldsNoCredential(t *testing.T) {
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	got := HookedEnv(p)([]string{"PATH=/bin", "GH_TOKEN=a-value-never-passed", "SSH_AUTH_SOCK=a-value-never-passed", "GIT_CONFIG_COUNT=1", "GIT_TERMINAL_PROMPT=0"})
	if strings.Contains(strings.Join(got, " "), "a-value-never-passed") {
		t.Fatalf("a credential reached a hook: %q", got)
	}
	for _, want := range []string{"PATH=/bin", "GIT_CONFIG_COUNT=1", "GIT_TERMINAL_PROMPT=0", "HOME=" + p.ShellHome} {
		if !slices.Contains(got, want) {
			t.Errorf("a hook's environment lacks %q: %q", want, got)
		}
	}
}

// TestPinningKeepsEachNamesFirstMatch: a program that stands ahead of the system's
// stays what its name means, whatever order the names are asked in.
func TestPinningKeepsEachNamesFirstMatch(t *testing.T) {
	gnu, system := t.TempDir(), t.TempDir()
	for _, p := range []string{filepath.Join(gnu, "make"), filepath.Join(system, "make"), filepath.Join(system, "git")} {
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", gnu+string(os.PathListSeparator)+system)
	PinPrograms("git", "make")
	if got, err := exec.LookPath("make"); err != nil || got != filepath.Join(gnu, "make") {
		t.Fatalf("make resolves to %q %v", got, err)
	}
}

// TestAPlantedGitConfigDoesNotReachTheDriversGit: the driver's hooked git reads the
// configuration the drive wrote, not what a stage left in its home.
func TestAPlantedGitConfigDoesNotReachTheDriversGit(t *testing.T) {
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	planted := filepath.Join(p.ShellHome, ".config", "git", "config")
	if err := os.MkdirAll(filepath.Dir(planted), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planted, []byte("[user]\n\tname = planted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.ShellHome, ".gitconfig"), []byte("[user]\n\tname = planted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "config", "--global", "user.name")
	// Not gittest.Env: what this reads is what the driver's own git is told.
	cmd.Env = HookedEnv(p)([]string{"PATH=" + os.Getenv("PATH")})
	out, _ := cmd.CombinedOutput()
	if strings.Contains(string(out), "planted") {
		t.Fatalf("the driver's git read what a stage left: %q", out)
	}
	if err := os.WriteFile(stageGitFile(p), []byte("[user]\n\tname = planted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, wallBin); err == nil {
		t.Fatal("a changed configuration for the driver's git goes unseen")
	}
}

// TestTheProofsCommandsAreOnesTheApproverPasses: the first is the one the guard
// refuses; the rest must reach the shell, or the proof can never hold.
func TestTheProofsCommandsAreOnesTheApproverPasses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	f := guard.NewFiles(t.TempDir()).WithStage(abcdhome.Path(home, "stage-homes", "walled"), []string{abcdhome.Path(home, "profiles", "walled")}, nil)
	approve := func(line string) guard.Approval {
		raw, _ := json.Marshal(map[string]string{"command": line})
		a, _ := guard.ApprovePrompt(guard.Loaded{Registry: guard.Defaults()}, f, "Bash", raw)
		return a
	}
	cmds := wallProofCommands()
	if len(cmds) != 3 {
		t.Fatalf("three commands: %q", cmds)
	}
	if a := approve(cmds[0]); a.Allow {
		t.Fatalf("the first is meant to be refused: %q", cmds[0])
	}
	for _, c := range cmds[1:] {
		if a := approve(c); !a.Allow {
			t.Errorf("%q was refused: %s", c, a.Reason)
		}
	}
}

// TestAResetLeavesThePersonsCheckDirectory: what a preflight reuses survives the
// emptying of a stage's home, a preflight's environment names it, and a stage's
// file tool does not write there.
func TestAResetLeavesThePersonsCheckDirectory(t *testing.T) {
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	check := abcdhome.Path(p.PersonHome, "tests", "root")
	if err := os.MkdirAll(check, abcdhome.DirMode); err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(check, "kept")
	if err := os.WriteFile(kept, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ResetStageHome(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("a reset reached the check directory: %v", err)
	}
	for _, want := range []string{"HOME=" + p.ShellHome, abcdhome.StageHomeEnv + "=" + p.ShellHome, abcdhome.PersonHomeEnv + "=" + p.PersonHome} {
		if !slices.Contains(PreflightEnv(p), want) {
			t.Errorf("a preflight is not told %s", want)
		}
	}
	if guard.StageWriteRefusal([]string{filepath.Join(p.PersonHome, "work")}, nil, kept) == "" {
		t.Fatal("a stage's file tool may write in the check directory")
	}
}

// TestAResetEmptiesAReadOnlyTreeAndFollowsNoLink: Go leaves its module directories read-only,
// and a stage can leave any; neither stops the next launch, and a link out is not walked.
func TestAResetEmptiesAReadOnlyTreeAndFollowsNoLink(t *testing.T) {
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Chmod(outside, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(outside, 0o700) })
	mod := filepath.Join(p.ShellHome, "go", "pkg", "mod", "example.com", "a@v1")
	if err := os.MkdirAll(mod, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "a.go"), []byte("package a\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(mod, "out")); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{mod, filepath.Dir(mod), filepath.Dir(filepath.Dir(mod))} {
		if err := os.Chmod(d, 0o500); err != nil {
			t.Fatal(err)
		}
	}
	shut := filepath.Join(p.ShellHome, "shut")
	if err := os.Mkdir(shut, 0); err != nil {
		t.Fatal(err)
	}
	if err := ResetStageHome(p); err != nil {
		t.Fatalf("the home is emptied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.ShellHome, "go")); !os.IsNotExist(err) {
		t.Fatalf("the tree stands: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.ShellHome, ".gitconfig")); err != nil {
		t.Fatalf("the git configuration is laid out again: %v", err)
	}
	fi, err := os.Stat(outside)
	if err != nil || fi.Mode().Perm() != 0o500 {
		t.Fatalf("a link out was followed: %v %v", fi, err)
	}
}

// TestAResetStaysInsideTheHome: a link at the top of the home and below it, pointing out,
// leaves its target's mode and contents alone, a link where the git configuration goes
// is no way out, and a home that is itself a link is refused and left.
func TestAResetStaysInsideTheHome(t *testing.T) {
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "key")
	if err := os.WriteFile(secret, []byte("kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(outside, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(outside, 0o700) })
	below := filepath.Join(p.ShellHome, "a", "b")
	if err := os.MkdirAll(below, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{filepath.Join(p.ShellHome, "top"), filepath.Join(below, "deep")} {
		if err := os.Symlink(outside, link); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(p.ShellHome, ".gitconfig")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(p.ShellHome, ".gitconfig")); err != nil {
		t.Fatal(err)
	}
	unchanged := func() {
		t.Helper()
		fi, err := os.Stat(outside)
		if err != nil || fi.Mode().Perm() != 0o500 {
			t.Fatalf("the target's mode changed: %v %v", fi, err)
		}
		if raw, err := os.ReadFile(secret); err != nil || string(raw) != "kept\n" {
			t.Fatalf("the target's contents changed: %q %v", raw, err)
		}
	}
	if err := ResetStageHome(p); err != nil {
		t.Fatalf("the home is emptied: %v", err)
	}
	unchanged()
	if fi, err := os.Lstat(filepath.Join(p.ShellHome, ".gitconfig")); err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("the git configuration is a file of its own: %v %v", fi, err)
	}
	if err := os.RemoveAll(p.ShellHome); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, p.ShellHome); err != nil {
		t.Fatal(err)
	}
	if err := ResetStageHome(p); err == nil {
		t.Fatal("a home that is a link is refused")
	}
	unchanged()
}

// TestOwnerAccessStaysInsideAndWidensNoMore: a directory swapped for a link out, after its
// mode was read, is refused; one with owner access is left as it is.
func TestOwnerAccessStaysInsideAndWidensNoMore(t *testing.T) {
	base := t.TempDir()
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	outside := t.TempDir()
	if err := os.Chmod(outside, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(outside, 0o700) })
	read, err := os.Stat(outside)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "swapped")); err != nil {
		t.Fatal(err)
	}
	ownerAccess(root, "swapped", read)
	if fi, err := os.Stat(outside); err != nil || fi.Mode().Perm() != 0o500 {
		t.Fatalf("the change of mode left the root: %v %v", fi, err)
	}
	for want, mode := range map[os.FileMode]os.FileMode{0o755: 0o755, 0o750: 0o050, 0o700: 0} {
		dir := fmt.Sprintf("d%o", mode)
		if err := os.Mkdir(filepath.Join(base, dir), mode); err != nil {
			t.Fatal(err)
		}
		restoreOwnerAccess(root, dir)
		if fi, err := os.Stat(filepath.Join(base, dir)); err != nil || fi.Mode().Perm() != want {
			t.Errorf("a directory of mode %o reads %v, want %o: %v", mode, fi.Mode().Perm(), want, err)
		}
	}
}

// TestAColdNoteNamesWhatIsMissedAndWhyTheGoStillRuns: a stage's shell and the preflight run
// that Go whatever abcd asked of it.
func TestAColdNoteNamesWhatIsMissedAndWhyTheGoStillRuns(t *testing.T) {
	refused := goRefusedNote([]string{"GOCACHE", "GOMODCACHE", "GOPATH"}, "go, or a directory it is reached through, is one group or other can write, so it is never run; chmod go-w it")
	for _, want := range []string{"did not ask that Go where its caches are", "one group or other can write", "a stage's shell and the preflight still run it", "chmod go-w it", "GOCACHE, GOMODCACHE, GOPATH"} {
		if !strings.Contains(refused, want) {
			t.Errorf("the note says %q: %s", want, refused)
		}
	}
	for _, never := range []string{"never run", "from cold"} {
		if strings.Contains(refused, never) {
			t.Errorf("the note says %q: %s", never, refused)
		}
	}
	partial := stageColdNote([]string{"GOPATH"}, "go names no absolute path for it")
	if !strings.Contains(partial, "GOPATH") || strings.Contains(partial, "GOCACHE") || strings.Contains(partial, "from cold") {
		t.Errorf("a partial miss is named as one: %s", partial)
	}
}
