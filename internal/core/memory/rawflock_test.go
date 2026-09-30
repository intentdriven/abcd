package memory

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// The store lock is a flock on .abcd/memory/.lock, and a holder of that flock —
// another process, or an abcd binary built before the lock moved onto
// fsutil.WithFileLock, which flocks the same file by hand — makes a writer fail
// closed at once with *StoreLockHeldError, and lets it in once released
// (iss-129). The holder takes the flock directly, as an older binary does, so
// the two are shown to exclude each other on the same path.
func TestStoreLockExcludesARawFlockOnTheLockFile(t *testing.T) {
	repo := t.TempDir()
	memDir := filepath.Join(repo, ".abcd", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	held, err := os.OpenFile(filepath.Join(memDir, ".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	ran := false
	err = WithStoreLock(repo, func() error { ran = true; return nil })
	if _, ok := err.(*StoreLockHeldError); !ok || ran {
		t.Fatalf("a writer under another holder's flock: ran=%v err=%v (%T); want *StoreLockHeldError", ran, err, err)
	}

	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := WithStoreLock(repo, func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("the writer once the flock was released: ran=%v err=%v", ran, err)
	}
}
