package ahoy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// rewriteLockTimeout bounds the wait for another abcd rewriting the same repo
// file. The rewrites it serialises are a read, an edit in memory and an atomic
// write, so a holder is done in milliseconds; a var so a test can shorten it.
var rewriteLockTimeout = 5 * time.Second

// rewriteLockPath names the lock that guards the read-modify-write of path: a
// dot-file beside it (.config.json.lock, .gitignore.lock, .CLAUDE.md.lock).
func rewriteLockPath(path string) string {
	name := filepath.Base(path)
	if !strings.HasPrefix(name, ".") {
		name = "." + name
	}
	return filepath.Join(filepath.Dir(path), name+".lock")
}

// withRewriteLock runs fn — a read, change and write of the repo file at path —
// holding that file's lock, so two abcd runs in one working tree never write
// from the same stale read and erase each other's change (iss-127). Every ahoy
// writer of .abcd/config.json, of abcd's .gitignore block and of the marker
// block in CLAUDE.md / AGENTS.md takes it, and each re-reads the file INSIDE fn:
// the lock is advisory, and a read taken before it proves nothing. It is never
// held across a prompt; a step that asks first re-reads under the lock after.
//
// The lock is fsutil.WithFileLock, the one inter-process load-modify-write
// primitive, on a lock file beside the guarded file. That file sits in the
// user's tree — beside a committed file, in a directory git lists — so the
// holder removes it before letting go (WithFileLock's revalidation makes that
// safe for a waiter that opened it first), and a rewrite leaves nothing behind.
//
// Each lock is a leaf: no holder takes a second rewrite lock, or any other
// lock, inside fn.
func withRewriteLock(path string, fn func() error) error {
	lock := rewriteLockPath(path)
	err := fsutil.WithFileLock(lock, rewriteLockTimeout, func() error {
		defer os.Remove(lock)
		return fn()
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("another abcd is rewriting %s (its lock %s was held past %s); nothing was written, retry: %w",
			filepath.Base(path), filepath.Base(lock), rewriteLockTimeout, err)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("the lock %s beside %s is a symlink or not a regular file, so it is refused and nothing was written; remove it: %w",
			filepath.Base(lock), filepath.Base(path), err)
	}
	return err
}

// withConfigLock is withRewriteLock for .abcd/config.json. The lock sits in
// .abcd/, which a first install has not made yet, so the directory is created
// first — contained, through an os.Root at the repo, the way the config write
// itself creates it (GHSA-xrf8-4432-gw2f).
func withConfigLock(cwd string, fn func() error) error {
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return err
	}
	err = root.MkdirAll(filepath.Dir(configRelPath), 0o755)
	root.Close()
	if err != nil {
		return err
	}
	return withRewriteLock(configPath(cwd), fn)
}
