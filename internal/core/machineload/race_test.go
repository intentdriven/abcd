package machineload

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// The waiter judges a race run by a go process's argv, never by a pattern over a
// command line. On 2026-10-11 a hand-written waiter, `pgrep -f "go test -race"`,
// matched its own shell, whose command line carries that string, so it saw a race
// run forever and never opened.

// waiterChain is a waiter started the way a session starts one: an agent's zsh
// whose command line holds the hand-written loop's words, running `go run`
// whose own arguments gate `go test -race`, running the built abcd (pid 60000).
func waiterChain() ([]Proc, map[int][]string) {
	procs := []Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: rootUID, Age: time.Hour, Name: "launchd"},
		{PID: 59000, PPID: 1, PGID: 59000, UID: callerUID, Age: time.Hour, Name: "zsh"},
		{PID: 59990, PPID: 59000, PGID: 59990, UID: callerUID, Age: time.Minute, Name: "go"},
		{PID: 60000, PPID: 59990, PGID: 59990, UID: callerUID, Age: time.Second, Name: "abcd"},
	}
	args := map[int][]string{
		59990: {"run", "./cmd/abcd", "implement", "load", "--wait", "--", "go", "test", "-race", "./internal/..."},
	}
	return procs, args
}

var waiter = Self{PID: 60000, UID: callerUID}

// TestWaiterNeverCountsItself is the self-match: the waiter's own chain carries
// "go test -race" in its command lines, which the hand-written pattern matched,
// and the waiter counts no race run.
func TestWaiterNeverCountsItself(t *testing.T) {
	procs, args := waiterChain()
	// The hand-written form's trap, kept here so the fixture stays honest: the
	// chain's command line does contain the words the old pattern looked for.
	cmdline := "go " + strings.Join(args[59990], " ")
	if !strings.Contains(cmdline, "go test -race") {
		t.Fatalf("fixture no longer reproduces the self-match: %q", cmdline)
	}
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}
	if got := RaceRuns(snap, waiter, args); len(got) != 0 {
		t.Fatalf("the waiter counted its own chain as race runs: %v", got)
	}

	// A PEER waiter of the same shape, outside the chain, is not a race run
	// either: its go command runs `run`, and `test -race` are its program's words.
	peer := Proc{PID: 61000, PPID: 1, PGID: 61000, UID: callerUID, Age: time.Minute, Name: "go"}
	snap.Procs = append(snap.Procs, peer)
	args[61000] = args[59990]
	if got := RaceRuns(snap, waiter, args); len(got) != 0 {
		t.Fatalf("a peer waiter counted as a race run: %v", got)
	}

	// A real race run, by any account, is.
	run := Proc{PID: 62000, PPID: 1, PGID: 62000, UID: otherUID, Age: time.Minute, Name: "go"}
	snap.Procs = append(snap.Procs, run)
	args[62000] = []string{"test", "-race", "-timeout", "20m", "./internal/..."}
	if got := RaceRuns(snap, waiter, args); !reflect.DeepEqual(got, []int{62000}) {
		t.Fatalf("race runs = %v, want [62000]", got)
	}
}

// TestRaceRunInTheWaitersAncestryIsExcluded: a waiter started from inside a race
// run (a test that shells out to abcd) does not wait for itself.
func TestRaceRunInTheWaitersAncestryIsExcluded(t *testing.T) {
	procs, args := waiterChain()
	args[59990] = []string{"test", "-race", "./internal/surface/cli/"}
	snap := Snapshot{HasLoad: true, Cores: 16, Procs: procs, HasProcs: true}
	if got := RaceRuns(snap, waiter, args); len(got) != 0 {
		t.Fatalf("an ancestor counted as a race run: %v", got)
	}
}

// TestRaceCandidatesAreGoProcessesOutsideTheChain: only processes the kernel
// records as executing go are asked for their arguments, and never the waiter's
// own chain.
func TestRaceCandidatesAreGoProcessesOutsideTheChain(t *testing.T) {
	procs, _ := waiterChain()
	procs = append(procs,
		Proc{PID: 70000, PPID: 1, Name: "go"},
		Proc{PID: 70001, PPID: 1, Name: "zsh"},
		Proc{PID: 70002, PPID: 1, Name: "gopls"},
	)
	got := raceCandidates(Snapshot{Procs: procs, HasProcs: true}, waiter)
	if !reflect.DeepEqual(got, []int{70000}) {
		t.Fatalf("candidates = %v, want [70000]", got)
	}
}

