package loop

// drain_hold.go is the drain's own judgement hold (spc-2610161044461448 step
// 2; iss-2610161015387318): the reservation the drain takes on an issue before
// it asks a judgement over it. The judgement awaited records exactly which
// reservation that was (DrainJudging.Hold), and a move releases it, by a
// tombstone over that commit alone, on every path that leaves it unused. Only
// two things keep it: the judgement still awaited over its issue, and the lane
// a no opens on it. A reservation this person made or renewed by hand is never
// the drain's, so a judgement never releases it.

import (
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
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
// hand), and is "" for any other (released, or another's).
func (h *judgeHolds) releaseOne(k *judgeHold, kind string) string {
	k.done = true
	if !k.fresh && h.view != nil {
		if r, ok := h.view.Lookup(k.issue); !ok || r.Commit != k.hold.Commit {
			if ok && r.Copy == claim.CopyThis && r.State == claim.StateLive {
				return "left the reservation on " + k.issue + " as it stands: it was renewed or reserved again since the drain took it"
			}
			return ""
		}
	}
	c, _ := releaseReservationAt(h.repoRoot, k.issue, k.hold.Commit, kind, h.now)
	return c
}

// release releases issue's holds now and returns the move's clause for them,
// "" when it knows none. A kind other than "" publishes the hand-back on the
// tombstone (releaseReservation).
func (h *judgeHolds) release(issue, kind string) string {
	clause := ""
	for i := range h.held {
		k := &h.held[i]
		if k.done || k.issue != issue {
			continue
		}
		c := h.releaseOne(k, kind)
		if c == "" {
			continue
		}
		if clause != "" {
			clause += "; "
		}
		clause += c
	}
	return clause
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

// settle releases every hold the move is not done with and awaited does not
// record, and returns the move's clause for each. A hold a lane was starting
// on is released even when awaited records it: the start failed, so no lane
// holds it, and the move's error leaves awaited on disk.
func (h *judgeHolds) settle(awaited *DrainJudging) []string {
	var out []string
	for i := range h.held {
		k := &h.held[i]
		if k.done {
			continue
		}
		if !k.starting && awaited != nil && awaited.Issue == k.issue && awaited.Hold != nil && awaited.Hold.Commit == k.hold.Commit {
			continue
		}
		c := h.releaseOne(k, "")
		if c == "" {
			continue
		}
		out = append(out, "nothing waits any longer on the reservation the drain took to judge "+k.issue+": "+c)
	}
	return out
}

// DrainLapse is a judgement request the drain dropped unanswered: its issue,
// when it was asked, and when the hold taken for it lapsed.
type DrainLapse struct {
	Issue       string    `json:"issue"`
	RequestedAt time.Time `json:"requested_at"`
	LapsedAt    time.Time `json:"lapsed_at"`
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
