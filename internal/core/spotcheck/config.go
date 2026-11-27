package spotcheck

// config.go is the spot-check's configuration (spc-2610301925429250
// approach 3): the spotcheck block of .abcd/config.json, read through the
// layered resolver. A draw and every reader of its cycle read it at the
// commit the window starts at, the repository layer alone (windowConfig).

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
func readConfig(ctx context.Context, r layered.Roots) (Config, error) {
	s, err := layered.LoadContext(ctx, layered.Config, r)
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
// as that commit holds it. A commit this checkout does not hold is a
// notHeldError.
// A read that failed for a reason that says nothing of the file (a git that
// did not answer, or ctx ended) is an error of its own and never the
// defaults, since another reader, or this one again, would read the block.
func windowConfig(ctx context.Context, root, commit string) (c Config, why string, err error) {
	if _, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", 256, "rev-parse", "--verify", "--quiet", commit+"^{commit}", "--"); err != nil {
		if ctx.Err() != nil {
			return Config{}, "", fmt.Errorf("the spot-check's window start %s was not read: %s", commit[:12], redact(ctx.Err()))
		}
		return Config{}, "", notHeldError{commit[:12]}
	}
	// A file over the size a configuration file may hold is a file read and
	// refused, the same for every reader, so its size is read from the
	// listing first: the capped read of its content would otherwise fail
	// as git does.
	size, err := blockSize(ctx, root, commit)
	if err != nil {
		return Config{}, "", fmt.Errorf("the spotcheck block at the window's start was not read (%s), so the window cannot be counted yet", redact(err))
	}
	if size > layered.MaxFileBytes {
		return Config{WindowDays: DefaultWindowDays, PerGroup: DefaultPerGroup},
			fmt.Sprintf("%s at %s is larger than %d bytes", layered.Config.RepoRel, commit[:12], layered.MaxFileBytes), nil
	}
	r, _ := layered.RootsFor(root)
	c, err = readConfig(ctx, layered.Roots{Repo: r.Repo, RepoAt: commit, RepoAtName: commit[:12], RepoAtAsStored: true, HomeRefused: true})
	var ue *layered.UnreadError
	switch {
	case err == nil:
		return c, "", nil
	case ctx.Err() != nil || errors.As(err, &ue) && ue.GitFailed():
		return Config{}, "", fmt.Errorf("the spotcheck block at the window's start was not read (%s), so the window cannot be counted yet", redact(err))
	}
	return Config{WindowDays: DefaultWindowDays, PerGroup: DefaultPerGroup}, redact(err), nil
}

// blockSize is the size of .abcd/config.json at commit as git lists it, -1
// when the commit holds no blob there (no file, or one the layered reader
// refuses by its kind). err is a git that did not answer.
func blockSize(ctx context.Context, root, commit string) (int, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", 4096, "ls-tree", "-l", "-z", commit, "--", layered.Config.RepoRel)
	if err != nil {
		return 0, err
	}
	meta, _, _ := strings.Cut(strings.TrimSuffix(string(out), "\x00"), "\t")
	f := strings.Fields(meta)
	if len(f) != 4 || f[1] != "blob" {
		return -1, nil
	}
	n, err := strconv.Atoi(f[3])
	if err != nil || n < 0 {
		return 0, errors.New("git listed the block's file with a size that does not read")
	}
	return n, nil
}

// notHeldError is a window's start this checkout does not hold, by its
// short name.
type notHeldError struct{ short string }

func (e notHeldError) Error() string {
	return "the spot-check's window starts at " + e.short + ", which this checkout does not hold"
}

// unreadError is a cycle whose check could not be made for a passing
// reason: the read that refuses the observation, never a cycle set aside.
type unreadError struct{ err error }

func (e *unreadError) Error() string { return e.err.Error() }

// defaultsNote is the note's tail for a window counted by the bundled
// defaults because its block did not read (why), "" for none.
func defaultsNote(why string) string {
	if why == "" {
		return ""
	}
	return fmt.Sprintf("; the spotcheck block at the window's start does not read (%s), so this window is counted by the bundled defaults (%d days, %d per group)",
		why, DefaultWindowDays, DefaultPerGroup)
}
