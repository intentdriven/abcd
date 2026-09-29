package ahoy

import (
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// displayPath renders p for any surface a human reads or pastes: a path inside
// the user's home directory becomes its tilde form (~/.local/bin/abcd), so a gap
// line or an uninstall receipt carries the location without carrying the
// username. Everything outside home is unchanged — a system path is not
// sensitive and shortening it would lose information.
//
// This is the same hygiene iss-177 applies inside the install receipt's note
// seam, and it is now the SAME primitive: displayPath routes through
// fsutil.RedactHome rather than keeping a bespoke home-prefix compare. Routing
// through the canonical redactor also folds the comparison on a case-folding
// filesystem, so a case-variant HOME spelling no longer leaks the username in a
// doctor gap or an uninstall receipt (iss-2608270908341622).
func displayPath(p string) string {
	return fsutil.RedactHome(p)
}

// displayText renders arbitrary text — an OS error string, which embeds whatever
// absolute path the syscall was given — for the same surfaces displayPath serves.
// The home prefix is replaced wherever it appears at a path boundary, because an
// error reads "symlink /a/b /home/alice/.local/bin/abcd: permission denied" and the
// username sits in the middle. A path that merely starts with the home string but
// continues into another name (/home/user2 against /home/user) is left alone. abcd-lint:allow
//
// It is the same canonical redactor displayPath uses (fsutil.RedactHome), so the
// two never disagree about what home looks like and both fold case on a
// case-folding filesystem (iss-2608270908341622).
func displayText(s string) string {
	return fsutil.RedactHome(s)
}

// errText renders an error for a human-facing note with the same hygiene.
func errText(err error) string {
	if err == nil {
		return ""
	}
	return displayText(err.Error())
}

// modeSymlink aliases os.ModeSymlink so detection reads naturally.
const modeSymlink = os.ModeSymlink

// fileExists reports whether path exists as a regular file.
func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

func lstat(p string) (os.FileInfo, error) { return os.Lstat(p) }

func isNotExist(err error) bool { return os.IsNotExist(err) }

func readlink(p string) (string, error) { return os.Readlink(p) }

// resolvePath returns a canonical absolute form of p for symlink-target
// comparison, tolerating non-existent paths.
func resolvePath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

// resolveSymlinkDest canonicalises a symlink's target for comparison. A RELATIVE
// target is interpreted relative to the symlink's OWN directory — the way the
// kernel resolves it — not the process working directory. Feeding a relative
// readlink result straight to resolvePath resolved it against the CWD, so a
// correct relative link (e.g. /usr/local/bin/abcd -> ../lib/abcd/abcd) was
// misclassified as foreign from any other directory, producing a bogus gap and
// making uninstall refuse to remove a link it owns.
func resolveSymlinkDest(symlinkPath, dest string) string {
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(filepath.Dir(symlinkPath), dest)
	}
	return resolvePath(dest)
}
