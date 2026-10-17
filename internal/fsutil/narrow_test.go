//go:build unix

package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// mkdirMode makes dir and pins its mode past the umask.
func mkdirMode(t *testing.T, dir string, perm os.FileMode) {
	t.Helper()
	if err := os.Mkdir(dir, perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, perm); err != nil {
		t.Fatal(err)
	}
}

func permOf(t *testing.T, p string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// TestNarrowToOwnerClosesADirectoryTheCallerOwns: a directory the caller owns
// that its group or every account can open is narrowed to the owner's bits
// alone, and one already the owner's alone is left as it is.
func TestNarrowToOwnerClosesADirectoryTheCallerOwns(t *testing.T) {
	for _, c := range []struct {
		have, want os.FileMode
		narrowed   bool
	}{
		{0o755, 0o700, true},
		{0o775, 0o700, true},
		{0o750, 0o700, true},
		{0o711, 0o700, true},
		{0o700, 0o700, false},
		{0o500, 0o500, false},
	} {
		dir := filepath.Join(t.TempDir(), "home")
		mkdirMode(t, dir, c.have)
		got, err := NarrowToOwner(dir)
		if err != nil {
			t.Fatalf("%04o: %v", c.have, err)
		}
		if got != c.narrowed || permOf(t, dir) != c.want {
			t.Errorf("%04o: narrowed %v, mode %04o; want %v, %04o", c.have, got, permOf(t, dir), c.narrowed, c.want)
		}
		_ = os.Chmod(dir, 0o700)
	}
}

// TestNarrowToOwnerNeverFollowsALink: a symlink at the path is not followed,
// so the directory it points at keeps its mode, and nothing is reported
// narrowed.
func TestNarrowToOwnerNeverFollowsALink(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "elsewhere")
	mkdirMode(t, target, 0o755)
	link := filepath.Join(base, "home")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	narrowed, err := NarrowToOwner(link)
	if narrowed || err != nil {
		t.Fatalf("NarrowToOwner through a link = %v, %v; want false, nil", narrowed, err)
	}
	if got := permOf(t, target); got != 0o755 {
		t.Fatalf("the link's target was re-moded to %04o", got)
	}
}

// TestNarrowToOwnerLeavesAnotherAccountsDirectory: a directory owned by any
// other account, or whose owner cannot be read, is never re-moded.
func TestNarrowToOwnerLeavesAnotherAccountsDirectory(t *testing.T) {
	me := uint32(os.Getuid())
	for name, owner := range map[string]func(os.FileInfo) (uint32, bool){
		"foreign":    func(os.FileInfo) (uint32, bool) { return me + 1, true },
		"unreadable": func(os.FileInfo) (uint32, bool) { return 0, false },
	} {
		dir := filepath.Join(t.TempDir(), "home")
		mkdirMode(t, dir, 0o755)
		restore := swapHomeScopeDirOwnerForTest(owner)
		narrowed, err := NarrowToOwner(dir)
		restore()
		if narrowed || err != nil || permOf(t, dir) != 0o755 {
			t.Errorf("%s owner: narrowed %v, err %v, mode %04o; want it left 0755", name, narrowed, err, permOf(t, dir))
		}
	}
}

// TestNarrowToOwnerIgnoresWhatIsNotADirectory: an absent path and a file are
// nothing to narrow, and neither is an error.
func TestNarrowToOwnerIgnoresWhatIsNotADirectory(t *testing.T) {
	base := t.TempDir()
	if narrowed, err := NarrowToOwner(filepath.Join(base, "absent")); narrowed || err != nil {
		t.Fatalf("absent: %v, %v", narrowed, err)
	}
	f := filepath.Join(base, "file")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f, 0o644); err != nil {
		t.Fatal(err)
	}
	if narrowed, err := NarrowToOwner(f); narrowed || err != nil || permOf(t, f) != 0o644 {
		t.Fatalf("file: %v, %v, %04o", narrowed, err, permOf(t, f))
	}
}
