package machineload

import (
	"math"
	"path"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// The waiter's gate is read from the process table, never from a file another
// account could write (iss-2610140638074587). The hold under ~/.abcd.noindex/
// is the caller's own, so a second account on the machine never sees it; and a
// file every account could write would let one account hold every other
// account's gate shut. The process table shows every account's processes, so
// the gate stays shut while any of these is live outside the waiter's own
// process and ancestors:
//
//   - a `make preflight` that is not queued at its own gate: it is running its
//     gates or its tests;
//   - an `abcd implement load --wait` queued ahead of this one, or running the
//     command it gates.
//
// Every waiter reads the same table and orders the queue the same way, by pid,
// lowest first, so two waiters at one gate agree on which goes first without
// writing anything. Each is judged from its arguments, never from a pattern
// over a command line, as a race run is (race.go).

// makePrograms are the base names make runs under.
var makePrograms = []string{"make", "gmake"}

// abcdProgram is the base name of abcd's binary; a plugin build carries a
// platform suffix (abcd-darwin-arm64, cut to fifteen characters by Linux).
const abcdProgram = "abcd"

// psProgram is the base name of the ps a waiter reads the machine with on
// macOS: its one child while it waits.
const psProgram = "ps"

func isMakeName(name string) bool { return slices.Contains(makePrograms, name) }

func isAbcdName(name string) bool {
	return name == abcdProgram || strings.HasPrefix(name, abcdProgram+"-")
}

// isProgramWord reports whether word names one of the programs the gate reads
// the arguments of: go, make, or abcd.
func isProgramWord(word string) bool {
	b := path.Base(word)
	return b == GoProgram || isMakeName(b) || isAbcdName(b)
}

// makeValueFlags are make's options whose value is the next word.
var makeValueFlags = []string{"-C", "-f", "-I", "-o", "-W", "--directory", "--file", "--makefile",
	"--include-dir", "--old-file", "--assume-old", "--new-file", "--assume-new", "--what-if"}

// isPreflightMake reports whether args, a make's arguments after the program
// name, name the goal preflight: a word that is neither a flag, the value of a
// flag that takes the next word, nor a variable assignment.
func isPreflightMake(args []string) bool {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case slices.Contains(makeValueFlags, a):
			i++
		case strings.HasPrefix(a, "-"), strings.Contains(a, "="):
		case a == "preflight":
			return true
		}
	}
	return false
}

// isLoadWaiter reports whether args, abcd's arguments after the program name,
// run `implement load` with --wait set before any `--` (after which the words
// are the gated command's). A flag before the verb path (--json) is skipped.
func isLoadWaiter(args []string) bool {
	var verb []string
	wait := false
	for _, a := range args {
		if a == "--" {
			break
		}
		if v, ok := strings.CutPrefix(a, "--wait="); ok {
			b, err := strconv.ParseBool(v)
			wait = err == nil && b
			continue
		}
		if a == "--wait" {
			wait = true
			continue
		}
		if !strings.HasPrefix(a, "-") && len(verb) < 2 {
			verb = append(verb, a)
		}
	}
	return wait && slices.Equal(verb, []string{"implement", "load"})
}

// GateCandidates returns, in pid order, the processes the gate asks for their
// arguments: those executing go (a race run), make (a preflight) or abcd (a
// waiter), outside self's process and its ancestors.
func GateCandidates(snap Snapshot, self Self) []int {
	chain := lineage(snap.Procs, self.PID)
	var out []int
	for _, p := range snap.Procs {
		if chain[p.PID] {
			continue
		}
		if p.Name == GoProgram || isMakeName(p.Name) || isAbcdName(p.Name) {
			out = append(out, p.PID)
		}
	}
	sort.Ints(out)
	return out
}

// Ahead is what the process table holds ahead of a waiter at the gate.
type Ahead struct {
	// Preflights are the make preflights running, not queued at their gate.
	Preflights []int
	// Waiters are the waiters queued ahead of this one (a lower pid) or
	// running the command they gate.
	Waiters []int
}

// QueueAhead reads what is ahead of self at the gate from one snapshot and the
// candidates' arguments (args, keyed by pid, the program name dropped). A
// waiter with a child other than ps is running its command. A preflight with a
// waiter beneath it that is not running a command is queued at its gate, and
// that waiter is its place in the queue.
func QueueAhead(snap Snapshot, self Self, args map[int][]string) Ahead {
	var out Ahead
	if !snap.HasProcs {
		return out
	}
	byPID := make(map[int]Proc, len(snap.Procs))
	running := map[int]bool{}
	for _, p := range snap.Procs {
		byPID[p.PID] = p
	}
	for _, p := range snap.Procs {
		if p.Name != psProgram {
			if parent, ok := byPID[p.PPID]; ok && isAbcdName(parent.Name) {
				running[p.PPID] = true
			}
		}
	}
	var waiting []int
	for _, pid := range GateCandidates(snap, self) {
		p := byPID[pid]
		if !isAbcdName(p.Name) || !isLoadWaiter(args[pid]) {
			continue
		}
		if !running[pid] {
			waiting = append(waiting, pid)
		}
		if running[pid] || pid < self.PID {
			out.Waiters = append(out.Waiters, pid)
		}
	}
	for _, pid := range GateCandidates(snap, self) {
		if !isMakeName(byPID[pid].Name) || !isPreflightMake(args[pid]) {
			continue
		}
		queued := false
		for _, w := range waiting {
			if Descends(snap.Procs, w, pid) {
				queued = true
				break
			}
		}
		if !queued {
			out.Preflights = append(out.Preflights, pid)
		}
	}
	return out
}

// GoTestParallelism is the -p a `go test` run started now should take, so one
// run cannot saturate the machine (iss-2610102020442624): the cores the
// machine can still give, which is twice the online cores (the waiter's
// default limit, DefaultWaitFactor) less the one-minute load rounded up,
// never more than the cores (go's own default) and never fewer than a quarter
// of them, or one, so a run on a loaded machine still makes progress. An
// unreadable load leaves go's default of the cores.
func GoTestParallelism(snap Snapshot) int {
	cores := snap.Cores
	if cores < 1 {
		cores = runtime.NumCPU()
	}
	if !snap.HasLoad {
		return cores
	}
	floor := max(1, cores/4)
	budget := DefaultWaitFactor*cores - int(math.Ceil(snap.Load1))
	return min(cores, max(floor, budget))
}
