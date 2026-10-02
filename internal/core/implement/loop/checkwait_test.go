package loop

// checkwait_test.go is ruling DR6d-2 (the product thinker, 2026-10-02): "a
// missing preflight receipt for one lane: CARRY ON, WAIT SHOWN — other lanes
// continue; the lane waits for its receipt like the merge wait (contend, not
// refuse), and the status shows 'waiting for its full check (since HH:MM)'."

import (
	"strings"
	"testing"
	"time"
)

// checkWaits counts the run record's entries naming a lane's wait for its full
// check.
func checkWaits(st State, laneID string) int {
	n := 0
	for _, e := range st.Record {
		if e.Lane == laneID && strings.Contains(e.Note, "waiting for its full check") {
			n++
		}
	}
	return n
}

// One lane reaches its push with no preflight receipt while another lane has
// an implementer out: the call is a wait for that lane alone (a contention, as
// the merge wait is), the time the wait began is written once and kept across
// calls, the other lane keeps moving with the wait named beside its move, and
// the status reads "waiting for its full check (since HH:MM)". The receipt
// minted, the lane pushes and the wait is over.
func TestALaneWithoutItsFullCheckWaitsWhileTheOthersCarryOn(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("3")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.roundPassed(t, "lane-1")

	// Lane 1 lands as far as its push; lane 2's implementer is still out, so
	// nothing else moves and the wait is the call's answer.
	f.now = time.Date(2026, 9, 30, 9, 30, 0, 0, time.UTC)
	var wait *Refusal
	for range 10 {
		_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
		if err != nil {
			wait = mustRefusal(t, err)
			break
		}
	}
	if wait == nil || !wait.Contention || wait.Lane != "lane-1" || !strings.Contains(wait.Reason, "waiting for its full check (since 09:30)") {
		t.Fatalf("the lane without its receipt waits, a contention naming the time the wait began: %+v", wait)
	}
	if !strings.Contains(wait.Remedy, "make preflight") {
		t.Fatalf("the remedy names how the receipt is minted: %s", wait.Remedy)
	}
	st := f.state(t)
	l1 := f.lane(t, "lane-1")
	if l1.Landing == nil || l1.Landing.Pushed != "" || l1.Landing.CheckWaitSince == nil || !l1.Landing.CheckWaitSince.Equal(f.now) || checkWaits(st, "lane-1") != 1 {
		t.Fatalf("the time the wait began is written on the lane's landing, and the record names it once: %+v\n%s", l1.Landing, recordText(st))
	}
	if got := l1.CheckWait(); got != "waiting for its full check (since 09:30)" {
		t.Fatalf("the lane's status: %q", got)
	}

	// Seven minutes on, lane 2's implementer hands back: the call moves lane
	// 2, names lane 1's wait beside it, and keeps the time the wait began.
	f.now = f.now.Add(7 * time.Minute)
	f.implement(t, "lane-2", "two.txt")
	res := f.step(t)
	if res.Lane != "lane-2" || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the other lane carries on: %+v", res)
	}
	if len(res.Blocked) != 1 || res.Blocked[0].Lane != "lane-1" || !strings.Contains(res.Next, "waiting for its full check (since 09:30)") {
		t.Fatalf("the wait is named beside the move, with the time it began: %+v", res)
	}
	st = f.state(t)
	if l1 := f.lane(t, "lane-1"); !l1.Landing.CheckWaitSince.Equal(time.Date(2026, 9, 30, 9, 30, 0, 0, time.UTC)) || checkWaits(st, "lane-1") != 1 {
		t.Fatalf("the time the wait began is kept across calls: %+v\n%s", l1.Landing, recordText(st))
	}
	lanes, err := StatusLanes(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, s := range lanes {
		for _, l := range s.Lanes {
			if l.Lane == "lane-1" {
				row = l.Waiting
			}
		}
	}
	if row != "waiting for its full check (since 09:30)" {
		t.Fatalf("the status block's row for lane 1: %q in %+v", row, lanes)
	}

	// The receipt minted, lane 1 pushes and its wait is over.
	preflighted(t, l1, l1.HeadSHA)
	if r := f.step(t); r.Lane != "lane-1" {
		t.Fatalf("lane 1 pushes on its receipt: %+v", r)
	}
	if l1 := f.lane(t, "lane-1"); l1.Landing.Pushed == "" || l1.Landing.CheckWaitSince != nil || l1.CheckWait() != "" {
		t.Fatalf("the push ends the wait: %+v", l1.Landing)
	}
}
