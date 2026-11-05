package implement

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/machineload"
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
//   - its account's own short hold, and its account's whole runs' settle.
//
// It takes no whole-run slot and no place in the whole-run queue, it reads no race
// run, and it takes no other gate's hold; a whole run counts it through the load
// figure alone. A race run is never narrow: it stays where the rule keeping race
// stages apart can see it.
//
// A narrow run asks for this gate with a mark in its argv, machineload.NarrowMark,
// which every reader of the table sees alike. Without it the same command is a whole
// run.
//
// A whole run keeps its place honestly. Sharing the load limit, narrow runs could
// otherwise take every dip the whole run waits for, and an older abcd also counts
// them in its slots. So while a whole-run waiter that started earlier has been
// queued WaitNarrowYieldAfter or more and a whole slot is free, narrow runs stop being admitted, for
// at most WaitNarrowYieldCap spent in any one wait. The gap in admissions lets the
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
// that abcd's waiter by its words, and none carries the mark, so each is counted in
// full and no whole run passes its limit. A narrow waiter queued ahead can be shut
// by its own account's stricter settings, which no other account can see, so waiters
// ahead that alone keep a narrow run shut are passed after WaitNarrowQueueCap.

// NarrowRunsKey is the machine configuration key of the narrow runs' limit. It has
// no environment variable of its own, and no repository file sets it; like
// load.preflights it follows the process's home.
const NarrowRunsKey = "load.narrow_runs"

// MaxNarrowRuns bounds the limit. The load limit and the memory floor are what
// bound the machine, so it only catches a slip of the keyboard.
const MaxNarrowRuns = 32

// NarrowRunCores is what one narrow run is bound to, in cores (NarrowEnv): a ceiling
// on what they took as measured on a 16-core Mac Studio on 2026-11-04, from about 0.2
// for a three-package run mostly waiting on git to under 2 for
// internal/core/implement, 14.5 seconds on at most 27 seconds of CPU. Almost none of
// abcd's tests are parallel, so a bound at two costs a test run little.
const NarrowRunCores = 2

// WaitNarrowHoldFileName is the narrow gate's hold record under ~/.abcd.noindex/.
// No abcd built before the class reads it.
const WaitNarrowHoldFileName = "load-wait-narrow.json"

// The narrow gate's intervals. They are variables only so the front door can read
// them into its help; nothing reassigns them.
var (
	// WaitNarrowSettle is how long a narrow run's hold lasts: its go test shows in
	// the table at once, and thirty seconds is enough for a one-minute load average
	// to show a good part of its two cores before the next opens, so four take at
	// least a minute and a half to open rather than a burst the load never saw.
	WaitNarrowSettle = 30 * time.Second
	// WaitNarrowJitterMin and WaitNarrowJitterMax bound the wait between the look
	// that finds the gate free and the re-check that opens it.
	WaitNarrowJitterMin = time.Second
	WaitNarrowJitterMax = 5 * time.Second
	// WaitNarrowYieldAfter is how long a whole-run waiter may be queued, with a slot
	// free, before narrow runs stand aside for it. It opens within a poll and a jitter of
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

// defaultNarrowRuns is the limit with none set: half the online cores divided by
// what one narrow run is bound to, and one at the least, so an account's look admits
// up to half the machine on four cores or more. Each other account can add one run over that, since a look
// always keeps its own account one place (narrowCount). Sixteen cores admit four,
// eight cores in all; how much of that a look grants beside whole runs is narrowRoom's.
func defaultNarrowRuns(cores int) int {
	return max(1, cores/2/NarrowRunCores)
}

// narrowRoom is how many narrow runs a look admits. Narrow runs and whole runs share
// one budget, the online cores, counted in the places each is bound to: a narrow run
// in its NarrowRunCores, a whole run in its stage's go test -p, the cores divided
// among the slots (machineload.GoTestParallelism). A machine the load shows has room
// for one more run gets the whole limit, since a slot's -p is a ceiling and a run can
// sit far below it, and a settled load (WaitNarrowSettle, WaitSettle) is what says so.
// A machine the load shows busy gets what the whole runs in their slots leave, and
// one place at the least, so a test is never behind them altogether.
func narrowRoom(limit, cores int, load float64, busy, slots int) int {
	if load+NarrowRunCores < float64(cores) {
		return limit
	}
	left := cores - busy*max(1, cores/max(slots, 1))
	return min(limit, max(1, left/NarrowRunCores))
}

// narrowCount is the running narrow runs a look counts against room: all of this
// account's, and of other accounts' never more than leave this account one place.
func narrowCount(snap machineload.Snapshot, self machineload.Self, running []int, room int) int {
	own, other := 0, 0
	for _, p := range snap.Procs {
		if !slices.Contains(running, p.PID) {
			continue
		}
		if p.UID == self.UID {
			own++
		} else {
			other++
		}
	}
	return own + min(other, max(room-1, 0))
}

// NarrowEnv is env as a narrow run's command runs under it, bound to what the gate
// counted it as: one package at a time and NarrowRunCores for the go command, its
// compiler and the test binaries, and a lower GOMAXPROCS of the person's own stands.
// The front door has proved the effective GOFLAGS empty, so this overrides nothing.
// Children a test starts outside go's reach, git among them, are not bound by it.
func NarrowEnv(env []string) []string {
	procs := NarrowRunCores
	out := make([]string, 0, len(env)+2)
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "GOFLAGS="):
		case strings.HasPrefix(kv, "GOMAXPROCS="):
			if n, err := strconv.Atoi(strings.TrimPrefix(kv, "GOMAXPROCS=")); err == nil && n >= 1 {
				procs = min(procs, n)
			}
		default:
			out = append(out, kv)
		}
	}
	return append(out, "GOFLAGS=-p=1", "GOMAXPROCS="+strconv.Itoa(procs))
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
	return defaultNarrowRuns(cores)
}

// NarrowRuns reads load.narrow_runs from the machine configuration alone
// (layered.MachineRoots), as PreflightSlots reads load.preflights, and returns it
// with where it came from; zero is unset, and a look then derives it from the
// cores (narrowLimit). A value outside one to MaxNarrowRuns is refused naming the
// key. A config.json it cannot read is read past: PreflightSlots, read at the same
// look, names the file.
func narrowRuns(home string) (int, string, error) {
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
// whole run, having spent less than WaitNarrowYieldCap of the wait doing so. Only the
// time between looks that each found one is spent: a whole waiter's age says how long
// it has queued, not how long a slot has been free for it, so one reads overdue the
// moment a slot frees and opens about a minute later, and that minute is all it costs.
func (w *holderWatch) standAside(t time.Time) bool {
	if !w.asideLast.IsZero() {
		w.asideSpent += t.Sub(w.asideLast)
	}
	w.asideLast = t
	return w.asideSpent < WaitNarrowYieldCap
}
