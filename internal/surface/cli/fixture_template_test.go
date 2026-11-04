package cli

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// Fixture repositories copied from one built once (iss-2610181757540736).
//
// The race pass this package runs in is bounded by its total work, and a
// fixture repository costs a git process per command that lays it out: an
// empty checkout from gitInitAt is one, gittest.NewRepo's five (git init and
// the four maintenance keys), a bare origin's five, a seeded world a dozen
// more. Every one of a kind is the same bytes, so each kind is built with git
// the first time a test in this process asks for it, into a directory of the
// package's own, and every later one is a copy of that (os.CopyFS), which
// spawns nothing. A copy is still the test's own repository, in the directory
// the test names, under the test's own isolated environment: every path that
// leads anywhere outside the copy is set again with git (git remote set-url),
// never by editing config text.
//
// A template is used only while the git on PATH is the git the templates were
// built with. A test that puts a stand-in git first on PATH, or takes git off
// it, gets its repository from that git exactly as before, so a stand-in that
// counts or refuses its calls still sees the git init, and a test with no git
// still skips.
var fixtureTemplates struct {
	mu    sync.Mutex
	root  string
	byKey map[string]string
}

// fixtureTemplateGit is the git on PATH as the process started, read before
// any test can put a stand-in first; the templates are built with it.
var fixtureTemplateGit, _ = exec.LookPath("git")

// fixtureTemplateTempDir is the temporary directory as the process started,
// read before any test can point TMPDIR at a directory of its own that its
// cleanup would remove with the templates inside.
var fixtureTemplateTempDir = os.TempDir()

// templateGit reports whether the git on PATH is the git the templates are
// built with.
func templateGit() bool {
	git, err := exec.LookPath("git")
	return err == nil && fixtureTemplateGit != "" && git == fixtureTemplateGit
}

// fixtureTemplate is the directory holding key's template, which build lays
// out with git in the directory it is handed the first time key is asked for.
// A build that fails or skips its test keeps nothing, so the next caller
// builds again.
func fixtureTemplate(t *testing.T, key string, build func(dir string)) string {
	t.Helper()
	fixtureTemplates.mu.Lock()
	dir, ok := fixtureTemplates.byKey[key]
	fixtureTemplates.mu.Unlock()
	if ok {
		return dir
	}
	root := templateRoot(t)
	dir, err := os.MkdirTemp(root, "tpl-")
	if err != nil {
		t.Fatal(err)
	}
	built := false
	defer func() {
		if !built {
			_ = os.RemoveAll(dir)
		}
	}()
	build(dir)
	built = true
	fixtureTemplates.mu.Lock()
	defer fixtureTemplates.mu.Unlock()
	if fixtureTemplates.byKey == nil {
		fixtureTemplates.byKey = map[string]string{}
	}
	fixtureTemplates.byKey[key] = dir
	return dir
}

// templateRoot is the directory every template is kept under, made on first
// use; TestMain removes it once the tests have run.
func templateRoot(t *testing.T) string {
	t.Helper()
	fixtureTemplates.mu.Lock()
	defer fixtureTemplates.mu.Unlock()
	if fixtureTemplates.root == "" {
		root, err := os.MkdirTemp(fixtureTemplateTempDir, "abcd-cli-fixtures-")
		if err != nil {
			t.Fatal(err)
		}
		fixtureTemplates.root = root
	}
	return fixtureTemplates.root
}

// removeFixtureTemplates deletes the kept templates; TestMain calls it once
// the tests have run.
func removeFixtureTemplates() {
	fixtureTemplates.mu.Lock()
	defer fixtureTemplates.mu.Unlock()
	if fixtureTemplates.root != "" {
		_ = os.RemoveAll(fixtureTemplates.root)
		fixtureTemplates.root = ""
		fixtureTemplates.byKey = nil
	}
}

// copyFixture copies the directory src into dst, creating dst if need be. It
// never overwrites: a file of src already in dst fails the test.
func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatalf("copying the fixture template %s: %v", src, err)
	}
}

// copyCheckout copies the checkout src into dst and refreshes the copy's
// index: the copy's files are new to the filesystem, so the stat data the
// index caches for them is not theirs, and a git command that reads the index
// without refreshing it (diff-index, diff-files) would name every one of them
// as changed. After the refresh the index is the one a checkout built in place
// holds.
func copyCheckout(t *testing.T, src, dst string) {
	t.Helper()
	copyFixture(t, src, dst)
	cmd := exec.Command("git", "-C", dst, "update-index", "-q", "--refresh")
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("refreshing the copied checkout's index: %v\n%s", err, out)
	}
}

// onlyGittestHome reports whether the home holds nothing but the Go
// telemetry mode file gittest.Env writes into a home it pins.
func onlyGittestHome(t *testing.T, home string) bool {
	t.Helper()
	only := true
	err := filepath.WalkDir(home, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(home, p)
		if err != nil {
			return err
		}
		switch filepath.ToSlash(rel) {
		case ".", ".config", ".config/go", ".config/go/telemetry", ".config/go/telemetry/mode",
			// os.UserConfigDir on darwin, where the macOS race job this cut is for runs.
			"Library", "Library/Application Support", "Library/Application Support/go",
			"Library/Application Support/go/telemetry", "Library/Application Support/go/telemetry/mode":
		default:
			only = false
		}
		return nil
	})
	return err == nil && only
}

// templateInit is the empty repository `git init --initial-branch=main` lays
// out (its .git directory), built once; ok is false when the git on PATH is
// not the templates' git, and the caller runs git init itself.
func templateInit(t *testing.T) (tpl string, ok bool) {
	t.Helper()
	if !templateGit() {
		return "", false
	}
	return fixtureTemplate(t, "init", func(dir string) {
		cmd := exec.Command("git", "-C", dir, "init", "--initial-branch=main")
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git init unavailable: %v (%s)", err, out)
		}
	}), true
}

// newRepo is gittest.NewRepo from a template: an empty repository on main
// with the maintenance keys off, under a per-test temporary directory, its
// commands run under gittest.Env(t). With a stand-in git on PATH, or none, it
// is gittest.NewRepo itself.
func newRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	if !templateGit() {
		return gittest.NewRepo(t)
	}
	tpl := fixtureTemplate(t, "newrepo", func(dir string) {
		r := gittest.NewRepo(t)
		copyFixture(t, r.Root(), dir)
	})
	root := t.TempDir()
	copyFixture(t, tpl, root)
	return gittest.OpenRepo(t, root)
}

// initBare is gittest.InitBare from a template: an empty bare repository on
// main at path, with the maintenance keys off. With a stand-in git on PATH,
// or none, it is gittest.InitBare itself.
func initBare(t *testing.T, path string) {
	t.Helper()
	if !templateGit() {
		gittest.InitBare(t, path)
		return
	}
	tpl := fixtureTemplate(t, "bare", func(dir string) {
		gittest.InitBare(t, filepath.Join(dir, "origin.git"))
	})
	gittest.Env(t)
	if _, err := os.Lstat(path); err == nil {
		gittest.InitBare(t, path)
		return
	}
	copyFixture(t, filepath.Join(tpl, "origin.git"), path)
}
