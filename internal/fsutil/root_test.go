package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rootFixture builds a directory that a hostile repository could ship: a real
// file, a symlinked leaf, and two symlinked ANCESTOR directories — one with an
// absolute target, one with a relative one — each pointing at a sibling
// directory the root does not contain.
func rootFixture(t *testing.T) (dir, outside string) {
	t.Helper()
	base := t.TempDir()
	outside = filepath.Join(base, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("OUTSIDE"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(base, "root")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.txt"), []byte("INSIDE"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", filepath.Join(dir, "leaflink.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "abslink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../outside", filepath.Join(dir, "rellink")); err != nil {
		t.Fatal(err)
	}
	return dir, outside
}

func openFixtureRoot(t *testing.T, dir string) *os.Root {
	t.Helper()
	r, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func TestReadGuardedInRootReadsAContainedFile(t *testing.T) {
	dir, _ := rootFixture(t)
	r := openFixtureRoot(t, dir)
	got, err := ReadGuardedInRoot(r, "real.txt", 1<<20)
	if err != nil {
		t.Fatalf("ReadGuardedInRoot: %v", err)
	}
	if string(got) != "INSIDE" {
		t.Fatalf("read %q, want %q", got, "INSIDE")
	}
}

// The containment guarantee ValidRelPath cannot give: neither path holds a ".."
// or a leading "/", so only the root refuses them.
func TestReadGuardedInRootRefusesASymlinkedAncestor(t *testing.T) {
	dir, _ := rootFixture(t)
	r := openFixtureRoot(t, dir)
	for _, rel := range []string{"abslink/secret.txt", "rellink/secret.txt"} {
		if !ValidRelPath(rel) {
			t.Fatalf("fixture is not the case under test: ValidRelPath(%q) is false", rel)
		}
		data, err := ReadGuardedInRoot(r, rel, 1<<20)
		if err == nil {
			t.Fatalf("ReadGuardedInRoot(%q) read outside the root: %q", rel, data)
		}
		if os.IsNotExist(err) {
			t.Fatalf("ReadGuardedInRoot(%q) reported an escape as absence: %v", rel, err)
		}
		if strings.Contains(string(data), "OUTSIDE") {
			t.Fatalf("ReadGuardedInRoot(%q) returned outside content", rel)
		}
	}
}

// The leaf guarantees ReadGuarded gives must survive the move into the root.
func TestReadGuardedInRootKeepsTheLeafGuards(t *testing.T) {
	dir, _ := rootFixture(t)
	r := openFixtureRoot(t, dir)

	if _, err := ReadGuardedInRoot(r, "leaflink.txt", 1<<20); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("symlinked leaf: err = %v, want ErrNotRegular", err)
	}
	if _, err := ReadGuardedInRoot(r, "sub", 1<<20); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("directory leaf: err = %v, want ErrNotRegular", err)
	}
	if _, err := ReadGuardedInRoot(r, "real.txt", 3); !errors.Is(err, ErrTooBig) {
		t.Fatalf("oversize: err = %v, want ErrTooBig", err)
	}
	if _, err := ReadGuardedInRoot(r, "absent.txt", 1<<20); !os.IsNotExist(err) {
		t.Fatalf("absent: err = %v, want an IsNotExist error", err)
	}
}

func TestWriteFileAtomicInRootWritesAndRefusesAnEscape(t *testing.T) {
	dir, outside := rootFixture(t)
	r := openFixtureRoot(t, dir)

	if err := WriteFileAtomicInRoot(r, "deep/nested/out.txt", []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomicInRoot: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "deep", "nested", "out.txt"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("read back = %q, %v", got, err)
	}
	if fi, serr := os.Stat(filepath.Join(dir, "deep", "nested", "out.txt")); serr != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, %v; want 0600", fi, serr)
	}
	// No temp file survives the write.
	entries, err := os.ReadDir(filepath.Join(dir, "deep", "nested"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("directory holds %d entries, want only the written file", len(entries))
	}

	for _, rel := range []string{"abslink/planted.txt", "rellink/planted.txt"} {
		if err := WriteFileAtomicInRoot(r, rel, []byte("planted"), 0o600); err == nil {
			t.Fatalf("WriteFileAtomicInRoot(%q) wrote through a symlinked ancestor", rel)
		}
	}
	if entries, err := os.ReadDir(outside); err == nil {
		for _, e := range entries {
			if e.Name() != "secret.txt" {
				t.Fatalf("a write landed outside the root: %s", e.Name())
			}
		}
	}
}

func TestWriteFileAtomicPreserveModeInRootKeepsTheExistingMode(t *testing.T) {
	dir, _ := rootFixture(t)
	r := openFixtureRoot(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "moded.txt"), []byte("a"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomicPreserveModeInRoot(r, "moded.txt", []byte("b")); err != nil {
		t.Fatalf("WriteFileAtomicPreserveModeInRoot: %v", err)
	}
	fi, err := os.Stat(filepath.Join(dir, "moded.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want 0640", fi.Mode().Perm())
	}

	// A new file defaults to 0644, and an escaping path refuses rather than
	// silently defaulting.
	if err := WriteFileAtomicPreserveModeInRoot(r, "fresh.txt", []byte("c")); err != nil {
		t.Fatalf("new file: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(dir, "fresh.txt")); err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("new-file mode = %v, %v; want 0644", fi, err)
	}
	if err := WriteFileAtomicPreserveModeInRoot(r, "rellink/planted.txt", []byte("d")); err == nil {
		t.Fatal("WriteFileAtomicPreserveModeInRoot wrote through a symlinked ancestor")
	}
}

// TestWriteFileAtomicModeOfInRootCarriesTheSourceMode: a write to a new path
// that takes its mode from another file (a record moved between status
// folders) carries that file's mode, defaults to 0644 when the source is
// absent, and refuses a source path that escapes the root.
func TestWriteFileAtomicModeOfInRootCarriesTheSourceMode(t *testing.T) {
	dir, _ := rootFixture(t)
	r := openFixtureRoot(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "from.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomicModeOfInRoot(r, "sub/to.txt", "from.txt", []byte("b")); err != nil {
		t.Fatalf("WriteFileAtomicModeOfInRoot: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(dir, "sub", "to.txt")); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, %v; want 0600", fi, err)
	}
	if err := WriteFileAtomicModeOfInRoot(r, "fresh.txt", "absent.txt", []byte("c")); err != nil {
		t.Fatalf("absent source: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(dir, "fresh.txt")); err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("absent-source mode = %v, %v; want 0644", fi, err)
	}
	if err := WriteFileAtomicModeOfInRoot(r, "other.txt", "rellink/secret.txt", []byte("d")); err == nil {
		t.Fatal("a source through a symlinked ancestor was read for its mode")
	}
}
