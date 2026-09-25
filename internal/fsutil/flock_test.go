package fsutil

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWithFileLockSurvivesAHolderUnlinkingThePath pins the inode revalidation
// that lets a holder retire its lock file. A waiter that opened the lock path
// before the holder unlinked it would otherwise acquire the flock on the
// orphaned inode while a newcomer holds the fresh file at the same path: two
// holders of one lock. After acquiring, a waiter must confirm the path still
// names the inode it locked, and start over on the current file when it does
// not.
func TestWithFileLockSurvivesAHolderUnlinkingThePath(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "k.lock")
	var inside atomic.Int32
	var overlap atomic.Bool
	critical := func(hold time.Duration) func() error {
		return func() error {
			if inside.Add(1) > 1 {
				overlap.Store(true)
			}
			time.Sleep(hold)
			inside.Add(-1)
			return nil
		}
	}

	holderIn := make(chan struct{})
	unlinked := make(chan struct{})
	newcomerIn := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() { // A: holds the original inode, unlinks it, waits for C to hold the fresh one
		defer wg.Done()
		_ = WithFileLock(lock, 5*time.Second, func() error {
			close(holderIn)
			time.Sleep(80 * time.Millisecond) // B opens the original inode and starts polling
			if err := os.Remove(lock); err != nil {
				t.Error(err)
			}
			close(unlinked)
			<-newcomerIn
			return nil
		})
	}()
	<-holderIn

	wg.Add(1)
	go func() { // B: opened the original inode before the unlink
		defer wg.Done()
		if err := WithFileLock(lock, 5*time.Second, critical(20*time.Millisecond)); err != nil {
			t.Error(err)
		}
	}()

	<-unlinked
	wg.Add(1)
	go func() { // C: creates and holds the fresh file at the same path
		defer wg.Done()
		if err := WithFileLock(lock, 5*time.Second, func() error {
			close(newcomerIn)
			return critical(300 * time.Millisecond)()
		}); err != nil {
			t.Error(err)
		}
	}()
	wg.Wait()
	if overlap.Load() {
		t.Fatal("two holders were inside the lock at once: a waiter locked the unlinked inode")
	}
}
