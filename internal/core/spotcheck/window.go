package spotcheck

// window.go is the trigger (spc-2610301925429250 approach 3): a window spans
// everything since the last cycle, or since the window opened, and closes at
// the first landing observed once either bound has passed.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// windowClosedBy is the bound a window that began at from has passed by now,
// changes first-parent commits later: ClosedByDays once WindowDays whole
// days have gone by, ClosedByChanges once WindowChanges (when set) commits
// have landed, the days first when both have; "" when neither has.
func windowClosedBy(c Config, from time.Time, changes int, now time.Time) string {
	if !now.Before(from.Add(time.Duration(c.WindowDays) * 24 * time.Hour)) {
		return ClosedByDays
	}
	if c.WindowChanges > 0 && changes >= c.WindowChanges {
		return ClosedByChanges
	}
	return ""
}

// changesSince counts the first-parent commits from from to tip, both full
// object names this checkout holds, as stored: one local git command.
func changesSince(ctx context.Context, root, from, tip string) (int, error) {
	if from == tip {
		return 0, nil
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", 64, "rev-list", "--first-parent", "--count", from+".."+tip, "--")
	if err != nil {
		return 0, fmt.Errorf("counting the changes since the spot-check's last window: %s", redact(err))
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("counting the changes since the spot-check's last window: git answered %q", strings.TrimSpace(string(out)))
	}
	return n, nil
}
