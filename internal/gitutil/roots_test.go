package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// datedRoot makes a commit with no parent, holding the empty tree, whose
// committer date is date, and returns its object name.
func datedRoot(t *testing.T, r *gittest.Repo, date, msg string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"commit-tree", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "-m", msg)
	cmd.Env = append(r.Env(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("commit-tree: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestDefaultBranchRootsListsEveryRoot: every root commit the default branch
// holds is listed, oldest committer date first, so a merge that brings in
// another root adds it to the list rather than changing which one is first,
// whichever root `rev-list` itself meets first.
func TestDefaultBranchRootsListsEveryRoot(t *testing.T) {
	setup := func(t *testing.T) (*gittest.Repo, string) {
		t.Helper()
		r := gittest.NewRepo(t)
		r.Write("README.md", "a project\n")
		r.Commit("c0")
		return r, r.Git("rev-parse", "HEAD")
	}
	t.Run("an ordinary repository lists its one root", func(t *testing.T) {
		r, root := setup(t)
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || !slices.Equal(got, []string{root}) || root != gitutil.RootCommit(r.Root()) {
			t.Fatalf("DefaultBranchRoots = %q, %v; want [%s], RootCommit's answer", got, err, root)
		}
	})
	t.Run("a newer root merged with an empty diff comes after", func(t *testing.T) {
		r, root := setup(t)
		other := datedRoot(t, r, "2090-01-01T00:00:00Z", "an unrelated root")
		r.Git("merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", other)
		if gitutil.RootCommit(r.Root()) != other {
			t.Fatal("the fixture's rev-list does not meet the merged root first")
		}
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || !slices.Equal(got, []string{root, other}) {
			t.Fatalf("DefaultBranchRoots = %q, %v; want [%s %s]", got, err, root, other)
		}
	})
	t.Run("an older root merged in comes first", func(t *testing.T) {
		r, root := setup(t)
		other := datedRoot(t, r, "2001-01-01T00:00:00Z", "an older unrelated root")
		r.Git("merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", other)
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || !slices.Equal(got, []string{other, root}) {
			t.Fatalf("DefaultBranchRoots = %q, %v; want [%s %s]", got, err, other, root)
		}
	})
	t.Run("roots of one date are ordered by object name", func(t *testing.T) {
		r := gittest.NewRepo(t)
		a := datedRoot(t, r, "2020-01-01T00:00:00Z", "a")
		b := datedRoot(t, r, "2020-01-01T00:00:00Z", "b")
		r.Git("update-ref", "refs/heads/main", a)
		r.Git("reset", "-q", "--hard", "main")
		r.Git("merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", b)
		want := []string{a, b}
		slices.Sort(want)
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("DefaultBranchRoots = %q, %v; want %q", got, err, want)
		}
	})
	t.Run("HEAD on an orphan branch lists the default branch's roots", func(t *testing.T) {
		r, root := setup(t)
		r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
		r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
		r.Git("checkout", "-q", "--orphan", "elsewhere")
		r.Commit("an unrelated root")
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || !slices.Equal(got, []string{root}) {
			t.Fatalf("DefaultBranchRoots = %q, %v; want the default branch's [%s]", got, err, root)
		}
	})
	t.Run("a replace ref does not change the answer", func(t *testing.T) {
		r, root := setup(t)
		r.Write("NEXT.md", "more\n")
		r.Commit("c1")
		tip := r.Git("rev-parse", "HEAD")
		r.Git("replace", tip, r.Git("commit-tree", r.Git("rev-parse", "HEAD^{tree}"), "-m", "a stand-in root"))
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || !slices.Equal(got, []string{root}) {
			t.Fatalf("DefaultBranchRoots = %q, %v; want the stored [%s], not the tip %s", got, err, root, tip)
		}
	})
	t.Run("a graft file does not change the answer", func(t *testing.T) {
		r, root := setup(t)
		r.Write("NEXT.md", "more\n")
		r.Commit("c1")
		tip := r.Git("rev-parse", "HEAD")
		if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "grafts"), []byte(tip+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || !slices.Equal(got, []string{root}) {
			t.Fatalf("DefaultBranchRoots = %q, %v; want the stored [%s], not the grafted tip %s", got, err, root, tip)
		}
	})
	t.Run("a repository with no commits lists none", func(t *testing.T) {
		r := gittest.NewRepo(t)
		got, err := gitutil.DefaultBranchRoots(r.Root())
		if err != nil || len(got) != 0 {
			t.Fatalf("DefaultBranchRoots = %q, %v; want none, no error", got, err)
		}
	})
}

// TestDefaultBranchRootsReadsEveryRootOfALargeMerge: a merge that brings in
// 1,400 parentless commits, about 73 KB of rev-list answer, is read whole and
// in order, never refused for its size: one merged change must not stop
// every reader of the list.
func TestDefaultBranchRootsReadsEveryRootOfALargeMerge(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("c0")
	root := r.Git("rev-parse", "HEAD")
	r.MergeParentlessRoots(1400)
	got, err := gitutil.DefaultBranchRoots(r.Root())
	if err != nil {
		t.Fatalf("DefaultBranchRoots: %v", err)
	}
	if len(got) != 1401 || got[0] != root {
		t.Fatalf("DefaultBranchRoots listed %d roots, first %q; want 1401, the repository's own root %s first", len(got), got[0], root)
	}
}
