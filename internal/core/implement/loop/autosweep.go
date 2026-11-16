package loop

// autosweep.go is where the loop runs the sweep abcd runs on its own
// (iss-2611150649557869, dcn-2611161331159806, dcn-2611161459048806): once a
// landing sees its pull request merged, after the step's save has written the
// lane done (a step's landing, the drain's own finish of a merged lane, and a
// build's start or resume that closes a merged lane), at the end of a drain
// move that ends the drain, and at the end of a terminal drive that was not
// killed, halted or stopped. A build run ends at its last landing, which has
// swept already. The sweep itself is the front door's to install
// (SetAutoSweeper): the sweep package reads the loop's runs, so the loop
// cannot import it. Whatever the sweep comes to, the act it follows stands:
// its answer is a line for that act's output, never an error.

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

// autoSweepTimeout bounds one automatic sweep as a whole: its fetch and the
// reservations' fetch, every read and forge call, and each delete's push on
// origin all end with it, and its worktree half starts no removal once it has
// ended (a removal begun is never cut off, so the bound can be passed by one
// removal's length, never by the run's). Two minutes is the sweep's own bound
// on any one read of origin or git (remoteNetTimeout, gitReadTimeout), so
// the whole sweep takes no longer than one of its reads may, and it leaves
// most of a lane's ten-minute command bound to the act it follows. A backlog
// that does not fit is not lost: every row is judged again, with the same
// proofs, by the next landing's sweep or the drain's end.
const autoSweepTimeout = 2 * time.Minute

// autoSweepFaultCap bounds the text of a panic the sweep is answered with,
// in runes.
const autoSweepFaultCap = 480

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

// autoSweep runs the installed sweep once, bounded by autoSweepTimeout, by
// ctx (nil is none) and by stop (nil is none), a person's stop of the drive
// it runs in: closed before the sweep starts, nothing is swept; closed while
// it runs, its context ends, so it stops between removals. It answers the
// sweep's line, cleaned for one line of the act's output. No sweep installed,
// or a ctx already ended, sweeps nothing and answers "". A panic in the sweep
// is answered as its fault: the act it follows stands.
func autoSweep(ctx context.Context, stop <-chan struct{}, repoRoot string) (line string) {
	autoSweepMu.Lock()
	f := autoSweeper
	autoSweepMu.Unlock()
	if f == nil {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil || closed(stop) {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, autoSweepTimeout)
	defer cancel()
	if stop != nil {
		go func() {
			select {
			case <-stop:
				cancel()
			case <-ctx.Done():
			}
		}()
	}
	defer func() {
		if p := recover(); p != nil {
			line = "abcd's own sweep stopped on a fault (" + bounded(fmt.Sprint(p), autoSweepFaultCap) + "); the act it followed stands"
		}
		line = termsafe.SanitizeLine(fsutil.RedactHome(line))
	}()
	return f(ctx, repoRoot)
}

// closed reports whether stop has been closed; a nil stop never is.
func closed(stop <-chan struct{}) bool {
	if stop == nil {
		return false
	}
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

// AutoSweepForTest is autoSweep, for a front door's test to prove that the
// sweep it installs is the one the loop runs.
func AutoSweepForTest(ctx context.Context, repoRoot string) string {
	return autoSweep(ctx, nil, repoRoot)
}
