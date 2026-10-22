package machineload

import (
	"math/rand/v2"
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

// TestQueueOrdersWaitersByStart: two waiters at one gate agree on one order,
// the earlier start first, whichever account each runs under. A preflight
// whose own waiter is queued at the gate is that waiter's place in the queue,
// not a running preflight.
func TestQueueOrdersWaitersByStart(t *testing.T) {
	procs, args := waiterChain()
	// This waiter started two minutes ago, before the other's 50 seconds.
	procs[3].Age = 2 * time.Minute
	other, otherArgs := otherPreflight(80000)
	// The other preflight is at its gate: its sh runs go run, whose abcd waits.
	other = other[:2]
	delete(otherArgs, 80002)
	w, wArgs := peerWaiter(80011, 80001, otherUID)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(append(procs, other...), w...), HasProcs: true}
	args = merge(merge(args, otherArgs), wArgs)
	args[60000] = []string{"implement", "load", "--wait", "--", "go", "test", "-race", "./internal/..."}

	// Its waiter started later: this waiter is first.
	if got := QueueAhead(snap, waiter, args); len(got.Preflights) != 0 || len(got.Waiters) != 0 {
		t.Fatalf("ahead = %+v, want nothing: the other preflight waits behind this waiter", got)
	}
	// And from the other side, this waiter is ahead of it.
	them := Self{PID: 80011, UID: otherUID}
	if got := QueueAhead(snap, them, args); !reflect.DeepEqual(got.Waiters, []int{60000}) {
		t.Fatalf("the other waiter's view = %+v, want this waiter 60000 ahead", got)
	}

	// A waiter that started earlier is ahead.
	low, lowArgs := peerWaiter(50001, 1, callerUID)
	low[0].Age, low[1].Age = 6*time.Minute, 5*time.Minute
	snap.Procs = append(snap.Procs, low...)
	args = merge(args, lowArgs)
	if got := QueueAhead(snap, waiter, args); !reflect.DeepEqual(got.Waiters, []int{50001}) {
		t.Fatalf("ahead = %+v, want the earlier waiter 50001", got)
	}
}

