package implement

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// A narrow run (machineload.IsNarrowCommand) waits at a gate of its own. On
// 2026-11-04 about thirty builders each queued one package's test behind whole
// preflights, at a load of 12 on 16 cores with 96% of memory free, until the
// 30-minute wait ran out; the machine's load and memory were the real limits and
// neither was near. So a narrow run is shut only by:
//
//   - the load limit and the memory floor, as every gate is;
//   - its own limit: narrow runs running, of any account, and those queued ahead
//     of it, reach load.narrow_runs;
//   - an overdue whole run (below);
//   - its account's short hold, so the next waiter sees this one in the table.
//
// It takes no whole-run slot and no place in the whole-run queue, it reads no
// race run, and it neither judges nor takes the first gate's hold; a whole run
// counts it through the load figure alone. A race run is never narrow: it stays
// where the rule keeping race stages apart can see it.
//
// A whole run keeps its place honestly. Sharing the load limit, narrow runs could
// otherwise take every dip the whole run waits for, and an older abcd also counts
// them in its slots. So while a whole-run waiter that started earlier has waited
// WaitNarrowYieldAfter with a whole slot free, narrow runs stop being admitted, for
// at most WaitNarrowYieldCap in any one wait. The gap in admissions lets the
// running ones finish and the load fall, which opens that whole run; one still
// shut after the cap is shut by something no narrow run does, such as its own
// account's stricter limit, and costs a narrow run minutes, not the wait.
//
// Mixed versions, since the gate crosses accounts and shares no file with them.
// Every wait here points at an earlier start or at a command already running, so
// no cycle forms with an abcd built before the class, which reads a narrow waiter
// as a waiter like any other: it queues behind one that started earlier and counts
// one running in a slot, more cautious than this code, never less. It reads no
// load-wait-narrow.json, and it names load.narrow_runs in a config.json as a key
// nothing reads, while load.preflights beside it still counts. This code reads
// that abcd's waiter by its words: a whole run's is counted in full, so no whole
// run passes its limit. One gating a narrow-shaped command queues as a whole run
// and may not open, so waiters ahead that alone keep a narrow run shut are passed
// after WaitNarrowQueueCap.

// NarrowRunsKey is the machine configuration key of the narrow runs' limit. It
// has no environment variable: a repository's Makefile can set one, and the limit
// is the person's own, as a repository file never sets it.
const NarrowRunsKey = "load.narrow_runs"

// MaxNarrowRuns bounds the limit. The load limit and the memory floor are what
// bound the machine, so it only catches a slip of the keyboard.
const MaxNarrowRuns = 32

// NarrowRunCores is what one narrow run takes, in cores, on the machine it was
// measured on, a 16-core Mac Studio: a one-package go test there averaged about
// two across the build and the tests.
const NarrowRunCores = 2

// WaitNarrowHoldFileName is the narrow gate's hold record under ~/.abcd.noindex/.
// No abcd built before the class reads it.
const WaitNarrowHoldFileName = "load-wait-narrow.json"

// The narrow gate's intervals. They are variables only so the front door can read
// them into its help; nothing reassigns them.
var (
	// WaitNarrowSettle is how long a narrow run's hold lasts: long enough for its
	// go test to show in the table, and a narrow run needs no load average to show
	// it, since it is counted against its own limit.
	WaitNarrowSettle = 5 * time.Second
	// WaitNarrowJitterMin and WaitNarrowJitterMax bound the wait between the look
	// that finds the gate free and the re-check that opens it.
	WaitNarrowJitterMin = time.Second
	WaitNarrowJitterMax = 5 * time.Second
	// WaitNarrowYieldAfter is how long a whole-run waiter may wait with a slot free
	// before narrow runs stand aside for it. It opens within a poll and a jitter of
	// a free slot and a load under the limit, about 75 seconds, so this is well past
	// a waiter that is simply about to open.
	WaitNarrowYieldAfter = 5 * time.Minute
	// WaitNarrowYieldCap is the most one narrow wait stands aside: a one-minute load
	// average shows most of a fall inside it.
	WaitNarrowYieldCap = 5 * time.Minute
	// WaitNarrowQueueCap is how long narrow runs queued ahead may alone keep one shut
	// with a place free. A waiter of this code takes a free place within a poll, a
	// jitter and its account's hold, under a minute even on a slow table, and each
	// that leaves starts the count again, so a queue that is moving never reaches
	// this; one overtaking another costs the other nothing it can notice.
	WaitNarrowQueueCap = 2 * time.Minute
)

// narrowJitter is a random wait from WaitNarrowJitterMin to WaitNarrowJitterMax.
func narrowJitter() time.Duration {
	return WaitNarrowJitterMin + rand.N(WaitNarrowJitterMax-WaitNarrowJitterMin+1)
}

// DefaultNarrowRuns is the limit with none set: half the online cores, which
// leaves the other half to whole runs, divided by what one narrow run takes, and
// one at the least. Sixteen cores admit four.
func DefaultNarrowRuns(cores int) int {
	return max(1, cores/2/NarrowRunCores)
}

// narrowLimit is the limit a look judges by: the setting (zero for none), or the
// default from the online cores.
func narrowLimit(set, cores int) int {
	if set > 0 {
		return set
	}
	if cores < 1 {
		cores = runtime.NumCPU()
	}
	return DefaultNarrowRuns(cores)
}

// NarrowRuns reads load.narrow_runs from the machine configuration alone
// (layered.MachineRoots), as PreflightSlots reads load.preflights, and returns it
// with where it came from; zero is unset, and a look then derives it from the
// cores (narrowLimit). A value outside one to MaxNarrowRuns is refused naming the
// key. A config.json it cannot read is read past: PreflightSlots, read at the same
// look, names the file.
func NarrowRuns(home string) (int, string, error) {
	roots, _ := layered.MachineRoots(home)
	if roots.HomeRefused {
		return 0, SlotsDefault, nil
	}
	stack, err := layered.Load(layered.Config, roots)
	if err != nil {
		return 0, SlotsDefault, nil
	}
	if _, err := stack.Lookup(NarrowRunsKey); err != nil {
		return 0, SlotsDefault, nil
	}
	_ = stack.Claim("load", loadKeys...)
	v, err := layered.Get(stack, NarrowRunsKey, 0, func(n int) error {
		if n < 1 || n > MaxNarrowRuns {
			return fmt.Errorf("want a whole number from 1 to %d", MaxNarrowRuns)
		}
		return nil
	})
	if err != nil {
		return 0, "", fmt.Errorf("the load gate stops rather than guess how many narrow runs it admits at once: %w", err)
	}
	if v.Layer == layered.Machine {
		return v.V, SlotsMachine, nil
	}
	return 0, SlotsDefault, nil
}

// WaitOnWholeRun: a whole-run waiter that started earlier is overdue, so this
// narrow run stands aside for it (machineload.OverdueWhole).
const WaitOnWholeRun = "whole-run"

// standAside reports whether this wait, at t, still stands aside for an overdue
// whole run: from the first look that finds one, until WaitNarrowYieldCap passes,
// once in a wait.
func (w *holderWatch) standAside(t time.Time) bool {
	if w.asideSince.IsZero() {
		w.asideSince = t
	}
	return t.Sub(w.asideSince) < WaitNarrowYieldCap
}
