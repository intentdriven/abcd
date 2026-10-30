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
// existing ancestor) has less than the floor free (diskFloorFor), and the
// load check (CheckLoad) reports the same finding as a warning, never
// failing. The waiter reads the disk when it starts and again just before it
// opens, since other work fills the disk while it waits; it refuses rather
// than waits, since nothing frees disk while it sleeps. The refusal names
// each low volume, its free space and the floor, and, when the cache's own
// volume is low, the cache's path and size and the age-based trim a person
// can run. abcd never trims the cache itself: deleting another tool's files
// is not the gate's call.
//
// A platform whose free space abcd cannot read (other than macOS and Linux),
// or a read that fails, is reported and never refused.

// DiskPerPreflight is the free space the floor keeps for each preflight the
// gate admits at once. Measured on 29 Oct 2026, the stages
// one `make preflight` of this module builds (go build, go vet, go test and
// go test -race over internal/), run into an empty build cache, wrote 1.9 GiB
// of cache entries, and their test binaries took another 1.9 GiB of the
// temporary directory at the peak: about 4 GiB for one cold preflight.
const DiskPerPreflight uint64 = 4 << 30

// DiskFloorMin is the lowest the floor goes, whatever the slots: one cold
// preflight's 4 GiB, and as much again for everything else writing to the
// volume while it runs.
const DiskFloorMin uint64 = 8 << 30

// diskFloorFor is the free space, in bytes, below which the load gate admits
// no run, for a gate admitting slots preflights at once (load.preflights,
// PreflightSlots): DiskPerPreflight for each, and DiskFloorMin at least. So
// 8 GiB at the default of one and 16 GiB at four.
// It follows only the person's own load.preflights (or its environment
// variable), never a repository file, so no checkout can lower it.
func diskFloorFor(slots int) uint64 {
	return max(uint64(max(slots, 0))*DiskPerPreflight, DiskFloorMin)
}

// diskFloorAt is the floor for the account whose home is home, at the slots
// its settings give; a setting that cannot be used (which the wait refuses
// for) reads as the default of one.
func diskFloorAt(getenv func(string) string, home string) uint64 {
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil || h == "" {
			return diskFloorFor(1)
		}
		home = h
	}
	slots, _, _, err := PreflightSlots(getenv, home)
	if err != nil {
		slots = 1
	}
	return diskFloorFor(slots)
}

// The disk statuses.
const (
	// DiskOK: every volume read has at least the floor free; Reason says
	// when a volume could not be read beside one that was.
	DiskOK = "ok"
	// DiskLow: at least one volume has less than the floor free.
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

	// goCacheShell is the cache's path as the trim command names it,
	// goCacheFuzz the pattern naming its fuzz corpora there, goCacheMeasured
	// whether its size was measured, and goCacheTrimmable whether it has a
	// Go build cache's shape and is not the home or above it, so the gate
	// offers a command deleting files in it (machineload.LooksLikeGoCache).
	goCacheShell     string
	goCacheFuzz      string
	goCacheMeasured  bool
	goCacheTrimmable bool
	goCacheHome      bool
}

// ReadDisk judges the free space for the account whose home is home ("" is
// this process's) against floor (diskFloorFor). It is a variable only so
// tests replace it: the core's own tests by default (load_disk_test.go), so
// none reads the real disk unless it asks.
var ReadDisk = func(home string, floor uint64) DiskCheck { return checkDisk(home, realDiskProbe(floor)) }

// diskProbe is what checkDisk reads the machine through.
type diskProbe struct {
	// goCache is the Go build cache's directory, "" when there is none.
	goCache func() string
	// volume reads the volume holding a path (machineload.ReadVolume).
	volume func(string) (machineload.Volume, error)
	// cacheSize measures the cache (machineload.GoCacheSize).
	cacheSize func(string) (machineload.CacheSize, error)
	// floor is the floor judged against (diskFloorFor).
	floor uint64
}

