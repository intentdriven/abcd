package history

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The records lock waits a bounded time (iss-129). It blocked on flock with no
// timeout, so a capture behind a holder that never let go — a hung capture, or
// a child that inherited the lock — hung with it, and the session-end hook that
// runs a capture hung too. Past repoLockTimeout the capture now gives up with
// fsutil.ErrLockContention naming the lock, writes nothing, and captures once
// the lock is free. The holder takes records/.lock with a raw flock, as an abcd
// binary built before the move does, so the test also shows the two exclude
// each other on the same path.
func TestCaptureGivesUpOnAHeldRecordsLockWithinItsBudget(t *testing.T) {
	repoRoot, _ := setupStore(t)
	store, err := Resolve(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(store.Records, ".lock")
	held, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	old := repoLockTimeout
	repoLockTimeout = 300 * time.Millisecond
	t.Cleanup(func() { repoLockTimeout = old })

	meta := CaptureMeta{SessionID: "sess-held-lock", Kind: "native"}
	done := make(chan error, 1)
	go func() {
		_, err := Capture(repoRoot, testRootSHA, []byte("assistant: hi\n"), meta)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, fsutil.ErrLockContention) {
			t.Fatalf("Capture behind a held records lock = %v; want fsutil.ErrLockContention", err)
		}
		if !strings.Contains(err.Error(), lockPath) {
			t.Errorf("the refusal %q does not name the lock %s", err, lockPath)
		}
	case <-time.After(10 * time.Second):
		_ = syscall.Flock(int(held.Fd()), syscall.LOCK_UN)
		<-done
		t.Fatal("Capture was still waiting on the records lock after 10s, past its 300ms budget: the wait is unbounded")
	}
	if recs, err := List(repoRoot, testRootSHA); err != nil || len(recs) != 0 {
		t.Fatalf("a refused capture wrote a record: %d record(s), err %v", len(recs), err)
	}

	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	res, err := Capture(repoRoot, testRootSHA, []byte("assistant: hi\n"), meta)
	if err != nil || !res.Wrote {
		t.Fatalf("Capture once the lock was released: wrote=%v err=%v", res.Wrote, err)
	}
}
