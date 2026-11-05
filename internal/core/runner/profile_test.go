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
	p, err := DriveProfile(home)
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
		_, err := DriveProfile(home)
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

// TestSeveralWalledDirectoriesRefuseNamingThemAndTheProfilesIntent: this cut
// runs on exactly one, and the intent that admits several is named.
func TestSeveralWalledDirectoriesRefuseNamingThemAndTheProfilesIntent(t *testing.T) {
	_, err := DriveProfile(profileHome(t, "alpha", "beta"))
	if !errors.Is(err, ErrSeveralProfiles) {
		t.Fatalf("two directories are ErrSeveralProfiles: %v", err)
	}
	for _, want := range []string{"alpha", "beta", "itd-2611040344170206"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not name %q", err, want)
		}
	}
}

// TestAPlainFileBesideTheWalledDirectoryIsNoSecondOne: a file the system
// leaves there counts for nothing.
func TestAPlainFileBesideTheWalledDirectoryIsNoSecondOne(t *testing.T) {
	home := profileHome(t, "only")
	if err := os.WriteFile(abcdhome.Path(home, "profiles", ".DS_Store"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DriveProfile(home); err != nil {
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
	if _, err := DriveProfile(home); err == nil || !strings.Contains(err.Error(), "linked") {
		t.Fatalf("a link is refused by name: %v", err)
	}

	home = profileHome(t, "open")
	dir := abcdhome.Path(home, "profiles", "open")
	if err := os.Chmod(dir, 0o770); err != nil {
		t.Fatal(err)
	}
	if _, err := DriveProfile(home); err == nil || !strings.Contains(err.Error(), "chmod go-w") {
		t.Fatalf("a group-writable directory is refused naming the fix: %v", err)
	}
	if filepath.Dir(dir) == dir {
		t.Fatal("unreachable")
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
