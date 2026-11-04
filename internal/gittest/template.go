package gittest

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Templates keeps repositories a package's tests start from: each is built with
// git once in a test process and held in memory, and every test gets its own
// copy of the tree. A fixture of many commits costs a process per git command
// when built and only file writes when copied, and the copy is the same
// repository: the same objects, refs, config and working tree. Nothing outlives
// the test process, so a package has nothing to clean up.
//
// A template must be a function of its key alone, since whichever test asks
// first builds it for all the others. A package declares one:
//
//	var fixtures gittest.Templates
type Templates struct {
	mu   sync.Mutex
	kept map[string]tree
}

// Repo is t's own copy of the repository kept under key, which build makes the
// first time the key is asked for. The copy is t's to change: it sits in its
// own temporary directory and shares nothing with the template or with any
// other test's copy. The first test gets a copy too, so no test is handed a
// repository of a different kind from the rest.
func (ts *Templates) Repo(t *testing.T, key string, build func(t *testing.T) *Repo) *Repo {
	t.Helper()
	root := t.TempDir()
	ts.tree(t, key, func() string { return build(t).Root() }).write(t, root)
	return OpenRepo(t, root)
}

// Dir is Repo for a tree that is not one fixture repository (a repository and
// its bare origin side by side, a tree holding no repository at all): t's own
// copy of the directory build returned the first time the key was asked for.
func (ts *Templates) Dir(t *testing.T, key string, build func(t *testing.T) string) string {
	t.Helper()
	root := t.TempDir()
	ts.tree(t, key, func() string { return build(t) }).write(t, root)
	return root
}

// tree is the kept tree for key, read from what build returns the first time.
// The lock is held across the build, so a second test waits for the template
// rather than building its own.
func (ts *Templates) tree(t *testing.T, key string, build func() string) tree {
	t.Helper()
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if kept, ok := ts.kept[key]; ok {
		return kept
	}
	kept := readTree(t, build())
	if ts.kept == nil {
		ts.kept = map[string]tree{}
	}
	ts.kept[key] = kept
	return kept
}

// tree is a directory's contents in memory: every entry, a parent before what
// it holds, with its exact mode.
type tree []treeEntry

type treeEntry struct {
	rel  string
	mode fs.FileMode
	link string
	body []byte
}

// readTree reads the directory root.
func readTree(t *testing.T, root string) tree {
	t.Helper()
	var entries tree
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		e := treeEntry{rel: rel, mode: fi.Mode()}
		switch {
		case fi.IsDir():
		case fi.Mode()&os.ModeSymlink != 0:
			if e.link, err = os.Readlink(p); err != nil {
				return err
			}
		case fi.Mode().IsRegular():
			if e.body, err = os.ReadFile(p); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: a template cannot hold mode %v", p, fi.Mode())
		}
		entries = append(entries, e)
		return nil
	})
	if err != nil {
		t.Fatalf("reading %s: %v", root, err)
	}
	return entries
}

// write lays the tree out under root, which it creates if need be. A folder's
// own mode comes last, so a read-only one can still be filled.
func (entries tree) write(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		p := filepath.Join(root, e.rel)
		var err error
		switch {
		case e.mode.IsDir():
			err = os.Mkdir(p, 0o700)
		case e.mode&os.ModeSymlink != 0:
			err = os.Symlink(e.link, p)
		default:
			if err = os.WriteFile(p, e.body, 0o600); err == nil {
				err = os.Chmod(p, e.mode.Perm())
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if e := entries[i]; e.mode.IsDir() {
			if err := os.Chmod(filepath.Join(root, e.rel), e.mode.Perm()); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// emptyRepositories keeps what `git init` and the maintenance config leave, so
// NewRepo and InitBare spawn their five git processes once in a test process.
var emptyRepositories Templates

// emptyRepositoryKey names everything that moves what git init makes, beyond the
// kind of repository: which git runs, and the variables of its own that survive
// Env. A test that puts another git first on PATH, or none, gets its own run of
// the real thing, so it still sees that git and a missing one still skips.
func emptyRepositoryKey(kind, format string, env []string) string {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		gitPath = "none"
	}
	parts := []string{kind, format, gitPath}
	for _, kv := range env {
		if strings.HasPrefix(kv, "GIT_DEFAULT_") || strings.HasPrefix(kv, "GIT_TEMPLATE_DIR=") {
			parts = append(parts, kv)
		}
	}
	return strings.Join(parts, "\x00")
}
