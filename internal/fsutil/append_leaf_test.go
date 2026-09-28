package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAppendLineInRefusesASymlinkedLeaf: the append primitive keeps the leaf
// refusal its read twin ReadGuardedInRoot applies — a symlink or a non-regular
// file at rel is refused, and the file a symlink names is left untouched, even
// when it lies inside the root (iss-2609230720193756).
func TestAppendLineInRefusesASymlinkedLeaf(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "claim.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("claim.json", filepath.Join(dir, "log.jsonl")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "adir"), 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := AppendLineIn(root, "log.jsonl", []byte(`{"a":1}`), 0o600); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("append through a symlinked leaf = %v; want ErrNotRegular", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "claim.json")); string(got) != "{}\n" {
		t.Fatalf("the symlink's target was appended to: %q", got)
	}
	if err := AppendLineIn(root, "adir", []byte(`{"a":1}`), 0o600); err == nil {
		t.Fatal("append to a directory succeeded")
	}
	// A real file, absent or present, is appended to as before.
	for i := 0; i < 2; i++ {
		if err := AppendLineIn(root, "real.jsonl", []byte(`{"a":1}`), 0o600); err != nil {
			t.Fatalf("append %d to a real file: %v", i, err)
		}
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "real.jsonl")); string(got) != "{\"a\":1}\n{\"a\":1}\n" {
		t.Fatalf("real file = %q", got)
	}
}

// TestLockFilesAreTheOwnersAlone: both lock primitives create their lock file
// 0600 — a lock carries nothing another account needs, and 0644 sat beside the
// 0600 files it guards (iss-2609230720193756).
func TestLockFilesAreTheOwnersAlone(t *testing.T) {
	dir := t.TempDir()
	if err := WithFileLock(filepath.Join(dir, "a.lock"), time.Second, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := WithFileLockIn(root, "b.lock", time.Second, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.lock", "b.lock"} {
		fi, err := os.Stat(filepath.Join(dir, n))
		if err != nil {
			t.Fatal(err)
		}
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("%s mode = %o, want 600", n, perm)
		}
	}
}

// TestAppendLineInRefusesALeafLinkedAfterItsLstat: a symlink planted at the leaf
// between the pre-open Lstat (which saw nothing) and the open is refused, and
// the file it names is left untouched. os.Root follows an in-root leaf link
// whatever flags the open carries, so the refusal rests on a post-open Lstat
// that must name the very file opened (iss-2609281229109140).
func TestAppendLineInRefusesALeafLinkedAfterItsLstat(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "claim.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	planted := false
	beforeAppendOpen = func(_ *os.Root, rel string) {
		if err := os.Symlink("claim.json", filepath.Join(dir, rel)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		planted = true
	}
	t.Cleanup(func() { beforeAppendOpen = nil })
	err = AppendLineIn(root, "log.jsonl", []byte(`{"a":1}`), 0o600)
	if !planted {
		t.Fatal("the seam never ran: the link was not planted between the Lstat and the open")
	}
	if !errors.Is(err, ErrNotRegular) {
		t.Fatalf("append through a leaf linked after its Lstat = %v; want ErrNotRegular", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "claim.json")); string(got) != "{}\n" {
		t.Fatalf("the planted link's target was appended to: %q", got)
	}
}
