package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateRunDirRefusesASymlinkedAncestor: a link at any level of the chain —
// not only the leaf — is refused, and nothing is created at its target.
func TestCreateRunDirRefusesASymlinkedAncestor(t *testing.T) {
	for _, level := range []string{".abcd", ".abcd/.work.local", ".abcd/.work.local/logs"} {
		t.Run(level, func(t *testing.T) {
			base, elsewhere := t.TempDir(), t.TempDir()
			link := filepath.Join(base, filepath.FromSlash(level))
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(elsewhere, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			_, err := CreateRunDir(base, ".abcd/.work.local/logs/memory", "lint-x", 0o755)
			if !errors.Is(err, ErrNotRealDir) {
				t.Fatalf("CreateRunDir through a symlinked %s = %v, want ErrNotRealDir", level, err)
			}
			if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
				t.Errorf("created %d entr(y|ies) through the symlink", len(entries))
			}
		})
	}
}

// TestCreateRunDirCreatesAFreshDirectoryPerRun is the ordinary path: the chain
// is created, and a second run with the same name gets name-001 rather than
// sharing the first run's directory.
func TestCreateRunDirCreatesAFreshDirectoryPerRun(t *testing.T) {
	base := t.TempDir()
	first, err := CreateRunDir(base, ".abcd/.work.local/logs/memory", "lint-x", 0o755)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := CreateRunDir(base, ".abcd/.work.local/logs/memory", "lint-x", 0o755)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if filepath.Base(first) != "lint-x" || filepath.Base(second) != "lint-x-001" {
		t.Errorf("run directories = %s, %s; want lint-x, lint-x-001", filepath.Base(first), filepath.Base(second))
	}
	for _, d := range []string{first, second} {
		if !IsRealDir(d) {
			t.Errorf("%s is not a real directory", d)
		}
	}
}

// TestCreateRunDirRefusesANameThatIsAPath keeps the run directory one level
// deep: a name carrying a separator or a traversal is refused before anything
// under the proved chain is created.
func TestCreateRunDirRefusesANameThatIsAPath(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"a/b", "..", "../x", ""} {
		if _, err := CreateRunDir(base, "logs", name, 0o755); !errors.Is(err, os.ErrInvalid) {
			t.Errorf("CreateRunDir(name=%q) = %v, want os.ErrInvalid", name, err)
		}
	}
}

// TestOpenRealDirRefusesALink: the write handle is opened on a real directory
// only, so a run directory swapped for a link is refused rather than written
// through.
func TestOpenRealDirRefusesALink(t *testing.T) {
	base, elsewhere := t.TempDir(), t.TempDir()
	link := filepath.Join(base, "run")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := OpenRealDir(link); !errors.Is(err, ErrNotRealDir) {
		t.Fatalf("OpenRealDir(link) = %v, want ErrNotRealDir", err)
	}
	root, err := OpenRealDir(base)
	if err != nil {
		t.Fatalf("OpenRealDir(real) = %v", err)
	}
	root.Close()
}
