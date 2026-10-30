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
// take every slot, and a preflight queued at its race gate takes none
// (Ahead.TakingSlots). A race waiter (`--race-gate`) is in a queue of its own
// (RaceAhead), and never in the first gate's.
//
// Every waiter reads the same table and orders the queue the same way, by when
// each waiter started, earliest first, with a tie going to the lower pid
// (queueOrder), and counts every queued waiter ahead of it in that order as
// ahead, whatever its pid, without writing anything. Pid order starved a
// waiter once the machine's process ids wrapped: every later waiter with a
// lower pid queued ahead of it (iss-2610161634043273). Counting a waiter ahead
// only when it had the lower pid as well, as abcd v0.15.1 does, starved one
// too: after a wrap an hour-old waiter with a high pid and a young one with a
// low pid waited on neither, so every young starter competed with the old one
// at each free moment, and the open look stood the old one back for each
// (iss-2610222125175640).
//
// A waiter ahead by start with the higher pid is crossed (Ahead.Crossed). A
// waiter built before the start order (v0.15.0 and earlier) waits on every
// lower pid, and nothing in the table tells it from one built after, since
// both run the same arguments: such a waiter, older with the higher pid,
// waits on this one while this one waits on it. Every cycle of waits passes
// through a crossed wait: every other wait, from this code, v0.15.1's or the
// pid-ordered code, points at a lower pid, and a cycle has to climb back up
// the pids. So implement.WaitLoad stops counting crossed waiters once they
// alone have held its gate shut for implement.WaitCrossedCap, longer than a
// waiter of this code or v0.15.1's, which waits on no later start, shows
// queued while the machine is free to another account. Each is judged from
// its arguments, never from a pattern over a command line, as a race run is
// (race.go).

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
	// Waiters are the waiters queued ahead of this one (ahead by start: earlier,
	// or in the same tie group with a lower pid; queueOrder) or running the
	// command they gate, in pid order.
	Waiters []int
	// Busy is the work running past its first gate: each preflight in
	// Preflights and each waiter in Waiters running its command, counted once
	// where one runs beneath the other (`--wait -- make preflight` is one
	// unit), in pid order.
	Busy []int
	// Queued is the waiters in Waiters queued ahead, not running a command.
	Queued []int
	// Crossed is the waiters in Queued with a higher pid than this one's:
	// ahead by start across a pid wrap. A waiter built before the start order
	// waits on every lower pid, so it may be waiting on this one in turn, and
	// implement.WaitLoad stops counting crossed waiters once they alone have
	// held the gate shut for implement.WaitCrossedCap (Uncrossed).
	Crossed []int
	// AtRaceGate is the preflights in Preflights and the units in Busy with a
	// race waiter queued beneath them, not running its command: their tests
	// are done and their race stage has not started, so they run nothing.
	// Above one slot they take none (TakingSlots, iss-2610282016250805).
	AtRaceGate []int
}

// TakingSlots is a as the first gate counts it against slots units of work
// at once (the load.preflights setting; zero or less reads as one). Above
// one slot the units at their race gate (AtRaceGate) are dropped from
// Preflights, Busy and Waiters: each is idle until its race
// run starts, and then it is busy again, beneath its make or as its race
// waiter running its command. Race runs are kept apart by the race gate
// (RaceAhead, RaceRuns and the race gate's hold), never by this count, and
// the load limit still judges the machine as a whole. With one slot the race
// gate opens at once, so a preflight at it is about to start its race stage,
// and a stays as it is: the gate reads as it did before slots existed.
func (a Ahead) TakingSlots(slots int) Ahead {
	if slots <= 1 || len(a.AtRaceGate) == 0 {
		return a
	}
	drop := func(pids []int) []int {
		return slices.DeleteFunc(slices.Clone(pids), func(pid int) bool { return slices.Contains(a.AtRaceGate, pid) })
	}
	a.Preflights, a.Busy, a.Waiters = drop(a.Preflights), drop(a.Busy), drop(a.Waiters)
	return a
}

