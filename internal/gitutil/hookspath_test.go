package gitutil_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestHooksPathsReadsTheRepositorysOwnValue: the value is the one the
// person's git reads, never the isolated environment's own override, and an
// unset key is an empty answer rather than an error.
func TestHooksPathsReadsTheRepositorysOwnValue(t *testing.T) {
	repo := newRepo(t, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	got, err := gitutil.HooksPaths(repo)
	if err != nil || len(got) != 0 {
		t.Fatalf("unset: %q, %v", got, err)
	}
	if out, err := runGit(t, repo, "config", "core.hooksPath", "tools/hooks"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	got, err = gitutil.HooksPaths(repo)
	if err != nil || !slices.Equal(got, []string{"tools/hooks"}) {
		t.Fatalf("set: %q, %v", got, err)
	}
}

// TestHooksPathsReadsThePersonsGlobalValue: a hooks directory the person's
// global configuration names runs on their next commit in this repository as
// surely as the repository's own, so it is read too, before the
// repository's.
func TestHooksPathsReadsThePersonsGlobalValue(t *testing.T) {
	repo := newRepo(t, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core]\n\thooksPath = ~/global-hooks\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := runGit(t, repo, "config", "core.hooksPath", "tools/hooks"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	got, err := gitutil.HooksPaths(repo)
	if err != nil || !slices.Equal(got, []string{filepath.Join(home, "global-hooks"), "tools/hooks"}) {
		t.Fatalf("got %q, %v", got, err)
	}
}
