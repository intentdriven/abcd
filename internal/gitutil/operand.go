package gitutil

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// OperandError is ProveOperandDir's refusal: Level, a directory inside the
// checkout rooted at Checkout and named relative to it, could not be proved a
// real directory. Err is the cause with no path in it (fsutil.ErrNotRealDir for
// a symlink or a non-directory), so a caller tests the class with errors.Is and
// phrases the message itself; Checkout is absolute and is for the caller to
// shorten, never to print whole.
type OperandError struct {
	Checkout string
	Level    string
	Err      error
}

func (e *OperandError) Error() string {
	return "operand level " + e.Level + " inside the checkout " + filepath.Base(e.Checkout) + ": " + e.Err.Error()
}

func (e *OperandError) Unwrap() error { return e.Err }

// ProveOperandDir proves an operand directory a person named on a command line
// — a lifeboat, an embark target, a pack destination, an --out directory —
// against a symlinked ANCESTOR, which a leaf-only fsutil.IsRealDir cannot see.
//
// The operand's declared base is the checkout it sits in: a checkout is the one
// place an untrusted commit can plant a link (a committed symlink beats
// .gitignore), so every level from that checkout's root down to the operand is
// proved a real directory with fsutil.ProbeRealDirAll, the read-only walk the
// local tier and the inbox use. A missing level ends the proof: nothing below it
// exists to be redirected, and a destination the verb creates starts there.
//
// The checkout root's own entry lives in the directory above it, so it is
// proved by the next round rather than this one: the proof moves out to the
// checkout enclosing that root, if any, and proves the chain down to it the same
// way. That is what refuses a committed link pointing INTO another checkout,
// which a proof stopping at the nearest .git would take as its base.
//
// Outside every checkout the path is the operator's own, taken as given under
// the trusted-worktree model: refusing there would refuse macOS's /var and /tmp
// links and every temporary directory, and protect nothing. The checkout is
// found with the .git marker walk (RepoShapedRoot) rather than git's answer:
// a marker read too widely only makes the proof stricter.
//
// A refusal is an *OperandError whose Level is relative to its Checkout, which
// is the checkout's absolute path; a caller reporting it names the checkout by
// its base name, never the absolute path.
//
// The operand is absolutised against the working directory first, whatever
// the caller passed: a relative path's marker walk ends at "." and never
// reaches the checkout above it, which would accept a link the proof exists to
// refuse, so the contract is held here rather than trusted to every caller
// (iss-2609262235543552).
func ProveOperandDir(operand string) error {
	abs, err := filepath.Abs(operand)
	if err != nil {
		return err
	}
	target := abs
	start := target
	for {
		base := RepoShapedRoot(start)
		if base == "" {
			return nil
		}
		if base != target {
			rel, err := filepath.Rel(base, target)
			if err != nil {
				return err
			}
			realBase := fsutil.RealExistingPath(base)
			if _, err := fsutil.ProbeRealDirAll(realBase, filepath.ToSlash(rel)); err != nil {
				// The proof's *os.PathError names the level absolutely; the
				// refusal carries it relative to the checkout and keeps only the
				// cause, so no caller can leak the absolute path by wrapping it.
				level, cause := filepath.ToSlash(rel), err
				var pe *os.PathError
				if errors.As(err, &pe) {
					if filepath.IsAbs(pe.Path) {
						level = fsutil.DisplayPath(realBase, pe.Path)
					}
					cause = pe.Err
				}
				return &OperandError{Checkout: base, Level: level, Err: cause}
			}
		}
		parent := filepath.Dir(base)
		if parent == base {
			return nil
		}
		target, start = base, parent
	}
}
