package peers_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// mergeUnrelatedRoot merges, with an empty diff, a root commit of an
// unrelated history dated far ahead, so `rev-list` meets it before the
// repository's own root, and returns it.
func (f *fixture) mergeUnrelatedRoot() string {
	f.t.Helper()
	const date = "2090-01-01T00:00:00Z"
	cmd := exec.Command("git", "-C", f.here(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"commit-tree", f.git(f.here(), "hash-object", "-t", "tree", "/dev/null"), "-m", "an unrelated root")
	cmd.Env = append(f.repo.Env(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("commit-tree: %v\n%s", err, out)
	}
	other := strings.TrimSpace(string(out))
	f.git(f.here(), "merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", other)
	if gitutil.RootCommit(f.here()) != other {
		f.t.Fatal("the fixture's rev-list does not meet the merged root first")
	}
	return other
}

// addStore adds a detached worktree holding a private record at store.
func (f *fixture) addStore(store string) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Dir(store), 0o700); err != nil {
		f.t.Fatal(err)
	}
	f.git(f.here(), "worktree", "add", "-q", "--detach", store, "main")
	f.write(store, ".abcd/work/issues/open/iss-700-private-report.md", issue("iss-700", "A sample private report"))
}

// wantStoresSkipped reads the listing both ways and wants n embargo stores
// skipped unread and nothing else.
func (f *fixture) wantStoresSkipped(n int) {
	f.t.Helper()
	for _, read := range []func(string) (peers.Report, error){peers.Read, peers.Scan} {
		rep, err := read(f.here())
		if err != nil {
			f.t.Fatal(err)
		}
		if rep.Live() != 0 || rep.IDCount() != 0 || len(rep.Locate("iss-700")) != 0 {
			f.t.Fatalf("an embargo store was read as a peer: %+v", rep)
		}
		if len(rep.Skipped) != n {
			f.t.Fatalf("skipped %+v, want %d embargo store(s)", rep.Skipped, n)
		}
		for _, s := range rep.Skipped {
			if s.Reason != peers.SkipEmbargo || s.Path != "" || s.Branch != "" {
				f.t.Fatalf("skipped %+v, want the embargo store alone, unnamed", rep.Skipped)
			}
		}
	}
}

// TestPeersSkipsEmbargoStoreWhicheverRootGitListsFirst: after an empty-diff
// merge of an unrelated root, which `rev-list` now meets first, the store
// made under the repository's own root is still skipped unread.
func TestPeersSkipsEmbargoStoreWhicheverRootGitListsFirst(t *testing.T) {
	f := newFixture(t)
	store, ok := storepath.PathForTest(f.home, f.here())
	if !ok {
		t.Fatal("the fixture repository has no root commit to key the embargo store on")
	}
	f.addStore(store)
	f.mergeUnrelatedRoot()
	f.wantStoresSkipped(1)
}

// TestPeersSkipsEveryEmbargoStoreUnderTwoRoots: with a store under each of
// two roots of the default branch, the store's owner refuses to choose, and
// the listing skips both unread.
func TestPeersSkipsEveryEmbargoStoreUnderTwoRoots(t *testing.T) {
	f := newFixture(t)
	first, ok := storepath.PathForTest(f.home, f.here())
	if !ok {
		t.Fatal("the fixture repository has no root commit to key the embargo store on")
	}
	f.addStore(first)
	other := f.mergeUnrelatedRoot()
	f.addStore(abcdhome.Path(f.home, "worktrees", other, storepath.Name))
	f.wantStoresSkipped(2)
}

// This file spawns git, so it imports gittest (TestTestGitCallsAreHermetic).
var _ = gittest.NewRepo
