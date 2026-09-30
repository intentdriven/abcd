package intent

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// The mint lock is a flock on intents/ itself, and a holder of that flock —
// another process, or an abcd binary built before the lock moved onto
// fsutil.WithDirLock, which flocks the same directory by hand — keeps a writer
// out for its budget with errIntentLockBusy in the words it always used, and
// lets it in once released (iss-129). The holder takes the flock directly, as
// an older binary does, so the two are shown to exclude each other.
func TestTheMintLockExcludesARawFlockOnTheStore(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, IntentsRelDir)
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

	ran := false
	err = withIntentMintLockWithin(root, 30*time.Millisecond, func() error { ran = true; return nil })
	if !errors.Is(err, errIntentLockBusy) || ran {
		t.Fatalf("a writer under another holder's flock: ran=%v err=%v; want errIntentLockBusy", ran, err)
	}
	if want := "intent: could not acquire mint lock within 30ms"; err.Error() != want {
		t.Errorf("the refusal = %q; want %q", err, want)
	}

	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := withIntentMintLockWithin(root, time.Second, func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("the writer once the flock was released: ran=%v err=%v", ran, err)
	}
}
