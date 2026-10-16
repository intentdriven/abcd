package machineload

import (
	"cmp"
	"math"
	"path"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
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
// Every waiter reads the same table and orders the queue the same way, by when
// each waiter started, earliest first, with a tie going to the lower pid
// (queueOrder), and counts a queued waiter as ahead only when that one is
// ahead by start and has the lower pid as well, so every wait on a queued
// waiter points at a lower pid and no two queued waiters wait on each other,
// without writing anything: two whose start and pid agree agree on which goes
// first, and two whose start and pid disagree wait on neither
// (implement.WaitLoad keeps such a pair from running at once). Pid order
// alone starved a waiter once the machine's process ids wrapped: every later
// waiter with a lower pid queued ahead of it (iss-2610161634043273). Start
// order alone deadlocked against waiters built before it, which still wait on
// every lower pid: a young waiter with a low pid waited on older ones with
// higher pids, and they on it. Each is judged from its arguments, never from
// a pattern over a command line, as a race run is (race.go).

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
// waiter), outside self's process and its ancestors, and the ancestors
// executing abcd, so a waiter can tell that an ancestor is a waiter running
// the command this one is part of.
func GateCandidates(snap Snapshot, self Self) []int {
	chain := lineage(snap.Procs, self.PID)
	var out []int
	for _, p := range snap.Procs {
		if p.PID == self.PID {
			continue
		}
		if chain[p.PID] {
			if isAbcdName(p.Name) {
				out = append(out, p.PID)
			}
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
	// Waiters are the waiters queued ahead of this one (ahead by start, earlier
	// or in the same tie group, and with a lower pid; queueOrder) or running
	// the command they gate, in pid order.
	Waiters []int
}

// QueueAhead reads what is ahead of self at the gate from one snapshot and the
// candidates' arguments (args, keyed by pid, the program name dropped). A
// waiter with a child other than ps is running its command. A preflight with a
// waiter beneath it that is not running a command is queued at its gate, and
// that waiter is its place in the queue. A waiter beneath a waiter running its
// command takes that ancestor's place, ahead of every queued waiter.
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
	chain := lineage(snap.Procs, self.PID)
	candidates := GateCandidates(snap, self)
	// A waiter beneath a waiter running its command (`--wait -- make
	// preflight`, whose preflight runs its own gate line) is part of work that
	// already won its turn, so it takes that ancestor's place, ahead of every
	// queued waiter. Queued behind them instead, it would wait on a waiter that
	// waits on its ancestor, and neither would move until its timeout.
	turnWon := false
	for _, pid := range candidates {
		if chain[pid] && isLoadWaiter(args[pid]) {
			turnWon = true
		}
	}
	var waiting []int
	for _, pid := range candidates {
		p := byPID[pid]
		if chain[pid] || !isAbcdName(p.Name) || !isLoadWaiter(args[pid]) {
			continue
		}
		if running[pid] {
			out.Waiters = append(out.Waiters, pid)
		} else {
			waiting = append(waiting, pid)
		}
	}
	if !turnWon {
		// This waiter's place among the queued ones. Absent from the table, it
		// has no start time, and queueOrder puts it behind every one that has.
		//
		// A waiter ahead by start is ahead only when its pid is lower too. A
		// waiter built before the start order (iss-2610161634043273) waits on
		// every lower pid, and this code cannot tell it from one built after:
		// were this waiter to wait on a higher pid that started earlier, and
		// that one run the older code, each would wait on the other until its
		// timeout. So every wait, whichever code makes it, points at a lower
		// pid, and no chain of waits can close into a cycle, whatever each
		// waiter's own read of the table says. Two waiters whose start and pid
		// disagree, as across a pid wrap, wait on neither: the earlier start,
		// with the higher pid, is no longer held behind every later start with
		// a lower one, and the waiter's settle wait and re-check, within one
		// account its hold, and across accounts the command form's look just
		// before it starts its command (JustOpened), keep the two from running
		// at once (implement.WaitLoad).
		queue := []Proc{{PID: self.PID}}
		if p, ok := byPID[self.PID]; ok {
			queue[0] = p
		}
		for _, pid := range waiting {
			queue = append(queue, byPID[pid])
		}
		for _, pid := range queueOrder(queue) {
			if pid == self.PID {
				break
			}
			if pid < self.PID {
				out.Waiters = append(out.Waiters, pid)
			}
		}
		sort.Ints(out.Waiters)
	}
	for _, pid := range candidates {
		if chain[pid] || !isMakeName(byPID[pid].Name) || !isPreflightMake(args[pid]) {
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

// queueOrder returns the pids of the queued waiters ws in the order they are
// admitted: the order they started in, earliest first, with start times the
// table cannot tell apart ordered by pid, lowest first.
//
// A start time is the read's moment less the process's age, and every age in
// one table is measured back from the same read, so the earlier start is the
// greater age; the moment itself, and how long the read took, which the
// session-name register allows for when it compares a start time with one an
// earlier read recorded, cancel between two waiters of one table, but for
// the read's own length (tens of milliseconds). What is left is the reading's
// grain, which StartTolerance absorbs, as it does for
// the register: two ages within it are a tie, broken by pid.
//
// A tie is not transitive: waiters A, B and C, each within the tolerance of
// the next and A and C apart by more, would make a cycle if each pair were
// judged alone (A before B and B before C by pid, C before A by start), and
// all three would wait on each other until their timeouts. So waiters linked
// by a chain of ties form one group, the groups are ordered by start, and the
// waiters within a group by pid: one order, the same from every waiter that
// reads the same table. Two waiters a hair either side of the tolerance on
// macOS, which reads ages in whole seconds, can see the gap differently at
// one look and disagree for that look, and because a group is a chain, a gap
// read differently can move waiters elsewhere in it; the next look reads it
// afresh. Outside a pid wrap start order and pid order agree, so a regrouping
// changes nothing. No cycle rests on the grouping any longer: QueueAhead
// counts a waiter ahead only when it has the lower pid too, and that alone
// keeps every wait pointing down the pids. The grouping keeps one order for
// the waiters a chain of ties links.
//
// A waiter whose start time cannot be read (no measurable age, which is also
// what Share takes as unknown, or absent from the table) cannot be placed by
// it, so every such waiter queues behind all whose start times are known, in
// pid order among themselves, as the whole queue was ordered before start
// times were read. On macOS a waiter under a second old also reads an age of
// zero, so for its first second it queues last as well.
func queueOrder(ws []Proc) []int {
	var known, unknown []Proc
	for _, w := range ws {
		if w.Age > 0 {
			known = append(known, w)
		} else {
			unknown = append(unknown, w)
		}
	}
	byPID := func(a, b Proc) int { return cmp.Compare(a.PID, b.PID) }
	slices.SortFunc(known, func(a, b Proc) int { return cmp.Or(cmp.Compare(b.Age, a.Age), byPID(a, b)) })
	const tie = StartTolerance * time.Second
	order := make([]int, 0, len(ws))
	for i := 0; i < len(known); {
		j := i + 1
		for j < len(known) && known[j-1].Age-known[j].Age <= tie {
			j++
		}
		group := known[i:j]
		slices.SortFunc(group, byPID)
		for _, w := range group {
			order = append(order, w.PID)
		}
		i = j
	}
	slices.SortFunc(unknown, byPID)
	for _, w := range unknown {
		order = append(order, w.PID)
	}
	return order
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

// JustOpened returns, in pid order, the waiters outside self's process and its
// ancestors that are running their command and opened within window of the
// snapshot: a waiter with a child other than ps is running its command, and it
// opened when its oldest such child started. An opener reads it just before it
// starts its command (implement.WaitLoad): two waiters whose start and pid
// disagree wait on neither, and across accounts no hold keeps them from
// opening in one moment, so each looks once more, and the one with the higher
// pid stands back from the other. A child under a second old on macOS reads an
// age of zero, which is within any window.
func JustOpened(snap Snapshot, self Self, args map[int][]string, window time.Duration) []int {
	if !snap.HasProcs {
		return nil
	}
	byPID := make(map[int]Proc, len(snap.Procs))
	for _, p := range snap.Procs {
		byPID[p.PID] = p
	}
	since := map[int]time.Duration{}
	for _, p := range snap.Procs {
		if p.Name == psProgram {
			continue
		}
		if parent, ok := byPID[p.PPID]; ok && isAbcdName(parent.Name) {
			if a, seen := since[p.PPID]; !seen || p.Age > a {
				since[p.PPID] = p.Age
			}
		}
	}
	chain := lineage(snap.Procs, self.PID)
	var out []int
	for pid, age := range since {
		if chain[pid] || age > window || !isLoadWaiter(args[pid]) {
			continue
		}
		out = append(out, pid)
	}
	sort.Ints(out)
	return out
}
