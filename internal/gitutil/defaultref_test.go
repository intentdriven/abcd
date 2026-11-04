package gitutil_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// TestTrackedDefaultBranchNamesOnlyTheRemotesRef: the remote's default branch
// as last fetched is its HEAD's target, then the conventional names under that
// remote, and never a local branch, so a checkout with no remote-tracking
// default answers "" rather than a branch the remote may not hold.
func TestTrackedDefaultBranchNamesOnlyTheRemotesRef(t *testing.T) {
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
			if got := gitutil.TrackedDefaultBranch(r.Root(), "origin"); got != c.want {
				t.Fatalf("TrackedDefaultBranch = %q, want %q", got, c.want)
			}
		})
	}
	// The remote named is the one read.
	r := gittest.NewRepo(t)
	r.Commit("c0")
	r.Git("update-ref", "refs/remotes/upstream/main", "HEAD")
	if got := gitutil.TrackedDefaultBranch(r.Root(), "upstream"); got != "refs/remotes/upstream/main" {
		t.Fatalf("TrackedDefaultBranch(upstream) = %q, want refs/remotes/upstream/main", got)
	}
	if got := gitutil.TrackedDefaultBranch(t.TempDir(), "origin"); got != "" {
		t.Fatalf("TrackedDefaultBranch outside a repository = %q, want \"\"", got)
	}
}

// oldDefaultRef is the resolution as a question per ref put it: the answer the
// one listing must keep.
func oldDefaultRef(root, remote string, local bool) string {
	exists := func(ref string) bool {
		_, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
		return err == nil
	}
	prefix := "refs/remotes/" + remote + "/"
	if out, err := gitutil.Run(root, "symbolic-ref", "--quiet", prefix+"HEAD"); err == nil && strings.HasPrefix(out, prefix) && exists(out) {
		return out
	}
	names := []string{"main", "master", "trunk", "develop"}
	for _, n := range names {
		if exists(prefix + n) {
			return prefix + n
		}
	}
	if local {
		for _, n := range names {
			if exists("refs/heads/" + n) {
				return "refs/heads/" + n
			}
		}
	}
	return ""
}

// TestTheListingAnswersAsAQuestionPerRefDid holds the one listing to the
// question per ref it replaced, over the shapes a default branch's refs come in:
// a tag, a tag of a tag, a tree, a HEAD that is no symref or points away, packed
// refs, a ref below a conventional name, a branch spelt in another case, and
// nothing at all.
func TestTheListingAnswersAsAQuestionPerRefDid(t *testing.T) {
	const origin, other = "refs/remotes/origin/", "refs/remotes/upstream/"
	shapes := map[string]func(r *gittest.Repo){
		"a local main": func(r *gittest.Repo) {},
		"nothing": func(r *gittest.Repo) {
			r.Git("branch", "-m", "main", "feature")
		},
		"a branch spelt in another case": func(r *gittest.Repo) {
			r.Git("branch", "-m", "main", "spelt")
			r.Git("branch", "-m", "spelt", "Main")
		},
		"the same, packed": func(r *gittest.Repo) {
			r.Git("branch", "-m", "main", "spelt")
			r.Git("branch", "-m", "spelt", "Main")
			r.Git("pack-refs", "--all")
		},
		"a remote main": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"main", "HEAD")
		},
		"remote names in the wrong order": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"develop", "HEAD")
			r.Git("update-ref", origin+"master", "HEAD")
		},
		"packed remote refs": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"trunk", "HEAD")
			r.Git("symbolic-ref", origin+"HEAD", origin+"trunk")
			r.Git("pack-refs", "--all")
		},
		"HEAD at an unconventional name": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"release", "HEAD")
			r.Git("symbolic-ref", origin+"HEAD", origin+"release")
		},
		"a dangling HEAD": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"master", "HEAD")
			r.Git("symbolic-ref", origin+"HEAD", origin+"gone")
		},
		"a HEAD that is no symref": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"HEAD", "HEAD")
			r.Git("update-ref", origin+"trunk", "HEAD")
		},
		"a HEAD pointing off the remote": func(r *gittest.Repo) {
			r.Git("symbolic-ref", origin+"HEAD", "refs/heads/main")
			r.Git("update-ref", origin+"develop", "HEAD")
		},
		"a HEAD at a symref": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"release", "HEAD")
			r.Git("symbolic-ref", origin+"next", origin+"release")
			r.Git("symbolic-ref", origin+"HEAD", origin+"next")
		},
		"a remote name that is a tag of a commit": func(r *gittest.Repo) {
			r.Git("tag", "-a", "-m", "t", "v1")
			r.Git("update-ref", origin+"main", "refs/tags/v1")
		},
		"a remote name that is a tag of a tag": func(r *gittest.Repo) {
			r.Git("tag", "-a", "-m", "t", "v1")
			r.Git("tag", "-a", "-m", "t", "v2", "v1")
			r.Git("update-ref", origin+"main", "refs/tags/v2")
		},
		"a remote name that is a tag of a tree": func(r *gittest.Repo) {
			r.Git("tag", "-a", "-m", "t", "tree", "HEAD^{tree}")
			r.Git("update-ref", origin+"main", "refs/tags/tree")
			r.Git("update-ref", origin+"master", "HEAD")
		},
		"a remote name that is a tree": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"main", r.Git("rev-parse", "HEAD^{tree}"))
		},
		"a HEAD at a tree": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"release", r.Git("rev-parse", "HEAD^{tree}"))
			r.Git("symbolic-ref", origin+"HEAD", origin+"release")
			r.Git("update-ref", origin+"trunk", "HEAD")
		},
		"a ref below a conventional name": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"main/sub", "HEAD")
			r.Git("branch", "-m", "main", "feature")
		},
		"a symlinked HEAD": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"release", "HEAD")
			r.Git("update-ref", origin+"main", "HEAD")
			r.Git("-c", "core.preferSymlinkRefs=true", "symbolic-ref", origin+"HEAD", origin+"release")
		},
		"a ref below a name whose object is missing": func(r *gittest.Repo) {
			r.Git("update-ref", origin+"release", "HEAD")
			r.Git("update-ref", origin+"main", "HEAD")
			r.Git("symbolic-ref", origin+"HEAD", origin+"release")
			writeLooseRef(t, r, "refs/heads/develop/lost", strings.Repeat("d", 40))
		},
		"another remote's refs": func(r *gittest.Repo) {
			r.Git("update-ref", other+"main", "HEAD")
			r.Git("branch", "-m", "main", "feature")
		},
	}
	for name, setup := range shapes {
		t.Run(name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Commit("c0")
			setup(r)
			if got, want := gitutil.DefaultRef(r.Root()), oldDefaultRef(r.Root(), "origin", true); got != want {
				t.Errorf("DefaultRef = %q, a question per ref said %q", got, want)
			}
			if got, want := gitutil.TrackedDefaultBranch(r.Root(), "origin"), oldDefaultRef(r.Root(), "origin", false); got != want {
				t.Errorf("TrackedDefaultBranch = %q, a question per ref said %q", got, want)
			}
		})
	}
	t.Run("an unborn branch", func(t *testing.T) {
		r := gittest.NewRepo(t)
		if got := gitutil.DefaultRef(r.Root()); got != "" {
			t.Errorf("DefaultRef = %q with no commit at all", got)
		}
	})
}

