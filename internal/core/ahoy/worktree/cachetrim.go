package worktree

// cachetrim.go trims the Go build cache when prune has removed a worktree
// (iss-2610290717018523). Go trims only entries unused for five days, and every
// store worktree, race build and pinned-toolchain build adds entries, so a
// machine running many worktrees filled its disk twice, and the load gate
// (implement/load_disk.go) then stood everything still until a person trimmed by
// hand. This deletes another tool's files, so it touches only what Go's own
// layout defines, in the one directory Go itself names. A build running beside
// it is safe: go treats a missing entry as a miss and rebuilds, so each file is
// unlinked plainly and no directory is removed.

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// CacheTrimAge is how long an entry goes unused before the trim deletes it.
const CacheTrimAge = 48 * time.Hour

// goCacheUseInterval is how stale Go lets an entry's modification time grow
// before a use refreshes it (mtimeInterval in cmd/go/internal/cache). An entry
// may have been used this long after its time says, so the trim allows for it,
// as Go's own trim does.
const goCacheUseInterval = time.Hour

// openCacheRoot and rootOwnerOf are how the trim opens the cache and reads the
// opened directory's owner. They are variables only so a test can stand in for a
// path that opens another directory and for another account's directory, which
// it cannot make without root.
var (
	openCacheRoot = os.OpenRoot
	rootOwnerOf   = fsutil.InfoOwnerUID
)

// CacheTrim is what one trim did. Under a dry run the counts are what a real run
// would have removed.
type CacheTrim struct {
	// Path is the cache's directory, home-redacted and masked; "" when go named
	// none.
	Path string `json:"path,omitempty"`
	// Skipped says why nothing was looked at; "" when the cache was walked.
	Skipped string `json:"skipped,omitempty"`
	// UnusedDays is the age an entry had to pass.
	UnusedDays     int    `json:"unused_days"`
	EntriesRemoved int    `json:"entries_removed"`
	BytesFreed     uint64 `json:"bytes_freed"`
	// Failed counts the entries that could not be removed and the
	// subdirectories that could not be read.
	Failed int `json:"failed,omitempty"`
}

// TrimGoCache deletes the entries of the cache `go env GOCACHE` names that have
// gone unused for CacheTrimAge, as of now, or under a dry run only counts them.
// It reports rather than returns a failure: a trim never fails the prune that
// asked for it.
func TrimGoCache(dryRun bool, now time.Time) CacheTrim {
	out := CacheTrim{UnusedDays: int(CacheTrimAge / (24 * time.Hour))}
	dir, skip := goCachePath()
	if skip != "" {
		out.Skipped = skip
		return out
	}
	shown := shownPath(dir)
	out.Path = shown
	checked, skip := cachePathInfo(dir, shown)
	if skip != "" {
		out.Skipped = skip
		return out
	}
	// os.Root keeps every open, stat and unlink below inside the cache.
	root, err := openCacheRoot(dir)
	if err != nil {
		out.Skipped = shown + " could not be opened: " + shownPath(causeOf(err))
		return out
	}
	defer root.Close()
	if skip := unusableRoot(root, checked, shown); skip != "" {
		out.Skipped = skip
		return out
	}
	subdirs, err := readRootDir(root, ".")
	if err != nil {
		out.Skipped = shown + " could not be read: " + shownPath(causeOf(err))
		return out
	}
	cutoff := now.Add(-CacheTrimAge - goCacheUseInterval)
	for _, sub := range subdirs {
		if sub.IsDir() && isCacheSubdir(sub.Name()) {
			trimCacheSubdir(root, sub.Name(), cutoff, dryRun, &out)
		}
	}
	return out
}

// goCachePath is the directory `go env GOCACHE` reports, or why there is none to
// trim. It never guesses: no go command, no trim.
func goCachePath() (dir, skip string) {
	got, err := machineload.GoEnvCache()
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return "", "no go command to say where the cache is"
	case err != nil:
		return "", "`go env GOCACHE` failed: " + shownPath(causeOf(err))
	case got == "" || got == "off":
		return "", "`go env GOCACHE` names no cache"
	case !filepath.IsAbs(got):
		return "", "`go env GOCACHE` names a relative path: " + shownPath(got)
	}
	return filepath.Clean(got), ""
}

