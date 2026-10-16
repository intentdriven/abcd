package embargo

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// mergeUnrelatedRoot merges, with an empty diff (`merge -s ours`), a root
// commit of an unrelated history dated far ahead, so `rev-list` meets it
// before the repository's own root. It returns the merged root.
func mergeUnrelatedRoot(t *testing.T, r *gittest.Repo) string {
	t.Helper()
	const date = "2090-01-01T00:00:00Z"
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"commit-tree", r.Git("hash-object", "-t", "tree", "/dev/null"), "-m", "an unrelated root")
	cmd.Env = append(r.Env(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("commit-tree: %v\n%s", err, out)
	}
	other := strings.TrimSpace(string(out))
	r.Git("merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", other)
	if gitutil.RootCommit(r.Root()) != other {
		t.Fatal("the fixture's rev-list does not meet the merged root first")
	}
	return other
}

// TestStoreFoundWhicheverRootGitListsFirst: after an empty-diff merge of an
// unrelated root, which `rev-list` now meets first, the store made before it
// is still found: the embargoed record is still refused, the check does
// not report "no store", and opening the store finds it and makes no second
// one under the merged root.
func TestStoreFoundWhicheverRootGitListsFirst(t *testing.T) {
	cf := newCheckFixture(t)
	other := mergeUnrelatedRoot(t, cf.f.repo)

	res := cf.text(t, "fixes "+ghsaOpen+"\n")
	if res.NoStore {
		t.Fatalf("the check reported no store after the merge: %+v", res)
	}
	wantKinds(t, res, KindAdvisoryID)

	s, created, err := openStore(cf.f.home, cf.f.repo.Root())
	if err != nil || created || s.Root != cf.s.Root {
		t.Fatalf("openStore = %+v, created %v, %v; want the store at %s, found", s, created, err, cf.s.Root)
	}
	second := abcdhome.Path(cf.f.home, "worktrees", other, StoreName)
	if _, err := os.Lstat(second); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a second store was made under the merged root (%v)", err)
	}
}

// TestCheckRefusesTheFoundStoresKeyedPath: the store-path form the check
// refuses names the root commit the found store is keyed on, not the root
// `rev-list` meets first.
func TestCheckRefusesTheFoundStoresKeyedPath(t *testing.T) {
	cf := newCheckFixture(t)
	key := filepath.Base(filepath.Dir(cf.s.Root))
	mergeUnrelatedRoot(t, cf.f.repo)
	wantKinds(t, cf.text(t, "under worktrees/"+key+"/embargo/.abcd\n"), KindStorePath)
}

// TestStoresUnderTwoRootsAreRefused: with a store under each of two roots of
// the default branch, which one holds the records cannot be told, so the
// check, the opening of an existing store and the making of one all refuse,
// naming neither store nor root, and nothing is made.
func TestStoresUnderTwoRootsAreRefused(t *testing.T) {
	cf := newCheckFixture(t)
	other := mergeUnrelatedRoot(t, cf.f.repo)
	second := abcdhome.Path(cf.f.home, "worktrees", other, StoreName)
	if err := os.MkdirAll(second, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"}, Stdin: strings.NewReader("a text\n")})
	mustRefuse(t, err, "more than one root commit")
	_, _, err = openExisting(cf.f.home, cf.f.repo.Root())
	mustRefuse(t, err, "more than one root commit")
	_, _, err = openStore(cf.f.home, cf.f.repo.Root())
	mustRefuse(t, err, "more than one root commit")
	for _, private := range []string{other, filepath.Base(filepath.Dir(cf.s.Root)), cf.f.home} {
		if strings.Contains(err.Error(), private) {
			t.Fatalf("the refusal %q names %s", err, private)
		}
	}
	entries, err := os.ReadDir(second)
	if err != nil || len(entries) != 0 {
		t.Fatalf("the second store's place was written to: %v, %v", entries, err)
	}
}

// TestOrdinaryCloneKeepsItsStorePath: in a clone with one root, the store is
// made where it always was, under that root.
func TestOrdinaryCloneKeepsItsStorePath(t *testing.T) {
	f := newFixture(t)
	f.repo.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	f.repo.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	s, created, err := openStore(f.home, f.repo.Root())
	if err != nil || !created {
		t.Fatalf("openStore: created %v, %v", created, err)
	}
	if want := abcdhome.Path(f.home, "worktrees", gitutil.RootCommit(f.repo.Root()), StoreName); s.Root != want {
		t.Fatalf("store at %s; want %s", s.Root, want)
	}
}
