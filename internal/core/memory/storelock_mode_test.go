package memory

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestStoreLockRefusesANonRegularLockFile pins iss-2608261133210491 on the
// behaviour rather than on a helper. The store-lock guard once tested
// st.Mode&S_IFREG != 0, but the file-type field is an enumeration, not a set
// of flags: a socket and a symlink both carry the S_IFREG bit, so the "regular
// file" assertion admitted them. The descriptor check is fsutil.WithFileLock's
// now (iss-129), which compares the type under the S_IFMT mask; a FIFO at the
// lock path — no S_IFREG bit of its own, openable without blocking — is refused
// as *UnsafeStorePathError and fn never runs.
func TestStoreLockRefusesANonRegularLockFile(t *testing.T) {
	repo := t.TempDir()
	memDir := filepath.Join(repo, ".abcd", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(memDir, ".lock"), 0o600); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	ran := false
	err := WithStoreLock(repo, func() error { ran = true; return nil })
	var unsafe *UnsafeStorePathError
	if !errors.As(err, &unsafe) || ran {
		t.Fatalf("a FIFO at the lock path: ran=%v err=%v (%T); want *UnsafeStorePathError", ran, err, err)
	}
}
