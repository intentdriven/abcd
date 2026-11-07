package worktree

// cachetrim.go keeps the Go build cache under a size limit, oldest entries first
// (iss-2611041338455722). Go trims only entries unused for five days, and every
// store worktree, race build and pinned-toolchain build adds entries, so a
// machine running many worktrees filled its disk twice, and the load gate
// (implement/load_disk.go) then stood everything still until a person trimmed by
// hand. An age does not bound it: on 6 Nov 2026 a cache held 324 GB in 951,386
// entries, all used within five days, and two days' age would have freed 87 GB.
// This deletes another tool's files, so it touches only what Go's own layout
// defines, in the one directory Go itself names. A build running beside it is
// safe, but for one narrow window: go treats an entry missing at lookup as a miss
// and rebuilds, each entry's time is read again just before it is unlinked so one
// a build used during the walk stays, and only a build looking an entry up in the
// instant between that read and the unlink can fail to open it (removeOldest).
// Each file is unlinked plainly and no directory is removed. Sizes are judged by the cache
// itself and never by free space, since a local snapshot can hold what a trim
// freed: on 7 Nov 2026 a trim of 263 GB moved the free disk by nothing.

import (
	"cmp"
	"errors"
	"fmt"
	"github.com/intentdriven/abcd/internal/abcdhome"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// CacheTrimKeep is how recently used an entry must be for the trim to keep it
// whatever the limit, so a running build keeps what it is using.
const CacheTrimKeep = 3 * time.Hour

// The limit, a setting of the person's own machine and never of a checkout, read
// as load.preflights is (implement/load_slots.go): the Go cache is the account's,
// and a repository that could lower the limit could delete from it.
const (
	// CacheLimitKey is the machine configuration key, in whole GiB.
	CacheLimitKey = "load.go_cache_limit_gib"
	// DefaultCacheLimitGiB is the limit the person chose on 6 Nov 2026.
	DefaultCacheLimitGiB = 100
	// MaxCacheLimitGiB bounds the setting at 1 PiB, so it cannot overflow.
	MaxCacheLimitGiB = 1 << 20
)

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
// would have removed. Sizes are of the cache's entries alone, as Go reports each
// file's length, the measure the load gate names (machineload.GoCacheSize).
type CacheTrim struct {
	// Path is the cache's directory, home-redacted and masked; "" when go named
	// none.
	Path string `json:"path,omitempty"`
	// Skipped says why nothing was looked at; "" when the cache was walked.
	Skipped string `json:"skipped,omitempty"`
	// LimitBytes is the limit it worked to, and KeepHours how recently used an
	// entry had to be to stay whatever the limit.
	LimitBytes uint64 `json:"limit_bytes,omitempty"`
	KeepHours  int    `json:"keep_hours,omitempty"`
	// SizeBytes is the size found, and LeftBytes the size it left.
	SizeBytes uint64 `json:"size_bytes,omitempty"`
	LeftBytes uint64 `json:"left_bytes,omitempty"`
	// OverLimit says LeftBytes is still over the limit: what remains was used
	// inside the keep, or could not be removed (Failed).
	OverLimit      bool   `json:"over_limit,omitempty"`
	EntriesRemoved int    `json:"entries_removed"`
	BytesFreed     uint64 `json:"bytes_freed"`
	// Failed counts the entries that could not be removed and the
	// subdirectories that could not be read.
	Failed int `json:"failed,omitempty"`
}

// TrimGoCache deletes the oldest entries of the cache `go env GOCACHE` names
// until the rest fit the person's limit, as of now, keeping every entry used
// inside CacheTrimKeep, or under a dry run only counts them. It reports rather
// than returns a failure: a trim never fails what asked for it.
func TrimGoCache(dryRun bool, now time.Time) CacheTrim {
	return trimGoCache(dryRun, now, goCacheLimit)
}

// TrimGoCacheToForTest is TrimGoCache to a limit the caller gives, for a test standing a
// small cache over one with no configuration file.
func TrimGoCacheToForTest(limit uint64, dryRun bool, now time.Time) CacheTrim {
	return trimGoCache(dryRun, now, func() (uint64, string) { return limit, "" })
}

// trimGoCache is TrimGoCache with where its limit comes from, read only once the
// cache has proved to be one: limit answers the bytes, or why it has none.
func trimGoCache(dryRun bool, now time.Time, limit func() (uint64, string)) CacheTrim {
	out := CacheTrim{KeepHours: int(CacheTrimKeep / time.Hour)}
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
	if out.LimitBytes, skip = limit(); skip != "" {
		out.Skipped = skip
		return out
	}
	subdirs, err := readRootDir(root, ".")
	if err != nil {
		out.Skipped = shown + " could not be read: " + shownPath(causeOf(err))
		return out
	}
	// An entry may have been used an hour after its time says, as Go's own trim
	// allows for, so what was used inside the keep is what is newer than this.
	cutoff := now.Add(-CacheTrimKeep - goCacheUseInterval)
	var removable []cacheEntry
	for _, sub := range subdirs {
		if sub.IsDir() && isCacheSubdir(sub.Name()) {
			removable = readCacheSubdir(root, sub.Name(), cutoff, removable, &out)
		}
	}
	out.LeftBytes = out.SizeBytes
	afterWalk()
	removeOldest(root, removable, cutoff, dryRun, &out)
	out.OverLimit = out.LeftBytes > out.LimitBytes
	return out
}

// goCacheLimit is the limit in bytes from the machine configuration alone, or why
// there is none. Where load.preflights reads past a file it cannot read, this
// stops: guessing a size to delete down to could remove what the person's own
// value would have kept.
func goCacheLimit() (uint64, string) {
	home, err := abcdhome.MachineHome()
	if err != nil || home == "" {
		return 0, "the home directory could not be read, so " + CacheLimitKey + " cannot be"
	}
	roots, _ := layered.MachineRoots(home)
	if roots.HomeRefused {
		return 0, "HOME lies inside a git working tree and is not this account's home, so " + CacheLimitKey + " cannot be read"
	}
	stack, err := layered.Load(layered.Config, roots)
	if err != nil {
		return 0, layered.Config.MachineOrigin() + " cannot be read for " + CacheLimitKey + ": " + shownPath(err.Error())
	}
	// A key under load that nothing reads may be this one misspelt, and the
	// default would then delete what the person's value meant to keep.
	if err := stack.Claim("load", machineload.LoadConfigKeys...); err != nil {
		return 0, "the limit cannot be read past a key nothing reads: " + shownPath(err.Error())
	}
	gib, err := layered.Get(stack, CacheLimitKey, DefaultCacheLimitGiB, func(n int) error {
		if n < 1 || n > MaxCacheLimitGiB {
			return fmt.Errorf("want a whole number of GiB from 1 to %d", MaxCacheLimitGiB)
		}
		return nil
	})
	if err != nil {
		return 0, "the limit cannot be used: " + shownPath(err.Error())
	}
	return uint64(gib.V) << 30, ""
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
	home, err := abcdhome.MachineHome()
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

// cacheEntry is one entry the limit may remove.
type cacheEntry struct {
	rel  string
	mod  time.Time
	size uint64
}

// readCacheSubdir adds up the entries of one subdirectory in out and appends
// those older than cutoff to removable. A cache of a million entries holds about
// a hundred megabytes here, and only those past the keep. An entry another
// process removed first is no failure.
func readCacheSubdir(root *os.Root, sub string, cutoff time.Time, removable []cacheEntry, out *CacheTrim) []cacheEntry {
	files, err := readRootDir(root, sub)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			out.Failed++
		}
		return removable
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
		if !fi.Mode().IsRegular() {
			continue
		}
		out.SizeBytes += uint64(fi.Size())
		if fi.ModTime().Before(cutoff) {
			removable = append(removable, cacheEntry{rel: rel, mod: fi.ModTime(), size: uint64(fi.Size())})
		}
	}
	return removable
}

