package implement

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/shellquote"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The disk floor (iss-2610281502581386). On 28 Oct 2026 the Go build cache
// had grown to 177 GiB and the development machine's data volume fell to 658
// MiB free; every preflight on the machine then failed with "no space left on
// device" or wrote truncated files, and no gate named the cause. Go trims
// only cache entries unused for five days, and every store worktree, race
// build and pinned-toolchain build adds entries, so days of many-worktree
// runs fill the disk.
//
// So the waiter (WaitLoad) refuses to admit a test stage, at either gate,
// while the volume holding the Go build cache (`go env GOCACHE`) or the one
// holding the worktree store (~/.abcd.noindex/worktrees, or its nearest
// existing ancestor) has less than DiskFloor free, and the load check
// (CheckLoad) reports the same finding as a warning, never failing. The
// waiter refuses rather than waits: nothing frees disk while it sleeps, so a
// wait would only hold the preflight until its timeout. The refusal names
// each low volume, its free space, the floor, the cache's path and size, and
// the age-based trim a person can run. abcd never trims the cache itself:
// deleting another tool's files is not the gate's call.
//
// A platform whose free space abcd cannot read (other than macOS and Linux),
// or a read that fails, is reported as unknown and never refused.

// DiskFloor is the free space, in bytes, below which the load gate admits no
// test stage. Measured on 29 Oct 2026, the stages one `make preflight` of
// this module builds (go build, go vet, go test and go test -race over
// internal/), run into an empty build cache, wrote 1.9 GiB of cache entries,
// and their test binaries took another 1.9 GiB of the temporary directory at
// the peak: about 4 GiB for one cold preflight. The person's load.preflights
// admits at most four at once (MaxPreflightSlots), so 16 GiB holds four cold
// preflights side by side before the volume runs out. It is a constant: the
// person-level load settings (load.preflights in ~/.abcd.noindex/config.json)
// take no other key today, and a floor a repository could lower would defeat
// it.
const DiskFloor uint64 = 16 << 30

// The disk statuses.
const (
	// DiskOK: every volume read has at least DiskFloor free.
	DiskOK = "ok"
	// DiskLow: at least one volume has less than DiskFloor free.
	DiskLow = "low"
	// DiskUnknown: no volume could be read (an unsupported platform, or reads
	// that failed); nothing is refused for it.
	DiskUnknown = "unknown"
)

// What a volume holds.
const (
	DiskHoldsGoCache       = "go-build-cache"
	DiskHoldsWorktreeStore = "worktree-store"
)

// DiskVolume is one volume the check read.
type DiskVolume struct {
	// Holds is what the gate reads it for: DiskHoldsGoCache,
	// DiskHoldsWorktreeStore, or both.
	Holds []string `json:"holds"`
	// Paths are the paths read on it, in the order of Holds, home-relative
	// where under the home.
	Paths []string `json:"paths"`
	// Mount is where the volume is mounted, when the platform says.
	Mount     string `json:"mount,omitempty"`
	FreeBytes uint64 `json:"free_bytes"`
	Low       bool   `json:"low"`
}

// DiskCheck is the disk half of the gate's judgement.
type DiskCheck struct {
	Status     string       `json:"status"`
	FloorBytes uint64       `json:"floor_bytes"`
	Volumes    []DiskVolume `json:"volumes"`
	// GoCache is the Go build cache's directory, home-relative where under the
	// home; "" when there is none to read (GOCACHE=off, or no cache yet).
	GoCache string `json:"go_cache,omitempty"`
	// GoCacheBytes is the cache's size, measured only when its volume is low;
	// GoCacheSampled and GoCacheSubdirs say how (machineload.GoCacheSize).
	GoCacheBytes   uint64 `json:"go_cache_bytes,omitempty"`
	GoCacheSampled int    `json:"go_cache_sampled,omitempty"`
	GoCacheSubdirs int    `json:"go_cache_subdirs,omitempty"`
	// Reason says why a volume could not be read, or why the cache was not
	// measured.
	Reason string `json:"reason,omitempty"`

	// goCacheShell is the cache's path as the trim command names it, and
	// goCacheMeasured whether its size was.
	goCacheShell    string
	goCacheMeasured bool
}

// ReadDisk judges the free space for the account whose home is home ("" is
// this process's). It is a variable only so tests replace it: the core's own
// tests by default (load_disk_test.go), so none reads the real disk unless it
// asks.
var ReadDisk = func(home string) DiskCheck { return checkDisk(home, realDiskProbe) }

