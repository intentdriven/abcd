package release

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// changelogLockTimeout bounds the wait for another cut of the same tree. The
// lock covers the whole ingest, the derivation's git reads included, so it is
// longer than a single file rewrite's; a var so a test can shorten it.
var changelogLockTimeout = 30 * time.Second

// changelogLockPath names the lock the cut's writes take: .CHANGELOG.md.lock,
// beside the file the tagging workflow reads.
func changelogLockPath(root string) string {
	return filepath.Join(root, "."+changelogFile+".lock")
}

// withChangelogLock runs fn holding the CHANGELOG's lock. It is taken by every
// writer of the release record — Ingest, whose writes are the archive move, the
// release page and the CHANGELOG section, and UndoPlan.Apply, which puts them
// back — so no two of them interleave in one working tree (iss-127).
//
// The lock is fsutil.WithFileLock, the one inter-process load-modify-write
// primitive. Its file sits in the repository root, beside a committed file, so
// the holder removes it before letting go (WithFileLock's revalidation makes
// that safe for a waiter that opened it first), and a cut leaves nothing behind
// for the dirty-tree gate of the next one to find. It is a leaf: no other lock
// is taken inside it.
func withChangelogLock(root string, fn func() error) error {
	lock := changelogLockPath(root)
	err := fsutil.WithFileLock(lock, changelogLockTimeout, func() error {
		defer os.Remove(lock)
		return fn()
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("another cut of this tree holds %s (past %s), so nothing was written; let it finish and cut again: %w",
			filepath.Base(lock), changelogLockTimeout, err)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("the lock %s is a symlink or not a regular file, so it is refused and nothing was written; remove it: %w",
			filepath.Base(lock), err)
	}
	return err
}
