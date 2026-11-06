package loop

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// A fixture here is a checkout, a home and often a bare origin, each built with
// git, and often a run already started in them, which is dozens of processes a
// test before it reaches its subject. Most tests start from one of a few such
// worlds, so each is built once a test process and copied after that: every
// test still gets its own files, and no two tests share one.
//
// A world's files name its paths: the run's state names its worktree, the
// config fingerprint names and salts the hooks and the origin's URL, the
// briefs and the worktree's gitdir hold absolute paths. So the world a test
// gets stands at one fixed place for the whole process rather than at a new
// temporary directory, and a copy needs nothing rewritten. Tests here never run
// in parallel, so one place is enough; a test's second world, or a subtest's
// under a parent's, is built in temporary directories with git.

// fixtureWorld is the directories one test's fixture stands in.
type fixtureWorld struct {
	root, home, gh, bare string
	// repo is the Repo over root, made when root first exists, before a
	// fixture puts a commit identity in the environment, so its commands run
	// as they did when the fixture itself made it.
	repo *gittest.Repo
	// at is the fixed place, and so what can be kept and copied; "" for a
	// world in temporary directories.
	at       string
	ghOnPath bool
	// lineage names the kept world this one stands as, and made when the
	// oldest walk in it began: a walk's copy is of that build and no other, so
	// a world built again takes its walks with it.
	lineage string
	made    time.Time
}

// keptWorld is one world's files, and what its build returned.
type keptWorld struct {
	dir        string
	value      any
	made       time.Time
	generation int
}

// keptWorldAge bounds how old a kept world's clock can read, since a run in one
// was started on the real clock. It stays below every span a test measures on
// that clock from a fixture (half a lease, a work window), and a whole pass of
// the package has been run on worlds built by the pass before.
const keptWorldAge = 30 * time.Minute

var fixtureWorlds struct {
	base     string
	kept     map[string]*keptWorld
	current  *fixtureWorld
	building *fixtureWorld
	made     int
}

// builtOnce stands up the world key names: a copy of it where this process has
// already built it, otherwise what build makes. A fixture called from inside
// another's build stands in the same world. The key must name everything build
// reads, and build must read nothing a test can have changed first; what it
// returns is handed to every later test, so it holds values only. HOME is the
// world's own on return; what else build put in the environment, a fixture
// puts there again after this returns.
func builtOnce[T any](t *testing.T, key string, build func(w *fixtureWorld) T) (T, *fixtureWorld) {
	t.Helper()
	w := fixtureWorlds.building
	if w == nil {
		w = claimWorld(t)
	}
	return walkedOnce(t, w, key, func() T { return build(w) }), w
}

// walkedOnce carries on from the world a fixture just stood up: the world
// after walk where this process has already walked it, otherwise walk's. It is
// only for a walk that follows the fixture directly: a test that has changed
// anything first (a file, the environment, a package's hook) walks itself. The
// name must say everything walk reads beyond the world it starts from.
func walkedOnce[T any](t *testing.T, w *fixtureWorld, name string, walk func() T) T {
	t.Helper()
	fw := &fixtureWorlds
	key := worldKey(w.lineage, name)
	if w.at != "" {
		if k, ok := fw.kept[key]; ok && time.Since(k.made) < keptWorldAge {
			w.restore(t, k)
			w.lineage = worldKey(key, strconv.Itoa(k.generation))
			return k.value.(T)
		}
	}
	outer := fw.building
	fw.building = w
	defer func() { fw.building = outer }()
	v := walk()
	if w.at != "" {
		w.keep(t, key, v)
	}
	return v
}

// onceAfter is walkedOnce for repo, the repository the fixture a test just
// called made; a repository from anywhere else just walks.
func onceAfter[T any](t *testing.T, repo *gittest.Repo, name string, walk func() T) T {
	t.Helper()
	if w := fixtureWorlds.current; w != nil && w.root == repo.Root() {
		return walkedOnce(t, w, name, walk)
	}
	return walk()
}

// outcome is what a call a test goes on to judge returned, kept with its world.
type outcome[T any] struct {
	value T
	err   error
}

// startedOnce is Start with no options, straight after the fixture.
func startedOnce(t *testing.T, repo *gittest.Repo, key string) (StartResult, error) {
	t.Helper()
	o := onceAfter(t, repo, worldKey("Start", key), func() outcome[StartResult] {
		v, err := Start(repo.Root(), key, Options{})
		return outcome[StartResult]{v, err}
	})
	return o.value, o.err
}

// implementedOnce is implemented on DefaultStages, straight after startedOnce.
func implementedOnce(t *testing.T, repo *gittest.Repo, runID, file string) Lane {
	t.Helper()
	onceAfter(t, repo, worldKey("implemented", file), func() struct{} {
		implemented(t, repo, runID, DefaultStages(), file)
		return struct{}{}
	})
	return currentLane(t, repo, runID)
}

