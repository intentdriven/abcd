package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// maxRunDirSuffix bounds the same-instant collision suffix: name, name-001 …
// name-999. A thousand runs in one timestamp is a runaway loop, not a workload.
const maxRunDirSuffix = 999

// CreateRunDir creates a fresh directory for one run's log under rel below base
// and returns its path. It is the local tier's run-log create path: every level
// from base down through rel is proved a real directory first (EnsureRealDirAll,
// creating what is missing), so a tier a checkout carries as a committed symlink
// — at any level — is refused rather than followed out of the checkout. The run
// directory itself is then created exclusively, never reused: name when it is
// free, else name-001, name-002 and so on, so two runs in one instant keep two
// logs.
//
// The directory is returned as a path because callers report it; a caller that
// writes into it opens it with OpenRealDir, which re-proves the leaf and pins the
// writes to the directory handle, so a level swapped for a link after this proof
// cannot carry them elsewhere.
func CreateRunDir(base, rel, name string, perm os.FileMode) (string, error) {
	if err := EnsureRealDirAll(base, rel, perm); err != nil {
		return "", err
	}
	if !ValidRelPath(name) || filepath.Base(name) != name {
		return "", &os.PathError{Op: "createrundir", Path: name, Err: os.ErrInvalid}
	}
	parent := filepath.Join(base, filepath.FromSlash(rel))
	for n := 0; n <= maxRunDirSuffix; n++ {
		candidate := name
		if n > 0 {
			candidate = fmt.Sprintf("%s-%03d", name, n)
		}
		dir := filepath.Join(parent, candidate)
		err := os.Mkdir(dir, perm)
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("fsutil: no free run directory for %s after %d attempts", name, maxRunDirSuffix+1)
}

// OpenRealDir opens dir as an os.Root only when it is a real directory rather
// than a symlink or a non-directory, so every write through the handle lands in
// the directory that was proved and nowhere its path might name later. The
// caller closes the root.
func OpenRealDir(dir string) (*os.Root, error) {
	if !IsRealDir(dir) {
		return nil, &os.PathError{Op: "openrealdir", Path: dir, Err: ErrNotRealDir}
	}
	return os.OpenRoot(dir)
}
