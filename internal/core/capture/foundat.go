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
// whether it names one at all, by its shape alone. It is a path when, with any
// locator suffix removed, it is a single path token that either contains a
// separator or ends in a file extension. Absolute and home-relative values are
// not repo-relative and are left alone, as is anything carrying a URL scheme.
// foundAtPathIn also asks the checkout, for a lone name with neither.
func foundAtPath(value string) (string, bool) {
	v, ok := foundAtToken(value)
	if !ok || (!strings.Contains(v, "/") && !foundAtExtension.MatchString(v)) {
		return "", false
	}
	return v, true
}

// foundAtToken is a found_at value trimmed and with any locator suffix
// dropped, and whether what is left is one repo-relative path token: not
// empty, not absolute, carrying no URL scheme.
func foundAtToken(value string) (string, bool) {
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
	return v, true
}

// foundAtPathIn is foundAtPath judged against the checkout at repoRoot
// (iss-2611190625535337): a lone name with no separator and no extension,
// such as Makefile, LICENSE or Dockerfile, names a path too when a regular
// file of exactly that name is at the checkout's root. A word naming no file
// there, a directory, and a name differing from the file only in case (which
// a case-insensitive filesystem would otherwise stat) stay a place in words,
// as does every lone name when the root cannot be read. This is the one
// verdict capture's check, the drain's ordering and every reservation read.
func foundAtPathIn(repoRoot, value string) (string, bool) {
	if p, ok := foundAtPath(value); ok {
		return p, true
	}
	v, ok := foundAtToken(value)
	if !ok || strings.Contains(v, "/") || !rootFile(repoRoot, v) {
		return "", false
	}
	return v, true
}

// rootFile reports whether a regular file named exactly name is an entry of
// the directory repoRoot. The entries are listed rather than the name stated,
// so the match is exact on a case-insensitive filesystem too.
func rootFile(repoRoot, name string) bool {
	if repoRoot == "" {
		return false
	}
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() == name {
			return e.Type().IsRegular()
		}
	}
	return false
}

// FoundAtPath is the repo-relative path a found_at value names by its shape,
// with any locator suffix dropped, and whether it names one. A caller with a
// checkout to hand asks FoundAtPathIn, which also knows a lone root file.
func FoundAtPath(value string) (string, bool) { return foundAtPath(value) }

// FoundAtPathIn is the repo-relative path a found_at value names in the
// checkout at repoRoot, a lone name included when it is a file at the root,
// and whether it names one: the file the build and the drain reserve an issue
// with.
func FoundAtPathIn(repoRoot, value string) (string, bool) { return foundAtPathIn(repoRoot, value) }

// checkFoundAt refuses a found_at that names a repo-relative path which does
// not resolve in the checkout being written to (iss-2609120511058115). The
// ledger records findings about the repository it lives in; a path that is not
// in that repository is the mechanical sign of a finding filed in the wrong
// place, and it is the one thing about "which repository is this about" that a
// gate can check without guessing. A conceptual location, and an absent value,
// are written as given. The check runs at capture time only: a record already
// in the ledger keeps naming the path it named when the tree moves on.
//
// A path that leaves the checkout or does not exist in it is the caller's input
// with nothing written, so it is refused (ErrRequestRefused, exit 2 at the
// surface) like any other flag value that names nothing (iss-2609261241119343).
// A path the check could not stat for any other reason — a directory on the way
// that cannot be searched, an I/O error — says nothing about the input: the
// checkout could not be read, so that one stays a fault.
func checkFoundAt(repoRoot, value string) error {
	rel, ok := foundAtPathIn(repoRoot, value)
	if !ok {
		return nil
	}
	given := strings.TrimSpace(value)
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return refused(fmt.Errorf("capture: found_at %q leaves this checkout; a finding is filed in the ledger of the repository it is about (nothing written)", given))
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, clean)); err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			names := ""
			if given != rel {
				names = " (the path " + rel + ")"
			}
			return refused(fmt.Errorf("capture: found_at %q%s does not exist in this checkout; a finding is filed in the ledger of the repository it is about — correct the path, or give a conceptual location in words (nothing written)", given, names))
		}
		// The stat error is not echoed: it carries the absolute path, and a
		// refusal names the repo-relative locator only (iss-81).
		return fmt.Errorf("capture: found_at %q could not be checked against this checkout (nothing written)", given)
	}
	return nil
}
