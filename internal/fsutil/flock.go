package fsutil

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// ErrLockContention and ErrLockPathUnsafe are the file-lock sentinels: the
// exclusive lock could not be acquired within the caller's timeout, and the lock
// path is a symlink or not a regular file.
var (
	ErrLockContention = errors.New("fsutil: lock contention")
	ErrLockPathUnsafe = errors.New("fsutil: lock path unsafe")
)

// WithFileLock acquires an exclusive advisory (flock) lock on lockPath, holds it
// across fn, and releases it when fn returns. The lock file is opened O_NOFOLLOW
// (a symlinked lock path is refused) and verified on the same descriptor to be a
// regular file, both reported as ErrLockPathUnsafe; the exclusive flock is polled
// for up to timeout, after which ErrLockContention is returned. fn's own error is
// returned unwrapped.
//
// It is the single inter-process load-modify-write lock primitive: capture's
// ledger allocator and ahoy's ~/.abcd history registry both route through it
// rather than keep divergent flock loops (the one-canonical-primitive invariant).
// The lock is advisory — every writer of the guarded state must take it — and its
// scope is one lock file; callers that must not deadlock keep their acquisitions
// unnested.
//
// A holder MAY unlink lockPath inside fn, which is how a caller retires a
// per-key lock file once the key has nothing left to guard. That is safe
// because acquisition is revalidated: after the flock is granted, the path must
// still name the inode that was locked. A waiter that opened the file before a
// holder unlinked it would otherwise be granted the flock on the orphaned inode
// while a newcomer holds the fresh file at the same path — two holders of one
// lock. On a mismatch the waiter releases, reopens the current file and waits
// again, all within the one timeout.
func WithFileLock(lockPath string, timeout time.Duration, fn func() error) error {
	deadline := time.Now().Add(timeout)
	for {
		fd, err := openLockFd(lockPath)
		if err != nil {
			return err
		}
		if err := acquireFlock(fd, deadline, timeout); err != nil {
			syscall.Close(fd)
			return err
		}
		current, err := lockStillNamesFd(lockPath, fd)
		if err != nil {
			syscall.Flock(fd, syscall.LOCK_UN)
			syscall.Close(fd)
			return err
		}
		if !current {
			syscall.Flock(fd, syscall.LOCK_UN)
			syscall.Close(fd)
			continue
		}
		defer syscall.Close(fd)
		defer syscall.Flock(fd, syscall.LOCK_UN)
		return fn()
	}
}

// lockStillNamesFd reports whether lockPath still names the inode fd holds. An
// absent path (a holder retired it) is false, not an error; a path that is now
// a symlink is refused as ErrLockPathUnsafe rather than followed.
func lockStillNamesFd(lockPath string, fd int) (bool, error) {
	var held, named syscall.Stat_t
	if err := syscall.Fstat(fd, &held); err != nil {
		return false, err
	}
	if err := syscall.Lstat(lockPath, &named); err != nil {
		if err == syscall.ENOENT {
			return false, nil
		}
		return false, err
	}
	if named.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		return false, fmt.Errorf("%w: lock path is a symlink: %s", ErrLockPathUnsafe, lockPath)
	}
	return held.Dev == named.Dev && held.Ino == named.Ino, nil
}

// openLockFd opens lockPath with O_CREAT|O_RDWR|O_NOFOLLOW and verifies, on the
// same descriptor, that it is a regular file — refusing a symlinked or
// non-regular lock path with ErrLockPathUnsafe.
func openLockFd(lockPath string) (int, error) {
	fd, err := syscall.Open(lockPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		if err == syscall.ELOOP {
			return -1, fmt.Errorf("%w: lock path is a symlink: %s", ErrLockPathUnsafe, lockPath)
		}
		return -1, err
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		syscall.Close(fd)
		return -1, err
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFREG {
		syscall.Close(fd)
		return -1, fmt.Errorf("%w: lock path is not a regular file: %s", ErrLockPathUnsafe, lockPath)
	}
	return fd, nil
}

// acquireFlock polls for an exclusive flock until deadline, returning
// ErrLockContention on timeout. The error names timeout, the caller's whole
// budget: a revalidation retry spends one deadline across more than one
// acquisition, and the slice left for the last one is not what was asked for.
func acquireFlock(fd int, deadline time.Time, timeout time.Duration) error {
	backoff := 5 * time.Millisecond
	for {
		err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			return err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("%w: could not acquire lock within %s", ErrLockContention, timeout)
		}
		if backoff > remaining {
			backoff = remaining
		}
		time.Sleep(backoff)
		if backoff < 100*time.Millisecond {
			backoff *= 2
		}
	}
}

// WithFileLockIn is WithFileLock with the lock file resolved INSIDE root: rel is
// a slash path relative to it, so an ancestor swapped for a symlink cannot carry
// the lock out of the containment scope between a caller's checks and the open.
// The leaf keeps WithFileLock's refusals: a symlink or a non-regular file at rel
// is ErrLockPathUnsafe, judged by Lstat and confirmed on the opened descriptor
// (os.SameFile), because os.Root follows a symlink that stays inside the root.
func WithFileLockIn(root *os.Root, rel string, timeout time.Duration, fn func() error) error {
	f, err := openLockIn(root, rel)
	if err != nil {
		return err
	}
	defer f.Close()
	fd := int(f.Fd())
	if err := acquireFlock(fd, time.Now().Add(timeout), timeout); err != nil {
		return err
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	return fn()
}

// openLockIn opens (creating when absent) the lock file at rel inside root and
// proves the descriptor is the regular, non-symlinked file that was checked.
func openLockIn(root *os.Root, rel string) (*os.File, error) {
	pre, lerr := root.Lstat(rel)
	switch {
	case lerr == nil && pre.Mode()&os.ModeSymlink != 0:
		return nil, fmt.Errorf("%w: lock path is a symlink: %s", ErrLockPathUnsafe, rel)
	case lerr == nil && !pre.Mode().IsRegular():
		return nil, fmt.Errorf("%w: lock path is not a regular file: %s", ErrLockPathUnsafe, rel)
	case lerr != nil && !errors.Is(lerr, os.ErrNotExist):
		return nil, lerr
	}
	f, err := openOrCreateIn(root, rel, os.O_RDWR|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !st.Mode().IsRegular() || (lerr == nil && !os.SameFile(pre, st)) {
		f.Close()
		return nil, fmt.Errorf("%w: lock path changed or is not a regular file: %s", ErrLockPathUnsafe, rel)
	}
	return f, nil
}

// openOrCreateIn opens rel inside root, creating it at perm when absent, using
// only the two opens that behave when several processes race to create the same
// file: a plain open, and an exclusive create exactly one racer wins. A single
// non-exclusive openat(O_CREAT) relative to a directory descriptor was observed
// on darwin to fail with ENOENT for some of several racers (openAppendIn's note),
// which here would fail a ledger verb on a lock file every racer was creating.
func openOrCreateIn(root *os.Root, rel string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := root.OpenFile(rel, flag, 0)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return f, err
	}
	f, err = root.OpenFile(rel, flag|os.O_CREATE|os.O_EXCL, perm)
	if err == nil || !errors.Is(err, os.ErrExist) {
		return f, err
	}
	return root.OpenFile(rel, flag, 0)
}
