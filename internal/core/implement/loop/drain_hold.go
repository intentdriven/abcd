package loop

// drain_hold.go is the drain's own judgement hold (spc-2610161044461448 step
// 2; iss-2610161015387318): the reservation the drain takes on an issue before
// it asks a judgement over it. The judgement awaited records exactly which
// reservation that was (DrainJudging.Hold), and a move releases it, by a
// tombstone over that commit alone, on every path that leaves it unused while
// the process lives, a panic included. Only two things keep it: the judgement
// still awaited over its issue, and the lane a no opens on it. The CLI's
// handler (drainSignalContext) takes only the first interrupt or termination,
// which cancels the move's context; a process killed outright (SIGKILL, or a
// second interrupt or termination) runs no release, so its hold lapses after
// one lease. A reservation this person
// holds live, made or renewed by hand, is never the drain's, so a judgement
// never releases it; one of theirs that had lapsed and that the drain's push
// renews is the drain's from that push on.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// judgeHolds are the drain's judgement holds one move knows: the one the
// awaited judgement carried into the move, and any the move takes.
type judgeHolds struct {
	repoRoot string
	now      time.Time
	held     []judgeHold
	// view is the reservations as this move fetched them, nil when it has
	// none: a hold carried into the move that the view no longer shows at
	// the tip has nothing left to release, since a reservation's tip only
	// ever moves forward, so it is passed over with no fetch and no push.
	view *claim.Snapshot
}

// judgeHold is one of them: its issue, the hold, whether the move is done
// with it (released, or kept by the lane opened on it), and whether the move
// was starting a lane on it, which an error leaves no lane on.
type judgeHold struct {
	issue    string
	hold     DrainHold
	done     bool
	starting bool
	// fresh is true for a hold this move took, after its view was read.
	fresh bool
}

// add records a hold the move knows of, fresh when this move took it; one
// already known is not added twice.
func (h *judgeHolds) add(issue string, hold *DrainHold, fresh bool) {
	if hold == nil {
		return
	}
	for _, k := range h.held {
		if k.issue == issue && k.hold.Commit == hold.Commit {
			return
		}
	}
	h.held = append(h.held, judgeHold{issue: issue, hold: *hold, fresh: fresh})
}

// releaseOne releases k, and returns the clause for it. When the view shows
// the tip has moved on there is nothing to release: the clause says so for a
// reservation this person holds from this copy (renewed or reserved again by
// hand), and is "" for any other (released, or another's). pending is why a
// hand-back kind asked to publish is not on origin and is still due
// (released.pending), "" when kind is "".
func (h *judgeHolds) releaseOne(k *judgeHold, kind string) (clause, pending string) {
	k.done = true
	if r, ok, moved := h.movedOn(*k); moved {
		if ok && r.Copy == claim.CopyThis && r.State == claim.StateLive {
			clause = "left the reservation on " + k.issue + " as it stands: it was renewed or reserved again since the drain took it"
		}
		if kind != "" {
			pending = "the reservation the drain took to judge " + k.issue + " is no longer its tip on " + claim.Remote + movedTo(r, ok)
		}
		return clause, pending
	}
	out := releaseAt(h.repoRoot, k.issue, k.hold.Commit, kind, "", h.now)
	return out.clause, out.pending
}

// movedOn reports whether the move's view shows k's tip moved off its hold,
// so releasing it pushes nothing, with the view's reservation of k's issue
// and whether the view holds one. A hold the move took itself, or one judged
// with no view, has not moved on: its release goes to origin.
func (h *judgeHolds) movedOn(k judgeHold) (r claim.Reservation, ok, moved bool) {
	if k.fresh || h.view == nil {
		return claim.Reservation{}, false, false
	}
	r, ok = h.view.Lookup(k.issue)
	return r, ok, !ok || r.Commit != k.hold.Commit
}

// movedTo says what the view shows at a tip the drain's hold moved off: a
// reservation this person renewed or made by hand, another's, a release, or
// none.
func movedTo(r claim.Reservation, ok bool) string {
	switch {
	case !ok:
		return ": it holds no reservation of it"
	case r.State == claim.StateReleased:
		return ": it was released by a tombstone that marks no hand-back"
	case r.Copy == claim.CopyThis:
		return ": this person renewed or reserved it again by hand"
	}
	return ": " + r.Holder + " holds it"
}

