package capture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ledgerroot.go is the ledger's containment scope (iss-2609012037143368). Every
// ledger write — a directory, the allocator lock, a placeholder, a record, a
// removal — resolves its path INSIDE an os.Root opened on the checkout, never
// as an absolute path. The per-segment Lstat walk (ledgerDirs) still refuses a
// committed symlink by name; the root is what closes the window between that
// check and the write it licenses, because a directory swapped for a symlink in
// that window cannot carry the write out of the root.

// ledgerBase is the directory a ledger's writes are contained in: the checkout
// when the ledger sits inside it (every front door), otherwise the ledger's own
// parent, for an operator-typed ledger operand that has no checkout around it.
func ledgerBase(repoRoot, issuesRoot string) string {
	if fsutil.PathWithin(issuesRoot, repoRoot, false) {
		return repoRoot
	}
	return filepath.Dir(issuesRoot)
}

// containedRel maps abs to the slash path an os.Root at base resolves, refusing
// a path that is not under base.
func containedRel(base, abs string) (string, error) {
	rel, err := filepath.Rel(base, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: %s is outside the ledger's containment root", ErrPathUnsafe, abs)
	}
	return filepath.ToSlash(rel), nil
}

// withContainedRoot opens an os.Root on base and runs fn with it and abs's path
// relative to it.
func withContainedRoot(base, abs string, fn func(root *os.Root, rel string) error) error {
	rel, err := containedRel(base, abs)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return err
	}
	defer root.Close()
	return mapEscape(fn(root, rel), abs)
}

// mapEscape reports an os.Root refusal as the ledger's own path-unsafe
// sentinel, so every caller that already tests ErrPathUnsafe sees one. The os
// package does not export its escape error, so its message is matched.
func mapEscape(err error, abs string) error {
	if err == nil || errors.Is(err, ErrPathUnsafe) || !strings.Contains(err.Error(), "path escapes from parent") {
		return err
	}
	return fmt.Errorf("%w: %s resolves outside the ledger's containment root: %v", ErrPathUnsafe, abs, err)
}

// writeContained writes data to abs atomically inside an os.Root on base,
// keeping the file's existing mode.
func writeContained(base, abs string, data []byte) error {
	return withContainedRoot(base, abs, func(root *os.Root, rel string) error {
		return fsutil.WriteFileAtomicPreserveModeInRoot(root, rel, data)
	})
}

// removeContained removes abs inside an os.Root on base; an absent file is not
// an error, matching os.Remove as its callers used it.
func removeContained(base, abs string) error {
	return withContainedRoot(base, abs, func(root *os.Root, rel string) error {
		if err := root.Remove(rel); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	})
}

// writeLedgerFile is writeContained for a path in this ledger.
func writeLedgerFile(repoRoot, issuesRoot, abs string, data []byte) error {
	return writeContained(ledgerBase(repoRoot, issuesRoot), abs, data)
}

// removeLedgerFile is removeContained for a path in this ledger.
func removeLedgerFile(repoRoot, issuesRoot, abs string) error {
	return removeContained(ledgerBase(repoRoot, issuesRoot), abs)
}
