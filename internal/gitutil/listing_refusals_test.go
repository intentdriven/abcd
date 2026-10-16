package gitutil_test

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestEachWorktreePartialNULListingIsAnError: a NUL-separated listing that
// hands over a record and then fails is an error, never a cue to read the
// newline-separated listing instead, though that one would answer whole: a
// git that understood -z and failed partway is a failure, not an old git.
func TestEachWorktreePartialNULListingIsAnError(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("c0")
	r.Git("worktree", "add", "-q", "--detach", filepath.Join(t.TempDir(), "wt"))
	if got, err := eachWorktree(t, r.Root()); err != nil || len(got) != 2 {
		t.Fatalf("the fixture's own listing = %+v, %v; want 2 worktrees", got, err)
	}
	gittest.ShimGit(t, []string{"worktree", "list", "-z"}, "worktree "+r.Root()+"\x00HEAD "+r.Git("rev-parse", "HEAD")+"\x00\x00worktree ", 1)
	if got, err := eachWorktree(t, r.Root()); err == nil {
		t.Fatalf("EachWorktree = %+v and no error; want the partial listing refused", got)
	}
}

// TestDefaultBranchRootsRefusesWhatGitNeverWrites: a root listing with a line
// of a shape git never writes, a root listing git fails, and a HEAD that git
// fails to resolve (other than "no such commit") are each an error, never a
// list of roots and never "no commits".
func TestDefaultBranchRootsRefusesWhatGitNeverWrites(t *testing.T) {
	for _, tc := range []struct {
		name string
		when []string
		out  string
		code int
	}{
		{"a line of an unexpected shape", []string{"rev-list", "--max-parents=0"}, "not-a-time " + "0123456789abcdef0123456789abcdef01234567\n", 0},
		{"a line with no date", []string{"rev-list", "--max-parents=0"}, "0123456789abcdef0123456789abcdef01234567\n", 0},
		{"a failed listing", []string{"rev-list", "--max-parents=0"}, "", 128},
		{"a HEAD git fails to resolve", []string{"rev-parse", "HEAD^{commit}"}, "", 128},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Write("README.md", "a project\n")
			r.Commit("c0")
			gittest.ShimGit(t, tc.when, tc.out, tc.code)
			if got, err := gitutil.DefaultBranchRoots(r.Root()); err == nil {
				t.Fatalf("DefaultBranchRoots = %q and no error", got)
			}
		})
	}
}
