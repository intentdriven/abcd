package loop

// drain_wait.go is the drain's wait at the cap and on held work
// (spc-2610161044461448 step 6; iss-2610161015371794, iss-2610161015422836;
// dcn-2610161040364312). A drain whose holder is at the cap of live
// reservations, counted in the move's view before any reservation or
// judgement, or whose every eligible issue left is held by another, neither
// refuses nor ends: it writes waiting_until, when it looks again, fifteen
// minutes on, names what it waits on (the holder's reservations and the pull
// requests waiting to merge, or the holds), and keeps its judgements. The
// wait is kept apart from next_eligible_at, so it never resets the pace
// window, and a move before it ends still looks: passing over held work
// fetches and pushes nothing, so the move takes any issue freed since, and a
// move that still waits keeps the time first written. The drain stops as
// empty only when nothing eligible is left, held or not.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// The waits a drain makes, in waiting.
const (
	// DrainWaitCap: the drain's holder is at the cap of live reservations.
	DrainWaitCap = "cap"
	// DrainWaitHeld: every eligible issue left is held by another.
	DrainWaitHeld = "held"
)

// drainWaitKinds are the waits the drain writes.
var drainWaitKinds = []string{DrainWaitCap, DrainWaitHeld}

// drainWaitMinutes is how long a waiting drain waits before it looks again:
// a bounded poll, short against the two-hour lease a held issue's holder
// renews at its half.
const drainWaitMinutes = 15

// drainWait is the wait a drain was in as its move began.
type drainWait struct {
	until *time.Time
	kind  string
}

// waitFrom takes the wait st records off it and returns it, so a move that
// takes work leaves none and one that still waits writes it again.
func waitFrom(st *DrainState) drainWait {
	w := drainWait{until: st.WaitingUntil, kind: st.Waiting}
	st.WaitingUntil, st.Waiting = nil, ""
	return w
}

// waitMove writes the wait of kind into st and the move's next step into
// res: waiting_until fifteen minutes on, or, while the same wait still
// stands, the time it was first written, or, for held work, a lapsed
// request's cool-down ending sooner (cooling, zero when none). It reports
// that the state is to be written.
func waitMove(st *DrainState, res *DrainResult, prev drainWait, kind, why string, now, cooling time.Time) bool {
	until := now.Add(drainWaitMinutes * time.Minute)
	if prev.until != nil && prev.kind == kind && now.Before(*prev.until) {
		until = *prev.until
	}
	if !cooling.IsZero() && cooling.Before(until) {
		until = cooling
	}
	st.WaitingUntil, st.Waiting = &until, kind
	at := stamp(until)
	res.Next = joinNext(res.Next, fmt.Sprintf("the drain waits until %s: %s; run `abcd drain` again at or after %s", at, why, at))
	return true
}

// capWaitText names a wait at the cap: the holder's live reservations, the
// pull requests among this checkout's runs waiting to merge on them, and the
// advice to wait for a merge rather than release one.
func capWaitText(held []string, runs []State) string {
	var prs []string
	for _, id := range held {
		for _, r := range runs {
			if !recordid.SameID(r.Key, id) || r.Complete() {
				continue
			}
			for _, l := range r.Lanes {
				if l.PR > 0 && l.Stage != StageDone && l.Stage != StageDiscarded {
					prs = append(prs, fmt.Sprintf("#%d (%s)", l.PR, id))
				}
			}
		}
	}
	slices.Sort(prs)
	prs = slices.Compact(prs)
	waiting := "none of them a lane's pull request in this checkout"
	if len(prs) > 0 {
		waiting = "the pull requests waiting to merge: " + strings.Join(prs, ", ")
	}
	return fmt.Sprintf("this person holds %d live reservations, the most one person may hold (%s), so it reserves and asks nothing; %s; wait for one of their pull requests to merge, which ends its reservation, rather than releasing one",
		len(held), strings.Join(held, ", "), waiting)
}

// heldWaitText names a wait on held work: each hold, and the judgements the
// drain keeps.
func heldWaitText(held []string, judgements int) string {
	return fmt.Sprintf("every eligible issue left is held by another (%s), so the drain does not end, and keeps its %d judgement(s)",
		strings.Join(held, "; "), judgements)
}

// waitWritten reports whether a drain state's wait is one the drain writes:
// none, or a known kind with the time it looks again.
func (st DrainState) waitWritten() bool {
	if st.Waiting == "" {
		return st.WaitingUntil == nil
	}
	return st.WaitingUntil != nil && slices.Contains(drainWaitKinds, st.Waiting)
}
