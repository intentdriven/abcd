package gitutil_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// lockReason writes reason as the lock reason of the linked worktree at wt,
// straight into its administrative folder, where `git worktree lock` keeps
// it: a reason past the command line's limit is still git's to print.
func lockReason(t *testing.T, wt, reason string) {
	t.Helper()
	gitfile, err := os.ReadFile(filepath.Join(wt, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	admin, ok := strings.CutPrefix(strings.TrimSpace(string(gitfile)), "gitdir: ")
	if !ok {
		t.Fatalf("%s/.git is not a gitfile: %q", wt, gitfile)
	}
	if err := os.WriteFile(filepath.Join(admin, "locked"), []byte(reason), 0o644); err != nil {
		t.Fatal(err)
	}
}

func eachWorktree(t *testing.T, root string) ([]gitutil.Worktree, error) {
	t.Helper()
	var got []gitutil.Worktree
	err := gitutil.EachWorktree(root, func(w gitutil.Worktree) { got = append(got, w) })
	return got, err
}

// TestEachWorktreeMatchesTheListing: the streamed listing hands over the
// same records, in the same order, as ListWorktrees parses.
func TestEachWorktreeMatchesTheListing(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("c0")
	base := t.TempDir()
	r.Git("worktree", "add", "-q", "-b", "feat/one", filepath.Join(base, "one"))
	r.Git("worktree", "add", "-q", "--detach", filepath.Join(base, "two"))
	want, err := gitutil.ListWorktrees(r.Root(), gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	got, err := eachWorktree(t, r.Root())
	if err != nil || !slices.Equal(got, want) || len(got) != 3 {
		t.Fatalf("EachWorktree = %+v, %v; want ListWorktrees' %+v", got, err, want)
	}
}

// TestEachWorktreeHasNoTotalCap: a listing far past the cap a whole-listing
// read takes (twenty worktrees, each locked with a 900 KiB reason, about 18
// MiB) is streamed whole: a repository with many worktrees is never refused
// for the listing's length.
func TestEachWorktreeHasNoTotalCap(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("c0")
	base := t.TempDir()
	reason := strings.Repeat("r", 900<<10)
	for i := range 20 {
		wt := filepath.Join(base, "wt"+string(rune('a'+i)))
		r.Git("worktree", "add", "-q", "--detach", wt)
		lockReason(t, wt, reason)
	}
	if _, err := gitutil.ListWorktrees(r.Root(), gitutil.MaxWorktreeListing); err == nil {
		t.Fatal("the fixture's listing fits the whole-listing cap, so it proves nothing")
	}
	got, err := eachWorktree(t, r.Root())
	if err != nil || len(got) != 21 {
		t.Fatalf("EachWorktree listed %d worktrees, %v; want 21", len(got), err)
	}
}

// TestEachWorktreeRefusesAFieldPastItsBound: one field longer than any path
// (a 2 MiB lock reason) is an error, never a record cut short.
func TestEachWorktreeRefusesAFieldPastItsBound(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("c0")
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "--detach", wt)
	lockReason(t, wt, strings.Repeat("r", 2<<20))
	if got, err := eachWorktree(t, r.Root()); err == nil {
		t.Fatalf("EachWorktree = %+v and no error; want the long field refused", got)
	}
}