// release releases issue's holds now and returns the move's clause for them,
// "" when it knows none. A kind other than "" publishes the hand-back on the
// tombstone (releaseReservation), and pending is why it is not on origin and
// still due, "" when it is or none was asked; known is false when the move
// knows no hold for issue, so nothing was released or published.
func (h *judgeHolds) release(issue, kind string) (clause, pending string, known bool) {
	for i := range h.held {
		k := &h.held[i]
		if k.done || k.issue != issue {
			continue
		}
		known = true
		c, p := h.releaseOne(k, kind)
		if p != "" {
			pending = p
		}
		if c == "" {
			continue
		}
		if clause != "" {
			clause += "; "
		}
		clause += c
	}
	return clause, pending, known
}

// forget marks issue's hold at commit as done with nothing to release: the
// move's view no longer shows it at the tip.
func (h *judgeHolds) forget(issue, commit string) {
	for i := range h.held {
		if h.held[i].issue == issue && h.held[i].hold.Commit == commit {
			h.held[i].done = true
		}
	}
}

// starting marks issue's holds as the ones a lane is about to start on.
func (h *judgeHolds) starting(issue string) {
	for i := range h.held {
		if h.held[i].issue == issue {
			h.held[i].starting = true
		}
	}
}

// keep marks issue's holds as the lane's, opened on them.
func (h *judgeHolds) keep(issue string) {
	for i := range h.held {
		if h.held[i].issue == issue {
			h.held[i].done = true
		}
	}
}

// addForming records the holds of the bundle forming carries into the move,
// the lead's and each member's (drain_form.go), as holds the move knows.
func (h *judgeHolds) addForming(f *DrainForming) {
	for _, e := range f.entries() {
		h.add(e.issue, e.hold, false)
	}
}

// moved records that issue's hold at from is now to: a renewal the move
// pushed (renewForming). The hold is the move's own from then on, taken after
// its view was read, so a release goes to origin rather than reading the
// view's older tip as moved on.
func (h *judgeHolds) moved(issue, from string, to DrainHold) {
	for i := range h.held {
		if h.held[i].issue == issue && h.held[i].hold.Commit == from {
			h.held[i].hold, h.held[i].fresh = to, true
		}
	}
}

// settle releases every hold the move is not done with and that neither
// awaited nor forming records, and returns the move's clause for each: a
// forming bundle's holds are kept across moves as the judgement awaited's is
// (drain_form.go). A hold a lane was starting on is released even when
// awaited or forming records it: the start failed, so no lane holds it, and
// the move's error leaves them on disk.
func (h *judgeHolds) settle(awaited *DrainJudging, forming *DrainForming) []string {
	out, _ := h.settleAfter(awaited, forming, nil)
	return out
}

// settleAfter is settle with before run once, ahead of the first release it
// pushes to origin: a move's finish writes the drain state there
// (iss-2611260523161723), so no push sits between a lane's run write and the
// drain state's. An error from before stops it with nothing more released,
// the clauses so far returned with it.
func (h *judgeHolds) settleAfter(awaited *DrainJudging, forming *DrainForming, before func() error) ([]string, error) {
	var out []string
	for i := range h.held {
		k := &h.held[i]
		if k.done {
			continue
		}
		if !k.starting && awaited != nil && awaited.Issue == k.issue && awaited.Hold != nil && awaited.Hold.Commit == k.hold.Commit {
			continue
		}
		if !k.starting && forming.holds(k.issue, k.hold.Commit) {
			continue
		}
		if _, _, moved := h.movedOn(*k); before != nil && !moved {
			if err := before(); err != nil {
				return out, err
			}
			before = nil
		}
		c, _ := h.releaseOne(k, "")
		if c == "" {
			continue
		}
		out = append(out, "nothing waits any longer on the reservation the drain took to judge "+k.issue+": "+c)
	}
	return out, nil
}

