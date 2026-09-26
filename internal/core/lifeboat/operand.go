package lifeboat

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// proveOperand proves an operand directory — the lifeboat a verb reads or
// writes into, the embark target, the pack destination — against a symlinked
// ANCESTOR inside a checkout, which the leaf-only IsRealDir each gate applies
// cannot see (iss-2609261232464351). The rule and its bound (outside every
// checkout the path is taken as given) are gitutil.ProveOperandDir's; this is
// the lifeboat's phrasing of its refusal, which wraps fsutil.ErrNotRealDir and
// names the refused level relative to its checkout, never an absolute path.
func proveOperand(role, abs string) error {
	err := gitutil.ProveOperandDir(abs)
	if err == nil {
		return nil
	}
	var oe *gitutil.OperandError
	if !errors.As(err, &oe) {
		return fmt.Errorf("%s %s: %w", role, filepath.Base(abs), err)
	}
	return fmt.Errorf("%s %s reaches its directory through %s inside the checkout %s, which is not a real directory; refusing to follow it: %w",
		role, filepath.Base(abs), oe.Level, filepath.Base(oe.Checkout), oe.Err)
}
