package autonomy

import (
	"fmt"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The live work standing for an owed item. An owed branch whose next step is
// its preflight cannot move until that run ends, which takes from minutes to
// hours, so while the person is away a set whose every capped item has live
// work is waiting, not stuck (Judge, row 3). What counts as live work is only
// what abcd can prove from state it owns and the process table: a whole test
// run, `make preflight` waiting at the load gate or running past it, which
// records itself under abcd's home (implement.WholeRunMarker). A marker counts
// only for a process in the table that is this account's, is named abcd and
// started when the marker says, whose checkout is the worktree the item's
// branch is checked out in, at the item's head, while no receipt names that
// head: with a receipt the next step is the push, not a wait. A checkout whose
// abcd predates the marker writes none, and its items keep the two-wake rule.
//
// Not counted, and must not be until something proves it: a build run's lane
// agent (the loop's judgement of its agents is its own, and answers unknown
// for a host-driven agent while the session that drives it lives), and a
// host's background agent abcd did not start. Nothing enforces that beyond
// this reader reading neither.

// readLiveWork marks each capped item on a branch with the live work standing
// for it, while away. Every read it makes is under the check's ctx and marks
// the reading cut short when that ctx has ended (cutShort): a ctx already
// ended reads no marker and starts no process read. A read that fails for any
// other reason is a note, and no item is marked: the two-wake rule applies.
// The process table is read once, only when a marker is found; the live
// sessions' read (holders.go) reads it apart, through occupancy, so a check
// that needs both reads it twice.
func (r *owedReader) readLiveWork() {
	if !r.away {
		return
	}
	var cands []int
	for i, it := range r.items {
		if it.Capped && it.Branch != "" && it.Head != "" {
			cands = append(cands, i)
		}
	}
	if len(cands) == 0 || r.cutShort(r.expired()) {
		return
	}
	marks, err := implement.ReadWholeRunMarkers(r.home)
	if err != nil {
		r.note("whether a whole test run is live for an owed item was not read (%s), so none is counted as waiting on one", termsafe.CleanProseLine(fsutil.RedactHome(err.Error()), 256))
		return
	}
	if len(marks) == 0 || r.cutShort(r.expired()) {
		return
	}
	read := r.procTable
	if read == nil {
		read = machineload.ReadContext
	}
	before := time.Now()
	snap, err := read(r.ctx)
	took := time.Since(before)
	if err == nil && !snap.HasProcs {
		err = fmt.Errorf("the process table was not read")
	}
	if r.cutShort(err != nil) {
		r.note("the process table was not read before the check's deadline, so no whole test run is counted as live")
		return
	}
	if err != nil {
		r.note("whether a whole test run is live for an owed item was not read (%s), so none is counted as waiting on one", termsafe.CleanProseLine(fsutil.RedactHome(err.Error()), 256))
		return
	}
	runs := liveWholeRuns(marks, snap, sessionname.Table{Procs: snap.Procs, ReadAt: before, Took: took}, uint32(os.Geteuid()))
	if len(runs) == 0 {
		return
	}
	wts, err := gitutil.ListWorktreesContext(r.ctx, r.root, gitutil.MaxWorktreeListing)
	if r.cutShort(err != nil) {
		return
	}
	if err != nil {
		r.note("the worktrees were not listed, so no whole test run is counted as live for an owed item")
		return
	}
	for _, i := range cands {
		it := &r.items[i]
		run, ok := runFor(runs, wts, it.Branch, it.Head)
		if !ok || r.hasReceipt(it.Head) || r.cut {
			continue
		}
		it.Live = &LiveWork{
			What:  fmt.Sprintf("a whole test run (abcd process %d), queued at the load gate or running, for %s at %s", run.pid, it.Branch, shortHead(it.Head)),
			Since: run.since,
		}
	}
}

// liveRun is a whole run's marker its process proves: the checkout it runs
// in and when the process started, as the table read it.
type liveRun struct {
	pid   int
	dir   string
	since time.Time
}

// liveWholeRuns are the markers whose pid the table holds as a process of
// uid, named abcd, started when the marker says (sessionname.Table.Alive,
// within the start tolerance and the read's slack). Any other marker counts
// for nothing: another account's process, a later process given the pid, or
// a run that has ended.
func liveWholeRuns(marks []implement.WholeRunMarker, snap machineload.Snapshot, tab sessionname.Table, uid uint32) []liveRun {
	byPID := make(map[int]machineload.Proc, len(snap.Procs))
	for _, p := range snap.Procs {
		byPID[p.PID] = p
	}
	var out []liveRun
	for _, m := range marks {
		p, ok := byPID[m.PID]
		if !ok || p.UID != uid || !machineload.IsAbcdName(p.Name) || m.Kind != implement.WholeRunKind {
			continue
		}
		if !tab.Alive(sessionname.Host{PID: m.PID, StartUnix: m.StartUnix}) {
			continue
		}
		out = append(out, liveRun{pid: m.PID, dir: m.Dir, since: tab.ReadAt.Add(-p.Age)})
	}
	return out
}

// runFor is the live run for branch at head: one whose checkout is the
// worktree branch is checked out in, that worktree's HEAD being head.
func runFor(runs []liveRun, wts []gitutil.Worktree, branch, head string) (liveRun, bool) {
	for _, wt := range wts {
		if wt.Bare || wt.Branch != "refs/heads/"+branch || wt.Head != head {
			continue
		}
		for _, run := range runs {
			if sameDir(run.dir, wt.Path) {
				return run, true
			}
		}
	}
	return liveRun{}, false
}
