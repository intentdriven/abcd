package machineload

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestIsNarrowCommand: a narrow run is go test naming at most MaxNarrowPackages
// explicit packages, read from the argv with no shell in between. Everything
// else is a whole run: the classification fails closed.
func TestIsNarrowCommand(t *testing.T) {
	for _, c := range []struct {
		cmd  string
		want bool
	}{
		{"go test ./internal/core/implement", true},
		{"go test -count=1 ./a ./b ./c", true},
		{"go test -count 1 -run TestX -v ./a", true},
		{"go test -run=TestX -skip TestY -short -failfast -json -timeout 10m .", true},
		{"go test ../sibling", true},
		{"go test github.com/intentdriven/abcd/internal/core/spec", true},
		{"go test ./all", true},
		{"go test ./x/../all", true},
		{"go test ./a -run TestX", true},
		{"go test -v=true ./a", true},
		{"go test --count=1 ./a", true},
		{"go -C /copy test -count=1 ./a", true},
		{"go -C=/copy test ./a", true},

		{"go test", false},
		{"go test -count=1", false},
		{"go test ./a ./b ./c ./d", false},
		{"go test ./...", false},
		{"go test ./a/...", false},
		{"go test ./a ./b/...", false},
		{"go test all", false},
		{"go test all/", false},
		{"go test std/", false},
		{"go test cmd/", false},
		{"go test work/", false},
		{"go test tool/", false},
		{"go test x/../all", false},
		{"go test ./a\\b", false},
		{"go test all\\", false},
		{"go test x/@v1", false},
		{"./tools/go test ./a", false},
		{"/usr/local/go/bin/go test ./a", false},
		{"go test std", false},
		{"go test fmt", false},
		{"go test ./a/*", false},
		{"go test ./a/b?", false},
		{"go test x_test.go", false},
		{"go test -race ./a", false},
		{"go test --race ./a", false},
		{"go test -race=false ./a", false},
		{"go test -count=2 ./a", false},
		{"go test -count 20 ./a", false},
		{"go test -count", false},
		{"go test -p 4 ./a", false},
		{"go test -parallel 4 ./a", false},
		{"go test -tags smoke ./a", false},
		{"go test -cover ./a", false},
		{"go test -exec sudo ./a", false},
		{"go test -c ./a", false},
		{"go test -bench . ./a", false},
		{"go test -test.count=99 ./a", false},
		{"go test -v=maybe ./a", false},
		{"go test ./a -args -x", false},
		{"go test -- ./a", false},
		{"go test -run", false},
		{"go -C", false},
		{"go -C sub", false},
		{"go -C sub vet ./a", false},
		{"go -C sub test ./...", false},
		{"go run ./a", false},
		{"go vet ./a", false},
		{"make preflight", false},
		{"make test", false},
		{"sh -c go test ./a", false},
		{"bash scripts/x.sh", false},
		{"env go test ./a", false},
		{"gotest ./a", false},
		{"", false},
	} {
		if got := IsNarrowCommand(strings.Fields(c.cmd)); got != c.want {
			t.Errorf("IsNarrowCommand(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
	// A word holding a space cannot be told apart in another account's reading
	// of the table on macOS, which splits on whitespace, so it is a whole run.
	if IsNarrowCommand([]string{"go", "test", "-run", "A B", "./a"}) {
		t.Error("a word holding a space reads as narrow")
	}
	if IsNarrowCommand(nil) {
		t.Error("no command reads as narrow")
	}
}

// narrowWaiter is a waiter at pid, of another account, gating go test on one
// package; running gives it its go test child. Its age is how long it has
// waited or run.
func narrowWaiter(pid int, age time.Duration, running bool) ([]Proc, map[int][]string) {
	cmd := []string{"implement", "load", "--wait", "--narrow", "--timeout", "6h", "--", "go", "test", "-count=1", "./internal/core/spec"}
	procs := []Proc{{PID: pid, PPID: 1, PGID: pid, UID: otherUID, Age: age, Name: "abcd"}}
	args := map[int][]string{pid: cmd}
	if running {
		procs = append(procs, Proc{PID: pid + 1, PPID: pid, PGID: pid, UID: otherUID, Age: age - time.Second, Name: "go"})
		args[pid+1] = []string{"test", "-count=1", "./internal/core/spec"}
	}
	return procs, args
}

// TestWholeRunsCountNoNarrowRun: a narrow waiter, queued or running, takes no
// place in the first gate's queue and no slot, so a whole run counts it through
// the load figure alone.
func TestWholeRunsCountNoNarrowRun(t *testing.T) {
	procs, args := ownPreflightWaiter()
	early, earlyArgs := narrowWaiter(50001, time.Hour, false)
	busy, busyArgs := narrowWaiter(51000, time.Hour, true)
	busy2, busy2Args := narrowWaiter(52000, time.Hour, true)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append(append(procs, early...), busy...), busy2...), HasProcs: true}
	args = merge(merge(merge(args, earlyArgs), busyArgs), busy2Args)
	got := QueueAhead(snap, waiter, args)
	if len(got.Waiters)+len(got.Queued)+len(got.Busy)+len(got.Crossed)+len(got.Preflights) != 0 {
		t.Fatalf("ahead = %+v, want nothing: narrow runs are no whole run's business", got)
	}
	if p, w := got.Shut(1); p || w {
		t.Fatalf("one slot: Shut = %v, %v, want open", p, w)
	}
	if q := QueuedAt(snap, waiter, args, false); len(q) != 0 {
		t.Fatalf("queued at the first gate = %v, want no narrow waiter", q)
	}
	if o := JustOpened(snap, waiter, args, 24*time.Hour, false); len(o) != 0 {
		t.Fatalf("just opened = %v, want no narrow waiter", o)
	}
}

// TestAWholeShapedCommandStaysAWholeRun: the same waiter gating a wildcard, or
// anything else, is counted as ever.
func TestAWholeShapedCommandStaysAWholeRun(t *testing.T) {
	for _, cmd := range [][]string{
		{"go", "test", "./a"},
		{"go", "test", "./..."},
		{"go", "test", "-race", "./a"},
		{"make", "test"},
		{"sh", "-c", "go test ./a"},
	} {
		procs, args := ownPreflightWaiter()
		busy, busyArgs := narrowWaiter(51000, time.Hour, true)
		busyArgs[51000] = append([]string{"implement", "load", "--wait", "--"}, cmd...)
		snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, busy...), HasProcs: true}
		got := QueueAhead(snap, waiter, merge(args, busyArgs))
		if !reflect.DeepEqual(got.Busy, []int{51000}) {
			t.Errorf("%v: busy = %v, want it counted as a whole run", cmd, got.Busy)
		}
	}
}

