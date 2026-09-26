package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWithFileLockInRefusesASymlinkedLeafAndAnEscape: the contained lock keeps
// WithFileLock's leaf refusal, and a lock path that resolves outside the root is
// refused by the root rather than opened (iss-2609012037143368).
func TestWithFileLockInRefusesASymlinkedLeafAndAnEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "real"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "linked.lock")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "away")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	ran := false
	body := func() error { ran = true; return nil }

	if err := WithFileLockIn(root, "linked.lock", time.Second, body); !errors.Is(err, ErrLockPathUnsafe) {
		t.Fatalf("symlinked leaf: want ErrLockPathUnsafe, got %v", err)
	}
	if err := WithFileLockIn(root, "away/x.lock", time.Second, body); err == nil {
		t.Fatal("a lock path escaping the root was opened")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("the escaping lock created %v outside the root", entries)
	}
	if ran {
		t.Fatal("fn ran under a refused lock")
	}
	if err := WithFileLockIn(root, "ok.lock", time.Second, body); err != nil || !ran {
		t.Fatalf("a plain lock path: err=%v ran=%v", err, ran)
	}
}
