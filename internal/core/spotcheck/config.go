package spotcheck

// config.go is the spot-check's configuration (spc-2610301925429250
// approach 3): the spotcheck block of .abcd/config.json, read through the
// layered resolver. A draw and every reader of its cycle read it at the
// commit the window starts at, the repository layer alone (windowConfig).

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
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
// values, none a later landing or one machine's own layer chose.
func windowConfig(root, commit string) (Config, error) {
	r, _ := layered.RootsFor(root)
	return readConfig(layered.Roots{Repo: r.Repo, RepoAt: commit, RepoAtName: commit[:12], RepoAtAsStored: true, HomeRefused: true})
}
