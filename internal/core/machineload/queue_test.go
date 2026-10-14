package machineload

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// The gate is read from the process table, which shows every account's
// processes, never from a file another account could write
// (iss-2610140638074587): a make preflight of any account keeps it shut, and so
// does a waiter queued ahead of this one or running its command.

// otherPreflight is another account's `make preflight` at pid, its sh and go
// test children beneath it.
func otherPreflight(pid int) ([]Proc, map[int][]string) {
	procs := []Proc{
		{PID: pid, PPID: 1, PGID: pid, UID: otherUID, Age: 4 * time.Minute, Name: "make"},
		{PID: pid + 1, PPID: pid, PGID: pid, UID: otherUID, Age: time.Minute, Name: "sh"},
		{PID: pid + 2, PPID: pid + 1, PGID: pid, UID: otherUID, Age: time.Minute, Name: "go"},
	}
	args := map[int][]string{
		pid:     {"preflight"},
		pid + 2: {"test", "./..."},
	}
	return procs, args
}

// peerWaiter is a waiter at pid started by `go run` under parent; its children
// are added by the caller.
func peerWaiter(pid, parent int, uid uint32) ([]Proc, map[int][]string) {
	procs := []Proc{
		{PID: pid - 1, PPID: parent, PGID: pid - 1, UID: uid, Age: time.Minute, Name: "go"},
		{PID: pid, PPID: pid - 1, PGID: pid - 1, UID: uid, Age: 50 * time.Second, Name: "abcd"},
	}
	args := map[int][]string{
		pid - 1: {"run", "./cmd/abcd", "implement", "load", "--wait", "--timeout", "2h"},
		pid:     {"implement", "load", "--wait", "--timeout", "2h"},
	}
	return procs, args
}

func merge(dst map[int][]string, src map[int][]string) map[int][]string {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// TestQueueSeesAnotherAccountsPreflight: another account's make preflight,
// past its gate or not yet at it, holds the gate shut; this waiter's own make
// preflight, an ancestor, never does.
func TestQueueSeesAnotherAccountsPreflight(t *testing.T) {
	procs, args := waiterChain()
	// This waiter runs inside its own make preflight: the zsh's child is make.
	procs[1].Name = "make"
	args[59000] = []string{"preflight"}
	other, otherArgs := otherPreflight(80000)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, other...), HasProcs: true}
	args = merge(args, otherArgs)

	got := QueueAhead(snap, waiter, args)
	if !reflect.DeepEqual(got.Preflights, []int{80000}) || len(got.Waiters) != 0 {
		t.Fatalf("ahead = %+v, want the other account's preflight 80000 and no waiter", got)
	}
	for _, pid := range []int{80000, 80002} {
		found := false
		for _, c := range GateCandidates(snap, waiter) {
			found = found || c == pid
		}
		if !found {
			t.Fatalf("pid %d is not asked for its arguments: %v", pid, GateCandidates(snap, waiter))
		}
	}
	for _, c := range GateCandidates(snap, waiter) {
		if c == 59000 || c == 59990 || c == 60000 {
			t.Fatalf("the waiter's own chain is a gate candidate: %v", GateCandidates(snap, waiter))
		}
	}

	// A make that runs another goal is not a preflight.
	args[80000] = []string{"-C", "preflight", "lint-issues"}
	if got := QueueAhead(snap, waiter, args); len(got.Preflights) != 0 {
		t.Fatalf("make -C preflight lint-issues read as a preflight: %+v", got)
	}
}

// TestQueueOrdersWaitersByPid: two waiters at one gate agree on one order, the
// lower pid first, whichever account each runs under. A preflight whose own
// waiter is queued at the gate is that waiter's place in the queue, not a
// running preflight.
func TestQueueOrdersWaitersByPid(t *testing.T) {
	procs, args := waiterChain()
	other, otherArgs := otherPreflight(80000)
	// The other preflight is at its gate: its sh runs go run, whose abcd waits.
	other = other[:2]
	delete(otherArgs, 80002)
	w, wArgs := peerWaiter(80011, 80001, otherUID)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append(procs, other...), w...), HasProcs: true}
	args = merge(merge(args, otherArgs), wArgs)
	args[60000] = []string{"implement", "load", "--wait", "--", "go", "test", "-race", "./internal/..."}

	// Its waiter has the higher pid: this waiter is first.
	if got := QueueAhead(snap, waiter, args); len(got.Preflights) != 0 || len(got.Waiters) != 0 {
		t.Fatalf("ahead = %+v, want nothing: the other preflight waits behind this waiter", got)
	}
	// And from the other side, this waiter is ahead of it.
	them := Self{PID: 80011, UID: otherUID}
	if got := QueueAhead(snap, them, args); !reflect.DeepEqual(got.Waiters, []int{60000}) {
		t.Fatalf("the other waiter's view = %+v, want this waiter 60000 ahead", got)
	}

	// A waiter with a lower pid is ahead.
	low, lowArgs := peerWaiter(50001, 1, callerUID)
	snap.Procs = append(snap.Procs, low...)
	args = merge(args, lowArgs)
	if got := QueueAhead(snap, waiter, args); !reflect.DeepEqual(got.Waiters, []int{50001}) {
		t.Fatalf("ahead = %+v, want the lower-pid waiter 50001", got)
	}
}

