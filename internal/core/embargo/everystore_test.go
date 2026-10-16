package embargo

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ghsaPendingElsewhere is the pending id sampleState holds.
const ghsaPendingElsewhere = "GHSA-mmmm-mmmm-mmmm"

// unrelatedStore makes a project that shares no history with any other
// fixture, opens its store under home, saves st there, and returns it.
func unrelatedStore(t *testing.T, home string, st State) Store {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/.gitkeep", "")
	r.Write("README.md", "an unrelated project\n")
	datedCommit(t, r, "2011-01-01T00:00:00Z", "its own root")
	s, created, err := openStore(home, r.Root())
	if err != nil || !created {
		t.Fatalf("the unrelated project's store: created %v, %v", created, err)
	}
	if err := s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	return s
}

// wantNoPathIn fails when err names any of private.
func wantNoPathIn(t *testing.T, err error, private ...string) {
	t.Helper()
	for _, p := range private {
		if strings.Contains(err.Error(), p) {
			t.Fatalf("the refusal %q names %q", err, p)
		}
	}
}

// TestCheckRefusesAStoreRegisteredUnderAnotherHome: this repository's store
// lives under another home (abcd run once under sudo, a sandbox or an
// aliased HOME). Checking from this home refuses, naming no path and saying
// to run abcd with that home, rather than reading clean; and no second store
// is made here.
func TestCheckRefusesAStoreRegisteredUnderAnotherHome(t *testing.T) {
	f := newFixture(t)
	elsewhere := t.TempDir()
	s, created, err := openStore(elsewhere, f.repo.Root())
	if err != nil || !created {
		t.Fatalf("the store under the other home: created %v, %v", created, err)
	}
	holdSample(t, s)
	res, err := checkStdin(f.home, f.repo.Root(), "fixes "+ghsaOpen+"\n")
	if err == nil {
		t.Fatalf("the check = %+v and no refusal", res)
	}
	mustRefuse(t, err, "another home")
	mustRefuse(t, err, "run abcd with that home")
	key := filepath.Base(filepath.Dir(s.Root))
	wantNoPathIn(t, err, elsewhere, f.home, key)
	_, _, err = openStore(f.home, f.repo.Root())
	mustRefuse(t, err, "another home")
	if _, err := os.Lstat(abcdhome.Path(f.home, "worktrees")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("something was made under this home's worktree store (%v)", err)
	}
}

// shallowClone clones the fixture's repository one commit deep, the way a
// hosted session checks a project out, and returns its root.
func shallowClone(t *testing.T, f fixture) string {
	t.Helper()
	clone := filepath.Join(t.TempDir(), "shallow")
	for _, args := range [][]string{
		{"clone", "-q", "--depth", "1", "file://" + f.repo.Root(), clone},
		{"-C", clone, "config", "--local", "gc.auto", "0"},
		{"-C", clone, "config", "--local", "maintenance.auto", "false"},
		{"-C", clone, "config", "--local", "core.fsmonitor", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Env = f.repo.Env()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return clone
}

// TestShallowSecondCloneChecksAgainstTheFirstClonesStore: a shallow second
// clone of the same project, whose only root is its boundary commit and so
// misses the key of the store the first clone made, still refuses the
// first clone's held id; and with nothing held anywhere it checks clean.
func TestShallowSecondCloneChecksAgainstTheFirstClonesStore(t *testing.T) {
	cf := newCheckFixture(t)
	cf.f.repo.Write("a.txt", "one\n")
	cf.f.repo.Commit("c1")
	cf.f.repo.Write("a.txt", "two\n")
	cf.f.repo.Commit("c2")
	clone := shallowClone(t, cf.f)
	if key := filepath.Base(filepath.Dir(cf.s.Root)); gitutil.RootCommit(clone) == key {
		t.Fatal("the shallow clone's root is the store's key, so it proves nothing")
	}
	wantEmbargoedIDRefused(t, cf.f.home, clone)

	quiet := newFixture(t)
	quiet.repo.Write("a.txt", "one\n")
	quiet.repo.Commit("c1")
	res, err := checkStdin(quiet.home, shallowClone(t, quiet), "a text\n")
	if err != nil || !res.NoStore || !res.Clean() {
		t.Fatalf("the shallow clone with no store anywhere = %+v, %v; want no store, clean", res, err)
	}
}

// TestCheckRefusesAnUnrelatedProjectsHeldID: a store of a project that
// shares no history with this one, under the same home, is read too: both
// are private to the same person, so its held id is refused here.
func TestCheckRefusesAnUnrelatedProjectsHeldID(t *testing.T) {
	f := newFixture(t)
	other := unrelatedStore(t, f.home, sampleState())
	before := listTree(t, other.Root)
	wantEmbargoedIDRefused(t, f.home, f.repo.Root())
	if listTree(t, other.Root) != before {
		t.Fatal("the check changed the unrelated project's store")
	}
}

// TestCheckReadsTwoPopulatedStoresAsOne: this repository's own store and an
// unrelated project's store, read together:
//   - an id one holds unpublished and the other published stays under
//     embargo, and --allow of it is refused;
//   - each store's held ids and each store's own path forms are refused;
//   - each id is counted once, however many stores hold it;
//   - the intake reported is the oldest.
func TestCheckReadsTwoPopulatedStoresAsOne(t *testing.T) {
	cf := newCheckFixture(t)
	older := fixedNow.Add(-72 * time.Hour)
	st := sampleState()
	st.IntakeAt = &older
	// sampleState holds ghsaOpen unpublished; here it is published, while
	// this repository's own store holds it unpublished.
	st.Advisories[0].PublishedAt = &older
	st.Advisories[0].ForgeState = "published"
	other := unrelatedStore(t, cf.f.home, st)

	// The one finding a held id gives, of the kind check_test.go pins.
	heldKind := kinds(cf.text(t, "fixes "+ghsaOpen+"\n"))
	if len(heldKind) != 1 {
		t.Fatalf("the held id gave %q; want one finding", heldKind)
	}
	_, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader("fixes " + ghsaOpen + "\n"), Allow: []string{ghsaOpen}})
	mustRefuse(t, err, "unpublished")

	wantKinds(t, cf.text(t, "fixes "+ghsaPendingElsewhere+"\n"), heldKind...)
	for _, s := range []Store{cf.s, other} {
		key := filepath.Base(filepath.Dir(s.Root))
		wantKinds(t, cf.text(t, "under worktrees/"+key+"/embargo/.abcd\n"), KindStorePath)
		res := cf.text(t, "see "+s.Root+"/notes\n")
		if len(res.Findings) == 0 {
			t.Fatalf("the store's absolute path is not refused: %+v", res)
		}
		for _, fd := range res.Findings {
			if fd.Kind != KindStorePath {
				t.Fatalf("the store's absolute path gave %+v", res.Findings)
			}
		}
	}

	res := cf.text(t, "a text\n")
	// Own: ghsaOpen, ghsaClosed, ghsaPublic. The other adds only the pending
	// id: its ghsaOpen is published, and its closed id is ghsaClosed.
	if res.UnderEmbargo != 4 {
		t.Fatalf("under embargo %d; want 4, each id once", res.UnderEmbargo)
	}
	if res.IntakeAt == nil || !res.IntakeAt.Equal(older) || !res.IntakeStale {
		t.Fatalf("intake %v, stale %v; want the older store's %v, stale", res.IntakeAt, res.IntakeStale, older)
	}
}

