package storepath_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/gittest"
)

var (
	key40 = strings.Repeat("a1", 20)
	key64 = strings.Repeat("b2", 32)
)

// newRepo is a repository with one commit, dated date, and returns it with
// its root.
func newRepo(t *testing.T, date string) (*gittest.Repo, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Git("add", "-A")
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false", "commit", "-q", "-m", "root")
	cmd.Env = append(r.Env(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
	return r, r.Git("rev-parse", "HEAD")
}

// addStore registers a detached worktree of r at the store place under key.
func addStore(t *testing.T, r *gittest.Repo, home, key string) string {
	t.Helper()
	p := abcdhome.Path(home, "worktrees", key, storepath.Name)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	r.Git("worktree", "add", "-q", "--detach", p)
	return p
}

func TestIsStorePath(t *testing.T) {
	home := t.TempDir()
	linked := filepath.Join(t.TempDir(), "home-link")
	if err := os.Symlink(home, linked); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		home, p string
		want    bool
	}{
		{home, abcdhome.Path(home, "worktrees", key40, "embargo"), true},
		{home, abcdhome.Path(home, "worktrees", key64, "embargo"), true},
		{home, abcdhome.Path(home, "worktrees", key40, "embargo") + "/", true},
		{home, abcdhome.Path(linked, "worktrees", key40, "embargo"), true},
		{linked, abcdhome.Path(home, "worktrees", key40, "embargo"), true},
		{home, abcdhome.Path(home, "worktrees", strings.ToUpper(key40), "embargo"), false},
		{home, abcdhome.Path(home, "worktrees", key40[:39], "embargo"), false},
		{home, abcdhome.Path(home, "worktrees", key40, "embargo-old"), false},
		{home, abcdhome.Path(home, "worktrees", key40, "embargo", "inner"), false},
		{home, abcdhome.Path(home, "elsewhere", key40, "embargo"), false},
		{home, abcdhome.Path(t.TempDir(), "worktrees", key40, "embargo"), false},
		{"", abcdhome.Path(home, "worktrees", key40, "embargo"), false},
	} {
		if got := storepath.IsStorePath(tc.home, tc.p); got != tc.want {
			t.Errorf("IsStorePath(%q, %q) = %v, want %v", tc.home, tc.p, got, tc.want)
		}
	}
}

func TestHasStoreShape(t *testing.T) {
	for _, tc := range []struct {
		p    string
		want bool
	}{
		{"/anywhere/.abcd.noindex/worktrees/" + key40 + "/embargo", true},
		{"/anywhere/.abcd.noindex/worktrees/" + key64 + "/embargo", true},
		{"/anywhere/.abcd/worktrees/" + key40 + "/embargo", false},
		{"/anywhere/.abcd.noindex/other/" + key40 + "/embargo", false},
		{"/anywhere/.abcd.noindex/worktrees/" + key40 + "/lane", false},
		{"/anywhere/.abcd.noindex/worktrees/main/embargo", false},
	} {
		if got := storepath.HasStoreShape(tc.p); got != tc.want {
			t.Errorf("HasStoreShape(%q) = %v, want %v", tc.p, got, tc.want)
		}
	}
}

// TestLocateNoStoreIsUnderTheOldestRoot: with no store registered, the store
// is to be made under the default branch's oldest root, read as stored.
func TestLocateNoStoreIsUnderTheOldestRoot(t *testing.T) {
	home := t.TempDir()
	r, root := newRepo(t, "2020-01-01T00:00:00Z")
	loc, err := storepath.Locate(home, r.Root())
	want := abcdhome.Path(home, "worktrees", root, storepath.Name)
	if err != nil || loc.Registered || loc.Path != want {
		t.Fatalf("Locate = %+v, %v; want %s, not registered", loc, err, want)
	}
}

// TestLocateFindsTheRegisteredStoreUnderAnyKey: a store registered as a
// worktree of the repository is the store, whatever its key, even one that
// is no commit of the repository at all, and with no folder there.
func TestLocateFindsTheRegisteredStoreUnderAnyKey(t *testing.T) {
	home := t.TempDir()
	r, _ := newRepo(t, "2020-01-01T00:00:00Z")
	p := addStore(t, r, home, key40)
	r.Git("worktree", "add", "-q", "--detach", filepath.Join(t.TempDir(), "a-peer"))
	loc, err := storepath.Locate(home, r.Root())
	if err != nil || !loc.Registered || loc.Path != p {
		t.Fatalf("Locate = %+v, %v; want %s, registered", loc, err, p)
	}
	if err := os.RemoveAll(p); err != nil {
		t.Fatal(err)
	}
	loc, err = storepath.Locate(home, r.Root())
	if err != nil || !loc.Registered || loc.Path != p {
		t.Fatalf("Locate after the folder went = %+v, %v; want %s, still registered", loc, err, p)
	}
}

// TestStoreUnderAnotherHomeIsRefused: a store-shaped worktree of the
// repository under some other home (sudo, a sandbox or an aliased HOME) is
// the store this repository registered, so both Locate and Reachable refuse:
// read as "no store", the check would pass clean and Locate would make a
// second store here.
func TestStoreUnderAnotherHomeIsRefused(t *testing.T) {
	home := t.TempDir()
	r, _ := newRepo(t, "2020-01-01T00:00:00Z")
	addStore(t, r, t.TempDir(), key40)
	if loc, err := storepath.Locate(home, r.Root()); !errors.Is(err, storepath.ErrStoreUnderAnotherHome) {
		t.Fatalf("Locate = %+v, %v; want ErrStoreUnderAnotherHome", loc, err)
	}
	if reach, err := storepath.Reachable(home, r.Root()); !errors.Is(err, storepath.ErrStoreUnderAnotherHome) {
		t.Fatalf("Reachable = %+v, %v; want ErrStoreUnderAnotherHome", reach, err)
	}
}

// TestReachableReadsEveryLiveStoreUnderTheHome: the stores the check reads
// are this repository's own and every live store under the home, whatever
// repository it belongs to and whatever its key, in path order; a store
// under another home is not read, and a folder under a key that is no full
// object name is no store and is passed over.
func TestReachableReadsEveryLiveStoreUnderTheHome(t *testing.T) {
	home := t.TempDir()
	r, root := newRepo(t, "2020-01-01T00:00:00Z")
	own := addStore(t, r, home, root)
	other, _ := newRepo(t, "2001-01-01T00:00:00Z")
	b := addStore(t, other, home, key64)
	addStore(t, other, t.TempDir(), key40) // under another home: not read
	third, _ := newRepo(t, "2010-01-01T00:00:00Z")
	c := addStore(t, third, home, key40)
	if err := os.MkdirAll(abcdhome.Path(home, "worktrees", "not-a-key", storepath.Name), 0o700); err != nil {
		t.Fatal(err)
	}
	reach, err := storepath.Reachable(home, r.Root())
	if err != nil || reach.Own != own || strings.Join(reach.Others, ",") != c+","+b {
		t.Fatalf("Reachable = %+v, %v; want own %s and others [%s %s]", reach, err, own, c, b)
	}
}

// TestReachableRefusesAPlaceThatIsNoLiveStore: a folder at a store's place
// under any key, whatever repository the key names, that is no live
// worktree, is refused, never passed over: it may be a store git lost track
// of.
func TestReachableRefusesAPlaceThatIsNoLiveStore(t *testing.T) {
	home := t.TempDir()
	r, _ := newRepo(t, "2020-01-01T00:00:00Z")
	if err := os.MkdirAll(abcdhome.Path(home, "worktrees", key64, storepath.Name), 0o700); err != nil {
		t.Fatal(err)
	}
	if reach, err := storepath.Reachable(home, r.Root()); !errors.Is(err, storepath.ErrPlaceUnread) {
		t.Fatalf("Reachable = %+v, %v; want ErrPlaceUnread", reach, err)
	}
}

// TestLocateRefusesRootsItCannotRead: with no store registered, a root
// listing git fails, one with a line of a shape git never writes, and a HEAD
// git fails to resolve are each ErrRootsUnread, and a repository with no
// commit is ErrNoRootCommit: none is a place to make the store.
func TestLocateRefusesRootsItCannotRead(t *testing.T) {
	for _, tc := range []struct {
		name string
		when []string
		out  string
		code int
	}{
		{"a failed listing", []string{"rev-list", "--max-parents=0"}, "", 128},
		{"a line of an unexpected shape", []string{"rev-list", "--max-parents=0"}, "yesterday " + key40 + "\n", 0},
		{"a HEAD git fails to resolve", []string{"rev-parse", "HEAD^{commit}"}, "", 128},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			r, _ := newRepo(t, "2020-01-01T00:00:00Z")
			gittest.ShimGit(t, tc.when, tc.out, tc.code)
			if loc, err := storepath.Locate(home, r.Root()); !errors.Is(err, storepath.ErrRootsUnread) {
				t.Fatalf("Locate = %+v, %v; want ErrRootsUnread", loc, err)
			}
		})
	}
	t.Run("no commit", func(t *testing.T) {
		r := gittest.NewRepo(t)
		if loc, err := storepath.Locate(t.TempDir(), r.Root()); !errors.Is(err, storepath.ErrNoRootCommit) {
			t.Fatalf("Locate = %+v, %v; want ErrNoRootCommit", loc, err)
		}
	})
}