// afterWalk runs between the walk and the removals; a variable only so a test can
// stand in for a build using an entry while the walk ran.
var afterWalk = func() {}

// removeOldest removes entries oldest first until what is left fits the limit,
// and stops there: an entry the limit does not need gone stays however old. The
// walk takes minutes on a large cache and Go refreshes an entry's time when a
// build uses it, so each entry is read again just before it goes: one used since
// the walk (its time no longer before cutoff) stays and counts as left, and a name
// that is no longer a regular file is no entry and is left alone. Between that read
// and the unlink a build may still look the entry up and then fail to open it; the
// window is one system call wide. An entry another process removed first is no
// failure and no removal of this one, and one that cannot be removed still counts
// as left.
func removeOldest(root *os.Root, removable []cacheEntry, cutoff time.Time, dryRun bool, out *CacheTrim) {
	slices.SortFunc(removable, func(a, b cacheEntry) int {
		return cmp.Or(a.mod.Compare(b.mod), strings.Compare(a.rel, b.rel))
	})
	for _, e := range removable {
		if out.LeftBytes <= out.LimitBytes {
			return
		}
		fi, err := root.Lstat(e.rel)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			out.LeftBytes -= e.size
			continue
		case err != nil:
			out.Failed++
			continue
		case !fi.Mode().IsRegular():
			out.LeftBytes -= e.size
			continue
		case !fi.ModTime().Before(cutoff):
			continue
		}
		if !dryRun {
			if err := root.Remove(e.rel); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					out.LeftBytes -= e.size
				} else {
					out.Failed++
				}
				continue
			}
		}
		out.LeftBytes -= e.size
		out.EntriesRemoved++
		out.BytesFreed += e.size
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