// TestQueueCountsAWaiterRunningItsCommand: a waiter with a child other than the
// ps it reads the machine with is running its command, and is ahead whatever its
// pid.
func TestQueueCountsAWaiterRunningItsCommand(t *testing.T) {
	procs, args := waiterChain()
	w, wArgs := peerWaiter(90001, 1, otherUID)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, w...), HasProcs: true}
	args = merge(args, wArgs)

	// Reading the machine: its one child is ps.
	snap.Procs = append(snap.Procs, Proc{PID: 90002, PPID: 90001, PGID: 90000, UID: otherUID, Name: "ps"})
	if got := QueueAhead(snap, waiter, args); len(got.Waiters) != 0 {
		t.Fatalf("a waiting higher-pid waiter is ahead: %+v", got)
	}
	// Running its command.
	snap.Procs = append(snap.Procs, Proc{PID: 90003, PPID: 90001, PGID: 90000, UID: otherUID, Name: "sleep"})
	if got := QueueAhead(snap, waiter, args); !reflect.DeepEqual(got.Waiters, []int{90001}) {
		t.Fatalf("ahead = %+v, want the waiter running its command, 90001", got)
	}
}

// TestIsPreflightMake reads make's goals: a word that is neither a flag, a
// flag's value nor a variable assignment.
func TestIsPreflightMake(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{"preflight"}, true},
		{[]string{"-j4", "preflight"}, true},
		{[]string{"-C", "/src/abcd", "preflight"}, true},
		{[]string{"--directory=/src", "preflight", "V=1"}, true},
		{[]string{"-f", "preflight"}, false},
		{[]string{"-C", "preflight", "test"}, false},
		{[]string{"preflight=1", "test"}, false},
		{[]string{"lint-issues"}, false},
		{nil, false},
	} {
		if got := isPreflightMake(c.args); got != c.want {
			t.Errorf("isPreflightMake(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

// TestIsLoadWaiter reads abcd's words: the verb path implement load, with --wait
// before any `--`.
func TestIsLoadWaiter(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{"implement", "load", "--wait"}, true},
		{[]string{"--json", "implement", "load", "--wait", "--", "make", "preflight"}, true},
		{[]string{"implement", "load", "--wait=true"}, true},
		{[]string{"implement", "load", "--site", "preflight"}, false},
		{[]string{"implement", "load", "--wait=false"}, false},
		{[]string{"implement", "load", "--", "--wait"}, false},
		{[]string{"implement", "status", "--wait"}, false},
		{nil, false},
	} {
		if got := isLoadWaiter(c.args); got != c.want {
			t.Errorf("isLoadWaiter(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

// TestParseDarwinArgsDropsMakeAndAbcdProgramWords: the gate reads make's and
// abcd's arguments too, so their program words are dropped as go's are, an
// install path with a space in it included.
func TestParseDarwinArgsDropsMakeAndAbcdProgramWords(t *testing.T) {
	out := []byte(` 30748 /Applications/Xcode.app/Contents/Developer/usr/bin/make preflight
 30750 /Volumes/My Disk/bin/gmake preflight
 31000 /var/folders/x/T/go-build1/b001/exe/abcd implement load --wait
 31001 /Users/a b/plugin/abcd-darwin-arm64 implement load --wait
`)
	want := map[int][]string{
		30748: {"preflight"},
		30750: {"preflight"},
		31000: {"implement", "load", "--wait"},
		31001: {"implement", "load", "--wait"},
	}
	if got := parseDarwinArgs(out); !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %v\nwant %v", got, want)
	}
}

// TestGoTestParallelism: go test's -p is the cores the machine can still give,
// up to twice the cores less the load, never more than the cores and never
// fewer than a quarter of them, so one run cannot saturate a busy machine and
// still makes progress on a loaded one.
func TestGoTestParallelism(t *testing.T) {
	for _, c := range []struct {
		load    float64
		has     bool
		cores   int
		want    int
		comment string
	}{
		{2, true, 16, 16, "an idle machine: every core"},
		{20, true, 16, 12, "twice the cores less the load"},
		{19.2, true, 16, 12, "a fractional load rounds up"},
		{28, true, 16, 4, "a quarter of the cores at the floor"},
		{300, true, 16, 4, "never below the floor"},
		{0, false, 16, 16, "an unreadable load: go's default"},
		{9, true, 2, 1, "a two-core machine: one at least"},
	} {
		snap := Snapshot{Load1: c.load, HasLoad: c.has, Cores: c.cores}
		if got := GoTestParallelism(snap); got != c.want {
			t.Errorf("%s: GoTestParallelism(load %v on %d) = %d, want %d", c.comment, c.load, c.cores, got, c.want)
		}
	}
}

// TestQueueInnerWaiterTakesItsRunningAncestorsPlace is the deadlock of
// `abcd implement load --wait -- make preflight`: the outer waiter has won its
// turn and runs make, whose own gate line starts an inner waiter minutes later;
// meanwhile another session's waiter Y, with a lower pid than the inner one,
// queues. Y is shut by the outer waiter running its command, and the inner
// waiter must not be shut by Y in turn, or neither moves until its timeout. The
// inner waiter takes its running ancestor's place, ahead of every queued
// waiter, and still sees what is running.
func TestQueueInnerWaiterTakesItsRunningAncestorsPlace(t *testing.T) {
	procs := []Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: rootUID, Age: time.Hour, Name: "launchd"},
		// Session X: zsh, go run, the outer waiter running make preflight, make's
		// sh, the gate line's go run, the inner waiter.
		{PID: 15000, PPID: 1, PGID: 15000, UID: callerUID, Age: time.Hour, Name: "go"},
		{PID: 15001, PPID: 15000, PGID: 15000, UID: callerUID, Age: 9 * time.Minute, Name: "abcd"},
		{PID: 15002, PPID: 15001, PGID: 15000, UID: callerUID, Age: 9 * time.Minute, Name: "make"},
		{PID: 15003, PPID: 15002, PGID: 15000, UID: callerUID, Age: time.Minute, Name: "sh"},
		{PID: 20000, PPID: 15003, PGID: 15000, UID: callerUID, Age: time.Minute, Name: "go"},
		{PID: 20001, PPID: 20000, PGID: 15000, UID: callerUID, Age: 50 * time.Second, Name: "abcd"},
		// Session Y, another account: a plain waiter, queued, with the lower pid.
		{PID: 10000, PPID: 1, PGID: 10000, UID: otherUID, Age: 5 * time.Minute, Name: "go"},
		{PID: 10001, PPID: 10000, PGID: 10000, UID: otherUID, Age: 5 * time.Minute, Name: "abcd"},
	}
	args := map[int][]string{
		15000: {"run", "./cmd/abcd", "implement", "load", "--wait", "--", "make", "preflight"},
		15001: {"implement", "load", "--wait", "--", "make", "preflight"},
		15002: {"preflight"},
		20000: {"run", "./cmd/abcd", "implement", "load", "--wait", "--timeout", "2h"},
		20001: {"implement", "load", "--wait", "--timeout", "2h"},
		10000: {"run", "./cmd/abcd", "implement", "load", "--wait"},
		10001: {"implement", "load", "--wait"},
	}
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}
	inner := Self{PID: 20001, UID: callerUID}
	y := Self{PID: 10001, UID: otherUID}

	for _, pid := range []int{15001} {
		if !slices.Contains(GateCandidates(snap, inner), pid) {
			t.Fatalf("the inner waiter does not read its ancestor waiter %d's arguments: %v", pid, GateCandidates(snap, inner))
		}
	}
	if got := QueueAhead(snap, inner, args); len(got.Waiters) != 0 || len(got.Preflights) != 0 {
		t.Fatalf("inner waiter's view = %+v, want nothing ahead: its ancestor already won the turn", got)
	}
	if got := QueueAhead(snap, y, args); !reflect.DeepEqual(got.Waiters, []int{15001}) {
		t.Fatalf("Y's view = %+v, want the outer waiter 15001, running its command, ahead", got)
	}

	// What is running still shuts the inner waiter: a third waiter running its
	// command, outside the inner waiter's chain.
	snap.Procs = append(snap.Procs,
		Proc{PID: 40001, PPID: 1, PGID: 40001, UID: otherUID, Age: time.Minute, Name: "abcd"},
		Proc{PID: 40002, PPID: 40001, PGID: 40001, UID: otherUID, Age: time.Minute, Name: "go"})
	args[40001] = []string{"implement", "load", "--wait", "--", "go", "test", "-race", "./..."}
	if got := QueueAhead(snap, inner, args); !reflect.DeepEqual(got.Waiters, []int{40001}) {
		t.Fatalf("inner waiter's view = %+v, want the waiter running its command, 40001", got)
	}
}
