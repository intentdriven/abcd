package worktree

// cachetrim_floor.go is the trim the load gate runs before it refuses at the
// disk floor (iss-2611220942266387). The trim kept the cache to its limit only
// when the build loop made a lane or a prune removed a worktree, so a session
// that drove its own worktrees and preflights reached neither: over a night of
// preflights on 21-22 Nov 2026 the free disk fell from 30 GiB to 12 GiB and a
// person cleared it by hand five times. The gate now runs the trim once before
// it refuses, inline, since the waiter has a terminal to say it on, and paced by
// the marker a lane's start claims (ClaimCacheTrim), so two waiters at the floor
// never both walk the cache and neither walks it again inside the hour.
//
// The trim is a step of each toolchain the repository uses, told as the needs
// list tells a tool's version file: a regular file from that toolchain's closed
// set at the repository's root (needs.VersionFiles). Go's build cache is the one
// step built in, and the step is TrimGoCache itself, so every guarantee in
// cachetrim.go's header holds: only the directory Go names, only Go's own
// layout, each entry's time read again just before it goes. abcd trims no cache
// whose layout it does not know, since there it cannot tell an entry from the
// tool's own index, so a repository with no known toolchain trims nothing and
// the answer says so.

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/needs"
)

// ToolchainTrim is what one toolchain's step did at the floor.
type ToolchainTrim struct {
	// Toolchain names the toolchain ("go"), and Cache what its trim clears, as a
	// sentence names it.
	Toolchain string `json:"toolchain"`
	Cache     string `json:"cache"`
	// Trim is what the trim answered, nil when it did not run.
	Trim *CacheTrim `json:"trim,omitempty"`
	// PacedSince is when the trim that paced this one out was claimed, zero
	// unless it was paced out; Last is that trim's answer, nil while it has not
	// finished (or when it never did).
	PacedSince time.Time  `json:"paced_since,omitzero"`
	Last       *CacheTrim `json:"last,omitempty"`
	// Problem says why the pacing record could not be claimed, so the trim did
	// not run, or why a trim's answer could not be saved in it.
	Problem string `json:"problem,omitempty"`
}

// FloorTrim is what the floor's trim did across the repository's toolchains.
type FloorTrim struct {
	Steps []ToolchainTrim `json:"steps,omitempty"`
	// None says why no step ran: the repository uses no toolchain abcd trims a
	// cache for. "" when a step ran.
	None string `json:"none,omitempty"`
}

// BytesFreed is what the steps that ran freed together, as their caches count
// it.
func (f FloorTrim) BytesFreed() uint64 {
	var n uint64
	for _, s := range f.Steps {
		if s.Trim != nil {
			n += s.Trim.BytesFreed
		}
	}
	return n
}

// trimStep is one toolchain's cache trim: the toolchain, what its trim clears,
// and the trim with its pacing record.
type trimStep struct {
	toolchain string
	cache     string
	trim      func(dryRun bool, now time.Time) CacheTrim
	claim     func(home string, now time.Time) (CacheTrimRun, bool, error)
	save      func(home string, started time.Time, t CacheTrim) error
}

// goTrimStep is Go's step, with trim as its trim: TrimGoCache in the binary.
func goTrimStep(trim func(dryRun bool, now time.Time) CacheTrim) trimStep {
	return trimStep{toolchain: "go", cache: "the Go build cache", trim: trim, claim: ClaimCacheTrim, save: SaveCacheTrim}
}

// TrimForFloor runs, as of now, the paced cache trim of each toolchain the
// repository whose root is dir uses, for the account whose home is home, calling
// before with the cache's name just before a trim starts, since a large cache
// takes minutes. It never fails: what it could not do is in the answer.
func TrimForFloor(home, dir string, now time.Time, before func(cache string)) FloorTrim {
	return trimForFloor(home, dir, now, before, []trimStep{goTrimStep(TrimGoCache)})
}

// trimForFloor is TrimForFloor over steps.
func trimForFloor(home, dir string, now time.Time, before func(string), steps []trimStep) FloorTrim {
	var out FloorTrim
	var looked []string
	for _, step := range steps {
		files := needs.VersionFiles(step.toolchain)
		looked = append(looked, strings.Join(files, " or ")+" for "+step.toolchain)
		if !holdsAny(dir, files) {
			continue
		}
		out.Steps = append(out.Steps, runTrimStep(step, home, now, before))
	}
	if len(out.Steps) == 0 {
		out.None = "the repository uses no toolchain abcd trims a cache for: it looks for " + strings.Join(looked, ", ") +
			" at the repository's root"
	}
	return out
}

// runTrimStep claims the step's pacing record and, when a trim is due, runs it
// and saves its answer there.
func runTrimStep(step trimStep, home string, now time.Time, before func(string)) ToolchainTrim {
	got := ToolchainTrim{Toolchain: step.toolchain, Cache: step.cache}
	last, due, err := step.claim(home, now)
	switch {
	case err != nil:
		got.Problem = "its pacing record could not be claimed: " + shownPath(err.Error())
		return got
	case !due:
		got.PacedSince, got.Last = last.StartedAt, last.Trim
		return got
	}
	if before != nil {
		before(step.cache)
	}
	t := step.trim(false, now)
	got.Trim = &t
	if err := step.save(home, now, t); err != nil {
		got.Problem = "its answer could not be saved in its pacing record: " + shownPath(err.Error())
	}
	return got
}

// holdsAny reports whether dir holds one of files as a regular file of its own,
// read without following a link.
func holdsAny(dir string, files []string) bool {
	if dir == "" {
		return false
	}
	for _, f := range files {
		if fi, err := os.Lstat(filepath.Join(dir, f)); err == nil && fi.Mode().IsRegular() {
			return true
		}
	}
	return false
}
