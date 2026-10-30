package implement

import (
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// A command hold lasts until its command exits, starts its own race run or
// WaitCommandCap passes (iss-2610241419324875). On
// 2026-10-24 a `go test -timeout 60m` of the loop suite, admitted through the
// first gate, held the slot for 45 minutes on a loaded machine, working
// through tests at 30 to 70 seconds each, most of its time in git
// subprocesses, so its test binary showed little CPU. About nine preflights
// queued behind it, and none of them could tell whether it was working; a peer
// asked its owner, who stopped it.
//
// So the gate watches the CPU the holder's whole process tree (the holder and
// every descendant by parent pid, git children included) uses between looks,
// from the cumulative CPU times the process table already carries
// (machineload.Proc.CPU), and when the tree has used no more than QuietShare
// of one core for QuietSpan, it says so as a fact: to each waiter in its
// waiting line, naming the holder's pid, the checkout its record names and how
// long it has held the slot, and to the holder's own standard error while
// others queue (HoldWatch). Little CPU is not a diagnosis: a slow run waiting
// on its subprocesses' I/O uses little too, so the notice leaves the judgement
// to the holder's owner. The hold stays in force, and nothing is ever
// signalled: a process is stopped by its owner, through its own handle, never
// by the gate (the LOAD rule). Each watcher judges from its own looks, so a
// waiter that joins the queue late watches a full span before it says
// anything.
//
// A child that exits between two looks takes its CPU out of the table, so a
// member gone since the baseline counts as work, as a new one counts all its
// CPU: a run that keeps starting short git children is not quiet. A tree with
// a waiter of its own queued at either gate (`--wait -- make preflight`, whose
// preflight waits at its own race gate) is waiting at that gate, and is not
// said to be quiet.

// QuietSpan is how long a holder's tree must have used little CPU before the
// waiters are told. A test stage that is compiling or running tests shows CPU
// within minutes even on a loaded machine, so ten minutes outlasts any pause
// in that work, and is well short of the 45 minutes the slow run that
// prompted it held its slot.
const QuietSpan = 10 * time.Minute

// QuietShare is the most CPU, in cores, a tree may average over QuietSpan and
// still be called quiet: a tenth of one core, sixty seconds of CPU in ten
// minutes. A go test that is compiling or running CPU-bound tests uses whole
// cores, so the line falls well below the least such work does; a run that
// is mostly waiting falls under it, which is what the waiters are told.
const QuietShare = 0.1

// HoldWatchEvery is how often the holder's own watch looks while its command
// runs. Each look is a run of ps, so it looks less often than a waiter polls.
const HoldWatchEvery = time.Minute

// QuietHolder is a command hold's holder whose tree has used little CPU for
// QuietSpan: a fact the waiters are told, not a verdict on the holder.
type QuietHolder struct {
	PID int `json:"pid"`
	// Checkout is the directory the holder ran in, as its record names it;
	// empty for a record an older abcd wrote.
	Checkout string `json:"checkout,omitempty"`
	// HeldS is how long it has held the slot, since its hold opened.
	HeldS float64 `json:"held_s"`
	// QuietS is how long its tree has used no more than QuietShare of a core.
	QuietS float64 `json:"quiet_s"`
}

// GateHolder is one holder ahead of a waiter, as each waiting line names it:
// the hold record's holder, or work the process table shows in its slot (a
// waiter running its command, a make preflight in its test stages, or at the
// race gate a race run).
//
// Quiet marks a holder this waiter has measured whose tree used no more than
// QuietShare of a core over the last QuietSpan and has no waiter of its own
// queued at either gate: the same fact the quiet notice states
// (judgeHolder), for every holder named, of any account. The notice speaks
// only of a command hold, which a run that is not a race run takes only at
// one preflight at a time; above one such a run holds its slot under the
// settle hold, so a stalled one is told to the waiters behind it by this
// marker alone (iss-2610241419324875). Like the notice, it is a fact, not a
// verdict. It is read from the meter (cpuMeter), which judges as the notice's
// watch (quietWatch) does: a child gone since the last look is work, so a
// run that keeps starting short git children is never marked quiet.
type GateHolder struct {
	PID int `json:"pid"`
	// Hold says it is the holder of this account's hold record; its age is
	// then how long it has held the slot, and otherwise how long it has run.
	Hold bool    `json:"hold,omitempty"`
	AgeS float64 `json:"age_s"`
	// Measured says this waiter has watched it for QuietSpan, and CPU is
	// then its tree's CPU over the last QuietSpan, in cores. Until then it
	// is not yet measured, and CPU is zero.
	Measured bool    `json:"measured"`
	CPU      float64 `json:"cpu_cores,omitempty"`
	// Quiet says it is measured, its CPU is at most QuietShare, no member
	// has left its tree within QuietSpan, and no waiter of its own is
	// queued at either gate.
	Quiet bool `json:"quiet,omitempty"`
}

// quietWatch follows one holder's tree between looks. Its zero value watches
// nothing.
type quietWatch struct {
	pid   int
	start int64
	// since is the baseline's moment, and base each tree member's CPU then.
	since time.Time
	base  map[int]time.Duration
	// quietSince is when the tree went quiet, zero while it is not.
	quietSince time.Time
}

// observe takes one look at the tree under pid, whose start time is start,
// at t, and returns how long it has been quiet, zero while it is not.
// waiting says the tree has a waiter of its own queued, which is never quiet.
//
// A look whose tree has used more than QuietSpan's worth of QuietShare since
// the baseline is busy: the baseline moves to it and any quiet ends. A look
// QuietSpan or more past the baseline that has not is quiet: the quiet begins
// at the baseline (or goes on), and the baseline moves to it, so a tree that
// picks up again ends the quiet within one span's worth of CPU. A member gone
// since the baseline exited, which is work too. A look that does not find pid
// at all measured nothing, so it starts the watch again rather than count an
// empty tree as quiet.
func (w *quietWatch) observe(procs []machineload.Proc, pid int, start int64, waiting bool, t time.Time) time.Duration {
	tree := treeCPU(procs, pid)
	if _, ok := tree[pid]; !ok {
		*w = quietWatch{}
		return 0
	}
	if w.base == nil || w.pid != pid || w.start != start || waiting {
		*w = quietWatch{pid: pid, start: start, since: t, base: tree}
		return 0
	}
	var used time.Duration
	for p, was := range w.base {
		now, ok := tree[p]
		if !ok {
			w.since, w.base, w.quietSince = t, tree, time.Time{}
			return 0
		}
		used += max(now-was, 0)
	}
	for p, now := range tree {
		if _, ok := w.base[p]; !ok {
			used += now
		}
	}
	if used > time.Duration(QuietShare*float64(QuietSpan)) {
		w.since, w.base, w.quietSince = t, tree, time.Time{}
		return 0
	}
	if t.Sub(w.since) >= QuietSpan {
		if w.quietSince.IsZero() {
			w.quietSince = w.since
		}
		w.since, w.base = t, tree
	}
	if w.quietSince.IsZero() {
		return 0
	}
	return t.Sub(w.quietSince)
}

// treeCPU is the cumulative CPU of pid and each of its descendants, by pid.
func treeCPU(procs []machineload.Proc, pid int) map[int]time.Duration {
	kids := map[int][]machineload.Proc{}
	out := map[int]time.Duration{}
	for _, p := range procs {
		if p.PID == pid {
			out[pid] = p.CPU
		} else {
			kids[p.PPID] = append(kids[p.PPID], p)
		}
	}
	if _, ok := out[pid]; !ok {
		return out
	}
	for next := []int{pid}; len(next) > 0; {
		parent := next[0]
		next = next[1:]
		for _, c := range kids[parent] {
			if _, seen := out[c.PID]; !seen {
				out[c.PID] = c.CPU
				next = append(next, c.PID)
			}
		}
	}
	return out
}

// holderWatch is what a wait keeps between its looks about the holders
// ahead of it: the quiet watch of the hold record's holder, and a CPU meter
// for each holder, by pid.
type holderWatch struct {
	quiet  quietWatch
	meters map[int]*cpuMeter
}

// cpuMeter follows one holder's tree between a waiter's looks: when it was
// first seen, with what start time, each member's CPU at the last look, and
// the CPU the tree used in each interval between looks, by the interval's
// end. A member that starts and exits between two looks is never seen, and
// one gone since the last look adds nothing more, so the meter can read low
// for a tree of short-lived children; the interval it left in is marked
// instead, and the tree is not settled (settled) while one such interval is
// within the last QuietSpan. A member new to the tree counts its
// whole CPU so far, so a long-running process moved under the tree reads high
// for the look that finds it.
type cpuMeter struct {
	first time.Time
	start int64
	prev  map[int]time.Duration
	used  []cpuInterval
}

type cpuInterval struct {
	end  time.Time
	used time.Duration
	// left says a member seen at the last look was gone at this one.
	left bool
}

// measure takes one look at the tree under pid, started at start, at t, and
// returns its CPU over the last QuietSpan in cores, and whether this meter
// has watched it that long.
func (m *cpuMeter) measure(procs []machineload.Proc, pid int, t time.Time) (float64, bool) {
	tree := treeCPU(procs, pid)
	var used time.Duration
	for p, now := range tree {
		if was, ok := m.prev[p]; ok {
			used += max(now-was, 0)
		} else {
			used += now
		}
	}
	left := false
	for p := range m.prev {
		if _, ok := tree[p]; !ok {
			left = true
		}
	}
	m.prev = tree
	m.used = append(m.used, cpuInterval{end: t, used: used, left: left})
	m.used = slices.DeleteFunc(m.used, func(i cpuInterval) bool { return !i.end.After(t.Add(-QuietSpan)) })
	if t.Sub(m.first) < QuietSpan {
		return 0, false
	}
	var total time.Duration
	for _, i := range m.used {
		total += i.used
	}
	return total.Seconds() / QuietSpan.Seconds(), true
}

// settled says no member of the tree has left it within the last QuietSpan:
// a child gone between looks did work the meter could not see, which the
// notice's watch counts as work too.
func (m *cpuMeter) settled() bool {
	return !slices.ContainsFunc(m.used, func(i cpuInterval) bool { return i.left })
}

// measureHolders names each holder ahead at this look, the hold record's
// holder first (when it holds the command form), then the work in the slots,
// each once, a slot beneath another holder named counting as part of it, as
// the first gate's slots count them (machineload.Ahead.Busy): at the race
// gate a race command's waiter and the go test -race under it are one holder,
// whose tree would otherwise be counted twice. The hold's holder and a slot
// are one preflight whichever way they nest, so each is named once: a slot
// beneath the holder (`--wait -- make preflight`, the make under the waiter)
// and a slot above it (a whole run, whose `make preflight` runs the holding
// waiter beneath it, iss-2610241419324875) are both left to the holder, which
// would otherwise be named twice and its CPU counted twice. Each is metered:
// a holder first seen at this look starts its meter, and one no longer ahead
// drops it. A pid whose start time moved past holdStartSlack is a new
// process, and starts a new meter.
//
// A measured holder whose tree used no more than QuietShare of a core over
// the last QuietSpan is marked Quiet, unless a waiter of its own is queued at
// either gate beneath it (queuedUnder, as judgeHolder excludes one, from
// self's look at the same snapshot and args), so a waiter's line marks a
// stalled holder whatever form its hold takes.
func measureHolders(w *holderWatch, h *WaitHolder, slots []int, snap machineload.Snapshot, self machineload.Self, args map[int][]string, t time.Time) []GateHolder {
	if w.meters == nil {
		w.meters = map[int]*cpuMeter{}
	}
	var out []GateHolder
	seen := map[int]bool{}
	add := func(pid int, hold bool, age time.Duration) {
		if seen[pid] {
			return
		}
		seen[pid] = true
		start, ok := procStart(snap, pid, t)
		if !ok {
			return
		}
		m := w.meters[pid]
		if m == nil || start-m.start > holdStartSlack || m.start-start > holdStartSlack {
			m = &cpuMeter{first: t, start: start, prev: treeCPU(withoutReaders(snap, pid).Procs, pid)}
			w.meters[pid] = m
			out = append(out, GateHolder{PID: pid, Hold: hold, AgeS: age.Seconds()})
			return
		}
		cpu, measured := m.measure(withoutReaders(snap, pid).Procs, pid, t)
		quiet := measured && cpu <= QuietShare && m.settled() && !queuedUnder(snap, self, args, pid)
		out = append(out, GateHolder{PID: pid, Hold: hold, AgeS: age.Seconds(), Measured: measured, CPU: cpu, Quiet: quiet})
	}
	if h != nil && h.Form == HoldCommand {
		add(h.PID, true, t.Sub(h.Opened))
	}
	beneath := func(pid int) bool {
		if h != nil && h.Form == HoldCommand && (machineload.Descends(snap.Procs, pid, h.PID) || machineload.Descends(snap.Procs, h.PID, pid)) {
			return true
		}
		return slices.ContainsFunc(slots, func(other int) bool { return machineload.Descends(snap.Procs, pid, other) })
	}
	for _, pid := range slots {
		if beneath(pid) {
			continue
		}
		for _, p := range snap.Procs {
			if p.PID == pid {
				add(pid, false, p.Age)
				break
			}
		}
	}
	for pid := range w.meters {
		if !seen[pid] {
			delete(w.meters, pid)
		}
	}
	return out
}

// readerName is the base name of the program machineload.Read runs on macOS
// to read the table.
const readerName = "ps"

// withoutReaders is snap without the processes that only read the table: any
// ps that is a direct child of holder, or of an abcd in holder's tree, which
// is that abcd's own read (on macOS ps lists itself, as a child of its caller)
// whether its tree or another's look took the snapshot. Each is gone by the
// next look, and a member gone between looks counts as work, so left in, a
// holder's own reads would keep it from ever being quiet. The abcd beneath
// is a whole run's waiter, under the make preflight a waiter names from the
// slots: its watch reads the table while its command runs, and those reads
// are no more the run's work than the holder's own. An abcd starts no ps but
// its reads, and another's reader is gone before its arguments could be asked
// for, so for it the name is the handle there is.
func withoutReaders(snap machineload.Snapshot, holder int) machineload.Snapshot {
	readers := map[int]bool{holder: true}
	for _, p := range snap.Procs {
		if machineload.IsAbcdName(p.Name) && machineload.Descends(snap.Procs, p.PID, holder) {
			readers[p.PID] = true
		}
	}
	snap.Procs = slices.DeleteFunc(slices.Clone(snap.Procs), func(p machineload.Proc) bool {
		return readers[p.PPID] && p.Name == readerName
	})
	return snap
}

// queuedUnder reports whether any of the waiters queued at either gate, as
// self reads the table, descends from pid.
func queuedUnder(snap machineload.Snapshot, self machineload.Self, args map[int][]string, pid int) bool {
	for _, race := range []bool{false, true} {
		for _, q := range machineload.QueuedAt(snap, self, args, race) {
			if machineload.Descends(snap.Procs, q, pid) {
				return true
			}
		}
	}
	return false
}

// judgeHolder says whether the holder of the command hold h, in force at
// this look, is quiet, from the snapshot and the arguments the look read.
func judgeHolder(w *quietWatch, h *WaitHolder, snap machineload.Snapshot, self machineload.Self, args map[int][]string, t time.Time) *QuietHolder {
	if h == nil || h.Form != HoldCommand {
		*w = quietWatch{}
		return nil
	}
	snap = withoutReaders(snap, h.PID)
	quiet := w.observe(snap.Procs, h.PID, h.StartUnix, queuedUnder(snap, self, args, h.PID), t)
	if quiet <= 0 {
		return nil
	}
	return &QuietHolder{PID: h.PID, Checkout: termsafe.Sanitize(h.Dir), HeldS: t.Sub(h.Opened).Seconds(), QuietS: quiet.Seconds()}
}

// HoldWatch is the holder's own watch while its command runs: the waiter
// that opened with a command hold looks every HoldWatchEvery, and once its
// tree has used little CPU for QuietSpan while others queue at its gate it
// says so on its own standard error, so the command's owner sees it beside
// the command's output and can judge whether it is still working. It reads
// only the process table, and signals nothing.
type HoldWatch struct {
	// Read and ReadArgs read the machine; Now is the clock.
	Read     func() (machineload.Snapshot, error)
	ReadArgs func([]int) (map[int][]string, error)
	Now      func() time.Time
	// Self is the holder.
	Self machineload.Self
	// Gates are the gates the holder holds (WaitResult.HoldGates): a race
	// command above one slot holds both, and a waiter at either waits on it.
	Gates []string
	w     quietWatch
	// quiet and waiting are the last answer a readable look gave.
	quiet   time.Duration
	waiting []int
}

// Look takes one look and returns how long the holder's tree has been quiet
// and the waiters queued at its gate outside the tree, in pid order; the
// quiet time is zero while the tree is not quiet. A table that cannot be read
// measured nothing, so the look gives the last readable look's answer again,
// as a waiter keeps its verdict: read as not quiet, it would end a quiet the
// tree never left and say it again at the next readable look.
func (hw *HoldWatch) Look() (time.Duration, []int) {
	snap, err := hw.Read()
	if err != nil || !snap.HasProcs {
		return hw.quiet, hw.waiting
	}
	args, err := hw.ReadArgs(machineload.GateCandidates(snap, hw.Self))
	if err != nil {
		return hw.quiet, hw.waiting
	}
	snap = withoutReaders(snap, hw.Self.PID)
	// The holder is this process, so its start needs no proving; one read
	// from its age would differ by a second between looks on macOS.
	quiet := hw.w.observe(snap.Procs, hw.Self.PID, 0, queuedUnder(snap, hw.Self, args, hw.Self.PID), hw.Now())
	var waiting []int
	for _, gate := range hw.Gates {
		for _, q := range machineload.QueuedAt(snap, hw.Self, args, gate == GateRace) {
			if !machineload.Descends(snap.Procs, q, hw.Self.PID) && !slices.Contains(waiting, q) {
				waiting = append(waiting, q)
			}
		}
	}
	slices.Sort(waiting)
	hw.quiet, hw.waiting = quiet, waiting
	return quiet, waiting
}
