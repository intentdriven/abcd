package machineload

import (
	"reflect"
	"testing"
	"time"
)

// A lone race run is gated with `abcd implement load --wait -- go test -race
// ./...`. Its waiter waits at the first gate, but its command is a race run, so
// from the moment it opens it counts as one for both gates (RaceCommandWaiters),
// before the go test it starts shows in the table.

// TestIsRaceCommand: a gated command is a race run when its program is go and
// its go command is `test` with -race, read as a race run's arguments are.
func TestIsRaceCommand(t *testing.T) {
	for _, c := range []struct {
		cmd  []string
		want bool
	}{
		{[]string{"go", "test", "-race", "./..."}, true},
		{[]string{"/usr/local/go/bin/go", "test", "-p", "4", "-race", "./internal/..."}, true},
		{[]string{"go", "-C", "sub", "test", "--race=true", "./..."}, true},
		{[]string{"go", "test", "./..."}, false},
		{[]string{"go", "run", "./cmd/x", "test", "-race"}, false},
		{[]string{"make", "preflight"}, false},
		{[]string{"sh", "-c", "go test -race ./..."}, false},
		{nil, false},
	} {
		if got := IsRaceCommand(c.cmd); got != c.want {
			t.Errorf("IsRaceCommand(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

// TestRaceCommandWaiters: a first gate's waiter running a command that is a
// race run is one, from its mark onwards; a queued one, a waiter running any
// other command, and the caller's own chain are not.
func TestRaceCommandWaiters(t *testing.T) {
	m := time.Minute
	raceCmd := []string{"implement", "load", "--wait", "--timeout", "2h", "--", "go", "test", "-race", "./..."}
	procs := []Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 50 * time.Hour, Name: "launchd"},
		// Running its race command, shown by its mark alone.
		{PID: 70001, PPID: 1, PGID: 70001, UID: otherUID, Age: 2 * m, Name: "abcd"},
		{PID: 70002, PPID: 70001, PGID: 70001, UID: otherUID, Age: time.Second, Name: "sleep"},
		// Queued, reading the table with ps.
		{PID: 70101, PPID: 1, PGID: 70101, UID: otherUID, Age: m, Name: "abcd"},
		{PID: 70102, PPID: 70101, PGID: 70101, UID: otherUID, Age: 0, Name: "ps"},
		// Running a preflight.
		{PID: 70201, PPID: 1, PGID: 70201, UID: otherUID, Age: 3 * m, Name: "abcd"},
		{PID: 70202, PPID: 70201, PGID: 70201, UID: otherUID, Age: 3 * m, Name: "make"},
		// The caller, itself a race command's waiter, and its own mark.
		{PID: 60000, PPID: 1, PGID: 60000, UID: 501, Age: m, Name: "abcd"},
		{PID: 60001, PPID: 60000, PGID: 60000, UID: 501, Age: 0, Name: "sleep"},
	}
	args := map[int][]string{
		70001: raceCmd,
		70101: raceCmd,
		70201: {"implement", "load", "--wait", "--", "make", "preflight"},
		70202: {"preflight"},
		60000: raceCmd,
	}
	snap := Snapshot{Procs: procs, HasProcs: true}
	if got := RaceCommandWaiters(snap, Self{PID: 60000, UID: 501}, args); !reflect.DeepEqual(got, []int{70001}) {
		t.Fatalf("RaceCommandWaiters = %v, want [70001]", got)
	}
	if got := RaceCommandWaiters(Snapshot{}, Self{PID: 60000, UID: 501}, args); got != nil {
		t.Fatalf("RaceCommandWaiters on an unread table = %v, want none", got)
	}
}