// Uncrossed is a without its crossed waiters: what is ahead once a wait no
// longer counts them (implement.WaitLoad, implement.WaitCrossedCap).
func (a Ahead) Uncrossed() Ahead {
	drop := func(pids []int) []int {
		return slices.DeleteFunc(slices.Clone(pids), func(pid int) bool { return slices.Contains(a.Crossed, pid) })
	}
	a.Waiters, a.Queued, a.Crossed = drop(a.Waiters), drop(a.Queued), nil
	return a
}

// Shut reports which of the queue's conditions hold the first gate shut when
// it admits slots units of work at once (the load.preflights setting; zero
// or less reads as one): every unit in Busy
// takes a slot, and the gate is shut on preflight or on waiter once they are
// all taken, by the kind of unit that takes them. A waiter queued ahead shuts
// it whatever slots is, so a free slot never lets a waiter jump the queue.
// With one slot this is the gate as it read before slots existed: any running
// preflight shuts it on preflight, and any waiter ahead or running its command
// on waiter. Above one, a unit at its race gate takes no slot (TakingSlots).
func (a Ahead) Shut(slots int) (preflight, waiter bool) {
	a = a.TakingSlots(slots)
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
// is one has passed its first gate. It stays in Preflights and Busy, and while
// its race waiter is queued it is in AtRaceGate too: idle, so above one slot
// it takes none (TakingSlots).
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
		out.Queued, out.Crossed = t.queuedAhead(waiting)
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
	// Each unit of work running past the gate once: a preflight or a waiter
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
	// A preflight, or a unit, with a race waiter queued beneath it is at its
	// race gate: a race waiter running its command is running its race run.
	_, raceWaiting := t.waiters(args, true)
	for _, pid := range append(append([]int(nil), out.Preflights...), out.Busy...) {
		if slices.Contains(out.AtRaceGate, pid) {
			continue
		}
		for _, w := range raceWaiting {
			if Descends(snap.Procs, w, pid) {
				out.AtRaceGate = append(out.AtRaceGate, pid)
				break
			}
		}
	}
	sort.Ints(out.AtRaceGate)
	return out
}