// TestLocateRefusesSeveralStores: two registered stores are refused.
func TestLocateRefusesSeveralStores(t *testing.T) {
	home := t.TempDir()
	r, root := newRepo(t, "2020-01-01T00:00:00Z")
	addStore(t, r, home, root)
	addStore(t, r, home, key64)
	if loc, err := storepath.Locate(home, r.Root()); !errors.Is(err, storepath.ErrSeveralStores) {
		t.Fatalf("Locate = %+v, %v; want ErrSeveralStores", loc, err)
	}
}

// TestLocatePassesOverAnotherRepositorysStore: bar, holding foo's older root,
// passes over foo's live store at that root's place and is to make its own
// under its own root.
func TestLocatePassesOverAnotherRepositorysStore(t *testing.T) {
	home := t.TempDir()
	foo, fooRoot := newRepo(t, "2001-01-01T00:00:00Z")
	addStore(t, foo, home, fooRoot)
	bar, barRoot := newRepo(t, "2020-01-01T00:00:00Z")
	bar.Git("fetch", "-q", foo.Root(), "main:refs/heads/foo")
	bar.Git("merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", "foo")
	loc, err := storepath.Locate(home, bar.Root())
	want := abcdhome.Path(home, "worktrees", barRoot, storepath.Name)
	if err != nil || loc.Registered || loc.Path != want {
		t.Fatalf("Locate = %+v, %v; want bar's own place %s", loc, err, want)
	}
}

