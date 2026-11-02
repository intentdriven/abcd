package machineload

import (
	"slices"
	"testing"
)

// TestQueuedUnderNamesTheWaitersARunnersTreeHolds: of the waiters in the
// tree a runner started, only those queued at a gate are named; a waiter
// running its command, one whose only child is its own errand, an abcd that
// is no waiter and a waiter outside the tree are told apart
// (iss-2611021751406429).
func TestQueuedUnderNamesTheWaitersARunnersTreeHolds(t *testing.T) {
	snap := Snapshot{HasProcs: true, Procs: []Proc{
		{PID: 100, PPID: 1, Name: "claude"},
		{PID: 110, PPID: 100, Name: "zsh"},
		{PID: 120, PPID: 110, Name: "abcd"}, // queued at the first gate
		{PID: 121, PPID: 120, Name: "ps"},   // its errand
		{PID: 130, PPID: 110, Name: "abcd"}, // running its command
		{PID: 131, PPID: 130, Name: "make"},
		{PID: 140, PPID: 100, Name: "abcd"},              // the approver, no waiter
		{PID: 150, PPID: 110, Name: "abcd-darwin-arm64"}, // queued at the race gate
		{PID: 200, PPID: 1, Name: "abcd"},                // another tree's waiter
	}}
	args := map[int][]string{
		120: {"implement", "load", "--wait", "--", "make", "preflight"},
		130: {"implement", "load", "--wait", "--", "make", "preflight"},
		140: {"guard", "approve"},
		150: {"implement", "load", "--race-gate"},
		200: {"implement", "load", "--wait"},
	}
	if got, want := AbcdUnder(snap, 100), []int{120, 130, 140, 150}; !slices.Equal(got, want) {
		t.Errorf("AbcdUnder = %v, want %v", got, want)
	}
	if got, want := QueuedUnder(snap, 100, args), []int{120, 150}; !slices.Equal(got, want) {
		t.Errorf("QueuedUnder = %v, want %v", got, want)
	}
	if got := QueuedUnder(Snapshot{}, 100, args); got != nil {
		t.Errorf("an unread table names %v, want none", got)
	}
}
