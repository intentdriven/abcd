package spotcheck

// config.go is the spot-check's configuration (spc-2610301925429250
// approach 3): the spotcheck block of .abcd/config.json, read through the
// layered resolver. A draw and every reader of its cycle read it at the
// commit the window starts at, the repository layer alone (windowConfig).

import (
	"context"
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The configuration's keys, bundled defaults and bounds.
const (
	KeyWindowDays     = "spotcheck.window_days"
	KeyWindowChanges  = "spotcheck.window_changes"
	KeyPerGroup       = "spotcheck.per_group"
	DefaultWindowDays = 7
	DefaultPerGroup   = 10
	maxWindowDays     = 90
	minWindowChanges  = 10
	maxWindowChanges  = 100000
	maxPerGroup       = 100
)

// Config is the spotcheck block as resolved. WindowChanges 0 is unset: only
// the days close the window.
type Config struct {
	WindowDays    int
	WindowChanges int
	PerGroup      int
}

// readConfig resolves the spotcheck block. A misspelt key, and a value out
// of its bounds or not a whole number, is refused naming its file and key,
// never replaced by the default.
func readConfig(r layered.Roots) (Config, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return Config{}, err
	}
	if err := s.Claim("spotcheck", "window_days", "window_changes", "per_group"); err != nil {
		return Config{}, err
	}
	days, err := layered.Get(s, KeyWindowDays, DefaultWindowDays, between(1, maxWindowDays, "days"))
	if err != nil {
		return Config{}, err
	}
	changes, err := layered.Get(s, KeyWindowChanges, 0, func(n int) error {
		if n == 0 {
			return nil
		}
		return between(minWindowChanges, maxWindowChanges, "changes")(n)
	})
	if err != nil {
		return Config{}, err
	}
	per, err := layered.Get(s, KeyPerGroup, DefaultPerGroup, between(1, maxPerGroup, "items"))
	if err != nil {
		return Config{}, err
	}
	return Config{WindowDays: days.V, WindowChanges: changes.V, PerGroup: per.V}, nil
}

// between is a check that n is a whole number of what from lo to hi.
func between(lo, hi int, what string) func(int) error {
	return func(n int) error {
		if n < lo || n > hi {
			return fmt.Errorf("want a whole number of %s from %d to %d", what, lo, hi)
		}
		return nil
	}
}

// windowConfig is the spotcheck block that governs the window starting at
// commit: the repository layer alone, as the default branch stores it at
// that commit, so every reader of a cycle counts its window by the same
// values, none a later landing or one machine's own layer chose. A block that
// does not read there, a file that does not parse or a value misspelt or out
// of its bounds, counts that window by the bundled defaults, and why says
// why, "" when the block read: the window is still drawn, the same by every
// reader, and the next window, starting at a later commit, reads the block
// as that commit holds it. A commit this checkout does not hold is an error.
func windowConfig(ctx context.Context, root, commit string) (c Config, why string, err error) {
	if _, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", 256, "rev-parse", "--verify", "--quiet", commit+"^{commit}", "--"); err != nil {
		return Config{}, "", fmt.Errorf("the spot-check's window starts at %s, which this checkout does not hold", commit[:12])
	}
	r, _ := layered.RootsFor(root)
	c, err = readConfig(layered.Roots{Repo: r.Repo, RepoAt: commit, RepoAtName: commit[:12], RepoAtAsStored: true, HomeRefused: true})
	if err != nil {
		return Config{WindowDays: DefaultWindowDays, PerGroup: DefaultPerGroup}, redact(err), nil
	}
	return c, "", nil
}

// defaultsNote is the note's tail for a window counted by the bundled
// defaults because its block did not read (why), "" for none.
func defaultsNote(why string) string {
	if why == "" {
		return ""
	}
	return fmt.Sprintf("; the spotcheck block at the window's start does not read (%s), so this window is counted by the bundled defaults (%d days, %d per group)",
		why, DefaultWindowDays, DefaultPerGroup)
}