// diskProbe is what checkDisk reads the machine through.
type diskProbe struct {
	// goCache is the Go build cache's directory, "" when there is none.
	goCache func() string
	// volume reads the volume holding a path (machineload.ReadVolume).
	volume func(string) (machineload.Volume, error)
	// cacheSize measures the cache (machineload.GoCacheSize).
	cacheSize func(string) (machineload.CacheSize, error)
	// floor is DiskFloor, a field so a test can state its own.
	floor uint64
}

var realDiskProbe = diskProbe{
	goCache: machineload.GoCacheDir,
	volume:  machineload.ReadVolume,
	cacheSize: func(dir string) (machineload.CacheSize, error) {
		return machineload.GoCacheSize(dir, machineload.GoCacheBudget)
	},
	floor: DiskFloor,
}

// checkDisk reads the volumes holding the Go build cache and the worktree
// store and judges each against the floor. Two paths on one volume are one
// volume, named once.
func checkDisk(home string, p diskProbe) DiskCheck {
	out := DiskCheck{Status: DiskOK, FloorBytes: p.floor, Volumes: []DiskVolume{}}
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	type target struct{ holds, path string }
	var targets []target
	cache := p.goCache()
	if cache != "" {
		targets = append(targets, target{DiskHoldsGoCache, cache})
		out.GoCache = displayPath(home, cache)
		out.goCacheShell = shellPath(home, cache)
	}
	if home != "" {
		targets = append(targets, target{DiskHoldsWorktreeStore, abcdhome.Path(home, "worktrees")})
	}
	var reasons []string
	byID := map[string]int{}
	cacheLow := false
	for _, t := range targets {
		v, err := p.volume(t.path)
		if err != nil {
			if errors.Is(err, machineload.ErrUnsupported) {
				reasons = appendOnce(reasons, "abcd reads free space on macOS and Linux only")
			} else {
				reasons = append(reasons, "the volume holding "+displayPath(home, t.path)+" could not be read: "+termsafe.Sanitize(fsRedact(err.Error(), home)))
			}
			continue
		}
		i, seen := byID[v.ID]
		if !seen || v.ID == "" {
			i = len(out.Volumes)
			byID[v.ID] = i
			out.Volumes = append(out.Volumes, DiskVolume{Mount: termsafe.Sanitize(v.Mount), FreeBytes: v.Free, Low: v.Free < p.floor})
		}
		out.Volumes[i].Holds = append(out.Volumes[i].Holds, t.holds)
		shown := t.path
		if t.holds == DiskHoldsWorktreeStore {
			shown = v.Path // its nearest existing ancestor, when it is not there yet
		}
		out.Volumes[i].Paths = append(out.Volumes[i].Paths, displayPath(home, shown))
		if out.Volumes[i].Low {
			out.Status = DiskLow
			cacheLow = cacheLow || t.holds == DiskHoldsGoCache
		}
	}
	if len(out.Volumes) == 0 && len(targets) > 0 {
		out.Status = DiskUnknown
	}
	if cacheLow {
		if size, err := p.cacheSize(cache); err != nil {
			reasons = append(reasons, "the Go build cache's size could not be measured: "+termsafe.Sanitize(fsRedact(err.Error(), home)))
		} else {
			out.GoCacheBytes, out.GoCacheSampled, out.GoCacheSubdirs = size.Bytes, size.Sampled, size.Subdirs
			out.goCacheMeasured = true
		}
	}
	out.Reason = strings.Join(reasons, "; ")
	return out
}

// Refusal is the waiter's refusal when a volume is low, nil otherwise.
func (d DiskCheck) Refusal() error {
	if d.Status != DiskLow {
		return nil
	}
	return errors.New("the disk is nearly full, so no test stage is admitted: a test stage writes gigabytes of build output and test binaries, " +
		"and on a full disk every preflight fails or writes truncated files. Waiting would not help: nothing frees disk while a waiter sleeps, " +
		"so this refuses at once; nothing was waited for.\n" + d.describe())
}

// Report is the load check's copy of the finding: nil when every volume read
// has room, so the check's report is unchanged then.
func (d DiskCheck) Report() *DiskCheck {
	if d.Status == DiskOK {
		return nil
	}
	return &d
}

