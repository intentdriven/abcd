package spec

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// The store lock is a flock on specs/ itself, and a holder of that flock —
// another process, or an abcd binary built before the lock moved onto
// fsutil.WithDirLock, which flocks the same directory by hand — keeps a writer
// out for its budget with ErrStoreLockBusy in the words it always used, and
// lets it in once released (iss-129). The holder takes the flock directly, as
// an older binary does, so the two are shown to exclude each other.
func TestTheStoreLockExcludesARawFlockOnTheStore(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root, "itd-9", "my-feature", ""); err != nil {
		t.Fatal(err)
	}
	held, err := os.Open(filepath.Join(root, SpecsRelDir))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	ran := false
	err = WithStoreLockWithin(root, 30*time.Millisecond, func() error { ran = true; return nil })
	if !errors.Is(err, ErrStoreLockBusy) || ran {
		t.Fatalf("a writer under another holder's flock: ran=%v err=%v; want ErrStoreLockBusy", ran, err)
	}
	if want := "spec: could not acquire the spec store's lock within 30ms"; err.Error() != want {
		t.Errorf("the refusal = %q; want %q", err, want)
	}

	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := WithStoreLockWithin(root, time.Second, func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("the writer once the flock was released: ran=%v err=%v", ran, err)
	}
}
