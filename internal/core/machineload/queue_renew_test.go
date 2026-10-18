package machineload

import (
	"reflect"
	"testing"
	"time"
)

// TestAWaitersRenewalIsNotItsCommand (iss-2610170926295770): a queued waiter
// renews its holder's reservations through git and, at most, gh, run as its
// own children. Such a child is the waiter's own errand, like the ps it reads
// the machine with, never the gated command: the waiter stays queued, and
// does not read as one that has just opened. A child that is the command (make
// here) still marks it running.
func TestAWaitersRenewalIsNotItsCommand(t *testing.T) {
	for _, errand := range []string{"git", "gh"} {
		t.Run(errand, func(t *testing.T) {
			procs, args := waiterChain()
			procs[3].Age = 2 * time.Minute
			w, wArgs := peerWaiter(90001, 1, otherUID)
			snap := Snapshot{HasLoad: true, Cores: 16, Procs: append(procs, w...), HasProcs: true}
			args = merge(args, wArgs)

			// Renewing: a fresh git or gh child, beside the ps.
			snap.Procs = append(snap.Procs,
				Proc{PID: 90002, PPID: 90001, PGID: 90000, UID: otherUID, Name: "ps"},
				Proc{PID: 90003, PPID: 90001, PGID: 90000, UID: otherUID, Age: 0, Name: errand})
			if got := QueueAhead(snap, waiter, args); len(got.Waiters) != 0 {
				t.Fatalf("a later waiter renewing through %s is ahead, as if running its command: %+v", errand, got)
			}
			if got := JustOpened(snap, waiter, args, 4*time.Second, false); len(got) != 0 {
				t.Fatalf("a waiter renewing through %s reads as just opened: %v", errand, got)
			}

			// Running its command.
			snap.Procs = append(snap.Procs, Proc{PID: 90004, PPID: 90001, PGID: 90000, UID: otherUID, Age: time.Second, Name: "make"})
			if got := QueueAhead(snap, waiter, args); !reflect.DeepEqual(got.Waiters, []int{90001}) {
				t.Fatalf("ahead = %+v, want the waiter running its command, 90001", got)
			}
			if got := JustOpened(snap, waiter, args, 4*time.Second, false); !reflect.DeepEqual(got, []int{90001}) {
				t.Fatalf("just opened = %v, want the waiter whose command started a second ago, 90001", got)
			}
		})
	}
}
