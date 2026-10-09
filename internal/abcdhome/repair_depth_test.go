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

// TestPrintedRepairReachesWorktreesAtAnyDepth is iss-2610050728100598: the
// first printed repair globbed worktrees/*/*, so it reached only a worktree
// exactly two levels down. One named after a slashed branch
// (<root-sha>/docs/some-branch) sits three levels down and was never repaired,
// git running in the docs/ folder above it ("not a git repository"); one left
// directly under worktrees/ had git run on each of its files ("Not a
// directory"). The store here holds all three shapes, a name with a space, an
// empty branch-prefix folder a removed worktree leaves, a stray log at the
// store's top, a nested .git file inside a worktree that names no repository,
// and a clone with a submodule, all made under the OLD folder; the folder is
// renamed as the stop says, and the exact printed command runs through sh.
// Every repository link must then name the worktree's new place, with no error
// line and exit 0, and the clone's submodule link must be left byte for byte
// as it was: the walk stops at a .git folder as it stops at a .git file.
func TestPrintedRepairReachesWorktreesAtAnyDepth(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	repo.Commit("init")
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")

	oldStore := filepath.Join(home, abcdhome.OldName, "worktrees")
	lanes := []struct{ rel, branch string }{
		{filepath.Join(sha, "lane one"), "lane-one"},
		{filepath.Join(sha, "docs", "some-branch"), "docs/some-branch"},
		{"legacy-lane", "legacy-lane"},
	}
	for _, l := range lanes {
		p := filepath.Join(oldStore, l.rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		repo.Git("worktree", "add", "-b", l.branch, p)
	}
	for _, dir := range []string{filepath.Join(oldStore, sha, "fix"), filepath.Join(oldStore, sha, "lane one", "vendor", "inner")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldStore, sha, "lane one", "vendor", "inner", ".git"), []byte("gitdir: /nonexistent/.git\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldStore, "test.log"), []byte("log\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A clone placed in the store, its .git a folder, holding a submodule whose
	// .git is a FILE naming its gitdir relatively. The repair has no business in
	// a clone: the walk must stop at its .git folder, or it reaches the
	// submodule and `git worktree repair` there rewrites the relative link to an
	// absolute one.
	sub := gittest.NewRepo(t)
	sub.Write("lib.txt", "lib\n")
	sub.Commit("init")
	clone := filepath.Join(oldStore, sha, "a-clone")
	gitIn(t, repo.Env(), oldStore, "clone", "--quiet", repo.Root(), clone)
	gitIn(t, repo.Env(), clone, "-c", "protocol.file.allow=always", "submodule", "--quiet", "add", sub.Root(), "vendored")

	if err := os.Rename(filepath.Join(home, abcdhome.OldName), abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	subGit := abcdhome.Path(home, "worktrees", sha, "a-clone", "vendored", ".git")
	subLinkBefore, err := os.ReadFile(subGit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(subLinkBefore), "gitdir: ../") {
		t.Fatalf("precondition: the submodule's .git names its gitdir as %q, want git's relative form", subLinkBefore)
	}
	if list := repo.Git("worktree", "list", "--porcelain"); strings.Count(list, "prunable") != len(lanes) {
		t.Fatalf("precondition: want all %d moved worktrees listed as prunable:\n%s", len(lanes), list)
	}

	cmd := exec.Command("sh", "-c", abcdhome.RepairCommand)
	cmd.Dir = home
	cmd.Env = repo.Env()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the printed repair %q exited %v:\n%s", abcdhome.RepairCommand, err, out)
	}
	for _, bad := range []string{"not a git repository", "Not a directory", "No such file", "fatal:", "error:"} {
		if strings.Contains(string(out), bad) {
			t.Errorf("the printed repair said %q:\n%s", bad, out)
		}
	}

	if list := repo.Git("worktree", "list", "--porcelain"); strings.Contains(list, "prunable") {
		t.Errorf("after the printed repair the repository still lists a prunable worktree:\n%s", list)
	}
	if after, err := os.ReadFile(subGit); err != nil || string(after) != string(subLinkBefore) {
		t.Errorf("the printed repair walked into a clone in the store and rewrote its submodule's .git from %q to %q (err %v)", subLinkBefore, after, err)
	}
	for _, l := range lanes {
		wt := abcdhome.Path(home, "worktrees", l.rel)
		if back := backLink(t, wt); !sameFile(back, filepath.Join(wt, ".git")) {
			t.Errorf("worktree %s: its repository's link names %q, not its own .git", l.rel, back)
		}
		if got := strings.TrimSpace(gitIn(t, repo.Env(), wt, "rev-parse", "--abbrev-ref", "HEAD")); got != l.branch {
			t.Errorf("worktree %s resolves branch %q after the repair, want %q", l.rel, got, l.branch)
		}
	}
}

// backLink is the path the repository's admin entry for the worktree at wt
// records for it (<admin>/gitdir), resolved against the admin folder when git
// wrote it relative.
func backLink(t *testing.T, wt string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(wt, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	admin := strings.TrimSpace(strings.TrimPrefix(string(raw), "gitdir:"))
	if !filepath.IsAbs(admin) {
		admin = filepath.Join(wt, admin)
	}
	b, err := os.ReadFile(filepath.Join(admin, "gitdir"))
	if err != nil {
		t.Fatal(err)
	}
	back := strings.TrimSpace(string(b))
	if !filepath.IsAbs(back) {
		back = filepath.Join(admin, back)
	}
	return back
}

func sameFile(a, b string) bool {
	sa, err := os.Stat(a)
	if err != nil {
		return false
	}
	sb, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(sa, sb)
}

// TestWorktreeRepairCommandRunsAsPrinted: the one-worktree repair abcd ahoy
// names is pasted into a shell as it is, so a worktree whose name holds a space
// or an apostrophe must reach git as the one folder it names, the tilde
// expanded to the home directory.
func TestWorktreeRepairCommandRunsAsPrinted(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	repo.Commit("init")
	rel := abcdhome.Rel("worktrees", "it's a lane")
	old := filepath.Join(home, abcdhome.OldName, "worktrees", "it's a lane")
	if err := os.MkdirAll(filepath.Dir(old), 0o700); err != nil {
		t.Fatal(err)
	}
	repo.Git("worktree", "add", "-b", "lane", old)
	if err := os.Rename(filepath.Join(home, abcdhome.OldName), abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", abcdhome.WorktreeRepairCommand(rel))
	cmd.Env = repo.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%q exited %v:\n%s", abcdhome.WorktreeRepairCommand(rel), err, out)
	}
	wt := filepath.Join(home, filepath.FromSlash(rel))
	if back := backLink(t, wt); !sameFile(back, filepath.Join(wt, ".git")) {
		t.Fatalf("after %q the repository's link names %q, not the worktree", abcdhome.WorktreeRepairCommand(rel), back)
	}
}