// TestCheckRefusesAnotherStoresStateNotThisAccountsAlone: another store's
// state file that a group can write, or that another account owns, is
// refused before it is read, never trusted.
func TestCheckRefusesAnotherStoresStateNotThisAccountsAlone(t *testing.T) {
	t.Run("a group can write it", func(t *testing.T) {
		f := newFixture(t)
		other := unrelatedStore(t, f.home, sampleState())
		if err := os.Chmod(other.StatePath(), 0o620); err != nil {
			t.Fatal(err)
		}
		_, err := checkStdin(f.home, f.repo.Root(), "a text\n")
		mustRefuse(t, err, "not this account's alone")
	})
	t.Run("another account owns it", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root owns the file the test links to")
		}
		f := newFixture(t)
		other := unrelatedStore(t, f.home, sampleState())
		foreign, err := filepath.EvalSymlinks("/etc/hosts")
		if err != nil {
			t.Skip("no file another account owns to link to")
		}
		if err := os.Remove(other.StatePath()); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(foreign, other.StatePath()); err != nil {
			t.Skipf("cannot link another account's file into the store: %v", err)
		}
		_, err = checkStdin(f.home, f.repo.Root(), "a text\n")
		mustRefuse(t, err, "not this account's alone")
	})
}

// TestCheckRefusesAFolderUnderAnyKeyThatIsNoLiveStore: a folder at a store's
// place under a key no root of this repository names refuses the check, as
// one under a root does: it may be a store git lost track of. A folder under
// a name that is no key, or a worktree of another name under a key, is no
// store and is passed over: with nothing else there, there is no store.
func TestCheckRefusesAFolderUnderAnyKeyThatIsNoLiveStore(t *testing.T) {
	f := newFixture(t)
	for _, p := range []string{
		abcdhome.Path(f.home, "worktrees", "not-a-key", StoreName),
		abcdhome.Path(f.home, "worktrees", strings.Repeat("ef", 20), "a-lane"),
	} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	res, err := checkStdin(f.home, f.repo.Root(), "a text\n")
	if err != nil || !res.NoStore {
		t.Fatalf("the check = %+v, %v; want no store", res, err)
	}
	if err := os.MkdirAll(abcdhome.Path(f.home, "worktrees", strings.Repeat("cd", 20), StoreName), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = checkStdin(f.home, f.repo.Root(), "a text\n")
	mustRefuse(t, err, "examined")
}
