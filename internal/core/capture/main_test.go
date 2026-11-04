package capture

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestMain shares the ledger scanner where sharing cannot be seen.
func TestMain(m *testing.M) {
	newLedgerScanner = sharedLedgerScanner(newLedgerScanner)
	os.Exit(m.Run())
}

// sharedLedgerScanner is the scanner seam with one scanner per identity source.
// Every write builds a scanner, and building one runs git to read the machine's
// identity: most of what this package's tests spawn, several times over in a test
// that seeds a ledger, and always the same answer for the same config.
//
// Sharing is limited to where a scanner is a function of the environment, the
// config files git reads and the root's own repository config. A root holding
// anything under .abcd/config (the per-repo pattern file) builds its own, and so
// do a .git that is not a folder and a root that is not one, the way the
// degraded-scanner tests make them. So does a scanner with an augmenter, which
// keeps state; capture wires none, and a test's own scanner arrives by replacing
// this seam. Once built, a scanner holds only what it reads, so tests sharing one
// share nothing they can change.
func sharedLedgerScanner(build func(string) (*scanner.Scanner, error)) func(string) (*scanner.Scanner, error) {
	var (
		mu     sync.Mutex
		shared = map[[sha256.Size]byte]*scanner.Scanner{}
	)
	return func(root string) (*scanner.Scanner, error) {
		key, ok := scannerSources(root)
		if !ok {
			return build(root)
		}
		mu.Lock()
		defer mu.Unlock()
		if sc, ok := shared[key]; ok {
			return sc, nil
		}
		sc, err := build(root)
		if err != nil {
			return sc, err
		}
		if unavailable, _ := sc.Unavailable(); !unavailable {
			shared[key] = sc
		}
		return sc, nil
	}
}

// scannerSources fingerprints what a scanner for root is built from, and reports
// false for a root whose scanner must be its own: one outside a test's temporary
// directory, not a plain folder, or holding scanner configuration.
func scannerSources(root string) ([sha256.Size]byte, bool) {
	tmp := strings.TrimRight(os.TempDir(), string(filepath.Separator)) + string(filepath.Separator)
	if !strings.HasPrefix(root, tmp) {
		return [sha256.Size]byte{}, false
	}
	if fi, err := os.Lstat(root); err != nil || !fi.IsDir() {
		return [sha256.Size]byte{}, false
	}
	// .abcd may be a link or a file: only a real folder whose config is absent
	// is plainly no configuration.
	if fi, err := os.Lstat(filepath.Join(root, ".abcd")); err == nil && !fi.IsDir() {
		return [sha256.Size]byte{}, false
	} else if err != nil && !os.IsNotExist(err) {
		return [sha256.Size]byte{}, false
	}
	if _, err := os.Lstat(filepath.Join(root, ".abcd", "config")); !os.IsNotExist(err) {
		return [sha256.Size]byte{}, false
	}
	// A repository's own config is part of the identity. A .git that is a file
	// or a link points elsewhere, which this does not follow.
	repoConfig := []byte("no repository")
	if fi, err := os.Lstat(filepath.Join(root, ".git")); err == nil {
		if !fi.IsDir() {
			return [sha256.Size]byte{}, false
		}
		if repoConfig, err = os.ReadFile(filepath.Join(root, ".git", "config")); err != nil {
			return [sha256.Size]byte{}, false
		}
	} else if !os.IsNotExist(err) {
		return [sha256.Size]byte{}, false
	}
	env := os.Environ()
	slices.Sort(env)
	h := sha256.New()
	h.Write([]byte(strings.Join(env, "\x00")))
	h.Write(repoConfig)
	for _, global := range []string{
		filepath.Join(os.Getenv("HOME"), ".gitconfig"),
		filepath.Join(os.Getenv("HOME"), ".config", "git", "config"),
		filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "git", "config"),
		os.Getenv("GIT_CONFIG_GLOBAL"),
	} {
		if global == "" {
			continue
		}
		raw, err := os.ReadFile(global)
		if err == nil {
			h.Write(raw)
		}
		h.Write([]byte{0})
	}
	return [sha256.Size]byte(h.Sum(nil)), true
}

// TestASharedLedgerScannerIsWhatTheRootWouldBuild holds the sharing to its
// reasons: a plain ledger directory gets the scanner a build would give it, which
// redacts the environment's identity, and each reason to build one of its own
// builds one. Two repositories with different identities never share one.
func TestASharedLedgerScannerIsWhatTheRootWouldBuild(t *testing.T) {
	builds := 0
	shared := sharedLedgerScanner(func(root string) (*scanner.Scanner, error) {
		builds++
		return scanner.New(root)
	})
	plain := func(t *testing.T) string {
		t.Helper()
		return t.TempDir()
	}
	first, err := shared(plain(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := shared(plain(t))
	if err != nil {
		t.Fatal(err)
	}
	if builds != 1 || first != second {
		t.Errorf("two plain directories built %d scanners, want one shared", builds)
	}
	for name, root := range map[string]func(t *testing.T) string{
		"a repository whose config names someone": func(t *testing.T) string {
			r := gittest.NewRepo(t)
			r.Git("config", "user.email", "zanzibar@example.invalid")
			return r.Root()
		},
		"a pattern file": func(t *testing.T) string {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".abcd", "config"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".abcd", "config", "pii.json"), []byte(`{"patterns": "not a list"}`), 0o644); err != nil {
				t.Fatal(err)
			}
			return dir
		},
		"a linked .abcd": func(t *testing.T) string {
			dir := t.TempDir()
			if err := os.Symlink(t.TempDir(), filepath.Join(dir, ".abcd")); err != nil {
				t.Skipf("no symlinks here: %v", err)
			}
			return dir
		},
		"a missing root": func(t *testing.T) string { return filepath.Join(t.TempDir(), "gone") },
	} {
		t.Run(name, func(t *testing.T) {
			root := root(t)
			was := builds
			if _, err := shared(root); err != nil {
				t.Fatal(err)
			}
			if builds != was+1 {
				t.Errorf("%s shared a scanner", name)
			}
		})
	}
	t.Run("the repository's identity reaches the scanner", func(t *testing.T) {
		r := gittest.NewRepo(t)
		const mail = "yarrow.quill@example.invalid"
		r.Git("config", "user.email", mail)
		sc, err := shared(r.Root())
		if err != nil {
			t.Fatal(err)
		}
		if len(sc.ScanText("written by "+mail, "issue")) == 0 {
			t.Errorf("the scanner does not know the repository's own %s", mail)
		}
	})
	t.Run("another environment", func(t *testing.T) {
		name, mail := "Zanzibar Quill", "zanzibar@example.invalid"
		t.Setenv("GIT_AUTHOR_NAME", name)
		t.Setenv("GIT_AUTHOR_EMAIL", mail)
		sc, err := shared(plain(t))
		if err != nil {
			t.Fatal(err)
		}
		if len(sc.ScanText("written by "+mail, "issue")) == 0 {
			t.Errorf("the scanner for this environment does not know its author %s", mail)
		}
	})
}