// openedUnrecorded is each lane a drain move opened on the drain's own holds
// and did not live to record (iss-2611260523161723): the lane a no opens
// starts on the drain's hold with no second push (heldReservation), so a move
// killed between the run's write and the drain state's leaves st recording,
// as the drain's, the judgement's hold or a forming bundle's that the new run
// now carries. Such a run is one of this checkout's that no lane of st names
// and no drive let go, holding one of those issues on the hold's own claim:
// its reserved_at for the issue is the hold's claimed_at. Each comes back as
// the lane the move would have recorded, in progress, and carried names each
// issue whose hold one of them carries. A run that cannot be read fails it,
// so a let-go that cannot tell releases nothing.
func openedUnrecorded(repoRoot string, st DrainState) (lanes []DrainLane, carried map[string]bool, err error) {
	var holds []formingEntry
	if j := st.Judging; j != nil && j.Hold != nil {
		holds = append(holds, formingEntry{issue: j.Issue, hold: j.Hold})
	}
	holds = append(holds, formingHolds(st.Forming)...)
	if len(holds) == 0 {
		return nil, nil, nil
	}
	runs, err := Runs(repoRoot)
	if err != nil {
		return nil, nil, err
	}
	carried = map[string]bool{}
	for _, run := range runs {
		if letGoByDrive(run) || slices.ContainsFunc(st.Lanes, func(l DrainLane) bool { return l.RunID == run.RunID }) {
			continue
		}
		held, opened := run.reservedIssues(), false
		for _, h := range holds {
			at := run.reservedAt(h.issue)
			if at == nil || !at.Equal(h.hold.ClaimedAt) || !slices.ContainsFunc(held, func(id string) bool { return recordid.SameID(id, h.issue) }) {
				continue
			}
			carried[h.issue], opened = true, true
		}
		if !opened {
			continue
		}
		l := DrainLane{Issue: run.Key, RunID: run.RunID, OpenedAt: run.CreatedAt, Outcome: DrainLaneInProgress}
		for _, m := range run.Bundle {
			l.Bundle = append(l.Bundle, m.Issue)
		}
		lanes = append(lanes, l)
	}
	return lanes, carried, nil
}

// openedLine is what a let-go says of a lane it recorded for the move that
// opened it and did not live to (openedUnrecorded).
func openedLine(l DrainLane) string {
	with := ""
	if len(l.Bundle) > 0 {
		with = " with its member(s) " + strings.Join(l.Bundle, ", ")
	}
	return fmt.Sprintf("%s: run %s opened its lane%s on the reservation the drain took, before the drain could record it, so the drain records the lane now and the run keeps what it holds: nothing of it is released, and each lapses with its lease %s",
		l.Issue, l.RunID, with, UnlessRunRenewed)
}

// CheckLapsed is the pass-over of an issue whose judgement request lapsed
// unanswered, while its cool-down runs.
const CheckLapsed = "lapsed"

// DrainLapse is a judgement request the drain dropped unanswered: its issue,
// when it was asked, when the hold taken for it lapsed, and when the move that
// dropped it released that hold. The issue is not asked again before its
// cool-down has passed (coolDownEnds), so another account has a window to
// take it, and then only after the drain's other eligible work; the entry is
// pruned once the issue is asked again or answered.
type DrainLapse struct {
	Issue       string    `json:"issue"`
	RequestedAt time.Time `json:"requested_at"`
	LapsedAt    time.Time `json:"lapsed_at"`
	DroppedAt   time.Time `json:"dropped_at"`
}

// coolDownEnds is when l's issue may be asked again: the take-over margin
// after the move that dropped it, which is never the move it lapsed in.
func (l DrainLapse) coolDownEnds() time.Time {
	return l.DroppedAt.Add(claim.TakeoverMargin)
}

// lapseOf is the lapse st records for issue, if any.
func lapseOf(st DrainState, issue string) (DrainLapse, bool) {
	i := slices.IndexFunc(st.Lapsed, func(l DrainLapse) bool { return l.Issue == issue })
	if i < 0 {
		return DrainLapse{}, false
	}
	return st.Lapsed[i], true
}

// pruneLapse drops issue's lapse from st: it was asked again, or answered.
func pruneLapse(st *DrainState, issue string) {
	st.Lapsed = slices.DeleteFunc(st.Lapsed, func(l DrainLapse) bool { return l.Issue == issue })
	if len(st.Lapsed) == 0 {
		st.Lapsed = nil
	}
}

// lapsedLast is the plan's verdicts in order, with the issues of lapsed
// requests moved after every other, the latest lapse last.
func lapsedLast(vs []capture.DrainVerdict, lapsed []DrainLapse) []capture.DrainVerdict {
	if len(lapsed) == 0 {
		return vs
	}
	out := make([]capture.DrainVerdict, 0, len(vs))
	for _, v := range vs {
		if !slices.ContainsFunc(lapsed, func(l DrainLapse) bool { return l.Issue == v.ID }) {
			out = append(out, v)
		}
	}
	for _, l := range lapsed {
		if i := slices.IndexFunc(vs, func(v capture.DrainVerdict) bool { return v.ID == l.Issue }); i >= 0 {
			out = append(out, vs[i])
		}
	}
	return out
}