// TestIsRaceTestReadsTheGoCommandLine: the subcommand is the first argument that
// is not a flag (after the go command's own -C dir), and -race counts only as a
// flag of `go test`, before -args hands the rest to the test binary.
func TestIsRaceTestReadsTheGoCommandLine(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{"test", "-race", "./..."}, true},
		{[]string{"test", "-count=1", "-race", "-timeout", "20m", "./internal/..."}, true},
		{[]string{"test", "--race", "./..."}, true},
		{[]string{"test", "-race=true", "./..."}, true},
		{[]string{"test", "-race=1", "./..."}, true},
		{[]string{"-C", "/src", "test", "-race", "./..."}, true},
		{[]string{"-C=/src", "test", "-race", "./..."}, true},
		{[]string{"test", "-race=false", "./..."}, false},
		{[]string{"test", "./..."}, false},
		{[]string{"test", "./...", "-args", "-race"}, false},
		{[]string{"run", "./cmd/abcd", "implement", "load", "--wait", "--", "go", "test", "-race"}, false},
		{[]string{"build", "-race", "./..."}, false},
		{[]string{"vet", "./..."}, false},
		{nil, false},
	} {
		if got := isRaceTest(c.args); got != c.want {
			t.Errorf("isRaceTest(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

// TestDescends: a grandchild descends from its grandparent; a sibling and the
// process itself do not; a cycle in a torn snapshot ends the walk.
func TestDescends(t *testing.T) {
	procs := []Proc{
		{PID: 1, PPID: 0}, {PID: 10, PPID: 1}, {PID: 11, PPID: 10}, {PID: 12, PPID: 11},
		{PID: 20, PPID: 1}, {PID: 30, PPID: 31}, {PID: 31, PPID: 30},
	}
	for _, c := range []struct {
		pid, anc int
		want     bool
	}{
		{12, 10, true}, {11, 10, true}, {10, 10, false}, {20, 10, false}, {30, 10, false},
	} {
		if got := Descends(procs, c.pid, c.anc); got != c.want {
			t.Errorf("Descends(%d, %d) = %v, want %v", c.pid, c.anc, got, c.want)
		}
	}
	if a := lineage(procs, 12); !a[12] || !a[11] || !a[10] || !a[1] || a[20] {
		t.Errorf("lineage(12) = %v", a)
	}
}

// TestParseDarwinArgs reads `ps -o pid=,args=`: the program's own words are
// dropped through the first word whose base name is go, so an install path with a
// space in it still leaves the go command's arguments.
func TestParseDarwinArgs(t *testing.T) {
	out := []byte(` 1284 go test ./...
 1594 /opt/sdk/go1.26.9/bin/go test -race -timeout 20m ./internal/...
 1600 /Volumes/My Disk/go/bin/go test -race ./...
`)
	got := parseDarwinArgs(out)
	want := map[int][]string{
		1284: {"test", "./..."},
		1594: {"test", "-race", "-timeout", "20m", "./internal/..."},
		1600: {"test", "-race", "./..."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %v\nwant %v", got, want)
	}
}

// TestReadCmdlines reads /proc/<pid>/cmdline: NUL-separated argv, the program
// dropped, and a process gone between the listing and the read skipped.
func TestReadCmdlines(t *testing.T) {
	proc := fstest.MapFS{
		"100/cmdline": {Data: []byte("/usr/local/go/bin/go\x00test\x00-race\x00./...\x00")},
		"101/cmdline": {Data: []byte("go\x00test\x00has space\x00")},
	}
	got := readCmdlines(proc, []int{100, 101, 102})
	want := map[int][]string{100: {"test", "-race", "./..."}, 101: {"test", "has space"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %v\nwant %v", got, want)
	}
}