// advancedToOnce is advanceTo, straight after startedOnce.
func advancedToOnce(t *testing.T, repo *gittest.Repo, runID string, want Stage) StepResult {
	t.Helper()
	return onceAfter(t, repo, worldKey("advanceTo", string(want)), func() StepResult { return advanceTo(t, repo, runID, want) })
}

// drainedOnce is Drain with no options but d, straight after the fixture.
func drainedOnce(t *testing.T, repo *gittest.Repo, d DrainOptions) (DrainResult, error) {
	t.Helper()
	o := onceAfter(t, repo, worldKey("Drain", fmt.Sprintf("%+v", d)), func() outcome[DrainResult] {
		v, err := Drain(repo.Root(), Options{}, d)
		return outcome[DrainResult]{v, err}
	})
	return o.value, o.err
}

// drainedToLaneOnce is drainToLane with no options, straight after the fixture.
func drainedToLaneOnce(t *testing.T, repo *gittest.Repo, d DrainOptions) DrainResult {
	t.Helper()
	return onceAfter(t, repo, worldKey("drainToLane", fmt.Sprintf("%+v", d)), func() DrainResult {
		return drainToLane(t, repo, Options{}, d)
	})
}

// claimWorld gives the test the fixed place, empty, unless a world of a test
// still running stands there, and then temporary directories.
func claimWorld(t *testing.T) *fixtureWorld {
	t.Helper()
	fw := &fixtureWorlds
	var w *fixtureWorld
	if fw.current != nil {
		w = &fixtureWorld{root: t.TempDir(), home: t.TempDir(), gh: t.TempDir(), bare: filepath.Join(t.TempDir(), "origin.git")}
	} else {
		if fw.base == "" {
			base, err := os.MkdirTemp("", "loop-worlds-")
			if err != nil {
				t.Fatal(err)
			}
			fw.base = base
		}
		at := filepath.Join(fw.base, "world")
		w = worldAt(at)
		for _, d := range []string{at, w.root, w.home, w.gh} {
			if err := os.Mkdir(d, 0o777); err != nil {
				t.Fatal(err)
			}
		}
		fw.current = w
		t.Cleanup(func() {
			fw.current = nil
			if err := os.RemoveAll(at); err != nil {
				t.Errorf("removing the test's world: %v", err)
			}
		})
		// The forge's answer about an account is kept for the process under
		// the checkout's path and the credential environment (claim's
		// holderCacheKey), PATH and HOME among it. At a fixed place every test
		// would read the first test's answer; a directory of its own on PATH
		// keeps each test's answer its own, as a temporary checkout did.
		t.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+t.TempDir())
	}
	t.Setenv("HOME", w.home)
	return w
}

func worldAt(at string) *fixtureWorld {
	return &fixtureWorld{at: at, root: filepath.Join(at, "root"), home: filepath.Join(at, "home"),
		gh: filepath.Join(at, "gh"), bare: filepath.Join(at, "origin.git")}
}

// newRepo makes the world's repository.
func (w *fixtureWorld) newRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	w.repo = gittest.NewRepoAt(t, w.root)
	return w.repo
}