// WarningText is the load check's warning: "" when there is nothing to say.
// The check never fails for it.
func (d *DiskCheck) WarningText(site string) string {
	if d == nil {
		return ""
	}
	switch d.Status {
	case DiskLow:
		return fmt.Sprintf("LOW DISK WARNING (%s): the load gate will admit no test stage until there is more room.\n%s", site, d.describe())
	case DiskUnknown:
		return fmt.Sprintf("disk check (%s): free space could not be read (%s); carrying on without checking it\n", site, d.Reason)
	}
	return ""
}

// describe is the finding's body: each low volume, the cache's size and the
// way to free space, one indented line each.
func (d DiskCheck) describe() string {
	var b strings.Builder
	cacheLow, storeLow := false, false
	for _, v := range d.Volumes {
		if !v.Low {
			continue
		}
		where := "the volume"
		if v.Mount != "" {
			where = "the volume mounted at " + v.Mount
		}
		var holds []string
		for i, h := range v.Holds {
			switch h {
			case DiskHoldsGoCache:
				cacheLow = true
				holds = append(holds, "the Go build cache ("+v.Paths[i]+")")
			case DiskHoldsWorktreeStore:
				storeLow = storeLow || len(v.Holds) == 1
				holds = append(holds, "the worktree store ("+v.Paths[i]+")")
			}
		}
		fmt.Fprintf(&b, "  %s free on %s, which holds %s; the floor is %s.\n",
			bytesText(v.FreeBytes), where, strings.Join(holds, " and "), bytesText(d.FloorBytes))
	}
	if cacheLow {
		switch {
		case !d.goCacheMeasured:
			b.WriteString("  The Go build cache's size could not be measured.\n")
		case d.GoCacheSampled < d.GoCacheSubdirs:
			fmt.Fprintf(&b, "  The Go build cache holds about %s (estimated from %d of its %d subdirectories).\n",
				bytesText(d.GoCacheBytes), d.GoCacheSampled, d.GoCacheSubdirs)
		default:
			fmt.Fprintf(&b, "  The Go build cache holds %s.\n", bytesText(d.GoCacheBytes))
		}
		b.WriteString("  Go trims only entries unused for five days. To delete the entries unused for two days:\n")
		fmt.Fprintf(&b, "    find %s -type f -mmin +2880 -delete\n", d.goCacheShell)
		b.WriteString("  or, all or nothing, to empty the cache, so the next build rebuilds everything:\n    go clean -cache\n")
	}
	if storeLow {
		b.WriteString("  Remove the store worktrees no longer in use (git worktree remove <path>), or free space on that volume otherwise.\n")
	}
	if d.Reason != "" {
		fmt.Fprintf(&b, "  Also: %s.\n", d.Reason)
	}
	return b.String()
}

// bytesText renders a size in GiB to one decimal, or in MiB below one GiB.
func bytesText(n uint64) string {
	if n < 1<<30 {
		return fmt.Sprintf("%d MiB", n>>20)
	}
	return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
}

// displayPath is p as a message names it: home-relative under home, and
// sanitized for the terminal.
func displayPath(home, p string) string {
	if home != "" {
		if rel, err := filepath.Rel(home, p); err == nil && rel != ".." && !strings.HasPrefix(rel, "../") {
			if rel == "." {
				return "~"
			}
			return termsafe.Sanitize("~/" + filepath.ToSlash(rel))
		}
	}
	return termsafe.Sanitize(p)
}

// shellUnquoted is what a POSIX shell takes unquoted.
var shellUnquoted = regexp.MustCompile(`^[A-Za-z0-9._/:@%+=-]+$`)

// shellPath is p as a command line a person pastes names it: home-relative
// under home, with the tilde left bare so the shell expands it, and the rest
// quoted when it needs it.
func shellPath(home, p string) string {
	prefix, rest := "", displayPath(home, p)
	if strings.HasPrefix(rest, "~/") {
		prefix, rest = "~/", strings.TrimPrefix(rest, "~/")
	}
	if shellUnquoted.MatchString(rest) {
		return prefix + rest
	}
	return prefix + shellquote.Single(rest)
}

// fsRedact replaces home in s with ~, so no message carries the home's path.
func fsRedact(s, home string) string {
	if home == "" {
		return s
	}
	return fsutil.RedactRoot(s, filepath.Clean(home), "~")
}

// appendOnce appends s unless it is already there.
func appendOnce(list []string, s string) []string {
	for _, have := range list {
		if have == s {
			return list
		}
	}
	return append(list, s)
}