// TestOnlyTheMarkMakesANarrowWaiter: the argv is the one truth every reader of
// the table shares, the waiter itself included, so only a waiter whose argv carries
// the mark and whose command is narrow counts as narrow, and a mark after `--` is
// the command's word.
func TestOnlyTheMarkMakesANarrowWaiter(t *testing.T) {
	for _, c := range []struct {
		args string
		want bool
	}{
		{"implement load --wait --narrow -- go test ./a", true},
		{"implement load --narrow --wait --timeout 6h -- go test ./a", true},
		{"implement load --wait --narrow=true -- go test ./a", true},
		{"--json implement load --wait --narrow -- go test ./a", true},
		{"implement load --wait -- go test ./a", false},
		{"implement load --wait --narrow=false -- go test ./a", false},
		{"implement load --wait -- go test --narrow ./a", false},
		{"implement load --wait --narrow -- go test ./...", false},
		{"implement load --wait --narrow -- make preflight", false},
		{"implement load --wait --narrow", false},
		{"implement load --race-gate --narrow -- go test ./a", false},
		{"implement status --wait --narrow -- go test ./a", false},
	} {
		if got := isNarrowWaiter(strings.Fields(c.args)); got != c.want {
			t.Errorf("isNarrowWaiter(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

// TestAnOlderAbcdStillReadsANarrowRunAsAWaiter: nothing about a narrow run's
// spelling changes for an abcd built before the class: it is an `implement load
// --wait` like any other, so that abcd queues behind it and counts it in a slot,
// the safe direction, and never takes it for nothing.
func TestAnOlderAbcdStillReadsANarrowRunAsAWaiter(t *testing.T) {
	_, args := narrowWaiter(51000, time.Minute, false)
	if !IsLoadWaiter(args[51000]) || isRaceWaiter(args[51000]) {
		t.Fatalf("%v does not read as a first gate's waiter", args[51000])
	}
}

// TestNarrowAhead names the narrow waiters running their commands and those
// queued ahead by start, and leaves out those started later, whole waiters,
// preflights and self's own chain.
func TestNarrowAhead(t *testing.T) {
	procs, args := waiterChain()
	args[60000] = []string{"implement", "load", "--wait", "--narrow", "--", "go", "test", "./a"}
	args[59990] = []string{"run", "./cmd/abcd", "implement", "load", "--wait", "--narrow", "--", "go", "test", "./a"}
	run, runArgs := narrowWaiter(51000, time.Minute, true)
	ahead, aheadArgs := narrowWaiter(52000, time.Hour, false)
	later, laterArgs := narrowWaiter(61000, time.Second, false)
	whole, wholeArgs := peerWaiter(54001, 1, otherUID)
	whole[0].Age, whole[1].Age = 2*time.Hour, 2*time.Hour
	pre, preArgs := otherPreflight(55000)
	all := procs
	for _, p := range [][]Proc{run, ahead, later, whole, pre} {
		all = append(all, p...)
	}
	for _, a := range []map[int][]string{runArgs, aheadArgs, laterArgs, wholeArgs, preArgs} {
		args = merge(args, a)
	}
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: all, HasProcs: true}
	gotRun, gotAhead, gotQueued := NarrowAhead(snap, waiter, args)
	if !reflect.DeepEqual(gotRun, []int{51000}) || !reflect.DeepEqual(gotAhead, []int{52000}) || !reflect.DeepEqual(gotQueued, []int{52000, 61000}) {
		t.Fatalf("NarrowAhead = running %v, ahead %v, queued %v", gotRun, gotAhead, gotQueued)
	}
}

// TestOverdueWhole: a whole waiter queued ahead for at least the span, with a
// slot of its own free, is overdue. One whose slots are all taken is shut by
// them and not by any narrow run, so nothing stands aside for it; a preflight
// parked at its race gate takes none above one slot.
func TestOverdueWhole(t *testing.T) {
	const after = 5 * time.Minute
	procs, args := waiterChain()
	whole, wholeArgs := peerWaiter(54001, 1, otherUID)
	whole[0].Age, whole[1].Age = 6*time.Minute, 6*time.Minute
	young, youngArgs := peerWaiter(54101, 1, otherUID)
	young[0].Age, young[1].Age = 4*time.Minute, 4*time.Minute
	all := append(append(procs, whole...), young...)
	args = merge(merge(args, wholeArgs), youngArgs)
	read := func(extra []Proc, extraArgs map[int][]string, slots int) []int {
		snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append([]Proc(nil), all...), extra...), HasProcs: true}
		return OverdueWhole(snap, QueueAhead(snap, waiter, merge(merge(map[int][]string{}, args), extraArgs)), slots, after)
	}
	if got := read(nil, nil, 2); !reflect.DeepEqual(got, []int{54001}) {
		t.Fatalf("a slot free: overdue = %v, want 54001 alone", got)
	}
	one, oneArgs := otherPreflight(80000)
	if got := read(one, oneArgs, 2); !reflect.DeepEqual(got, []int{54001}) {
		t.Fatalf("one of two slots taken: overdue = %v, want 54001", got)
	}
	if got := read(one, oneArgs, 1); got != nil {
		t.Fatalf("the only slot taken: overdue = %v, want none", got)
	}
	two, twoArgs := otherPreflight(81000)
	if got := read(append(append([]Proc(nil), one...), two...), merge(merge(map[int][]string{}, oneArgs), twoArgs), 2); got != nil {
		t.Fatalf("both slots taken: overdue = %v, want none", got)
	}
	parked, parkedArgs := raceStagePreflight(82000, 2*time.Minute)
	if got := read(append(append([]Proc(nil), one...), parked...), merge(merge(map[int][]string{}, oneArgs), parkedArgs), 2); !reflect.DeepEqual(got, []int{54001}) {
		t.Fatalf("one busy and one at its race gate: overdue = %v, want 54001", got)
	}
}