// TestLocateHasNoFreePlaceBesideAnotherCheckoutsStore: a second clone of a
// repository, whose one root keys a store the first clone holds, has no
// store and no free place.
func TestLocateHasNoFreePlaceBesideAnotherCheckoutsStore(t *testing.T) {
	home := t.TempDir()
	first, root := newRepo(t, "2020-01-01T00:00:00Z")
	addStore(t, first, home, root)
	second := filepath.Join(t.TempDir(), "second")
	cmd := exec.Command("git", "clone", "-q", first.Root(), second)
	cmd.Env = first.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}
	if loc, err := storepath.Locate(home, second); !errors.Is(err, storepath.ErrNoFreePlace) {
		t.Fatalf("Locate = %+v, %v; want ErrNoFreePlace", loc, err)
	}
}

// TestLocateRefusesATakenPlace: with no store registered, a place holding
// something not shown to be another repository's store is refused, never
// answered as free: a plain folder, and a worktree whose repository is gone.
func TestLocateRefusesATakenPlace(t *testing.T) {
	t.Run("a plain folder", func(t *testing.T) {
		home := t.TempDir()
		r, root := newRepo(t, "2020-01-01T00:00:00Z")
		if err := os.MkdirAll(abcdhome.Path(home, "worktrees", root, storepath.Name), 0o700); err != nil {
			t.Fatal(err)
		}
		if loc, err := storepath.Locate(home, r.Root()); !errors.Is(err, storepath.ErrPlaceTaken) {
			t.Fatalf("Locate = %+v, %v; want ErrPlaceTaken", loc, err)
		}
	})
	t.Run("a store whose repository is gone", func(t *testing.T) {
		home := t.TempDir()
		r, root := newRepo(t, "2020-01-01T00:00:00Z")
		gone := filepath.Join(t.TempDir(), "gone")
		cmd := exec.Command("git", "clone", "-q", r.Root(), gone)
		cmd.Env = r.Env()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("clone: %v\n%s", err, out)
		}
		p := abcdhome.Path(home, "worktrees", root, storepath.Name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		add := exec.Command("git", "-C", gone, "worktree", "add", "-q", "--detach", p)
		add.Env = r.Env()
		if out, err := add.CombinedOutput(); err != nil {
			t.Fatalf("worktree add: %v\n%s", err, out)
		}
		if err := os.RemoveAll(gone); err != nil {
			t.Fatal(err)
		}
		if loc, err := storepath.Locate(home, r.Root()); !errors.Is(err, storepath.ErrPlaceTaken) {
			t.Fatalf("Locate = %+v, %v; want ErrPlaceTaken", loc, err)
		}
	})
}

// TestLocateRefusesAPlaceThatCannotBeExamined: with no store registered, a
// place that cannot be examined counts as taken, never as free.
func TestLocateRefusesAPlaceThatCannotBeExamined(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root searches any folder")
	}
	home := t.TempDir()
	r, _ := newRepo(t, "2020-01-01T00:00:00Z")
	parent := abcdhome.Path(home, "worktrees")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	if loc, err := storepath.Locate(home, r.Root()); !errors.Is(err, storepath.ErrPlaceTaken) {
		t.Fatalf("Locate = %+v, %v; want ErrPlaceTaken", loc, err)
	}
}

// TestLocateRefusesWhenTheListingFails: a worktree listing git cannot hand
// over whole is ErrWorktreesUnread, never "no store".
func TestLocateRefusesWhenTheListingFails(t *testing.T) {
	home := t.TempDir()
	r, root := newRepo(t, "2020-01-01T00:00:00Z")
	p := addStore(t, r, home, root)
	gitfile, err := os.ReadFile(filepath.Join(p, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	admin := strings.TrimPrefix(strings.TrimSpace(string(gitfile)), "gitdir: ")
	if err := os.WriteFile(filepath.Join(admin, "locked"), []byte(strings.Repeat("r", 2<<20)), 0o644); err != nil {
		t.Fatal(err)
	}
	if loc, err := storepath.Locate(home, r.Root()); !errors.Is(err, storepath.ErrWorktreesUnread) {
		t.Fatalf("Locate = %+v, %v; want ErrWorktreesUnread", loc, err)
	}
}
