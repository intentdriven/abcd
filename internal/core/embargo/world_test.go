package embargo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// Most tests here start from a checkout and a home with the store already open,
// about forty git processes before a test reaches its subject. Each such start
// is built once a test process and written out again for every later test; every
// test still gets its own files.
//
// A store is a worktree, so its files and the checkout's name each other by
// absolute path, and the store's place follows from the home. So the start a
// test gets stands at one fixed place for the whole process rather than at a new
// temporary directory, and nothing needs rewriting. Tests here never run in
// parallel, so one place is enough; a test's second fixture, or a subtest's under
// a parent's, is built in temporary directories with git. A test whose subject
// is opening a store calls newFixture and openStore itself.

var openedWorlds struct {
	base string
	kept map[string]openedWorld
	busy bool
}

// openedWorld is one kept start: its files and the store it opened.
type openedWorld struct {
	tree  gittest.Tree
	store Store
}

func TestMain(m *testing.M) {
	code := m.Run()
	if openedWorlds.base != "" {
		_ = os.RemoveAll(openedWorlds.base)
	}
	os.Exit(code)
}

// openedFixture is newFixture with the store open.
func openedFixture(t *testing.T) (fixture, Store) {
	t.Helper()
	f, s, _ := standOpened(t, "", func(fixture) {})
	return f, s
}

// openedIntakeFixture is intakeFixture with the store open.
func openedIntakeFixture(t *testing.T) (fixture, *ghFake, Store) {
	t.Helper()
	f, s, _ := standOpened(t, "github origin", func(f fixture) {
		f.repo.Git("remote", "add", "origin", "https://github.com/example/example.git")
	})
	g := newGHFake(t)
	g.answer("GET", listPath, "triage", 0, "[]")
	g.answer("GET", listPath, "draft", 0, "[]")
	return f, g, s
}

// standOpened stands the start key names up for t: written out where this
// process has already built it, otherwise built by prepare and openStore. The
// result says whether it was written out.
func standOpened(t *testing.T, key string, prepare func(fixture)) (fixture, Store, bool) {
	t.Helper()
	w := &openedWorlds
	if w.busy {
		f := newFixture(t)
		prepare(f)
		s, _, err := openStore(f.home, f.repo.Root())
		if err != nil {
			t.Fatal(err)
		}
		return f, s, false
	}
	if w.base == "" {
		base, err := os.MkdirTemp("", "embargo-worlds-")
		if err != nil {
			t.Fatal(err)
		}
		w.base = base
	}
	at := filepath.Join(w.base, "world")
	home, root := filepath.Join(at, "home"), filepath.Join(at, "checkout")
	w.busy = true
	t.Cleanup(func() {
		w.busy = false
		if err := os.RemoveAll(at); err != nil {
			t.Errorf("removing the test's world: %v", err)
		}
	})
	if kept, ok := w.kept[key]; ok {
		kept.tree.Write(t, at)
		t.Setenv("HOME", home)
		r := gittest.OpenRepo(t, root)
		// A plumbing diff reads a new inode as an edit.
		r.Git("update-index", "-q", "--refresh")
		f := fixture{repo: r, home: home}
		if out, err := storeGit(t, f, kept.store.Root, "update-index", "-q", "--refresh"); err != nil {
			t.Fatalf("refreshing the copied store's index: %v\n%s", err, out)
		}
		return f, kept.store, true
	}
	for _, d := range []string{at, home, root} {
		if err := os.Mkdir(d, 0o777); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	r := gittest.NewRepoAt(t, root)
	r.Write(".abcd/work/issues/open/.gitkeep", "")
	r.Write("README.md", "a project\n")
	r.Commit("initial")
	f := fixture{repo: r, home: home}
	prepare(f)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if w.kept == nil {
		w.kept = map[string]openedWorld{}
	}
	w.kept[key] = openedWorld{tree: gittest.ReadTree(t, at), store: s}
	return f, s, false
}

// TestAnOpenedFixtureIsTheTestsOwn: what one test does to a world written out
// from a template reaches no later test's, a written-out store and checkout are
// the same worktree pair openStore made, with the store's folder closed to other
// accounts and a clean index, and a second fixture in one test stands apart.
func TestAnOpenedFixtureIsTheTestsOwn(t *testing.T) {
	first := func(t *testing.T) {
		f, s := openedFixture(t)
		st := emptyState()
		st.Repo = "example/changed"
		if err := s.SaveState(st); err != nil {
			t.Fatal(err)
		}
		f.repo.Write("marker.txt", "left by a test\n")
		f.repo.Commit("a test's own commit")
		if err := os.WriteFile(filepath.Join(s.Root, "stray"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 3 {
		t.Run(fmt.Sprintf("use %d", i), func(t *testing.T) {
			first(t)
		})
	}
	var written bool
	t.Run("a later world", func(t *testing.T) {
		f, s, was := standOpened(t, "", func(fixture) {})
		written = was
		if _, err := os.Stat(filepath.Join(f.repo.Root(), "marker.txt")); err == nil {
			t.Error("an earlier test's file reached this world's checkout")
		}
		if _, err := os.Stat(filepath.Join(s.Root, "stray")); err == nil {
			t.Error("an earlier test's file reached this world's store")
		}
		if got := f.repo.Git("rev-list", "--count", "HEAD"); got != "1" {
			t.Errorf("the checkout holds %s commits, want the one it started with", got)
		}
		if _, err := s.Verify(); err != nil {
			t.Errorf("a written-out store does not verify: %v", err)
		}
		fi, err := os.Stat(s.Root)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm()&0o077 != 0 {
			t.Errorf("a written-out store's folder has mode %v, open to other accounts", fi.Mode().Perm())
		}
		if _, err := storeGit(t, f, s.Root, "diff-files", "--quiet"); err != nil {
			t.Errorf("a plumbing diff reads the written-out store as edited: %v", err)
		}
		f.repo.Git("diff-files", "--quiet")
		again, reopened, err := openStore(f.home, f.repo.Root())
		if err != nil || reopened || again.Root != s.Root {
			t.Errorf("openStore over a written-out store: %+v, created %v, %v; want the same store, not made anew", again, reopened, err)
		}

		other, otherStore := openedFixture(t)
		if other.repo.Root() == f.repo.Root() || otherStore.Root == s.Root {
			t.Errorf("a second fixture in one test stands in the first's place: %s", other.repo.Root())
		}
	})
	if !written {
		t.Error("the later world was built with git, so nothing here judged a written-out one")
	}
}
