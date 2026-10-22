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
//     command it gates (a race gate's waiter, `--race-gate`, never is).
//
// That is with one slot, the default (Ahead.Shut): above one, the running
// preflights and the waiters running their commands shut it only once they
// take every slot. A race waiter (`--race-gate`) is in a queue of its own
// (RaceAhead), and never in the first gate's.
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
// macOS: one of its children while it waits.
const psProgram = "ps"

// errandPrograms are the base names of a waiter's own children while it
// waits: the ps it reads the machine with, and the git and gh it renews its
// holder's reservations through (iss-2610170926295770). A child named one of
// them is the waiter's errand, never its gated command, so it marks no waiter
// running; the gated command is make or go in practice, and a waiter gating
// git or gh itself would read as queued while it runs.
var errandPrograms = []string{psProgram, "git", "gh"}

func isErrandName(name string) bool { return slices.Contains(errandPrograms, name) }

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
// run a waiter at either gate: `implement load` with --wait (the first gate)
// or --race-gate (the race gate) set before any `--` (after which the words
// are the gated command's). A flag before the verb path (--json) is skipped.
func isLoadWaiter(args []string) bool {
	wait, race := waiterFlags(args)
	return wait || race
}

// isRaceWaiter reports whether args are a waiter's at the race gate:
// `implement load --race-gate`, the flag before any `--`. A race waiter waits
// for the race stage and is never in the first gate's queue (RaceAhead).
//
// The race gate is spelt without --wait on purpose. Every abcd released
// before it reads any `implement load --wait` as a waiter at its first gate,
// so a preflight at a race gate spelt `--wait --race` would read to an older
// waiter as queued, not in its test stages, and that waiter would open beside
// it, its own ungated race stage overlapping this one's.
func isRaceWaiter(args []string) bool {
	_, race := waiterFlags(args)
	return race
}

// waiterFlags reads args as isLoadWaiter does: whether they run `implement
// load` with --wait set, and whether --race-gate is set, each before any `--`.
func waiterFlags(args []string) (wait, race bool) {
	var verb []string
	for _, a := range args {
		if a == "--" {
			break
		}
		if v, ok := strings.CutPrefix(a, "--wait="); ok {
			b, err := strconv.ParseBool(v)
			wait = err == nil && b
			continue
		}
		if v, ok := strings.CutPrefix(a, "--race-gate="); ok {
			b, err := strconv.ParseBool(v)
			race = err == nil && b
			continue
		}
		if a == "--wait" {
			wait = true
			continue
		}
		if a == "--race-gate" {
			race = true
			continue
		}
		if !strings.HasPrefix(a, "-") && len(verb) < 2 {
			verb = append(verb, a)
		}
	}
	if !slices.Equal(verb, []string{"implement", "load"}) {
		return false, false
	}
	return wait, race
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
	// Busy is the work in its test stages: each preflight in Preflights and
	// each waiter in Waiters running its command, counted once where one runs
	// beneath the other (`--wait -- make preflight` is one unit), in pid order.
	Busy []int
	// Queued is the waiters in Waiters queued ahead, not running a command.
	Queued []int
}

// Shut reports which of the queue's conditions hold the first gate shut when
// slots units of work may be in their test stages at once (the
// load.preflights setting; zero or less reads as one): every unit in Busy
// takes a slot, and the gate is shut on preflight or on waiter once they are
// all taken, by the kind of unit that takes them. A waiter queued ahead shuts
// it whatever slots is, so a free slot never lets a waiter jump the queue.
// With one slot this is the gate as it read before slots existed: any running
// preflight shuts it on preflight, and any waiter ahead or running its command
// on waiter.
func (a Ahead) Shut(slots int) (preflight, waiter bool) {
	full := len(a.Busy) >= max(slots, 1)
	running := len(a.Waiters) > len(a.Queued)
	return full && len(a.Preflights) > 0, len(a.Queued) > 0 || (full && running)
}

