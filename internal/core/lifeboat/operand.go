package lifeboat

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// proveOperand proves an operand directory — the lifeboat a verb reads or
// writes into, the embark target, the pack destination — against a symlinked
// ANCESTOR, which the leaf-only IsRealDir each gate applies cannot see
// (iss-2609261232464351).
//
// The operand's declared base is the checkout it sits in: a checkout is the one
// place an untrusted commit can plant a link (a committed symlink beats
// .gitignore), so every level from that checkout's root down to the operand is
// proved a real directory with fsutil.ProbeRealDirAll, the read-only walk the
// local tier and the inbox use. A missing level ends the proof: nothing below
// it exists to be redirected, and a destination the verb creates starts there.
//
// The checkout root's own entry lives in the directory above it, so it is
// proved by the next round rather than this one: the walk moves out to the
// checkout enclosing that root, if any, and proves the chain down to it the
// same way. That is what refuses a committed link pointing INTO another
// checkout, which a proof stopping at the nearest .git would take as the base.
//
// Outside every checkout the path is the operator's own, taken as given under
// the trusted-worktree model: refusing there would refuse macOS's /var and /tmp
// links and every temporary directory, and protect nothing. The checkout is
// found with the .git marker walk (gitutil.RepoShapedRoot) rather than git's
// answer, because a marker read too widely only makes the proof stricter.
//
// The refusal wraps fsutil.ErrNotRealDir and names the refused level relative
// to its checkout, never an absolute path.
func proveOperand(role, abs string) error {
	target := filepath.Clean(abs)
	start := target
	for {
		base := gitutil.RepoShapedRoot(start)
		if base == "" {
			return nil
		}
		if base != target {
			rel, err := filepath.Rel(base, target)
			if err != nil {
				return err
			}
			if _, err := fsutil.ProbeRealDirAll(fsutil.RealExistingPath(base), filepath.ToSlash(rel)); err != nil {
				return operandRefusal(role, abs, base, err)
			}
		}
		parent := filepath.Dir(base)
		if parent == base {
			return nil
		}
		target, start = base, parent
	}
}

// operandRefusal phrases a failed proof. The refused level is named relative to
// the checkout that holds it, and the checkout by its own name.
func operandRefusal(role, abs, base string, err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) && errors.Is(err, fsutil.ErrNotRealDir) {
		return fmt.Errorf("%s %s reaches its directory through %s inside the checkout %s, which is not a real directory (a symlink or a file); refusing to follow it: %w",
			role, filepath.Base(abs), fsutil.DisplayPath(fsutil.RealExistingPath(base), pe.Path), filepath.Base(base), fsutil.ErrNotRealDir)
	}
	return fmt.Errorf("%s %s cannot be proved a real directory inside the checkout %s: %w", role, filepath.Base(abs), filepath.Base(base), err)
}