// putGHFirst puts the world's stub forge client first on PATH, once.
func (w *fixtureWorld) putGHFirst(t *testing.T) {
	t.Helper()
	if !w.ghOnPath {
		w.ghOnPath = true
		t.Setenv("PATH", w.gh+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
}

// keep copies the world, just built, under key.
func (w *fixtureWorld) keep(t *testing.T, key string, value any) {
	t.Helper()
	fw := &fixtureWorlds
	if was, ok := fw.kept[key]; ok {
		if err := os.RemoveAll(was.dir); err != nil {
			t.Fatal(err)
		}
	}
	fw.made++
	dir := filepath.Join(fw.base, "kept", strconv.Itoa(fw.made))
	if err := os.MkdirAll(filepath.Dir(dir), 0o777); err != nil {
		t.Fatal(err)
	}
	copyWorld(t, w.at, dir)
	if fw.kept == nil {
		fw.kept = map[string]*keptWorld{}
	}
	if w.made.IsZero() {
		w.made = time.Now()
	}
	fw.kept[key] = &keptWorld{dir: dir, value: value, made: w.made, generation: fw.made}
	w.lineage = worldKey(key, strconv.Itoa(fw.made))
}

// restore replaces what stands at the fixed place with the copy at dir. A
// copied file has a new inode and ctime, which git's index takes for an edit
// until it looks, and a plumbing diff does not look; so each index is refreshed,
// as a checkout that had just been written would read.
func (w *fixtureWorld) restore(t *testing.T, k *keptWorld) {
	t.Helper()
	if err := os.RemoveAll(w.at); err != nil {
		t.Fatal(err)
	}
	copyWorld(t, k.dir, w.at)
	w.made = k.made
	w.repo = gittest.OpenRepo(t, w.root)
	w.repo.Git("update-index", "-q", "--refresh")
	admin, err := filepath.Glob(filepath.Join(w.root, ".git", "worktrees", "*", "gitdir"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range admin {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		gittest.OpenRepo(t, filepath.Dir(strings.TrimSpace(string(b)))).Git("update-index", "-q", "--refresh")
	}
}

// copyWorld copies the tree at src to dst, which must not exist, keeping every
// mode a store checks and every symbolic link. A copy's times are its own: the
// files read as just written, as a world built now would.
func copyWorld(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.Mkdir(to, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, to)
		default:
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(to, b, info.Mode().Perm())
		}
	})
	if err != nil {
		t.Fatalf("copying a fixture's world: %v", err)
	}
}

// stateOnDisk is the run's state for a helper that is walking a run to its
// test's subject. ReadState asks git, twice, whether the local tier is tracked
// before it reads, which every step of the run asks again; the helpers ask it
// nothing, and the tests of the question call ReadState (drain_tracked_state,
// localstate).
func stateOnDisk(t *testing.T, repoRoot, runID string) State {
	t.Helper()
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	st, err := readListedState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// removeFixtureWorlds deletes the fixed place and the kept worlds; TestMain
// calls it once the tests have run.
func removeFixtureWorlds() {
	if fixtureWorlds.base != "" {
		_ = os.RemoveAll(fixtureWorlds.base)
	}
}

// worldKey joins what a world's build reads into its key.
func worldKey(parts ...string) string { return strings.Join(parts, "\x00") }

// sortedFiles is a fixture's files in a key's stable order.
func sortedFiles(files map[string]string) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		b.WriteString(n + "\x01" + files[n] + "\x01")
	}
	return b.String()
}

// TestACopiedWorldIsTheTestsOwn holds the copies to what they replace: a test
// that writes in its world leaves the next test's untouched, a copy reads as the
// world did to git and to a store that checks modes, and each test's credential
// environment is its own, so the forge's cached answer does not cross tests.
func TestACopiedWorldIsTheTestsOwn(t *testing.T) {
	const key = "TestACopiedWorldIsTheTestsOwn"
	build := func(t *testing.T) (*gittest.Repo, string) {
		t.Helper()
		sha, w := builtOnce(t, key, func(w *fixtureWorld) string {
			repo := w.newRepo(t)
			repo.Write("tracked.txt", "as built\n")
			repo.Commit("the world")
			store := filepath.Join(w.home, "store")
			if err := os.Mkdir(store, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("store", filepath.Join(w.home, "link")); err != nil {
				t.Fatal(err)
			}
			return repo.Git("rev-parse", "HEAD")
		})
		return w.repo, sha
	}
	var paths []string
	for _, name := range []string{"built", "copied and changed", "copied again"} {
		t.Run(name, func(t *testing.T) {
			repo, sha := build(t)
			paths = append(paths, os.Getenv("PATH"))
			if head := repo.Git("rev-parse", "HEAD"); head != sha {
				t.Fatalf("HEAD is %s; want the world's %s", head, sha)
			}
			if got := repo.Git("status", "--porcelain"); got != "" {
				t.Fatalf("an earlier test's work is in this world:\n%s", got)
			}
			// A plumbing diff does not refresh the index, as porcelain does.
			repo.Git("diff-index", "--quiet", "HEAD")
			home := os.Getenv("HOME")
			fi, err := os.Lstat(filepath.Join(home, "store"))
			if err != nil || fi.Mode().Perm() != 0o700 {
				t.Fatalf("the store's mode did not survive the copy: %v %v", fi, err)
			}
			if link, err := os.Readlink(filepath.Join(home, "link")); err != nil || link != "store" {
				t.Fatalf("the link did not survive the copy: %q %v", link, err)
			}
			repo.Write("tracked.txt", "changed by "+name+"\n")
			repo.Commit("this test's own")
		})
	}
	if paths[0] == paths[1] || paths[1] == paths[2] {
		t.Fatalf("two tests share a credential environment: %q", paths)
	}
}

// TestASecondWorldInOneTestIsBuiltApart: a test's second fixture, and a
// subtest's under its parent's, stand in directories of their own, since the
// fixed place is taken.
func TestASecondWorldInOneTestIsBuiltApart(t *testing.T) {
	first := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	second := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	if first.Root() == second.Root() {
		t.Fatalf("two worlds at %s", first.Root())
	}
	t.Run("a subtest's", func(t *testing.T) {
		third := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		if third.Root() == first.Root() {
			t.Fatalf("a subtest's world replaced its parent's at %s", third.Root())
		}
	})
	if got := first.Git("rev-parse", "HEAD"); got == "" {
		t.Fatal("the first world's repository is gone")
	}
}