// TestQueueCountsAWaiterRunningItsCommand: a waiter with a child other than the
// ps it reads the machine with is running its command, and is ahead whenever it
// started.
func TestQueueCountsAWaiterRunningItsCommand(t *testing.T) {
	procs, args := waiterChain()
	// This waiter started two minutes ago, before the other's 50 seconds.
	procs[3].Age = 2 * time.Minute
	w, wArgs := peerWaiter(90001, 1, otherUID)
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, w...), HasProcs: true}
	args = merge(args, wArgs)

	// Reading the machine: its one child is ps.
	snap.Procs = append(snap.Procs, Proc{PID: 90002, PPID: 90001, PGID: 90000, UID: otherUID, Name: "ps"})
	if got := QueueAhead(snap, waiter, args); len(got.Waiters) != 0 {
		t.Fatalf("a waiting later waiter is ahead: %+v", got)
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
		if got := GoTestParallelism(snap, 1); got != c.want {
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

// queuedWaiter is a plain waiter at pid, a child of launchd, that has existed
// for age; a zero age is a start time the table could not give.
func queuedWaiter(pid int, uid uint32, age time.Duration) (Proc, []string) {
	return Proc{PID: pid, PPID: 1, PGID: pid, UID: uid, Age: age, Name: "abcd"},
		[]string{"implement", "load", "--wait", "--timeout", "3h"}
}

// queueSnap is a snapshot holding launchd and the waiters, with their args.
func queueSnap(ws ...struct {
	pid int
	age time.Duration
}) (Snapshot, map[int][]string) {
	snap := Snapshot{HasLoad: true, Cores: 16, HasProcs: true,
		Procs: []Proc{{PID: 1, PPID: 0, PGID: 1, UID: rootUID, Age: 50 * time.Hour, Name: "launchd"}}}
	args := map[int][]string{}
	for _, w := range ws {
		p, a := queuedWaiter(w.pid, otherUID, w.age)
		snap.Procs = append(snap.Procs, p)
		args[w.pid] = a
	}
	return snap, args
}

type qw = struct {
	pid int
	age time.Duration
}

// TestQueueOrdersWaitersByStartAcrossAPidWrap is the peer report of
// 2026-10-16: after a reboot the process ids wrapped, and waiters started
// between 16:07 and 17:26 with pids 50460 to 71218 all queued ahead of one
// started at 16:04 with pid 96274, which starved while lower pids kept
// arriving. A waiter is ahead by start alone, so the 16:04 waiter is first
// whatever its pid, the 16:07 waiter waits on it, and the 17:26 waiter on
// both. A waiter ahead with the higher pid is crossed: the one wait a waiter
// built before the start order can close into a cycle, which implement.WaitLoad
// bounds (WaitCrossedCap).
func TestQueueOrdersWaitersByStartAcrossAPidWrap(t *testing.T) {
	// Read at 17:30.
	snap, args := queueSnap(
		qw{96274, 86 * time.Minute}, // 16:04
		qw{50460, 83 * time.Minute}, // 16:07
		qw{71218, 4 * time.Minute},  // 17:26
	)
	for pid, want := range map[int]struct{ ahead, crossed []int }{
		96274: {nil, nil},
		50460: {[]int{96274}, []int{96274}},
		71218: {[]int{50460, 96274}, []int{96274}},
	} {
		got := QueueAhead(snap, Self{PID: pid, UID: otherUID}, args)
		if !slices.Equal(got.Waiters, want.ahead) || !slices.Equal(got.Queued, want.ahead) || !slices.Equal(got.Crossed, want.crossed) {
			t.Fatalf("waiter %d's view = %+v, want %v ahead, %v of them crossed: every earlier start is ahead, whatever its pid", pid, got, want.ahead, want.crossed)
		}
	}
}

// TestQueueAfterTheWrapOf20261022 is the table abcd-ae read on 2026-10-22:
// waiters started 19 s and 9 min earlier held pids 8034 and 12051, and
// waiters an hour old held 58628 and 73003. Counting a waiter ahead only with
// the lower pid as well, neither side waited on the other, and the open look
// stood the hour-old waiters back for every young low pid. By start alone the
// hour-old waiters are first, and both young ones wait on them.
func TestQueueAfterTheWrapOf20261022(t *testing.T) {
	snap, args := queueSnap(
		qw{58628, 71 * time.Minute},
		qw{73003, 63 * time.Minute},
		qw{12051, 9 * time.Minute},
		qw{8034, 19 * time.Second},
	)
	if got, want := queueOrder(snap.Procs[1:]), []int{58628, 73003, 12051, 8034}; !slices.Equal(got, want) {
		t.Fatalf("queue order = %v, want %v: earliest start first", got, want)
	}
	for pid, want := range map[int][]int{
		58628: nil,
		73003: {58628},
		12051: {58628, 73003},
		8034:  {12051, 58628, 73003},
	} {
		if got := QueueAhead(snap, Self{PID: pid, UID: otherUID}, args); !slices.Equal(got.Queued, want) {
			t.Fatalf("waiter %d sees %v queued ahead, want %v", pid, got.Queued, want)
		}
	}
	if got := QueueAhead(snap, Self{PID: 12051, UID: otherUID}, args); !slices.Equal(got.Crossed, []int{58628, 73003}) {
		t.Fatalf("waiter 12051's crossed = %v, want the two hour-old waiters, whose pids are higher", got.Crossed)
	}
	if got := QueueAhead(snap, Self{PID: 8034, UID: otherUID}, args); !slices.Equal(got.Crossed, []int{12051, 58628, 73003}) {
		t.Fatalf("waiter 8034's crossed = %v, want every waiter ahead of it", got.Crossed)
	}
}

// TestQueueOrderIsOneFromEveryRead: each waiter reads the table at its own
// moment, so every age in its read is shifted by the same amount from another
// waiter's read (macOS reads them in whole seconds), and a start is derived
// from one read's ages alone: the order, the tie groups and every waiter's view
// come out the same from each.
func TestQueueOrderIsOneFromEveryRead(t *testing.T) {
	base := []qw{{40000, 90 * time.Second}, {30000, 89 * time.Second}, {20000, 60 * time.Second}, {10000, 58 * time.Second}, {50000, 10 * time.Second}}
	var first []int
	var firstViews map[int][]int
	for shift := time.Duration(0); shift <= 5*time.Second; shift += time.Second {
		ws := slices.Clone(base)
		for i := range ws {
			ws[i].age += shift
		}
		snap, args := queueSnap(ws...)
		order := queueOrder(snap.Procs[1:])
		views := map[int][]int{}
		for _, w := range ws {
			views[w.pid] = QueueAhead(snap, Self{PID: w.pid, UID: otherUID}, args).Queued
		}
		if first == nil {
			first, firstViews = order, views
			if want := []int{30000, 40000, 10000, 20000, 50000}; !slices.Equal(order, want) {
				t.Fatalf("queue order = %v, want %v: two tie groups by pid, then the later start", order, want)
			}
			continue
		}
		if !slices.Equal(order, first) || !reflect.DeepEqual(views, firstViews) {
			t.Fatalf("read %s later: order %v and views %v, want %v and %v as from the first read", shift, order, views, first, firstViews)
		}
	}
}

// TestQueueBreaksAStartTieByPid: start times the table cannot tell apart, the
// same or within StartTolerance of each other, are ordered by pid, lowest
// first, so every waiter still agrees on one order.
func TestQueueBreaksAStartTieByPid(t *testing.T) {
	for _, gap := range []time.Duration{0, time.Second, StartTolerance * time.Second} {
		// The higher pid reads as started earlier, by gap.
		snap, args := queueSnap(qw{40000, time.Minute}, qw{30000, time.Minute - gap})
		if got := QueueAhead(snap, Self{PID: 40000, UID: otherUID}, args); !reflect.DeepEqual(got.Waiters, []int{30000}) || len(got.Crossed) != 0 {
			t.Fatalf("gap %s: the higher pid's view = %+v, want the lower pid 30000 ahead, not crossed", gap, got)
		}
		if got := QueueAhead(snap, Self{PID: 30000, UID: otherUID}, args); len(got.Waiters) != 0 {
			t.Fatalf("gap %s: the lower pid's view = %+v, want nothing ahead", gap, got)
		}
	}
	// Past the tolerance the start times are told apart: the earlier start is
	// ahead however high its pid, and the later is not ahead however low.
	snap, args := queueSnap(qw{40000, time.Minute}, qw{30000, time.Minute - (StartTolerance+1)*time.Second})
	if got := QueueAhead(snap, Self{PID: 30000, UID: otherUID}, args); !slices.Equal(got.Waiters, []int{40000}) || !slices.Equal(got.Crossed, []int{40000}) {
		t.Fatalf("past the tolerance the later start's view = %+v, want 40000 ahead, crossed", got)
	}
	if got := QueueAhead(snap, Self{PID: 40000, UID: otherUID}, args); len(got.Waiters) != 0 {
		t.Fatalf("past the tolerance the earlier start's view = %+v, want nothing ahead", got)
	}
}

// TestQueueSortsAnUnknownStartLast: a waiter whose start time the table could
// not give (no measurable age) queues behind every waiter whose start time it
// could, in pid order among the others like it, so it holds back no known
// start even with the lower pid, and every known start is ahead of it.
func TestQueueSortsAnUnknownStartLast(t *testing.T) {
	snap, args := queueSnap(qw{20000, time.Minute}, qw{30000, 0}, qw{40000, 0}, qw{90000, time.Minute})
	for pid, want := range map[int][]int{
		20000: nil,
		30000: {20000, 90000},
		40000: {20000, 30000, 90000},
		// The unknown starts 30000 and 40000 queue behind it, lower pids or not.
		90000: {20000},
	} {
		if got := QueueAhead(snap, Self{PID: pid, UID: otherUID}, args); !slices.Equal(got.Waiters, want) {
			t.Fatalf("waiter %d sees %v ahead, want %v", pid, got.Waiters, want)
		}
	}
	// A waiter missing from the table has no start time either: every known
	// start is ahead of it, and the unknown ones with the lower pid.
	if got := QueueAhead(snap, Self{PID: 35000, UID: otherUID}, args); !reflect.DeepEqual(got.Waiters, []int{20000, 30000, 90000}) {
		t.Fatalf("a waiter absent from the table sees %+v, want 20000, 30000 and 90000 ahead", got)
	}
}

// TestQueueTakesAZeroAgeAsUnknown: macOS reads ages in whole seconds, so a
// waiter under a second old reads zero, which is an unknown start, and queues
// behind a waiter one second old even with the lower pid: read as a start one
// second apart, it would tie and lead by pid. A second later both are known
// and tie, and the lower pid is ahead: the order the two settle on.
func TestQueueTakesAZeroAgeAsUnknown(t *testing.T) {
	snap, args := queueSnap(qw{10, 0}, qw{20, time.Second})
	if got := QueueAhead(snap, Self{PID: 10, UID: otherUID}, args); !slices.Equal(got.Waiters, []int{20}) {
		t.Fatalf("waiter 10 sees %v ahead, want [20]: a zero age is an unknown start, behind every known one", got.Waiters)
	}
	if got := QueueAhead(snap, Self{PID: 20, UID: otherUID}, args); len(got.Waiters) != 0 {
		t.Fatalf("waiter 20 sees %v ahead, want nothing: 10's start is unknown", got.Waiters)
	}
	snap, args = queueSnap(qw{10, time.Second}, qw{20, 2 * time.Second})
	if got := QueueAhead(snap, Self{PID: 20, UID: otherUID}, args); !slices.Equal(got.Waiters, []int{10}) {
		t.Fatalf("a second later 20 sees %v ahead, want [10]: the two tie and the lower pid leads", got.Waiters)
	}
}

// TestQueueChainedTiesHoldOneOrder: a tie is not transitive, so three waiters
// each within StartTolerance of the next, the first and last apart by more,
// would make a cycle if each pair were judged alone (A before B and B before
// C by pid, C before A by start), and all three would wait on each other
// until their timeouts. Waiters linked by ties form one group ordered by pid,
// so every view agrees: one is first, one second, one third.
func TestQueueChainedTiesHoldOneOrder(t *testing.T) {
	snap, args := queueSnap(qw{100, 10 * time.Second}, qw{200, 11500 * time.Millisecond}, qw{300, 13 * time.Second})
	want := map[int][]int{100: nil, 200: {100}, 300: {100, 200}}
	for pid, w := range want {
		got := QueueAhead(snap, Self{PID: pid, UID: otherUID}, args)
		if !slices.Equal(got.Waiters, w) {
			t.Fatalf("waiter %d sees %v ahead, want %v", pid, got.Waiters, w)
		}
	}
}

// pidRuleAhead is the queue as a waiter built before iss-2610161634043273
// (abcd v0.15.0 and earlier) reads it, restated from that code for fixtures
// whose waiters are all queued (none runs its command or sits beneath another
// waiter): every queued waiter with a lower pid is ahead.
func pidRuleAhead(snap Snapshot, self Self, args map[int][]string) []int {
	var out []int
	for _, pid := range GateCandidates(snap, self) {
		if isLoadWaiter(args[pid]) && pid < self.PID {
			out = append(out, pid)
		}
	}
	return out
}

// bothRuleAhead is the queue as abcd v0.15.1 reads it, restated from that
// code for the same fixtures: a queued waiter is ahead when it is ahead by
// start (queueOrder) and has the lower pid as well.
func bothRuleAhead(snap Snapshot, self Self, args map[int][]string) []int {
	var queue []Proc
	for _, p := range snap.Procs {
		if p.PID == self.PID || isLoadWaiter(args[p.PID]) {
			queue = append(queue, p)
		}
	}
	var out []int
	for _, pid := range queueOrder(queue) {
		if pid == self.PID {
			break
		}
		if pid < self.PID {
			out = append(out, pid)
		}
	}
	slices.Sort(out)
	return out
}

// TestQueueMixedVersionsCycleOnlyThroughACrossedWait is the deadlock of the
// night of 2026-10-16, which this code meets again: a waiter built from it,
// the youngest with the lowest pid, and two built from the pid-ordered code
// (v0.15.0 and earlier), older with higher pids. This one sees the two older
// starts ahead, and they see its lower pid ahead, so each waits on the other.
// Every such wait from this code is crossed (ahead by start with the higher
// pid), and implement.WaitLoad stops counting a crossed waiter once it alone
// has held the gate shut for WaitCrossedCap: without its crossed waits the
// cycle is gone. Against v0.15.1, which waits only on an earlier start with
// the lower pid, there is no cycle at all.
func TestQueueMixedVersionsCycleOnlyThroughACrossedWait(t *testing.T) {
	snap, args := queueSnap(
		qw{30000, 5 * time.Minute}, // this code: youngest, lowest pid
		qw{60000, 3 * time.Hour},   // older code
		qw{70000, 2 * time.Hour},   // older code
	)
	self := Self{PID: 30000, UID: otherUID}
	got := QueueAhead(snap, self, args)
	if !slices.Equal(got.Waiters, []int{60000, 70000}) || !slices.Equal(got.Crossed, []int{60000, 70000}) {
		t.Fatalf("this code's view = %+v, want both older starts ahead, both crossed", got)
	}
	for name, rule := range map[string]func(Snapshot, Self, map[int][]string) []int{"v0.15.0": pidRuleAhead, "v0.15.1": bothRuleAhead} {
		view := map[int][]int{30000: got.Waiters}
		for _, pid := range []int{60000, 70000} {
			view[pid] = rule(snap, Self{PID: pid, UID: otherUID}, args)
		}
		cycle := waitCycle(view)
		if name == "v0.15.1" && cycle != nil {
			t.Fatalf("beside %s: the waits %v close the cycle %v", name, view, cycle)
		}
		if name == "v0.15.0" && cycle == nil {
			t.Fatalf("beside %s: the waits %v close no cycle; want the one the crossed waits make", name, view)
		}
		view[30000] = got.Uncrossed().Waiters
		if cycle := waitCycle(view); cycle != nil {
			t.Fatalf("beside %s, past the cap: the waits %v close the cycle %v", name, view, cycle)
		}
	}
}

// TestQueueWaitsFormNoCycle: across many tables of a few waiters, each built
// from this code, from v0.15.1's (ahead by start and with the lower pid) or
// from the pid-ordered code at random, no waiter waits on itself through any
// chain of waits once this code's crossed waits are dropped, as WaitLoad drops
// them past WaitCrossedCap, even with each waiter of this code reading its own
// table a second either side of the others' (ages read at different looks, on
// macOS in whole seconds): every wait left points at a lower pid. With no
// pid-ordered waiter among them and one table read alike, none waits on
// itself even before: every wait then points at an earlier start, and a cycle
// needs one that comes back down to a later start, which only the pid-ordered
// code makes. A view read a second off another can disagree about a tie at
// the tolerance's edge for that one look; the next look reads afresh.
func TestQueueWaitsFormNoCycle(t *testing.T) {
	const (
		thisCode = iota
		bothRule
		pidRule
	)
	rng := rand.New(rand.NewPCG(2610222125, 175640))
	for iter := 0; iter < 5000; iter++ {
		n := 2 + rng.IntN(4)
		pids := rng.Perm(900)[:n]
		ages := make([]int, n)
		code := make([]int, n)
		anyPidRule := false
		for i := range pids {
			pids[i] += 100
			ages[i] = rng.IntN(21) // seconds; zero is an unknown start
			code[i] = rng.IntN(3)
			anyPidRule = anyPidRule || code[i] == pidRule
		}
		ahead := func(v int, shifted, uncrossed bool) []int {
			ws := make([]qw, n)
			for i := range pids {
				age := ages[i]
				if shifted && age > 0 && code[v] == thisCode {
					age = max(1, age+rng.IntN(3)-1)
				}
				ws[i] = qw{pids[i], time.Duration(age) * time.Second}
			}
			snap, args := queueSnap(ws...)
			self := Self{PID: pids[v], UID: otherUID}
			switch code[v] {
			case pidRule:
				return pidRuleAhead(snap, self, args)
			case bothRule:
				return bothRuleAhead(snap, self, args)
			}
			a := QueueAhead(snap, self, args)
			for _, c := range a.Crossed {
				if c < self.PID {
					t.Fatalf("table %d: waiter %d counts %d crossed, whose pid is lower", iter, self.PID, c)
				}
			}
			if uncrossed {
				return a.Uncrossed().Waiters
			}
			return a.Waiters
		}
		past, full := map[int][]int{}, map[int][]int{}
		for v := range pids {
			past[pids[v]] = ahead(v, true, true)
			full[pids[v]] = ahead(v, false, false)
		}
		if cycle := waitCycle(past); cycle != nil {
			t.Fatalf("table %d: pids %v, ages %v s, code %v: past the cap the waits %v close the cycle %v", iter, pids, ages, code, past, cycle)
		}
		if cycle := waitCycle(full); cycle != nil && !anyPidRule {
			t.Fatalf("table %d: pids %v, ages %v s, code %v: with no pid-ordered waiter the waits %v close the cycle %v", iter, pids, ages, code, full, cycle)
		}
	}
}

// waitCycle returns a cycle in the wait graph edges (waiter to the waiters it
// sees ahead), or nil.
func waitCycle(edges map[int][]int) []int {
	const (
		unseen = iota
		open
		done
	)
	state := map[int]int{}
	var path []int
	var visit func(int) []int
	visit = func(v int) []int {
		state[v] = open
		path = append(path, v)
		for _, w := range edges[v] {
			switch state[w] {
			case open:
				i := slices.Index(path, w)
				return append(slices.Clone(path[i:]), w)
			case unseen:
				if c := visit(w); c != nil {
					return c
				}
			}
		}
		path = path[:len(path)-1]
		state[v] = done
		return nil
	}
	for v := range edges {
		if state[v] == unseen {
			if c := visit(v); c != nil {
				return c
			}
		}
	}
	return nil
}

// TestQueueAPidWrapStarvesNoWaiter: two waiters started before the machine's
// process ids wrapped, with high pids, and a stream of later waiters after it,
// one a minute with low pids, every one built from this code. Each minute one
// waiter at the front (one that sees nothing queued ahead of it) is admitted,
// chosen at random as the settle jitter chooses. A waiter is only ever held
// behind waiters that started before it, so later arrivals never delay it, and
// every waiter is admitted in the order it started. Ordered by pid, every
// later low pid queued ahead of the early two and they starved.
func TestQueueAPidWrapStarvesNoWaiter(t *testing.T) {
	rng := rand.New(rand.NewPCG(96274, 50460))
	type waiter struct {
		pid     int
		started int // minute
	}
	queue := []waiter{{96274, 0}, {96300, 1}}
	admitted := map[int]int{}
	next := 300
	for minute := 2; minute < 60; minute++ {
		queue = append(queue, waiter{next, minute})
		next++
		var ws []qw
		started := map[int]int{}
		for _, w := range queue {
			ws = append(ws, qw{w.pid, time.Duration(minute-w.started)*time.Minute + time.Second})
			started[w.pid] = w.started
		}
		snap, args := queueSnap(ws...)
		var front []int
		for _, w := range queue {
			ahead := QueueAhead(snap, Self{PID: w.pid, UID: otherUID}, args).Waiters
			for _, a := range ahead {
				if started[a] > w.started {
					t.Fatalf("minute %d: waiter %d (started minute %d) is held behind %d, which started later, at minute %d", minute, w.pid, w.started, a, started[a])
				}
			}
			if len(ahead) == 0 {
				front = append(front, w.pid)
			}
		}
		if len(front) != 1 || front[0] != queue[0].pid {
			t.Fatalf("minute %d: the front is %v, want only the earliest start %d", minute, front, queue[0].pid)
		}
		pick := front[rng.IntN(len(front))]
		admitted[pick] = minute
		queue = slices.DeleteFunc(queue, func(w waiter) bool { return w.pid == pick })
	}
	for pid, want := range map[int]int{96274: 2, 96300: 3} {
		if m := admitted[pid]; m != want {
			t.Fatalf("the early waiter %d was admitted at minute %d (admitted: %v), want minute %d, first in its turn", pid, m, admitted, want)
		}
	}
}

// TestStandBackFromAnOpenerAheadOrLower: two waiters that open in one moment
// each look once more (implement.WaitLoad), and one goes on only past an
// opener it precedes by start and by pid alike. An opener that started
// earlier is ahead in the queue, so a later start stands back from it however
// low its pid: standing the lower pid back instead let a stream of young
// low-pid starters hold an hour-old waiter back (iss-2610222125175640). An
// opener with the lower pid that started later may run an older abcd (v0.15.1
// stands back only from a lower pid, v0.15.0 takes no such look), which goes
// on past this one; were
// this one to go on too, both commands would run and under `--wait -- make
// preflight` each preflight's own gate would wait on the other's waiter until
// its timeout, so this one stands back from it as well. Two waiters of this
// code seldom open in one moment, since the later one waits on the earlier;
// where they do, both stand back and re-draw their jitters. Inside one tie
// the order is by pid, so exactly one goes on.
func TestStandBackFromAnOpenerAheadOrLower(t *testing.T) {
	for _, c := range []struct {
		name       string
		self, peer qw
		standBack  bool
	}{
		{"a later start with the lower pid stands back from the earlier", qw{8034, 19 * time.Second}, qw{58628, 71 * time.Minute}, true},
		{"an earlier start with the higher pid stands back from a later lower pid", qw{58628, 71 * time.Minute}, qw{8034, 19 * time.Second}, true},
		{"an earlier start with the lower pid goes on", qw{8034, 71 * time.Minute}, qw{58628, 19 * time.Second}, false},
		{"a later start with the higher pid stands back", qw{58628, 19 * time.Second}, qw{8034, 71 * time.Minute}, true},
		{"inside one tie the lower pid goes on", qw{8034, 60 * time.Second}, qw{58628, 61 * time.Second}, false},
		{"inside one tie the higher pid stands back", qw{58628, 61 * time.Second}, qw{8034, 60 * time.Second}, true},
		{"inside one tie the higher pid stands back at the tolerance", qw{58628, 60*time.Second + StartTolerance*time.Second}, qw{8034, 60 * time.Second}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			snap, _ := queueSnap(c.self, c.peer)
			got := StandBackFrom(snap, Self{PID: c.self.pid, UID: otherUID}, []int{c.peer.pid})
			if c.standBack != slices.Equal(got, []int{c.peer.pid}) || (!c.standBack && len(got) != 0) {
				t.Fatalf("stand back from %v, want standing back %v", got, c.standBack)
			}
		})
	}
}

// TestJustOpenedNamesWaitersThatOpenedWithinTheWindow: an opener's look just
// before it starts its command (implement.WaitLoad) names the other waiters
// that are running their command and opened within the window: a waiter's
// oldest child other than ps is no older than it. A waiter that has run its
// command for longer opened before this one's open look and would have shut
// it, a waiter whose one child is ps is reading the machine, and the waiter's
// own ancestors never count.
func TestJustOpenedNamesWaitersThatOpenedWithinTheWindow(t *testing.T) {
	const window = 4 * time.Second
	procs, args := waiterChain()
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}
	add := func(pid int, children ...Proc) {
		w, a := peerWaiter(pid, 1, otherUID)
		snap.Procs = append(snap.Procs, w...)
		args = merge(args, a)
		snap.Procs = append(snap.Procs, children...)
	}
	// Opened a second ago: its marker child, then its command beside it.
	add(30001, Proc{PID: 30002, PPID: 30001, UID: otherUID, Age: time.Second, Name: "sleep"},
		Proc{PID: 30003, PPID: 30001, UID: otherUID, Name: "make"})
	// Opened at the window's edge.
	add(31001, Proc{PID: 31002, PPID: 31001, UID: otherUID, Age: window, Name: "make"})
	// Running its command for ten minutes.
	add(32001, Proc{PID: 32002, PPID: 32001, UID: otherUID, Age: 10 * time.Minute, Name: "make"})
	// Reading the machine.
	add(33001, Proc{PID: 33002, PPID: 33001, UID: otherUID, Name: "ps"})
	// Queued.
	add(34001)
	// An abcd that is no waiter, with a fresh child.
	snap.Procs = append(snap.Procs,
		Proc{PID: 35001, PPID: 1, UID: otherUID, Age: time.Minute, Name: "abcd"},
		Proc{PID: 35002, PPID: 35001, UID: otherUID, Name: "git"})
	args[35001] = []string{"status"}
	if got := JustOpened(snap, waiter, args, window, false); !reflect.DeepEqual(got, []int{30001, 31001}) {
		t.Fatalf("just opened = %v, want the two waiters that opened within %s, 30001 and 31001", got, window)
	}

	// An inner waiter's own ancestor opened a moment ago; it is the work this
	// waiter is part of, never another opener.
	inner := Self{PID: 20001, UID: callerUID}
	snap = Snapshot{HasLoad: true, Cores: 16, HasProcs: true, Procs: []Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: rootUID, Age: time.Hour, Name: "launchd"},
		{PID: 15001, PPID: 1, PGID: 15001, UID: callerUID, Age: time.Hour, Name: "abcd"},
		{PID: 15002, PPID: 15001, PGID: 15001, UID: callerUID, Age: time.Second, Name: "make"},
		{PID: 20001, PPID: 15002, PGID: 15001, UID: callerUID, Name: "abcd"},
	}}
	args = map[int][]string{
		15001: {"implement", "load", "--wait", "--", "make", "preflight"},
		15002: {"preflight"},
		20001: {"implement", "load", "--wait", "--", "go", "test", "-race", "./..."},
	}
	if got := JustOpened(snap, inner, args, window, false); len(got) != 0 {
		t.Fatalf("just opened = %v for an inner waiter, want nothing: its ancestor is its own work", got)
	}
}

