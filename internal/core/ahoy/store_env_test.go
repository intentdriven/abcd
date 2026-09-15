package ahoy

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestRunGitIgnoresInheritedGitDir proves an inherited GIT_DIR cannot redirect
// runGit at a different repository: RootCommit must answer about `cwd`, not
// the env-selected repo. Without the IsolatedEnv scrub, GIT_DIR overrides `-C`,
// so the wrong root-commit SHA (and origin URL) would be registered against the
// cross-repo history store under a supposedly immutable key.
func TestRunGitIgnoresInheritedGitDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repoA := idGitRepo(t, "A", "a@example.com")
	idMustGit(t, repoA, "commit", "--allow-empty", "-m", "root-A")
	revList := exec.Command("git", "-C", repoA, "rev-list", "--max-parents=0", "HEAD")
	revList.Env = gittest.Env(t)
	outA, err := revList.Output()
	if err != nil {
		t.Fatalf("ground-truth rev-list on repoA: %v", err)
	}
	shaA := strings.TrimSpace(string(outA))

	repoB := idGitRepo(t, "B", "b@example.com")
	idMustGit(t, repoB, "commit", "--allow-empty", "-m", "root-B-different")

	// A leftover GIT_DIR (the bare-repo dotfiles pattern, or a rebase -x descendant)
	// pointing at repoB.
	t.Setenv("GIT_DIR", filepath.Join(repoB, ".git"))

	if got := gitutil.RootCommit(repoA); got != shaA {
		t.Errorf("RootCommit(repoA) = %q under inherited GIT_DIR=repoB; want repoA's root %q (GIT_DIR hijacked the query)", got, shaA)
	}
}