// writeLooseRef writes a ref's file directly, which is what lets a test hold a
// ref git would refuse to make, or thousands of them without a process each.
func writeLooseRef(t *testing.T, r *gittest.Repo, ref, sha string) {
	t.Helper()
	path := filepath.Join(r.Root(), ".git", filepath.FromSlash(ref))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(sha+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestABoundedListingNeverAnswersFromPartOfItself: a pattern for a conventional
// name also lists every ref below it, and local refs sort first, so branches
// under refs/heads/trunk/ can fill a bounded read before the remote's rows. The
// answer must stay what a question per ref gives, not what the part read says.
func TestABoundedListingNeverAnswersFromPartOfItself(t *testing.T) {
	const origin = "refs/remotes/origin/"
	shapes := map[string]struct {
		setup func(r *gittest.Repo)
		want  string
	}{
		"HEAD at an unconventional name": {func(r *gittest.Repo) {
			r.Git("update-ref", origin+"release", "HEAD")
			r.Git("update-ref", origin+"main", "HEAD")
			r.Git("symbolic-ref", origin+"HEAD", origin+"release")
		}, origin + "release"},
		"no HEAD, the remote's main": {func(r *gittest.Repo) {
			r.Git("update-ref", origin+"main", "HEAD")
		}, origin + "main"},
	}
	for name, c := range shapes {
		t.Run(name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Commit("c0")
			c.setup(r)
			sha := r.Git("rev-parse", "HEAD")
			for i := range 2000 {
				writeLooseRef(t, r, fmt.Sprintf("refs/heads/trunk/b%04d", i), sha)
			}
			if got := gitutil.DefaultRefContext(context.Background(), r.Root()); got != c.want {
				t.Errorf("DefaultRefContext = %q, want %q", got, c.want)
			}
		})
	}
}

// TestTheDefaultBranchIsAListingAndOneQuestion: where a default branch resolves,
// the resolution is one git process with origin/HEAD in the listing, and two
// without it, since only symbolic-ref can see a HEAD the listing does not show.
func TestTheDefaultBranchIsAListingAndOneQuestion(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("c0")
	r.Git("branch", "-m", "main", "develop")
	real, err := gitPath()
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	shim := "#!/bin/sh\necho x >>'" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if got := gitutil.DefaultRef(r.Root()); got != "refs/heads/develop" {
		t.Fatalf("DefaultRef = %q, want the local develop", got)
	}
	calls, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(calls), "x"); n != 2 {
		t.Errorf("the resolution ran %d git processes with no origin/HEAD, want two", n)
	}

	r.Git("update-ref", "refs/remotes/origin/release", "HEAD")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/release")
	if err := os.Remove(log); err != nil {
		t.Fatal(err)
	}
	if got := gitutil.DefaultRef(r.Root()); got != "refs/remotes/origin/release" {
		t.Fatalf("DefaultRef = %q, want origin/HEAD's target", got)
	}
	calls, err = os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(calls), "x"); n != 1 {
		t.Errorf("the resolution ran %d git processes with origin/HEAD, want one", n)
	}
}

func gitPath() (string, error) { return exec.LookPath("git") }
