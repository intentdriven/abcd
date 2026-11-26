package layered

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAnUnreadLayerSaysWhetherGitFailed: a repository layer read at a commit
// fails either because git did not answer, which says nothing of the file, or
// because the file the commit holds is refused (not JSON, not a regular file),
// which says something of it. GitFailed tells the two apart; the message is
// Load's either way.
func TestAnUnreadLayerSaysWhetherGitFailed(t *testing.T) {
	commit := func(t *testing.T, content string) (*gittest.Repo, string) {
		repo := gittest.NewRepo(t)
		repo.Write(".abcd/config.json", content)
		repo.Commit("config")
		return repo, repo.Git("rev-parse", "HEAD")
	}
	unreadAt := func(t *testing.T, repo *gittest.Repo, at string) *UnreadError {
		t.Helper()
		_, err := Load(Config, Roots{Repo: repo.Root(), Home: t.TempDir(), RepoAt: at, RepoAtName: "main"})
		var u *UnreadError
		if !errors.As(err, &u) {
			t.Fatalf("err = %v (%T), want an *UnreadError", err, err)
		}
		return u
	}
	t.Run("not JSON", func(t *testing.T) {
		repo, at := commit(t, "{")
		if u := unreadAt(t, repo, at); u.GitFailed() || !strings.Contains(u.Error(), "not valid JSON") {
			t.Errorf("GitFailed = %t over %q; want false, the file refused", u.GitFailed(), u.Error())
		}
	})
	t.Run("not a regular file", func(t *testing.T) {
		repo, _ := commit(t, "{}")
		repo.Remove(".abcd/config.json")
		repo.Git("-c", "core.symlinks=true", "update-index", "--add", "--cacheinfo", "120000,"+strings.TrimSpace(hashObject(t, repo, "/etc/passwd"))+",.abcd/config.json")
		repo.Git("commit", "-q", "-m", "a symlink in its place")
		if u := unreadAt(t, repo, repo.Git("rev-parse", "HEAD")); u.GitFailed() || !strings.Contains(u.Error(), "not a regular file") {
			t.Errorf("GitFailed = %t over %q; want false, the file refused", u.GitFailed(), u.Error())
		}
	})
	for _, verb := range []string{"ls-tree", "cat-file"} {
		t.Run(verb+" fails", func(t *testing.T) {
			repo, at := commit(t, `{"facilitator":{"mode":"auto"}}`)
			gittest.ShimGit(t, []string{verb}, "", 128)
			if u := unreadAt(t, repo, at); !u.GitFailed() {
				t.Errorf("GitFailed = false over %q; want true, git did not answer", u.Error())
			}
		})
	}
}