// TestQueuedAtNamesTheQueueAtOneGate: QueuedAt is every waiter queued at one
// gate, ahead of self or not: a first gate's waiter, a race-command waiter
// (`--wait -- go test -race`) still queued at the first gate, and at the race
// gate only race waiters. A waiter running its command is not queued, and
// self's own ancestors never count.
func TestQueuedAtNamesTheQueueAtOneGate(t *testing.T) {
	self := Self{PID: 20001, UID: callerUID}
	snap := Snapshot{HasLoad: true, Cores: 16, HasProcs: true, Procs: []Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: rootUID, Age: time.Hour, Name: "launchd"},
		// self beneath its own waiter running `make preflight`
		{PID: 15001, PPID: 1, PGID: 15001, UID: callerUID, Age: time.Hour, Name: "abcd"},
		{PID: 15002, PPID: 15001, PGID: 15001, UID: callerUID, Age: time.Minute, Name: "make"},
		{PID: 20001, PPID: 15002, PGID: 15001, UID: callerUID, Age: time.Second, Name: "abcd"},
		{PID: 30001, PPID: 1, PGID: 30001, UID: otherUID, Age: 9 * time.Minute, Name: "abcd"},
		{PID: 30002, PPID: 1, PGID: 30002, UID: otherUID, Age: 8 * time.Minute, Name: "abcd"},
		{PID: 30003, PPID: 1, PGID: 30003, UID: otherUID, Age: 7 * time.Minute, Name: "abcd"},
		{PID: 30004, PPID: 30003, PGID: 30003, UID: otherUID, Age: time.Minute, Name: "sh"},
		{PID: 30005, PPID: 1, PGID: 30005, UID: otherUID, Age: 6 * time.Minute, Name: "abcd"},
		{PID: 30006, PPID: 1, PGID: 30006, UID: otherUID, Age: 5 * time.Minute, Name: "abcd"},
		{PID: 30007, PPID: 30006, PGID: 30006, UID: otherUID, Age: time.Minute, Name: "go"},
		{PID: 30008, PPID: 1, PGID: 30008, UID: otherUID, Age: 4 * time.Minute, Name: "abcd"},
		{PID: 30009, PPID: 30008, PGID: 30008, UID: otherUID, Name: "ps"},
	}}
	args := map[int][]string{
		15001: {"implement", "load", "--wait", "--", "make", "preflight"},
		15002: {"preflight"},
		20001: {"implement", "load", "--wait", "--timeout", "2h"},
		30001: {"implement", "load", "--wait", "--timeout", "2h"},
		30002: {"implement", "load", "--race-gate", "--timeout", "2h"},
		30003: {"implement", "load", "--wait", "--", "sh", "-c", "work"},
		30005: {"implement", "load", "--wait", "--", "go", "test", "-race", "./..."},
		30006: {"implement", "load", "--wait", "--", "go", "test", "-race", "./..."},
		30007: {"test", "-race", "./..."},
		// Reading the machine: its one child is ps, so it is still queued.
		30008: {"implement", "load", "--wait", "--timeout", "2h"},
	}
	for _, c := range []struct {
		race bool
		want []int
	}{
		{false, []int{30001, 30005, 30008}},
		{true, []int{30002}},
	} {
		if got := QueuedAt(snap, self, args, c.race); !slices.Equal(got, c.want) {
			t.Fatalf("race gate %v: queued %v, want %v", c.race, got, c.want)
		}
	}
	if got := QueuedAt(Snapshot{}, self, args, false); got != nil {
		t.Fatalf("an unread table: queued %v, want none", got)
	}
}