// realDiskProbe reads the machine, judging against floor.
func realDiskProbe(floor uint64) diskProbe {
	return diskProbe{
		goCache: machineload.GoCacheDir,
		volume:  machineload.ReadVolume,
		cacheSize: func(dir string) (machineload.CacheSize, error) {
			return machineload.GoCacheSize(dir, machineload.GoCacheBudget)
		},
		floor: floor,
	}
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
		out.goCacheFuzz = shellWord(globEscape(displayPath(home, cache)) + "/fuzz/*")
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
		out.goCacheHome = holdsHome(cache, home)
		out.goCacheTrimmable = !out.goCacheHome && machineload.LooksLikeGoCache(cache)
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

// Refusal is the waiter's refusal when a volume is low as it starts, nil
// otherwise.
func (d DiskCheck) Refusal() error {
	if d.Status != DiskLow {
		return nil
	}
	return errors.New("the disk is nearly full, so the gate admits no run: a test stage writes gigabytes of build output and test binaries, " +
		"and on a full disk every preflight fails or writes truncated files. Waiting would not help: nothing frees disk while a waiter sleeps, " +
		"so this refuses at once; nothing was waited for.\n" + d.describe())
}

// RefusalAtOpen is the waiter's refusal when a volume is low at its last
// look before it opens, after it has waited. Nil when no volume is low.
// firstFloor is the floor the first look judged against, and slots, read from
// setting, the count this look read: when this look's floor is above the
// first's and every low volume still has the first's floor free, only the
// floor moved, because the count rose, and the refusal leads with that
// (dcn-2610300618371321); otherwise other work filled the disk while it
// waited.
func (d DiskCheck) RefusalAtOpen(firstFloor uint64, slots int, setting string) error {
	if d.Status != DiskLow {
		return nil
	}
	lead := "the disk fell below the floor while this waited"
	if d.FloorBytes > firstFloor && d.lowAbove(firstFloor) {
		lead = fmt.Sprintf("the disk floor rose to %s while this waited, because %s is now %d (the floor was %s when this began to wait)",
			bytesText(d.FloorBytes), setting, slots, bytesText(firstFloor))
	}
	return errors.New(lead + ", so the gate admits no run: a test stage writes gigabytes of build output and test binaries, " +
		"and on a full disk every preflight fails or writes truncated files. The gate did not open and took no hold.\n" + d.describe())
}

// lowAbove reports whether every low volume has at least floor free.
func (d DiskCheck) lowAbove(floor uint64) bool {
	for _, v := range d.Volumes {
		if v.Low && v.FreeBytes < floor {
			return false
		}
	}
	return true
}

// Note is one line saying what the check could not read, "" when it read
// every volume or found one low (the refusal or warning then says it). An
// unknown check, or one that read one volume but not the other, refuses
// nothing; the note is how the person learns the disk went unchecked.
func (d DiskCheck) Note() string {
	switch {
	case d.Status == DiskUnknown:
		return "the disk was not checked: " + orUnread(d.Reason) + "; the gate does not refuse for a disk it cannot read"
	case d.Status == DiskOK && d.Reason != "":
		return "the disk was checked only in part: " + d.Reason + "; the volume read has room, and the gate does not refuse for the one it could not read"
	}
	return ""
}

// orUnread is reason, or a stand-in when there is none.
func orUnread(reason string) string {
	if reason == "" {
		return "no volume could be read"
	}
	return reason
}

// Report is the load check's copy of the finding: nil when every volume was
// read and has room, so the check's report is unchanged then.
func (d DiskCheck) Report() *DiskCheck {
	if d.Status == DiskOK && d.Reason == "" {
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
		return fmt.Sprintf("LOW DISK WARNING (%s): the load gate will admit no run until there is more room.\n%s", site, d.describe())
	}
	if note := d.Note(); note != "" {
		return fmt.Sprintf("disk check (%s): %s\n", site, note)
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
	fmt.Fprintf(&b, "  The floor is %s for each preflight the gate admits at once (%s in %s, or %s), and %s at least.\n",
		bytesText(DiskPerPreflight), PreflightSlotsKey, abcdhome.Display("config.json"), PreflightSlotsEnv, bytesText(DiskFloorMin))
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
		b.WriteString("  Go trims only entries unused for five days.")
		if d.goCacheTrimmable {
			// -H follows a symlinked cache; the fuzz corpora, which go clean
			// -cache keeps, are skipped by path, since -delete turns on
			// depth-first order, in which -prune does nothing.
			b.WriteString(" To delete the entries unused for two days, keeping the fuzz corpora:\n")
			fmt.Fprintf(&b, "    find -H %s -type f ! -path %s -mmin +2880 -delete\n", d.goCacheShell, d.goCacheFuzz)
			b.WriteString("  or, all or nothing, to empty the cache, so the next build rebuilds everything:\n    go clean -cache\n")
		} else {
			why := "does not have a Go build cache's shape"
			if d.goCacheHome {
				why = "is the home directory or holds it"
			}
			fmt.Fprintf(&b, " %s %s, so no trim of it is offered. go clean -cache empties a Go build cache, but it deletes from whatever GOCACHE names, so check that it names one first:\n    go clean -cache\n", d.GoCache, why)
		}
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
	return shellWord(displayPath(home, p))
}

// shellWord is a displayed path (displayPath) as one shell word: the tilde of
// a home-relative path left bare so the shell expands it, and the rest
// quoted when it needs it.
func shellWord(display string) string {
	prefix, rest := "", display
	if strings.HasPrefix(rest, "~/") {
		prefix, rest = "~/", strings.TrimPrefix(rest, "~/")
	}
	if shellUnquoted.MatchString(rest) {
		return prefix + rest
	}
	return prefix + shellquote.Single(rest)
}

// globEscape escapes the characters find's -path reads as a pattern, so a
// path holding them matches only itself.
func globEscape(p string) string {
	var b strings.Builder
	for _, c := range p {
		if strings.ContainsRune(`*?[\`, c) {
			b.WriteByte('\\')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// holdsHome reports whether dir is the home or a directory above it.
func holdsHome(dir, home string) bool {
	if home == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(home))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
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
