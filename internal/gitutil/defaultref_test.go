package gitutil_test

import (
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestDefaultRefResolvesInItsDeclaredOrder pins the one resolution the peers
// reader and the reviews board share: origin/HEAD's target first, then the
// conventional names on the remote, then the same names locally, and "" when
// nothing resolves (the board then counts against HEAD). A dangling origin/HEAD
// is no answer, so the probe falls through rather than naming a ref git cannot
// read.
func TestDefaultRefResolvesInItsDeclaredOrder(t *testing.T) {
	fresh := func(t *testing.T) *gittest.Repo {
		t.Helper()
		r := gittest.NewRepo(t)
		r.Commit("c0")
		return r
	}
	cases := []struct {
		name  string
		setup func(r *gittest.Repo)
		want  string
	}{
		{"origin/HEAD wins over every name", func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/trunk", "HEAD")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
		}, "refs/remotes/origin/trunk"},
		{"a dangling origin/HEAD falls through", func(r *gittest.Repo) {
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
		}, "refs/heads/main"},
		{"the remote's main before the local one", func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
		}, "refs/remotes/origin/main"},
		{"a local main", func(r *gittest.Repo) {}, "refs/heads/main"},
		{"a local master", func(r *gittest.Repo) {
			r.Git("branch", "-m", "main", "master")
		}, "refs/heads/master"},
		{"no default branch at all", func(r *gittest.Repo) {
			r.Git("branch", "-m", "main", "feature")
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := fresh(t)
			c.setup(r)
			if got := gitutil.DefaultRef(r.Root()); got != c.want {
				t.Fatalf("DefaultRef = %q, want %q", got, c.want)
			}
		})
	}
}

// TestDefaultRefOutsideARepositoryIsEmpty: no repository resolves no branch,
// and says so with "" rather than an error the caller would have to invent a
// meaning for.
func TestDefaultRefOutsideARepositoryIsEmpty(t *testing.T) {
	if got := gitutil.DefaultRef(t.TempDir()); got != "" {
		t.Fatalf("DefaultRef outside a repository = %q, want \"\"", got)
	}
}

// TestRemoteDefaultBranchNamesOnlyTheRemotesRef: the remote's default branch
// as last fetched is its HEAD's target, then the conventional names under that
// remote, and never a local branch, so a checkout with no remote-tracking
// default answers "" rather than a branch the remote may not hold.
func TestRemoteDefaultBranchNamesOnlyTheRemotesRef(t *testing.T) {
	cases := []struct {
		name  string
		setup func(r *gittest.Repo)
		want  string
	}{
		{"the remote's HEAD wins", func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/trunk", "HEAD")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
		}, "refs/remotes/origin/trunk"},
		{"a dangling HEAD falls to the conventional names", func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/master", "HEAD")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
		}, "refs/remotes/origin/master"},
		{"a local main alone is no answer", func(r *gittest.Repo) {}, ""},
		{"another remote's branch is no answer", func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/upstream/main", "HEAD")
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Commit("c0")
			c.setup(r)
			if got := gitutil.RemoteDefaultBranch(r.Root(), "origin"); got != c.want {
				t.Fatalf("RemoteDefaultBranch = %q, want %q", got, c.want)
			}
		})
	}
	if got := gitutil.RemoteDefaultBranch(t.TempDir(), "origin"); got != "" {
		t.Fatalf("RemoteDefaultBranch outside a repository = %q, want \"\"", got)
	}
}
