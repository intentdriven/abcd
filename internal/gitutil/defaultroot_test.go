package gitutil_test

import (
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestDefaultBranchRootCommitReadsOriginsDefaultBranchAsStored: the root
// commit a repository-wide store is keyed on is read from origin's default
// branch, the branch every checkout of the repository shares, with objects
// read as stored. In an ordinary repository it is RootCommit's answer; with
// HEAD on an orphan branch, or a replace ref that cuts the history short,
// it is still the default branch's own root; and with no default branch of
// origin it falls back to HEAD.
func TestDefaultBranchRootCommitReadsOriginsDefaultBranchAsStored(t *testing.T) {
	setup := func(t *testing.T) (*gittest.Repo, string) {
		t.Helper()
		r := gittest.NewRepo(t)
		r.Write("README.md", "a project\n")
		r.Commit("c0")
		root := r.Git("rev-parse", "HEAD")
		r.Write("NEXT.md", "more\n")
		r.Commit("c1")
		return r, root
	}
	t.Run("no origin falls back to HEAD", func(t *testing.T) {
		r, root := setup(t)
		if got := gitutil.DefaultBranchRootCommit(r.Root()); got != root || got != gitutil.RootCommit(r.Root()) {
			t.Fatalf("DefaultBranchRootCommit = %q; want HEAD's root %s", got, root)
		}
	})
	t.Run("an ordinary clone", func(t *testing.T) {
		r, root := setup(t)
		r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
		r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
		if got := gitutil.DefaultBranchRootCommit(r.Root()); got != root || got != gitutil.RootCommit(r.Root()) {
			t.Fatalf("DefaultBranchRootCommit = %q; want %s, RootCommit's answer", got, root)
		}
	})
	t.Run("HEAD on an orphan branch", func(t *testing.T) {
		r, root := setup(t)
		r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
		r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
		r.Git("checkout", "-q", "--orphan", "elsewhere")
		r.Commit("an unrelated root")
		if gitutil.RootCommit(r.Root()) == root {
			t.Fatal("the fixture's HEAD still has the default branch's root")
		}
		if got := gitutil.DefaultBranchRootCommit(r.Root()); got != root {
			t.Fatalf("DefaultBranchRootCommit = %q; want the default branch's root %s", got, root)
		}
	})
	for _, withOrigin := range []bool{true, false} {
		name := map[bool]string{true: "a replace ref cutting the default branch short", false: "a replace ref cutting HEAD short"}[withOrigin]
		t.Run(name, func(t *testing.T) {
			r, root := setup(t)
			if withOrigin {
				r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
				r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
			}
			// The tip is stood in for by a commit with no parent, so read through
			// the replace ref the tip itself is the root.
			tip := r.Git("rev-parse", "HEAD")
			r.Git("replace", tip, r.Git("commit-tree", r.Git("rev-parse", "HEAD^{tree}"), "-m", "a stand-in root"))
			if got := gitutil.DefaultBranchRootCommit(r.Root()); got != root {
				t.Fatalf("DefaultBranchRootCommit = %q; want the stored root %s, not the tip %s", got, root, tip)
			}
		})
	}
}
