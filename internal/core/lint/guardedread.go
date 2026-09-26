package lint

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxRepoFileBytes caps a single file the lint reads out of the repository by
// a path its own config names or its own walk found: a registry, a snapshot, a
// runbook, a workflow, a record. It is the citation page cap, which is many
// times the largest committed file of any of those shapes, so it bounds a
// `-> /dev/zero` link without ever constraining a real one.
const maxRepoFileBytes = citationPageSizeLimit

// readRepoFile reads a repo-relative path that the lint config names. The config
// is committed, so a cloned repository controls both the path and the file it
// names, and CI's record-lint is what reads them: the path is refused if it is
// absolute or climbs out of the tree, the resolved leaf is refused if a symlink
// carries it outside the repository, and the read itself is fsutil.ReadGuarded,
// so a FIFO does not hang the gate and an endless device is not read unbounded.
// A missing file keeps its os.IsNotExist error, so a caller that treats absence
// as a state still can (iss-2608211914592726).
func readRepoFile(repoRoot, rel string, limit int64) ([]byte, error) {
	if err := containedRepoPath(rel); err != nil {
		return nil, &configError{quote(rel) + " " + err.Error() + "; the lint reads only inside the repository"}
	}
	return readRepoAbs(repoRoot, filepath.Join(repoRoot, filepath.FromSlash(rel)), limit)
}

// readRepoAbs is readRepoFile for a leaf a directory walk found, whose path is
// already under repoRoot lexically: the containment that is left to check is the
// symlink one, exactly as the roots walk checks its own leaves. An in-repo link
// (a bridge file) still reads, through the resolved path containment judged.
func readRepoAbs(repoRoot, abs string, limit int64) ([]byte, error) {
	realPath, err := containedRealPath(repoRoot, abs)
	if err != nil {
		return nil, &configError{"file " + quote(repoRel(repoRoot, abs)) + " " + err.Error() +
			"; the lint reads only inside the repository"}
	}
	return fsutil.ReadGuarded(realPath, limit)
}

// readRepoLeaf is readRepoFile for a configured file that is never legitimately
// a link — a baseline, an exemption list — so the leaf is refused as a link
// rather than resolved, and every ancestor is resolved inside the repository on
// the descriptor that is read (fsutil.ReadGuardedInRoot through an os.Root), so
// a directory linked out of the tree cannot carry the read with it. The
// directory is also judged up front, so the refusal names the repository rather
// than the os.Root error. A missing file keeps its os.IsNotExist error.
func readRepoLeaf(repoRoot, rel string, limit int64) ([]byte, error) {
	if err := containedRepoPath(rel); err != nil {
		return nil, &configError{quote(rel) + " " + err.Error() + "; the lint reads only inside the repository"}
	}
	native := filepath.FromSlash(rel)
	if err := resolvedInsideRoot(repoRoot, filepath.Join(repoRoot, filepath.Dir(native))); err != nil {
		return nil, &configError{quote(rel) + " " + err.Error() + "; the lint reads only inside the repository"}
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return fsutil.ReadGuardedInRoot(root, native, limit)
}

// maxReceiptBytes caps a semantic-pass receipt and the release-gate manifest.
// Both are small JSON documents a tool writes; the cap is thousands of times the
// largest committed one.
const maxReceiptBytes = 4 << 20

// guardedReason renders why a guarded read refused a file, for a finding that
// names it rather than an error that aborts the whole lint.
func guardedReason(err error) string {
	switch {
	case errors.Is(err, fsutil.ErrTooBig):
		return "larger than the read cap"
	case errors.Is(err, fsutil.ErrNotRegular), errors.Is(err, syscall.ELOOP):
		return "not a regular file: a symlink, FIFO, device or directory"
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	}
	return "unreadable: " + bareCause(err)
}
