package runner

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// profileHome is a home directory holding the named walled directories.
func profileHome(t *testing.T, names ...string) string {
	t.Helper()
	home := t.TempDir()
	for _, n := range names {
		if err := os.MkdirAll(abcdhome.Path(home, "profiles", n), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

// TestTheDriveRunsOnTheOneWalledDirectory: the one directory under profiles/
// is the profile, with the launch's environment and arguments.
func TestTheDriveRunsOnTheOneWalledDirectory(t *testing.T) {
	home := profileHome(t, "first")
	p, err := DriveProfile(home, "")
	if err != nil {
		t.Fatal(err)
	}
	dir := abcdhome.Path(home, "profiles", "first")
	if p.Dir != dir {
		t.Fatalf("the profile's directory is %q, want %q", p.Dir, dir)
	}
	for _, want := range []string{"CLAUDE_CONFIG_DIR=" + dir, "CLAUDE_CODE_DISABLE_CLAUDE_MDS=1", "CLAUDE_CODE_DISABLE_AUTO_MEMORY=1"} {
		if !slices.Contains(p.Env, want) {
			t.Errorf("the profile's environment %q lacks %q", p.Env, want)
		}
	}
	for _, want := range []string{"--no-session-persistence", "--strict-mcp-config", "--setting-sources=user"} {
		if !slices.Contains(p.Args, want) {
			t.Errorf("the profile's arguments %q lack %q", p.Args, want)
		}
	}
}

// TestNoWalledDirectoryRefusesNamingTheOneToMake: with none, or no profiles
// folder at all, the refusal names the folder and the login to run.
func TestNoWalledDirectoryRefusesNamingTheOneToMake(t *testing.T) {
	for _, home := range []string{t.TempDir(), func() string {
		h := t.TempDir()
		if err := os.MkdirAll(abcdhome.Path(h, "profiles"), 0o700); err != nil {
			t.Fatal(err)
		}
		return h
	}()} {
		_, err := DriveProfile(home, "")
		if !errors.Is(err, ErrNoProfile) {
			t.Fatalf("no directory is ErrNoProfile: %v", err)
		}
		for _, want := range []string{abcdhome.Display("profiles") + "/", "CLAUDE_CONFIG_DIR="} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("the refusal %q does not name %q", err, want)
			}
		}
	}
}

// TestSeveralWalledDirectoriesRefuseNamingThemAndTheFlag: with no name a drive
// does not choose among several, and the refusal says how to.
func TestSeveralWalledDirectoriesRefuseNamingThemAndTheFlag(t *testing.T) {
	_, err := DriveProfile(profileHome(t, "alpha", "beta"), "")
	if !errors.Is(err, ErrSeveralProfiles) {
		t.Fatalf("two directories are ErrSeveralProfiles: %v", err)
	}
	for _, want := range []string{"alpha", "beta", "--profile <name>"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not name %q", err, want)
		}
	}
}

// TestANameChoosesItsDirectory: every part of the profile that names a place is
// the named directory's, among several and with one.
func TestANameChoosesItsDirectory(t *testing.T) {
	home := profileHome(t, "alpha", "beta")
	p, err := DriveProfile(home, "beta")
	if err != nil {
		t.Fatal(err)
	}
	dir := abcdhome.Path(home, "profiles", "beta")
	if p.Dir != dir || p.Name() != "beta" {
		t.Fatalf("the profile is %q, named %q", p.Dir, p.Name())
	}
	for _, want := range []string{"CLAUDE_CONFIG_DIR=" + dir, StageProtectEnv + "=" + dir} {
		if !slices.Contains(p.Env, want) {
			t.Errorf("the profile's environment %q lacks %q", p.Env, want)
		}
	}
	there := abcdhome.Path(home, "stage-homes", "beta") + string(filepath.Separator)
	if !strings.HasPrefix(p.StageRoot, there) || !strings.HasPrefix(p.ForCheckout("/a/checkout").StageRoot, there) {
		t.Fatalf("the stage's files lie under the named profile: %q", p.StageRoot)
	}

	one, err := DriveProfile(profileHome(t, "only"), "only")
	if err != nil || one.Name() != "only" {
		t.Fatalf("one directory can be named too: %q %v", one.Name(), err)
	}
}

// TestANameThatIsNoPlainElementIsRefusedAndNotRepeated: a name is one path element,
// so nothing reaches outside the folder or a sibling by another spelling, and a
// refusal never repeats what was typed.
func TestANameThatIsNoPlainElementIsRefusedAndNotRepeated(t *testing.T) {
	home := profileHome(t, "alpha", "beta")
	for _, name := range []string{"../profiles/beta", "alpha/../beta", "beta/", "/abs/beta", ".", "..", "a:b", "be\x1b[0mta", "be\nta"} {
		_, err := DriveProfile(home, name)
		if !errors.Is(err, ErrProfileName) {
			t.Errorf("%q is no plain name: %v", name, err)
			continue
		}
		if len(name) > 2 && strings.Contains(err.Error(), name) {
			t.Errorf("the refusal repeats %q: %v", name, err)
		}
	}
}

