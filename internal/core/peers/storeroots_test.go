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

// ownStore is where this repository's embargo store is made: under its own
// root commit.
func (f *fixture) ownStore() string {
	f.t.Helper()
	return abcdhome.Path(f.home, "worktrees", gitutil.RootCommit(f.here()), storepath.Name)
}

// TestPeersSkipsEmbargoStoreWhicheverRootGitListsFirst: after an empty-diff
// merge of an unrelated root, which `rev-list` now meets first, the store
// made under the repository's own root is still skipped unread.
func TestPeersSkipsEmbargoStoreWhicheverRootGitListsFirst(t *testing.T) {
	f := newFixture(t)
	f.addStore(f.ownStore())
	f.mergeUnrelatedRoot()
	f.wantStoresSkipped(1)
}

// TestPeersSkipsEveryEmbargoStore: with two embargo stores registered, one
// under a merged root and one under a key that is no root at all, the
// store's owner refuses to choose, and the listing skips every one unread.
func TestPeersSkipsEveryEmbargoStore(t *testing.T) {
	f := newFixture(t)
	f.addStore(f.ownStore())
	other := f.mergeUnrelatedRoot()
	f.addStore(abcdhome.Path(f.home, "worktrees", other, storepath.Name))
	f.addStore(abcdhome.Path(f.home, "worktrees", strings.Repeat("ab", 20), storepath.Name))
	f.wantStoresSkipped(3)
}

// TestPeersSkipsEmbargoStorePastManyMergedRoots: a merge that brings in
// 1,400 parentless commits does not expose the store: it is skipped unread.
func TestPeersSkipsEmbargoStorePastManyMergedRoots(t *testing.T) {
	f := newFixture(t)
	f.addStore(f.ownStore())
	f.repo.MergeParentlessRoots(1400)
	f.wantStoresSkipped(1)
}

// TestPeersSkipsEmbargoStoreWhenItCannotBeLocated: when the store's own
// lookup fails (here the store's lock reason is longer than any field the
// lookup's listing reads), the listing still skips the store unread.
func TestPeersSkipsEmbargoStoreWhenItCannotBeLocated(t *testing.T) {
	f := newFixture(t)
	store := f.ownStore()
	f.addStore(store)
	gitfile, err := os.ReadFile(filepath.Join(store, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	admin, ok := strings.CutPrefix(strings.TrimSpace(string(gitfile)), "gitdir: ")
	if !ok {
		t.Fatal("the store's .git is not a gitfile")
	}
	if err := os.WriteFile(filepath.Join(admin, "locked"), []byte(strings.Repeat("r", 2<<20)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := storepath.Locate(f.home, f.here()); err == nil {
		t.Fatal("the store's lookup succeeded, so the fixture proves nothing")
	}
	f.wantStoresSkipped(1)
}

// This file spawns git, so it imports gittest (TestTestGitCallsAreHermetic).
var _ = gittest.NewRepo
