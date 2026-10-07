package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// movedStore stands up, under home, a store whose worktrees were made under
// the OLD folder name and then moved with the folder, as the rename does: a
// lane two levels down, one named after a slashed branch three levels down,
// and a legacy one directly under worktrees/. A nested .git file inside a
// worktree names no repository, and a .git file in an empty prefix folder
// names an admin folder that does not exist, so a detector that walks into
// worktrees or trusts any .git file reports what is not there. It returns the
// repository and the worktrees' paths relative to home.
func movedStore(t *testing.T, home string) (*gittest.Repo, []string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	repo.Commit("init")
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
	old := filepath.Join(home, ".abcd", "worktrees")
	lanes := []struct{ rel, branch string }{
		{sha + "/lane one", "lane-one"},
		{sha + "/docs/some-branch", "docs/some-branch"},
		{"legacy-lane", "legacy-lane"},
	}
	var rels []string
	for _, l := range lanes {
		p := filepath.Join(old, filepath.FromSlash(l.rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		repo.Git("worktree", "add", "-b", l.branch, p)
		rels = append(rels, abcdhome.Rel("worktrees", l.rel))
	}
	nested := filepath.Join(old, sha, "lane one", "vendor", "inner")
	stray := filepath.Join(old, sha, "fix", "gone")
	for _, dir := range []string{nested, stray} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /nonexistent/.git/worktrees/x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The hermetic home already holds the new folder, so the store alone moves
	// into it, which leaves git with the same stale links the rename leaves.
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(old, abcdhome.Path(home, "worktrees")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	return repo, rels
}

func storeWorktreeGaps(gaps []Gap) []Gap {
	var out []Gap
	for _, g := range gaps {
		if g.ID == StoreWorktreeUnlinkedGapID {
			out = append(out, g)
		}
	}
	return out
}

// TestAhoyNamesEachStoreWorktreeTheRenameLeftUnlinked (iss-2610050728100598):
// after the rename a worktree the printed repair missed is listed by its
// repository as prunable, and a prune drops it. abcd ahoy names each one, by
// its place under the home, with the one command that repairs it, and only
// reads: the repository's admin records are byte-identical afterwards. Running
// each named command, as printed, through sh clears the gap.
func TestAhoyNamesEachStoreWorktreeTheRenameLeftUnlinked(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	home, _ := setupHermetic(t)
	repo, rels := movedStore(t, home)
	adminBefore := treeBytes(t, filepath.Join(repo.Root(), ".git", "worktrees"))

	res, err := Detect(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	gaps := storeWorktreeGaps(res.Gaps)
	if len(gaps) != len(rels) {
		t.Fatalf("want one %s gap per unlinked worktree (%d), got %d: %+v", StoreWorktreeUnlinkedGapID, len(rels), len(gaps), gaps)
	}
	for _, rel := range rels {
		var found *Gap
		for i := range gaps {
			if strings.Contains(gaps[i].Detail, "~/"+rel+":") {
				found = &gaps[i]
			}
		}
		if found == nil {
			t.Errorf("no gap names ~/%s: %+v", rel, gaps)
			continue
		}
		if found.Category != UserState || found.Scope != "machine" || found.Required || found.Resolvable {
			t.Errorf("gap for %s is not a machine-scope report-only note: %+v", rel, *found)
		}
		cmd := abcdhome.WorktreeRepairCommand(rel)
		if !strings.Contains(found.FixHint, "`"+cmd+"`") {
			t.Errorf("gap for %s does not give its repair %q: %q", rel, cmd, found.FixHint)
		}
		if strings.Contains(found.Detail, home) || strings.Contains(found.FixHint, home) {
			t.Errorf("gap for %s names the home directory in full: %+v", rel, *found)
		}
	}
	if after := treeBytes(t, filepath.Join(repo.Root(), ".git", "worktrees")); after != adminBefore {
		t.Fatalf("detection changed the repository's worktree records:\nbefore:\n%s\nafter:\n%s", adminBefore, after)
	}

	for _, rel := range rels {
		sh := exec.Command("sh", "-c", abcdhome.WorktreeRepairCommand(rel))
		sh.Env = repo.Env()
		if out, err := sh.CombinedOutput(); err != nil {
			t.Fatalf("%q exited %v:\n%s", abcdhome.WorktreeRepairCommand(rel), err, out)
		}
	}
	res, err = Detect(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if left := storeWorktreeGaps(res.Gaps); len(left) != 0 {
		t.Fatalf("after each named repair ran, ahoy still reports: %+v", left)
	}
}

// TestAhoyReportsNoStoreWorktreeWhereEveryLinkHolds: worktrees made under the
// current folder, at every depth, are linked back, so nothing is reported, and
// a home with no store reports nothing either.
func TestAhoyReportsNoStoreWorktreeWhereEveryLinkHolds(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	repo.Commit("init")
	if got := storeWorktreeGaps(detectStoreWorktrees()); len(got) != 0 {
		t.Fatalf("a home with no store reports %+v", got)
	}
	for _, rel := range []string{"sha/lane", "sha/docs/deep/branch", "legacy"} {
		p := abcdhome.Path(home, "worktrees", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		repo.Git("worktree", "add", "-b", strings.ReplaceAll(rel, "/", "-"), p)
	}
	if got := storeWorktreeGaps(detectStoreWorktrees()); len(got) != 0 {
		t.Fatalf("a store whose every worktree is linked reports %+v", got)
	}
}

// treeBytes renders every file under root with its content, so a test can
// prove a pass changed nothing there.
func treeBytes(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		b.WriteString(rel + " " + info.Mode().String())
		if info.Mode().IsRegular() {
			c, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			b.WriteString(" " + string(c))
		}
		b.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
