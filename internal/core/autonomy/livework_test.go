package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// livework_test.go: the owed reading marks a capped item with the live work
// standing for it: a whole test run of this account, queued at the load gate
// or running, in the worktree the item's branch is checked out in, at the
// item's head, while no receipt names that head.

const livePID = 4242

// liveFx is a fixture whose own branch feat/live, with commits not pushed,
// is checked out in the fixture's own working copy.
func liveFx(t *testing.T) *fx {
	t.Helper()
	f := newFx(t)
	f.git("checkout", "-q", "-b", "feat/live")
	f.commitOn("feat/live", "work", "")
	return f
}

// plantMarker writes a whole run's marker for pid under the fixture's home.
func (f *fx) plantMarker(pid int, start time.Time, dir string) {
	f.t.Helper()
	d := abcdhome.Path(f.home, implement.WholeRunsLeaf)
	if err := os.MkdirAll(d, 0o700); err != nil {
		f.t.Fatal(err)
	}
	raw, _ := json.Marshal(implement.WholeRunMarker{SchemaVersion: 1, PID: pid, StartUnix: start.Unix(), Dir: dir, Kind: implement.WholeRunKind, Began: start})
	if err := os.WriteFile(filepath.Join(d, strconv.Itoa(pid)+".json"), raw, 0o600); err != nil {
		f.t.Fatal(err)
	}
}

// table is a process table holding procs, each aged from its start at the
// moment the table is read, and how many times it was read.
type table struct {
	procs  map[int]tableProc
	reads  int
	broken error
}

type tableProc struct {
	uid   uint32
	name  string
	start time.Time
}

func (tb *table) read(ctx context.Context) (machineload.Snapshot, error) {
	tb.reads++
	if err := ctx.Err(); err != nil {
		return machineload.Snapshot{}, err
	}
	if tb.broken != nil {
		return machineload.Snapshot{}, tb.broken
	}
	snap := machineload.Snapshot{HasProcs: true, HasLoad: true}
	for pid, p := range tb.procs {
		snap.Procs = append(snap.Procs, machineload.Proc{PID: pid, PPID: 1, PGID: pid, UID: p.uid, Age: time.Since(p.start), Name: p.name})
	}
	return snap, nil
}

func me() uint32 { return uint32(os.Geteuid()) }

// readLive is ReadOwed while away with tb as the process table.
func (f *fx) readLive(tb *table) Owed {
	f.t.Helper()
	o := f.opts()
	o.Record = f.awayRecord(false)
	o.ProcTable = tb.read
	return f.readWith(o)
}

func liveItem(t *testing.T, o Owed) OwedItem {
	t.Helper()
	it, ok := find(o, OwedUnpushed, "feat/live")
	if !ok {
		t.Fatalf("feat/live is not owed: %s", describe(o))
	}
	return it
}

// TestLiveWholeRunMarksTheItem: a live whole run of this account, named
// abcd, with the marker's start time, in the worktree feat/live is checked
// out in at its head, marks the item with what is running and since when.
func TestLiveWholeRunMarksTheItem(t *testing.T) {
	f := liveFx(t)
	start := time.Now().Add(-30 * time.Minute).Truncate(time.Second)
	f.plantMarker(livePID, start, f.root)
	tb := &table{procs: map[int]tableProc{livePID: {me(), "abcd", start}}}
	it := liveItem(t, f.readLive(tb))
	if it.Live == nil {
		t.Fatal("a whole run live for the item's head did not mark it")
	}
	if !strings.Contains(it.Live.What, "abcd process 4242") || !strings.Contains(it.Live.What, "feat/live") {
		t.Errorf("Live.What = %q, want the process and the branch named", it.Live.What)
	}
	if d := it.Live.Since.Sub(start); d < -3*time.Second || d > 3*time.Second {
		t.Errorf("Live.Since = %s, want about %s, the process's start", it.Live.Since, start)
	}
	if tb.reads != 1 {
		t.Errorf("the process table was read %d times, want once", tb.reads)
	}
}