// cachePathInfo reads what the path dir names, without following a link, for
// unusableRoot to compare with what opens; skip says why it is not trimmed,
// when it is no real directory or is the home or above it. The home is compared
// through links, and one that cannot be read is a skip.
func cachePathInfo(dir, shown string) (checked fs.FileInfo, skip string) {
	fi, err := os.Lstat(dir)
	switch {
	case err != nil:
		return nil, shown + " is not an existing directory"
	case fi.Mode()&fs.ModeSymlink != 0:
		return nil, shown + " is a symbolic link"
	case !fi.IsDir():
		return nil, shown + " is not a directory"
	}
	home, err := os.UserHomeDir()
	switch {
	case err != nil || home == "":
		return nil, "the home directory could not be read, so " + shown + " cannot be told from it"
	case machineload.HoldsHome(fsutil.RealExistingPath(dir), fsutil.RealExistingPath(home)):
		return nil, shown + " is the home directory or holds it"
	}
	return fi, ""
}

// unusableRoot is why the opened root is not trimmed, or "" when it is a
// directory, the same one the path named when checked, the caller's own, and of
// a Go build cache's shape: the load gate's guards, read from the opened root
// through root.Stat and root.FS. An owner that cannot be read is a skip.
func unusableRoot(root *os.Root, checked fs.FileInfo, shown string) string {
	fi, err := root.Stat(".")
	if err != nil {
		return shown + " could not be read: " + shownPath(causeOf(err))
	}
	if !fi.IsDir() || !os.SameFile(checked, fi) {
		return shown + " did not open as the directory it names"
	}
	uid, err := rootOwnerOf(fi)
	switch {
	case err != nil:
		return "the owner of " + shown + " could not be read"
	case uid != uint32(os.Getuid()):
		return shown + " belongs to another account"
	case !machineload.LooksLikeGoCacheFS(root.FS()):
		return shown + " does not have a Go build cache's shape"
	}
	return ""
}

// trimCacheSubdir removes the old entries of one subdirectory. An entry another
// process removed first is no failure and no removal of this one.
func trimCacheSubdir(root *os.Root, sub string, cutoff time.Time, dryRun bool, out *CacheTrim) {
	files, err := readRootDir(root, sub)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			out.Failed++
		}
		return
	}
	for _, f := range files {
		if !f.Type().IsRegular() || !isCacheEntry(f.Name()) {
			continue
		}
		rel := filepath.Join(sub, f.Name())
		fi, err := root.Lstat(rel)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				out.Failed++
			}
			continue
		}
		if !fi.Mode().IsRegular() || !fi.ModTime().Before(cutoff) {
			continue
		}
		if !dryRun {
			if err := root.Remove(rel); err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					out.Failed++
				}
				continue
			}
		}
		out.EntriesRemoved++
		out.BytesFreed += uint64(fi.Size())
	}
}

// readRootDir lists the directory name in root.
func readRootDir(root *os.Root, name string) ([]fs.DirEntry, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}

// isCacheSubdir reports whether name is two lowercase hex digits, the name of
// one of the cache's subdirectories.
func isCacheSubdir(name string) bool { return len(name) == 2 && isLowerHex(name) }

// isCacheEntry reports whether name is <hex>-a or <hex>-d, the cache's two kinds
// of entry.
func isCacheEntry(name string) bool {
	hex, ok := strings.CutSuffix(name, "-a")
	if !ok {
		hex, ok = strings.CutSuffix(name, "-d")
	}
	return ok && isLowerHex(hex)
}

// isLowerHex reports whether s is non-empty and only lowercase hex digits.
func isLowerHex(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}

// shownPath is a path as the answer names it: go's answer is another tool's
// string, so it is masked as well as home-redacted.
func shownPath(p string) string { return termsafe.Sanitize(fsutil.RedactHome(p)) }
