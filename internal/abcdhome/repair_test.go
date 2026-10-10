package abcdhome_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestPrintedRepairReconnectsMovedWorktrees (iss-2610040147016103, the spec's
// open question 6): git records a worktree's location in absolute form, so the
// rename the stop names leaves every worktree in the store listed by its
// repository as prunable, and a prune would delete its link. The stop's lines
// print one loop the person runs after the rename; this runs that exact loop,
// through sh, against a store worktree moved as the rename moves it, and
// asserts the repository lists the worktree again at its new path and no
// longer as prunable.
func TestPrintedRepairReconnectsMovedWorktrees(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	repo.Commit("init")
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")

	// A lane worktree in the store under the OLD folder, made with plain git as
	// a session makes one today.
	oldLane := filepath.Join(home, abcdhome.OldName, "worktrees", sha, "lane-one")
	if err := os.MkdirAll(filepath.Dir(oldLane), 0o700); err != nil {
		t.Fatal(err)
	}
	repo.Git("worktree", "add", "-b", "lane-one", oldLane)

	// The person's rename: the whole home, as `mv ~/.abcd ~/.abcd.noindex` does.
	if err := os.Rename(filepath.Join(home, abcdhome.OldName), abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	newLane := abcdhome.Path(home, "worktrees", sha, "lane-one")
	if list := repo.Git("worktree", "list", "--porcelain"); !strings.Contains(list, "prunable") {
		t.Fatalf("precondition: the moved worktree is not listed as prunable, so the fixture did not stage iss-2610040147016103:\n%s", list)
	}

	cmd := exec.Command("sh", "-c", abcdhome.RepairCommand)
	cmd.Dir = home
	cmd.Env = repo.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the printed repair %q failed: %v\n%s", abcdhome.RepairCommand, err, out)
	}

	list := repo.Git("worktree", "list", "--porcelain")
	if strings.Contains(list, "prunable") {
		t.Fatalf("after the printed repair the repository still lists a prunable worktree:\n%s", list)
	}
	real, err := filepath.EvalSymlinks(newLane)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(list, "worktree "+real+"\n") && !strings.Contains(list, "worktree "+newLane+"\n") {
		t.Fatalf("after the printed repair the worktree is not listed at its new path %q:\n%s", newLane, list)
	}
	if got := strings.TrimSpace(gitIn(t, repo.Env(), newLane, "rev-parse", "--abbrev-ref", "HEAD")); got != "lane-one" {
		t.Fatalf("the moved worktree does not resolve its own branch after the repair: %q", got)
	}
}

func gitIn(t *testing.T, env []string, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return string(out)
}
