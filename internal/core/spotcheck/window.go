package spotcheck

// window.go is the trigger (spc-2610301925429250 approach 3): a window spans
// the default branch's first-parent line since the last cycle, or since the
// window opened, and its close is read from that line alone, so every
// checkout that reads it finds the same close whenever it looks. By days, the
// window closes at its start plus WindowDays, once a commit of the line was
// committed after that moment, and ends at the last commit committed before
// it; by changes, it closes at the WindowChanges-th commit of the line, at
// that commit's time. Whichever comes first on the line closes it.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// closing is a window's close as its line gives it: the bound that closed it
// ("" while the window is open), the commit it ends at, the moment it
// closed, which fixes its round, and the first-parent commits it spans,
// oldest first.
type closing struct {
	by      string
	commit  string
	at      time.Time
	commits []string
}

// windowClose reads the close of the window that began at from (fromTime)
// from origin's default branch up to tip, as stored: one local git command.
// A start the first-parent line does not hold is an error, never an open
// window.
func windowClose(ctx context.Context, root string, c Config, from string, fromTime time.Time, tip string) (closing, error) {
	if from == tip {
		return closing{}, nil
	}
	on, err := isAncestor(ctx, root, from, tip)
	if err != nil {
		return closing{}, err
	}
	if !on {
		return closing{}, fmt.Errorf("the spot-check's window starts at %s, which origin's default branch does not hold", from[:12])
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", maxListingBytes, "log", "--first-parent", "--reverse", "--format=%H %ct %P", from+".."+tip, "--")
	if err != nil {
		return closing{}, fmt.Errorf("reading the spot-check's window: %s", redact(err))
	}
	daysAt := fromTime.Add(time.Duration(c.WindowDays) * 24 * time.Hour)
	last := from
	var commits []string
	for i, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		var secs int64
		if len(f) >= 2 {
			secs, err = strconv.ParseInt(f[1], 10, 64)
		}
		if len(f) < 3 || err != nil || !gitutil.IsFullSHA(f[0]) {
			return closing{}, errors.New("reading the spot-check's window: git answered a line of an unexpected shape")
		}
		if i == 0 && f[2] != from {
			return closing{}, fmt.Errorf("the spot-check's window starts at %s, which origin's default branch's first-parent line does not hold", from[:12])
		}
		at := time.Unix(secs, 0).UTC()
		if at.After(daysAt) {
			return closing{by: ClosedByDays, commit: last, at: daysAt, commits: commits}, nil
		}
		last, commits = f[0], append(commits, f[0])
		if c.WindowChanges > 0 && len(commits) == c.WindowChanges {
			return closing{by: ClosedByChanges, commit: f[0], at: later(at, fromTime), commits: commits}, nil
		}
	}
	return closing{}, nil
}

// later is the later of a and b.
func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