// TestLiveWholeRunCountsOnlyItsOwn: a marker counts for nothing when its pid
// is another account's process, is not abcd, started at another time, or is
// gone; when it names another worktree; or when a receipt names the head.
func TestLiveWholeRunCountsOnlyItsOwn(t *testing.T) {
	start := time.Now().Add(-30 * time.Minute).Truncate(time.Second)
	cases := map[string]func(f *fx) *table{
		"another account's process": func(f *fx) *table {
			f.plantMarker(livePID, start, f.root)
			return &table{procs: map[int]tableProc{livePID: {me() + 1, "abcd", start}}}
		},
		"a process not named abcd": func(f *fx) *table {
			f.plantMarker(livePID, start, f.root)
			return &table{procs: map[int]tableProc{livePID: {me(), "zsh", start}}}
		},
		"a pid now held by a later process": func(f *fx) *table {
			f.plantMarker(livePID, start.Add(-time.Hour), f.root)
			return &table{procs: map[int]tableProc{livePID: {me(), "abcd", start}}}
		},
		"a pid that is gone": func(f *fx) *table {
			f.plantMarker(livePID, start, f.root)
			return &table{procs: map[int]tableProc{}}
		},
		"a run in another worktree": func(f *fx) *table {
			other := filepath.Join(t.TempDir(), "other")
			f.git("worktree", "add", "-q", "--detach", other, "main")
			f.plantMarker(livePID, start, other)
			return &table{procs: map[int]tableProc{livePID: {me(), "abcd", start}}}
		},
		"a head a receipt names": func(f *fx) *table {
			f.receipt(f.head("feat/live"))
			f.plantMarker(livePID, start, f.root)
			return &table{procs: map[int]tableProc{livePID: {me(), "abcd", start}}}
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := liveFx(t)
			tb := setup(f)
			if it := liveItem(t, f.readLive(tb)); it.Live != nil {
				t.Fatalf("%s marked the item live: %+v", name, it.Live)
			}
		})
	}
}

// TestLiveWorkReadsNothingItNeedNot: present, with no marker, or with no
// capped item on a branch, the process table is not read.
func TestLiveWorkReadsNothingItNeedNot(t *testing.T) {
	start := time.Now().Add(-time.Minute)
	f := liveFx(t)
	tb := &table{procs: map[int]tableProc{livePID: {me(), "abcd", start}}}
	if f.readLive(tb); tb.reads != 0 {
		t.Errorf("with no marker the table was read %d times", tb.reads)
	}
	f.plantMarker(livePID, start, f.root)
	o := f.opts()
	o.ProcTable = tb.read
	if owed := f.readWith(o); tb.reads != 0 || liveItem(t, owed).Live != nil {
		t.Errorf("present, the table was read %d times", tb.reads)
	}
}

// TestLiveWorkUnreadIsANote: a process table that cannot be read, and a
// marker folder that is refused (a symlink), are notes, and the item is not
// live: today's rule.
func TestLiveWorkUnreadIsANote(t *testing.T) {
	start := time.Now().Add(-time.Minute)
	f := liveFx(t)
	f.plantMarker(livePID, start, f.root)
	owed := f.readLive(&table{broken: errors.New("ps failed")})
	if liveItem(t, owed).Live != nil || owed.Incomplete || !strings.Contains(strings.Join(owed.Notes, "\n"), "whole test run") {
		t.Fatalf("an unread table: %+v", owed)
	}

	f = liveFx(t)
	real := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(f.home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, abcdhome.Path(f.home, implement.WholeRunsLeaf)); err != nil {
		t.Fatal(err)
	}
	f.plantMarker(livePID, start, f.root) // lands in real, through the link
	tb := &table{procs: map[int]tableProc{livePID: {me(), "abcd", start}}}
	owed = f.readLive(tb)
	if liveItem(t, owed).Live != nil || tb.reads != 0 || !strings.Contains(strings.Join(owed.Notes, "\n"), "whole test run") {
		t.Fatalf("a symlinked marker folder was read: %+v, table reads %d", owed, tb.reads)
	}
}

// TestLiveWorkSpentStartsNothing: once the check's deadline has ended the
// live-work read reads no marker and no process table, and marks the reading
// cut short.
func TestLiveWorkSpentStartsNothing(t *testing.T) {
	f := liveFx(t)
	f.plantMarker(livePID, time.Now(), f.root)
	spent, cancel := context.WithCancel(context.Background())
	cancel()
	tb := &table{}
	r := &owedReader{ctx: spent, root: f.root, home: f.home, away: true, procTable: tb.read,
		items: []OwedItem{{Kind: OwedUnpushed, Capped: true, Branch: "feat/live", Head: f.head("feat/live")}}}
	r.readLiveWork()
	if !r.cut || tb.reads != 0 || r.items[0].Live != nil {
		t.Fatalf("a spent deadline: cut %v, table reads %d, live %+v; want cut short with nothing read", r.cut, tb.reads, r.items[0].Live)
	}
}
