package lifeboat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// embark_walk_hardening_test.go covers the GH #337/#343 hardening of
// walkLifeboatFiles: an untrusted lifeboat made only of directories (deep or
// wide) must be REFUSED before it can exhaust memory, mirroring the already
// bounded probe.go sibling (walkFilesBounded / readDirBounded, iss-112/114/116).
// The bounds are exercised through walkLifeboatFilesBounded at an affordable
// scale — the shipped walkLifeboatFiles injects the const ceilings.

// TestWalkLifeboatFilesCountsDirectoriesTowardCap is the GH #337 core: the old
// walk incremented its count only on regular files, so a lifeboat holding no
// regular file — only directories — never reached maxEmbarkFiles and walked
// unbounded. Ten empty directories under an entry cap of 5 must be refused.
func TestWalkLifeboatFilesCountsDirectoriesTowardCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		if err := os.MkdirAll(filepath.Join(dir, fmt.Sprintf("d%02d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	// perDir 50 (> the 10 siblings) and depth 64 leave only the entry cap to fire:
	// with directories counted, the 6th directory trips a cap of 5.
	if _, err := walkLifeboatFilesBounded(root, 5, 50, 64); err == nil {
		t.Fatal("a directory-only lifeboat (10 empty dirs) was not refused under an entry cap of 5: directories are not counted toward the cap")
	}
}

// TestWalkLifeboatFilesBoundsDeepChain is the GH #337 deep-chain facet: a chain
// a/a/a/… of directories holds no regular file, so the entry cap alone cannot
// bound it — the depth cap must. A 40-deep chain under a depth cap of 8 is
// refused.
func TestWalkLifeboatFilesBoundsDeepChain(t *testing.T) {
	dir := t.TempDir()
	deep := dir
	for i := 0; i < 40; i++ {
		deep = filepath.Join(deep, "a")
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	// A generous entry cap (never reached — the chain is 40 dirs) so only the depth
	// cap can fire.
	if _, err := walkLifeboatFilesBounded(root, 1000, 50, 8); err == nil {
		t.Fatal("a 40-deep directory-only chain was not refused under a depth cap of 8: no depth bound")
	}
}

// TestWalkLifeboatFilesBoundsPerDirectoryWidth is the GH #343 facet: one very
// wide directory must not be fully materialised before the cap applies. The old
// fs.WalkDir(root.FS(), …) did ReadDir(-1)+sort on the whole directory before the
// per-entry callback ever ran. A 30-entry directory under a per-directory bound
// of 5 must be refused after materialising at most the bound.
func TestWalkLifeboatFilesBoundsPerDirectoryWidth(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 30; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%02d", i)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	// A generous entry cap and depth so only the per-directory width bound can fire.
	if _, err := walkLifeboatFilesBounded(root, 1000, 5, 64); err == nil {
		t.Fatal("a 30-entry directory was not refused under a per-directory bound of 5: the whole listing is materialised before the cap")
	}
}

// TestWalkLifeboatFilesReturnsRegularFilesWithinBounds is the positive control: a
// small, well-formed tree walks to completion and returns its regular files, so
// the bounding did not break the ordinary path (the reseal / synthesis path).
func TestWalkLifeboatFilesReturnsRegularFilesWithinBounds(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"a.txt":       "x\n",
		"sub/b.txt":   "y\n",
		"sub/c/d.txt": "z\n",
	})
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	rels, err := walkLifeboatFilesBounded(root, 1000, 50, 64)
	if err != nil {
		t.Fatalf("well-formed tree refused: %v", err)
	}
	got := strings.Join(rels, ",")
	if got != "a.txt,sub/b.txt,sub/c/d.txt" {
		t.Fatalf("walk = %q, want the three regular files sorted", got)
	}
}

// TestWalkLifeboatFilesNamesTheEntryItCannotOpen is iss-2609252004013212: a
// directory the descent cannot open refused with the raw open error, whose path
// carries the "/." suffix openWalkDir adds and names only the last component, so
// the refusal named a path that does not exist. It names the entry by its path
// in the lifeboat instead. An unreadable directory stands in for the swapped-in
// FIFO the suffix exists to refuse, because both fail the same open.
func TestWalkLifeboatFilesNamesTheEntryItCannotOpen(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root opens a directory whatever its mode")
	}
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"sub/locked/a.txt": "x\n"})
	locked := filepath.Join(dir, "sub", "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = walkLifeboatFilesBounded(root, 1000, 50, 64)
	if err == nil {
		t.Fatal("an unopenable lifeboat directory was walked as if it were readable")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"sub/locked"`) {
		t.Errorf("the refusal must name the entry by its lifeboat path, got %q", msg)
	}
	if strings.Contains(msg, "/.") {
		t.Errorf("the refusal leaks the descent's internal \"/.\" suffix: %q", msg)
	}
}
