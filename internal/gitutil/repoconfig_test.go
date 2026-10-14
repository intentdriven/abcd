package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestRepoConfigReadsTheRepositorysOwnEntries: the entries the repository's
// config and its includes set are listed with their scope, canonical key and
// value; the -c pins the isolated command carries (core.hooksPath,
// core.fsmonitor) and the person's global config are not the repository's and
// are left out; nothing matching is an empty list.
func TestRepoConfigReadsTheRepositorysOwnEntries(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	const pattern = `^(filter\..*|core\.(hookspath|sshcommand))$`

	got, err := gitutil.RepoConfig(repo, pattern)
	if err != nil || len(got) != 0 {
		t.Fatalf("nothing set: got %+v, %v; want an empty list", got, err)
	}

	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core]\n\tsshCommand = global-ssh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "inc"), []byte("[core]\n\tsshCommand = included-ssh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("config", "filter.Big.clean", "clean-it")
	git("config", "include.path", "../inc")

	got, err = gitutil.RepoConfig(repo, pattern)
	if err != nil {
		t.Fatal(err)
	}
	want := []gitutil.ConfigEntry{
		{Scope: "local", Key: "filter.Big.clean", Value: "clean-it"},
		{Scope: "local", Key: "core.sshcommand", Value: "included-ssh"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
