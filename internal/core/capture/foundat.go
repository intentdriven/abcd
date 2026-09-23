package capture

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// foundAtPathToken is the shape of a value that names a repo-relative path: one
// token of path characters, with no whitespace, list separator or bracket. A
// value outside it is a conceptual location ("git history", "internal (capture,
// lint)", a list of paths) and is never judged against the tree.
var foundAtPathToken = regexp.MustCompile(`^[A-Za-z0-9._@+/-]+$`)

// foundAtExtension matches a final path segment that carries a file extension
// beginning with a letter, so "README.md" and ".github" are path-shaped while a
// version string such as "v0.9.0" is not.
var foundAtExtension = regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9_-]*$`)

// foundAtLocator matches the source-locator suffix a path may carry: a line
// (":103"), a range (":66-120"), a list (":72,92") or a symbol
// (":scanAllPatterns").
var foundAtLocator = regexp.MustCompile(`:[A-Za-z0-9_,-]+$`)

// foundAtPath reports the repo-relative path a found_at value names, and
// whether it names one at all. It is a path when, with any locator suffix
// removed, it is a single path token that either contains a separator or ends
// in a file extension. Absolute and home-relative values are not repo-relative
// and are left alone, as is anything carrying a URL scheme.
func foundAtPath(value string) (string, bool) {
	v := strings.TrimSpace(value)
	if v == "" || strings.Contains(v, "://") {
		return "", false
	}
	if loc := foundAtLocator.FindStringIndex(v); loc != nil {
		v = v[:loc[0]]
	}
	if v == "" || strings.HasPrefix(v, "/") || !foundAtPathToken.MatchString(v) {
		return "", false
	}
	if !strings.Contains(v, "/") && !foundAtExtension.MatchString(v) {
		return "", false
	}
	return v, true
}

// checkFoundAt refuses a found_at that names a repo-relative path which does
// not resolve in the checkout being written to (iss-2609120511058115). The
// ledger records findings about the repository it lives in; a path that is not
// in that repository is the mechanical sign of a finding filed in the wrong
// place, and it is the one thing about "which repository is this about" that a
// gate can check without guessing. A conceptual location, and an absent value,
// are written as given. The check runs at capture time only: a record already
// in the ledger keeps naming the path it named when the tree moves on.
func checkFoundAt(repoRoot, value string) error {
	rel, ok := foundAtPath(value)
	if !ok {
		return nil
	}
	given := strings.TrimSpace(value)
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("capture: found_at %q leaves this checkout; a finding is filed in the ledger of the repository it is about (nothing written)", given)
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, clean)); err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return fmt.Errorf("capture: found_at %q names %s, which does not exist in this checkout; a finding is filed in the ledger of the repository it is about — correct the path, or give a conceptual location in words (nothing written)", given, rel)
		}
		// The stat error is not echoed: it carries the absolute path, and a
		// refusal names the repo-relative locator only (iss-81).
		return fmt.Errorf("capture: found_at %q could not be checked against this checkout (nothing written)", given)
	}
	return nil
}
