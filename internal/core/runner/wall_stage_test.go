package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
	p, err := DriveProfile(home)
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(home, "work", "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
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
	cfg := "[includeIf \"gitdir:" + filepath.Join(home, "work") + "/\"]\n\tpath = " + inc + "\n"
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
	for _, want := range []string{"HOME=" + p.ShellHome, abcdhome.PersonHomeEnv + "=" + p.PersonHome, "GOFLAGS=-p=1"} {
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
	for _, name := range CredentialVars() {
		if stageShellAllowed(name) {
			t.Errorf("%s is allowed in a stage's shell", name)
		}
	}
}