// TestANamedDirectoryMeetsEveryCheckTheOneDirectoryMeets: a name lifts none of them,
// and a sibling that fails them is no matter to a directory that holds.
func TestANamedDirectoryMeetsEveryCheckTheOneDirectoryMeets(t *testing.T) {
	home := profileHome(t, "alpha", "open")
	folder := abcdhome.Path(home, "profiles")
	if err := os.Chmod(filepath.Join(folder, "open"), 0o770); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(folder, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "plain"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, want string }{
		{"linked", "is a link"},
		{"open", "chmod go-w"},
		{"plain", "no profile directory"},
		{"absent", "no profile directory"},
		// The folder holds the name exactly or not at all, on a file system that
		// folds case too: a run records the name a directory has.
		{"ALPHA", "no profile directory"},
	} {
		_, err := DriveProfile(home, c.name)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	_, err := DriveProfile(home, "absent")
	if !errors.Is(err, ErrNoProfile) {
		t.Fatalf("a missing name is ErrNoProfile: %v", err)
	}
	for _, want := range []string{"alpha", "linked", "open"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q names what the folder holds, %q among it", err, want)
		}
	}
	if strings.Contains(err.Error(), "absent") {
		t.Errorf("the refusal repeats what was typed: %v", err)
	}
	if p, err := DriveProfile(home, "alpha"); err != nil || p.Name() != "alpha" {
		t.Fatalf("a sibling that fails is no matter: %q %v", p.Name(), err)
	}
}

// TestAPlainFileBesideTheWalledDirectoryIsNoSecondOne: a file the system
// leaves there counts for nothing.
func TestAPlainFileBesideTheWalledDirectoryIsNoSecondOne(t *testing.T) {
	home := profileHome(t, "only")
	if err := os.WriteFile(abcdhome.Path(home, "profiles", ".DS_Store"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DriveProfile(home, ""); err != nil {
		t.Fatalf("a plain file is no directory: %v", err)
	}
}

// TestAWalledDirectoryOnlyItsOwnerCanChangeIsAdmitted: a link, or a directory
// another account can write, is refused, since its settings decide what the
// drive's agents run.
func TestAWalledDirectoryOnlyItsOwnerCanChangeIsAdmitted(t *testing.T) {
	home := profileHome(t)
	target := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home, "profiles"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, abcdhome.Path(home, "profiles", "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := DriveProfile(home, ""); err == nil || !strings.Contains(err.Error(), "linked") {
		t.Fatalf("a link is refused by name: %v", err)
	}

	home = profileHome(t, "open")
	dir := abcdhome.Path(home, "profiles", "open")
	if err := os.Chmod(dir, 0o770); err != nil {
		t.Fatal(err)
	}
	if _, err := DriveProfile(home, ""); err == nil || !strings.Contains(err.Error(), "chmod go-w") {
		t.Fatalf("a group-writable directory is refused naming the fix: %v", err)
	}
	if filepath.Dir(dir) == dir {
		t.Fatal("unreachable")
	}
}

// TestTwoProfilesInTwoCheckoutsShareNoStageFile: what two drives at once write for
// their stages lies apart, by profile and by checkout.
func TestTwoProfilesInTwoCheckoutsShareNoStageFile(t *testing.T) {
	home := profileHome(t, "alpha", "beta")
	alpha, err := DriveProfile(home, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := DriveProfile(home, "beta")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, p := range []Profile{alpha.ForCheckout("/one"), beta.ForCheckout("/two"), beta.ForCheckout("/one")} {
		for _, path := range []string{p.StageRoot, p.ShellHome, p.EnvFile} {
			if seen[path] {
				t.Errorf("%s is shared", path)
			}
			seen[path] = true
		}
	}
}

// TestHeldCredentialsNamesEachVariableAndNoValue: what the harness would bill
// through is reported by name.
func TestHeldCredentialsNamesEachVariableAndNoValue(t *testing.T) {
	for _, k := range credentialVars {
		t.Setenv(k, "")
	}
	if got := HeldCredentials(); len(got) != 0 {
		t.Fatalf("nothing set holds nothing: %q", got)
	}
	for _, k := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN"} {
		if !slices.Contains(credentialVars, k) {
			t.Fatalf("%s is a variable the harness bills through", k)
		}
	}
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "a-value-never-shown")
	got := HeldCredentials()
	if !slices.Equal(got, []string{"ANTHROPIC_AUTH_TOKEN"}) {
		t.Fatalf("the one set is named: %q", got)
	}
}
