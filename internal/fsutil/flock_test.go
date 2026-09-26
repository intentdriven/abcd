package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// TestWithFileLockContentionNamesTheCallersTimeout: a waiter that loses its
// inode to a retired lock file starts over on the current one within the same
// deadline, and when that deadline passes the contention error must name the
// timeout the caller gave, not the slice that was left for the retry.
func TestWithFileLockContentionNamesTheCallersTimeout(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "k.lock")
	const timeout = 400 * time.Millisecond
	holderIn := make(chan struct{})
	unlinked := make(chan struct{})
	newcomerIn := make(chan struct{})
	release := make(chan struct{})
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

	got := make(chan error, 1)
	go func() { // B: retries onto the fresh file and times out behind C
		got <- WithFileLock(lock, timeout, func() error { return nil })
	}()

	<-unlinked
	wg.Add(1)
	go func() { // C: holds the fresh file until B has given up
		defer wg.Done()
		_ = WithFileLock(lock, 5*time.Second, func() error {
			close(newcomerIn)
			<-release
			return nil
		})
	}()
	err := <-got
	close(release)
	wg.Wait()
	if !errors.Is(err, ErrLockContention) {
		t.Fatalf("B = %v; want ErrLockContention behind the newcomer", err)
	}
	if !strings.Contains(err.Error(), "within "+timeout.String()) {
		t.Fatalf("contention error %q does not name the caller's %s timeout", err, timeout)
	}
}
