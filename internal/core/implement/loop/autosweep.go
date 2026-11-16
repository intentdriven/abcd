package loop

// autosweep.go is where the loop runs the sweep abcd runs on its own
// (iss-2611150649557869, dcn-2611161331159806): once a landing sees its pull
// request merged, after the step's save has written the lane done (a step's
// landing, and the drain's own finish of a merged lane), at the end of a drain
// move that ends the drain, and at the end of a terminal drive. A build run
// ends at its last landing, which has swept already. The sweep itself is the
// front door's to install (SetAutoSweeper): the sweep package reads the loop's
// runs, so the loop cannot import it. Whatever the sweep comes to, the act it
// follows stands: its answer is a line for that act's output, never an error.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// AutoSweeper runs the automatic sweep of the repository at repoRoot under
// ctx and answers the line the act it follows reports, "" for none. It reads
// its own setting: off, it removes nothing.
type AutoSweeper func(ctx context.Context, repoRoot string) string

var (
	autoSweepMu sync.Mutex
	autoSweeper AutoSweeper
)

// autoSweepTimeout bounds one automatic sweep, so a slow forge or origin
// never holds up the act it follows for longer.
const autoSweepTimeout = 10 * time.Minute

// SetAutoSweeper installs f as the automatic sweep and answers a restore of
// the one before. nil installs none: nothing is swept.
func SetAutoSweeper(f AutoSweeper) (restore func()) {
	autoSweepMu.Lock()
	defer autoSweepMu.Unlock()
	prev := autoSweeper
	autoSweeper = f
	return func() {
		autoSweepMu.Lock()
		defer autoSweepMu.Unlock()
		autoSweeper = prev
	}
}

// AutoSweep runs the installed sweep once, bounded by autoSweepTimeout and by
// ctx (nil is none), and answers its line, cleaned for the act's output. No
// sweep installed, or a ctx already ended, sweeps nothing and answers "". A
// panic in the sweep is answered as its fault: the act it follows stands.
func AutoSweep(ctx context.Context, repoRoot string) (line string) {
	autoSweepMu.Lock()
	f := autoSweeper
	autoSweepMu.Unlock()
	if f == nil {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, autoSweepTimeout)
	defer cancel()
	defer func() {
		if p := recover(); p != nil {
			line = fmt.Sprintf("abcd's own sweep stopped on a fault (%v); the act it followed stands", p)
		}
		line = termsafe.Sanitize(fsutil.RedactHome(line))
	}()
	return f(ctx, repoRoot)
}