// RaceAhead returns, in pid order, the race waiters (isRaceWaiter) ahead of
// self at the race gate: those queued ahead of it, by the rule the first
// gate's queue keeps (ahead by start, queuedAhead), and those running their
// command, outside self's process and its ancestors; and, as crossed, those
// queued ahead of it with a higher pid (Ahead.Crossed). A waiter at a first
// gate, and a preflight in its unit stage, are not ahead of it there; the race
// runs that shut it are RaceRuns.
func RaceAhead(snap Snapshot, self Self, args map[int][]string) (ahead, crossed []int) {
	if !snap.HasProcs {
		return nil, nil
	}
	t := readGate(snap, self, args)
	runningW, waiting := t.waiters(args, true)
	queued, crossed := t.queuedAhead(waiting)
	ahead = append(append([]int(nil), runningW...), queued...)
	sort.Ints(ahead)
	return ahead, crossed
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

// QueuedAt returns, in pid order, the waiters queued at one gate, the race
// gate's when race is set and the first gate's otherwise, outside self's
// process and its ancestors: every one not running its command, ahead of self
// or not. implement.WaitLoad reads it to tell a crossed waiter that has left
// the queue (exited, or opened) from one a tie read differently has moved
// behind self.
func QueuedAt(snap Snapshot, self Self, args map[int][]string, race bool) []int {
	if !snap.HasProcs {
		return nil
	}
	t := readGate(snap, self, args)
	_, waiting := t.waiters(args, race)
	return waiting
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
// self by start (queueOrder), at the first gate (QueueAhead) and the race gate
// (RaceAhead) alike, and those of them with a higher pid than self's, the
// crossed ones (Ahead.Crossed). Absent from the table, self has no start time,
// and queueOrder puts it behind every one that has.
func (t gateTable) queuedAhead(waiting []int) (ahead, crossed []int) {
	queue := []Proc{{PID: t.self.PID}}
	if p, ok := t.byPID[t.self.PID]; ok {
		queue[0] = p
	}
	for _, pid := range waiting {
		queue = append(queue, t.byPID[pid])
	}
	for _, pid := range queueOrder(queue) {
		if pid == t.self.PID {
			break
		}
		ahead = append(ahead, pid)
		if pid > t.self.PID {
			crossed = append(crossed, pid)
		}
	}
	sort.Ints(ahead)
	sort.Ints(crossed)
	return ahead, crossed
}

// queueOrder returns the pids of the queued waiters ws in the order they are
// admitted: the order they started in, earliest first, with start times the
// table cannot tell apart ordered by pid, lowest first.
//
// A start time is the read's moment less the process's age, and every age in
// one table is measured back from the same read (one ps run on macOS, one
// uptime read on Linux), so the earlier start is the greater age and the
// order is derived from the ages alone, never from a clock read beside them:
// every waiter that reads one table derives the same order from it, and a
// read at another moment shifts every age alike and orders the queue the same
// way. The moment itself, and how long the read took, which the session-name
// register allows for when it compares a start time with one an earlier read
// recorded, cancel between two waiters of one table, but for the read's own
// length (tens of milliseconds). What is left is the reading's grain (macOS
// reads ages in whole seconds), which StartTolerance absorbs, as it does for
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
// changes nothing.
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
// never more than the cores divided among the preflights the first gate
// admits at once (load.preflights; zero or less reads as one, and one is go's
// own default of the cores) and never fewer than a
// quarter of the cores, or one, so a run on a loaded machine still makes
// progress. With n slots, n concurrent stages each take at most the cores
// divided by n (half at two, a quarter at four, where on most machines the
// share and the floor meet): the load a later stage starts under lags the
// earlier stages' real demand by up to a minute, so the load alone would give
// each the whole machine. A race stage may run beside n unit stages, since a
// preflight at its race gate takes no slot (Ahead.TakingSlots), so the load
// limit is what bounds that one extra stage. An unreadable load leaves the
// cores divided among the slots.
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
// before it starts its command (implement.WaitLoad): across accounts no hold
// keeps two waiters from opening in one moment, so each looks once more and
// stands back from the others StandBackFrom names. A waiter at the other gate did not
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

// StandBackFrom returns, in pid order, the waiters of opened, read from snap
// beside self as having opened in the same moment (JustOpened), that self
// stands back from: every one but those self precedes both in the queue's
// order (queueOrder, from snap's ages) and by pid. So the later start stands
// back from the earlier whatever their pids, as the queue orders them; inside
// one tie, where the order is by pid, exactly one goes on.
//
// The earlier start stands back from a lower pid that opened beside it too.
// That opener started later, so a waiter of this code would have stood back
// from it; an older abcd does not (v0.15.1 stands back only from a lower pid,
// and v0.15.0 takes no such look), and goes on. Were this one to go on as
// well, both would run their commands, and under `--wait -- make preflight`
// each preflight's own gate would wait on the other's waiter until its
// timeout.
//
// So whenever the pair is crossed (the earlier start has the higher pid), the
// later start stands back, and so does the earlier start if it runs this code
// or v0.15.1, which stands back from the lower pid: both stand back and
// re-draw their jitters, at most implement.WaitBackoffCap times each. Only a
// later start of v0.15.1 or v0.15.0 goes on past a crossed pair, and then this
// code's earlier start standing back is what keeps the two from running at
// once. A pair that is not crossed agrees: exactly one goes on, unless the
// other runs v0.15.0. Two waiters of this code meet here seldom, since the
// later start waits on the earlier in the queue: only after a wrap, for a
// crossed pair, where a tie at the tolerance's edge reads differently to each
// (macOS reads ages in whole seconds) or the later has stopped counting the
// earlier past its cap (implement.WaitCrossedCap).
func StandBackFrom(snap Snapshot, self Self, opened []int) []int {
	byPID := make(map[int]Proc, len(snap.Procs))
	for _, p := range snap.Procs {
		byPID[p.PID] = p
	}
	queue := make([]Proc, 0, len(opened)+1)
	for _, pid := range append([]int{self.PID}, opened...) {
		p, ok := byPID[pid]
		if !ok {
			p = Proc{PID: pid}
		}
		queue = append(queue, p)
	}
	order := queueOrder(queue)
	at := slices.Index(order, self.PID)
	var out []int
	for i, pid := range order {
		if pid != self.PID && (i < at || pid < self.PID) {
			out = append(out, pid)
		}
	}
	sort.Ints(out)
	return out
}
