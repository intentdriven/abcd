package gitutil

import (
	"errors"
	"fmt"
	"strings"
)

// RefIsSafe reports whether a ref is safe to hand git as a POSITIONAL argument:
// non-empty and not option-shaped. A ref beginning with '-' parses as a flag —
// `--output=<path>` on a diff writes a file, `--upload-pack=<cmd>` on a fetch
// runs one — and no legitimate ref name starts with one, so refusing them drops
// only hostile or mistyped input. Every caller that interpolates a ref it did
// not write itself (a CI base sha, a user-named branch, a ref read out of a
// repository) checks it here first.
func RefIsSafe(ref string) bool {
	return ref != "" && !strings.HasPrefix(ref, "-")
}

// IsNullOID reports whether s is git's all-zeroes object name, under SHA-1 or
// SHA-256. A forge hands it to a range gate when a push has no predecessor to
// name — a branch the push created, or a force-push whose old tip it declines
// to cite. It is well formed and resolves to nothing.
func IsNullOID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	return strings.Trim(s, "0") == ""
}

// ResolveRangeBase is the one derivation of the base a range-scoped gate checks
// from. raw is whatever the caller was handed — a CI event's base sha, or a ref
// a developer named — and there are three outcomes:
//
//   - raw is empty (after trimming) or the null object name: there is no usable
//     base for this event. It returns ("", false, nil), and the caller SAYS it
//     skipped; it never reads the empty range as a clean one.
//   - raw resolves to a commit: it returns that commit's full object name.
//   - anything else — an option-shaped value, a name no commit answers to, a
//     repository git cannot read — is an error. A base a gate cannot resolve is
//     one it cannot judge from, and the error is the "refusing rather than
//     reporting a vacuous pass" answer, never a quiet downgrade.
//
// It exists so the zero-sha and absent-commit guard lives once, in Go, rather
// than hand-copied into every workflow step that runs a range gate.
func ResolveRangeBase(root, raw string) (string, bool, error) {
	ref := strings.TrimSpace(raw)
	if ref == "" || IsNullOID(ref) {
		return "", false, nil
	}
	sha, err := ResolveCommit(root, ref)
	if err != nil {
		return "", false, err
	}
	return sha, true, nil
}

// ResolveCommit resolves ref to the full object name of the commit it names,
// refusing an option-shaped ref before git sees it.
func ResolveCommit(root, ref string) (string, error) {
	if !RefIsSafe(ref) {
		return "", fmt.Errorf("refusing ref %q: an empty or option-shaped ref would reach git as a flag", ref)
	}
	sha, err := Run(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil || sha == "" {
		return "", fmt.Errorf("%q names no commit in this repository", ref)
	}
	return sha, nil
}

// ErrShallowCheckout is RequireFullHistory's refusal: the checkout is shallow,
// so history past the graft is absent.
var ErrShallowCheckout = errors.New("shallow checkout")

// RequireFullHistory refuses a shallow checkout. A gate that compares commits
// with their parents is blinded by one: past the graft a commit's parent is not
// present, so every change reads as a whole-file add and the check covers
// nothing. The probe is itself fail-closed — a git that cannot say whether the
// checkout is shallow is an error, never an assumed "full".
func RequireFullHistory(root string) error {
	out, err := Run(root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return fmt.Errorf("whether the checkout is shallow could not be read: %w", err)
	}
	switch out {
	case "false":
		return nil
	case "true":
		return fmt.Errorf("%w: history past the graft is absent — run 'git fetch --unshallow' first (CI checks out with fetch-depth: 0)", ErrShallowCheckout)
	default:
		return fmt.Errorf("unexpected 'git rev-parse --is-shallow-repository' output %q; refusing rather than guessing", out)
	}
}

// RunCappedBytes is RunCapped without the trim: git's stdout exactly as
// written, or an error when it exceeds maxBytes. It is for a caller that reads
// FILE CONTENT through git — a blob's trailing blank lines and its last line's
// trailing whitespace are content, and a trimmed blob has a different line count
// from the committed one.
func RunCappedBytes(root string, maxBytes int, args ...string) ([]byte, error) {
	out, overflowed, err := runBoundedBytes(root, maxBytes, args...)
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated, and a truncated answer is a wrong one",
			strings.Join(args, " "), maxBytes)
	}
	return out, nil
}