// QueueAhead reads what is ahead of self at the first gate from one snapshot
// and the candidates' arguments (args, keyed by pid, the program name
// dropped). A waiter with a child other than ps is running its command. A
// preflight with a first gate's waiter beneath it that is not running a
// command is queued at its gate, and that waiter is its place in the queue. A
// waiter beneath a waiter running its command takes that ancestor's place,
// ahead of every queued waiter. A race waiter (isRaceWaiter) is never in this
// queue: it waits at the race gate (RaceAhead), and a preflight whose waiter
// is one has passed its first gate and is in its test stages.
func QueueAhead(snap Snapshot, self Self, args map[int][]string) Ahead {
	var out Ahead
	if !snap.HasProcs {
		return out
	}
	t := readGate(snap, self, args)
	// A waiter beneath a waiter running its command (`--wait -- make
	// preflight`, whose preflight runs its own gate line) is part of work that
	// already won its turn, so it takes that ancestor's place, ahead of every
	// queued waiter. Queued behind them instead, it would wait on a waiter that
	// waits on its ancestor, and neither would move until its timeout.
	turnWon := false
	for _, pid := range t.candidates {
		if t.chain[pid] && isLoadWaiter(args[pid]) {
			turnWon = true
		}
	}
	runningW, waiting := t.waiters(args, false)
	if !turnWon {
		out.Queued = t.queuedAhead(waiting)
	}
	out.Waiters = append(append([]int(nil), runningW...), out.Queued...)
	sort.Ints(out.Waiters)
	for _, pid := range t.candidates {
		if t.chain[pid] || !isMakeName(t.byPID[pid].Name) || !isPreflightMake(args[pid]) {
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
	// Each unit of work in its test stages once: a preflight or a waiter
	// running its command, unless it runs beneath another of them.
	units := append(append([]int(nil), out.Preflights...), runningW...)
	for _, pid := range units {
		beneath := false
		for _, other := range units {
			if Descends(snap.Procs, pid, other) {
				beneath = true
				break
			}
		}
		if !beneath {
			out.Busy = append(out.Busy, pid)
		}
	}
	sort.Ints(out.Busy)
	return out
}

// RaceAhead returns, in pid order, the race waiters (isRaceWaiter) ahead of
// self at the race gate: those queued ahead of it, by the rule the first
// gate's queue keeps (ahead by start and with the lower pid, queuedAhead), and
// those running their command, outside
// self's process and its ancestors. A waiter at a first gate, and a preflight
// in its unit stage, are not ahead of it there; the race runs that shut it are
// RaceRuns.
func RaceAhead(snap Snapshot, self Self, args map[int][]string) []int {
	if !snap.HasProcs {
		return nil
	}
	t := readGate(snap, self, args)
	runningW, waiting := t.waiters(args, true)
	out := append(append([]int(nil), runningW...), t.queuedAhead(waiting)...)
	sort.Ints(out)
	return out
}

// IsRaceCommand reports whether cmd, a command a waiter gates (the words after
// `--`, its program first), is a race run: its program is go and its go
// command is `test` with -race, read as a race run's arguments are
// (isRaceTest). A shell or a make that runs one is not read into.
func IsRaceCommand(cmd []string) bool {
	return len(cmd) > 0 && path.Base(cmd[0]) == GoProgram && isRaceTest(cmd[1:])
}

// gatedCommand is the command a waiter's args gate: the words after the first
// `--`, or none.
func gatedCommand(args []string) []string {
	for i, a := range args {
		if a == "--" {
			return args[i+1:]
		}
	}
	return nil
}

// RaceCommandWaiters returns, in pid order, the first gate's waiters outside
// self's process and its ancestors that are running a command that is a race
// run (`--wait -- go test -race`, IsRaceCommand). Such a waiter is a race run
// for both gates from the moment it opens: its mark shows it running before
// the go test it starts is in the table, and once that go test is, RaceRuns
// names it too. A queued one is not.
func RaceCommandWaiters(snap Snapshot, self Self, args map[int][]string) []int {
	if !snap.HasProcs {
		return nil
	}
	t := readGate(snap, self, args)
	running, _ := t.waiters(args, false)
	var out []int
	for _, pid := range running {
		if IsRaceCommand(gatedCommand(args[pid])) {
			out = append(out, pid)
		}
	}
	return out
}

// gateTable is one snapshot read for a gate: the processes by pid, the waiters
// running a command, self's process and ancestors, and the gate candidates.
type gateTable struct {
	self       Self
	byPID      map[int]Proc
	running    map[int]bool
	chain      map[int]bool
	candidates []int
}

func readGate(snap Snapshot, self Self, args map[int][]string) gateTable {
	t := gateTable{self: self, byPID: make(map[int]Proc, len(snap.Procs)), running: map[int]bool{}}
	for _, p := range snap.Procs {
		t.byPID[p.PID] = p
	}
	for _, p := range snap.Procs {
		if !isErrandName(p.Name) {
			if parent, ok := t.byPID[p.PPID]; ok && isAbcdName(parent.Name) {
				t.running[p.PPID] = true
			}
		}
	}
	t.chain = lineage(snap.Procs, self.PID)
	t.candidates = GateCandidates(snap, self)
	return t
}

// waiters splits the waiters outside self's chain at one gate, the race gate's
// when race is set and the first gate's otherwise, into those running their
// command and those queued, each in pid order.
func (t gateTable) waiters(args map[int][]string, race bool) (running, waiting []int) {
	for _, pid := range t.candidates {
		if t.chain[pid] || !isAbcdName(t.byPID[pid].Name) || !isLoadWaiter(args[pid]) || isRaceWaiter(args[pid]) != race {
			continue
		}
		if t.running[pid] {
			running = append(running, pid)
		} else {
			waiting = append(waiting, pid)
		}
	}
	return running, waiting
}

// queuedAhead returns, in pid order, the queued waiters at one gate ahead of
// self: ahead by start (queueOrder) and with the lower pid as well, at the
// first gate (QueueAhead) and the race gate (RaceAhead) alike. Absent from the
// table, self has no start time, and queueOrder puts it behind every one that
// has.
//
// A waiter ahead by start is ahead only when its pid is lower too. A waiter
// built before the start order (iss-2610161634043273) waits on every lower
// pid, and this code cannot tell it from one built after: were this waiter to
// wait on a higher pid that started earlier, and that one run the older code,
// each would wait on the other until its timeout. So every wait, whichever
// code makes it, points at a lower pid, and no chain of waits can close into
// a cycle, whatever each waiter's own read of the table says. Two waiters
// whose start and pid disagree, as across a pid wrap, wait on neither: the
// earlier start, with the higher pid, is no longer held behind every later
// start with a lower one, and the waiter's settle wait and re-check, within
// one account its hold, and across accounts the command form's look just
// before it starts its command (JustOpened), keep the two from running at
// once (implement.WaitLoad).
func (t gateTable) queuedAhead(waiting []int) []int {
	queue := []Proc{{PID: t.self.PID}}
	if p, ok := t.byPID[t.self.PID]; ok {
		queue[0] = p
	}
	for _, pid := range waiting {
		queue = append(queue, t.byPID[pid])
	}
	var out []int
	for _, pid := range queueOrder(queue) {
		if pid == t.self.PID {
			break
		}
		if pid < t.self.PID {
			out = append(out, pid)
		}
	}
	sort.Ints(out)
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
// changes nothing. No cycle rests on the grouping any longer: QueueAhead and
// RaceAhead count a waiter ahead only when it has the lower pid too
// (queuedAhead), and that alone
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
// never more than the cores divided among the slots preflights may be in
// their test stages at once (load.preflights; zero or less reads as one, and
// one is go's own default of the cores) and never fewer than a quarter of the
// cores, or one, so a run on a loaded machine still makes progress. With two
// slots, two concurrent stages each take at most half the cores: the load a
// second stage starts under lags the first stage's real demand by up to a
// minute, so the load alone would give each the whole machine. An unreadable
// load leaves the cores divided among the slots.
func GoTestParallelism(snap Snapshot, slots int) int {
	cores := snap.Cores
	if cores < 1 {
		cores = runtime.NumCPU()
	}
	share := max(1, cores/max(slots, 1))
	if !snap.HasLoad {
		return share
	}
	floor := max(1, cores/4)
	budget := DefaultWaitFactor*cores - int(math.Ceil(snap.Load1))
	return min(share, max(floor, budget))
}

// JustOpened returns, in pid order, the waiters at one gate, the race gate's
// when race is set and the first gate's otherwise, outside self's process and
// its ancestors, that are running their command and opened within window of
// the snapshot: a waiter with a child other than ps is running its command,
// and it opened when its oldest such child started. An opener reads it just
// before it starts its command (implement.WaitLoad): two waiters whose start
// and pid disagree wait on neither, and across accounts no hold keeps them
// from opening in one moment, so each looks once more, and the one with the
// higher pid stands back from the other. A waiter at the other gate did not
// open beside this one: a race waiter's preflight passed its first gate
// earlier, and a first gate's opener never stands back from it, as an abcd
// released before the race gate, which reads no race waiter at all, does not.
// A child under a second old on macOS reads an age of zero, which is within
// any window.
func JustOpened(snap Snapshot, self Self, args map[int][]string, window time.Duration, race bool) []int {
	if !snap.HasProcs {
		return nil
	}
	byPID := make(map[int]Proc, len(snap.Procs))
	for _, p := range snap.Procs {
		byPID[p.PID] = p
	}
	since := map[int]time.Duration{}
	for _, p := range snap.Procs {
		if isErrandName(p.Name) {
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
		if chain[pid] || age > window || !isLoadWaiter(args[pid]) || isRaceWaiter(args[pid]) != race {
			continue
		}
		out = append(out, pid)
	}
	sort.Ints(out)
	return out
}
