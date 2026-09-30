package decide

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// The mint lock is a flock on the ADR store's own directory, and a holder of
// that flock — another process, or an abcd binary built before the lock moved
// onto fsutil.WithDirLock, which flocks the same directory by hand — keeps a
// mint out for mintLockTimeout, refused in the words the mint always used,
// and lets it in once released (iss-129). The holder here takes the flock
// directly, as an older binary does, so the test pins that the two exclude
// each other and not merely that the new lock excludes itself.
func TestTheMintLockExcludesARawFlockOnTheStore(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(ADRsRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	held, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	old := mintLockTimeout
	mintLockTimeout = 200 * time.Millisecond
	t.Cleanup(func() { mintLockTimeout = old })

	ran := false
	err = withMintLock(root, func() error { ran = true; return nil })
	if ran || err == nil {
		t.Fatalf("a mint under another holder's flock: ran=%v err=%v", ran, err)
	}
	if want := "decide: could not acquire mint lock within 200ms"; err.Error() != want {
		t.Errorf("the refusal = %q; want %q", err, want)
	}

	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := withMintLock(root, func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("the mint once the flock was released: ran=%v err=%v", ran, err)
	}
}
