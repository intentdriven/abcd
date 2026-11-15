package loop

// handback.go is the fix-round bound (ruling DR1, 2026-09-29, on itd-50's
// criterion 2): a lane that has taken its run's cap of fix rounds and still
// does not pass is stopped and handed back to the person with the last round's
// findings. The loop starts nothing further for it: every later step refuses,
// naming the hand-back, and building the intent again resumes the run and says
// the same. A run `abcd build next` started records its pick as falsified
// (itd-2609211116005482); the intent's grounds entry is not edited.
//
// Moving the intent to drafts/ with its replan reason (itd-50, criterion 3) is
// the landing's side of the hand-back, and is not made here: the lane's record
// is what the person replans from.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/spotcheck"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// handBackLane stops lane with hb and records it, loudly, in the run record.
func handBackLane(st *State, lane *Lane, hb HandBack, note string, now time.Time) {
	hb.At = now
	lane.HandBack = &hb
	lane.Stage = StageHandedBack
	lane.Awaits = nil
	if note == "" {
		note = handBackSummary(st.Intent, hb)
	}
	st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: string(StageHandedBack), Note: note})
	if st.Pick != nil && hb.Kind == "" && hb.Verdict != VerdictCellFailed {
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: "pick",
			Note: fmt.Sprintf("the pick of %s is falsified: %s was handed back as %s after %s; the intent's grounds entry is left as it was written",
				keyOf(*st), lane.ID, hb.Verdict, fixRoundsPhrase(hb.FixRounds))})
	}
}

// keyOf is the record a run builds, for a sentence.
func keyOf(st State) string {
	if st.Intent != "" {
		return st.Intent
	}
	return st.Key
}

// handBackSummary is the one sentence a hand-back is recorded and reported
// with: the intent, the verdict, the cap, and what the last round found.
func handBackSummary(key string, hb HandBack) string {
	var b strings.Builder
	if hb.Verdict == VerdictCellFailed {
		fmt.Fprintf(&b, "%s's trial copy is handed back as %s: %s; nothing else ran the role, so the copy counts as the cell's failure", key, hb.Verdict, hb.Reason)
		return b.String()
	}
	if hb.Kind == HandBackStuck || hb.Kind == HandBackClosed {
		fmt.Fprintf(&b, "%s is handed back as %s: %s; its home: %s", key, hb.Kind, hb.Reason, hb.Home)
		return b.String()
	}
	if hb.Kind != "" {
		fmt.Fprintf(&b, "%s is handed back by its lane as %s: %s", key, hb.Kind, hb.Reason)
		if hb.Home != "" {
			fmt.Fprintf(&b, "; its home: %s", hb.Home)
		}
		if hb.Discarded != "" {
			fmt.Fprintf(&b, "; the lane's work at %s is discarded with its worktree and branch", shortSHA(hb.Discarded))
		} else {
			b.WriteString("; the lane's work is discarded")
		}
		return b.String()
	}
	if hb.Verdict == HandBackNotPlanned {
		// An auto-plan lane (autoplan.go): the substitute did not hold, or a
		// plan was found that the run cannot vouch for, so the run records no
		// plan; the reason says where the record stands.
		fmt.Fprintf(&b, "%s is handed back as %s: %s", key, hb.Verdict, hb.Reason)
		if hb.Verdicts != "" {
			fmt.Fprintf(&b, " (plan reviews: %s)", hb.Verdicts)
		}
		if len(hb.Findings) > 0 {
			fmt.Fprintf(&b, "; the findings: %s", strings.Join(hb.Findings, ", "))
		}
		return b.String()
	}
	fmt.Fprintf(&b, "%s is handed back as %s: round %d did not pass after %s, the run's cap (%s)",
		key, hb.Verdict, hb.Round, fixRoundsPhrase(hb.FixRounds), hb.Verdicts)
	if len(hb.NotMet) > 0 {
		fmt.Fprintf(&b, "; not met: %s", strings.Join(hb.NotMet, ", "))
	}
	if len(hb.Undecided) > 0 {
		fmt.Fprintf(&b, "; undecided: %s", strings.Join(hb.Undecided, ", "))
	}
	if len(hb.Findings) > 0 {
		fmt.Fprintf(&b, "; the last findings: %s", strings.Join(hb.Findings, ", "))
	}
	return b.String()
}

// handBackMove is what the caller is told once a lane is handed back.
func handBackMove(st State, lane Lane) string {
	if lane.HandBack.Verdict == VerdictCellFailed {
		return "stop: " + handBackSummary(keyOf(st), *lane.HandBack) + ". The loop starts nothing further for " + lane.ID +
			": a trial's failed copy is never retried, on its runner or on the host"
	}
	return "stop: " + handBackSummary(keyOf(st), *lane.HandBack) + ". The loop starts nothing further for " + lane.ID +
		"; " + keyOf(st) + " is the person's to replan from those findings"
}

// handedBackRefusal is a step's answer once a lane of the run was handed back
// and nothing is left to move but wait for the person (ruling DR6c): it names
// the hand-back and every held lane with the way out for each.
func handedBackRefusal(st State) error {
	var lane Lane
	for _, l := range st.Lanes {
		if l.Stage == StageHandedBack {
			lane = l
			break
		}
	}
	reason := lane.ID + " was handed back"
	if lane.HandBack != nil {
		reason = handBackSummary(keyOf(st), *lane.HandBack)
	}
	var held []string
	for _, l := range st.Lanes {
		if l.Stage == StageHeld && l.Hold != nil {
			held = append(held, fmt.Sprintf("%s is held at %s before its %s (%s)", l.ID, shortSHA(l.Hold.Head), l.Hold.Before, heldWayOut(l.ID)))
		}
	}
	if len(held) > 0 {
		reason += "; " + strings.Join(held, "; ")
	}
	return refuse(string(StageHandedBack), "", lane.ID, reason,
		"the loop starts nothing further for this lane; "+keyOf(st)+" is the person's to replan from the findings the reason names; "+
			handedBackWayOut(st))
}

// heldWayOut names the person's two choices over a held lane.
func heldWayOut(laneID string) string {
	return "land it as it is with `abcd implement step --release " + laneID + "`, or discard it with `abcd implement step --discard " + laneID + "`"
}

// heldMove is what the caller is told about a held lane.
func heldMove(lane Lane) string {
	if lane.Hold == nil {
		return lane.ID + " is held"
	}
	return fmt.Sprintf("%s is held after %s's hand-back, at %s before its %s: %s", lane.ID, lane.Hold.Cause, shortSHA(lane.Hold.Head), lane.Hold.Before, heldWayOut(lane.ID))
}

// handedBackWayOut names the one way past a handed-back run. The run stays
// live by construction (Complete is false while a lane sits at handed-back, so
// the run resumes and the pick excludes its intent) until terminal liveness
// lands with itd-50's move of the intent to drafts/; making it terminal before
// then would let the pick choose the falsified intent again. No verb clears
// it, so the run's own directory is named (iss-2609301303434847).
func handedBackWayOut(st State) string {
	return "to build it afresh once it is replanned, remove the run's directory, " + RunRelDir + "/" + st.RunID
}

// handedBack reports whether any lane of the run was handed back.
func (s State) handedBack() bool {
	for _, l := range s.Lanes {
		if l.Stage == StageHandedBack {
			return true
		}
	}
	return false
}

// tombstoneKind is the hand-back kind the tombstone over a handed-back
// issue's reservation names for hb (claim.HandBackKinds): the kind the lane's
// receipt or the judgement named, or, for a lane stopped after its fix rounds,
// unachievable.
func tombstoneKind(hb HandBack) string {
	if hb.Kind != "" {
		return hb.Kind
	}
	return claim.HandBackUnachievable
}

// spotcheckRecords keeps the spot-check's record of what the drain decided
// with no person involved (spc-2610301925429250 step 1): each hand-back st
// has routed, under handbacks/, and each judgement a model answered, under
// decisions/, each as its own file on the store's local copy
// (internal/core/spotcheck), all of a move's in one commit, reaching no
// network. The records are rebuilt from st, which keeps every route and
// judgement of the drain, and only those the local copy does not hold are
// written, so a record one move could not write is tried again by the next.
// It never fails, holds or changes the move: what it could not write is said
// in the note it returns, "" when everything is kept, and a checkout with no
// remote named origin keeps no record.
func spotcheckRecords(parent context.Context, repoRoot string, st DrainState) string {
	holder := spotcheck.Holder()
	var b spotcheck.Batch
	for _, r := range st.HandBacks {
		b.HandBacks = append(b.HandBacks, spotcheckHandBack(st, r, holder))
	}
	for _, j := range st.Judgements {
		if j.Route == nil {
			// A person's answer is not the automated facilitator's.
			continue
		}
		b.Decisions = append(b.Decisions, spotcheck.Decision{Kind: spotcheck.KindDrainJudgement, At: j.At, Holder: holder, Issue: j.Issue,
			Question: drainQuestion().Ask, Answer: j.Answer, AnswerKind: j.Kind, Model: j.Route.Model, Rule: st.Rule, Reason: j.Reason})
	}
	return spotcheckNote(spotcheck.Write(ctxOr(parent), repoRoot, b))
}

// spotcheckHandBack is r as the spot-check keeps it, with the lane's pull
// request and run (routedLane).
func spotcheckHandBack(st DrainState, r DrainRoute, holder string) spotcheck.HandBack {
	h := spotcheck.HandBack{Issue: r.Issue, From: r.From, Kind: r.Kind, Route: r.Route, Rule: r.Rule,
		DrainRule: st.Rule, Reason: r.Reason, Blob: r.Blob, Holder: holder}
	if r.At != nil {
		// A route with no time cannot be kept, and the write says so.
		h.At = *r.At
	}
	if l, ok := routedLane(st, r); ok {
		h.PR, h.RunID = l.PR, l.RunID
	}
	return h
}

// routedLane is the drain lane a lane's hand-back r came from: the lane
// on its issue opened latest at or before the hand-back, since an issue
// handed back and changed since may be taken up again. ok is false for a
// hand-back no lane made (the judgement's, the rule's, a reservation's).
func routedLane(st DrainState, r DrainRoute) (DrainLane, bool) {
	if r.From != "lane" {
		return DrainLane{}, false
	}
	var found DrainLane
	ok := false
	for _, l := range st.Lanes {
		if l.Issue != r.Issue || (r.At != nil && l.OpenedAt.After(*r.At)) {
			continue
		}
		if !ok || l.OpenedAt.After(found.OpenedAt) {
			found, ok = l, true
		}
	}
	return found, ok
}

// spotcheckNote is what a move says of its write of the spot-check's
// record: how many records it tried to write and could not, and why; and,
// apart, how many records of the drain's state the store cannot keep at all
// (a route with no time, an id outside the store's form), a number that
// stays the same from move to move while the state holds them. "" when every
// record is kept, and for a checkout with no remote named origin.
func spotcheckNote(w spotcheck.Written) string {
	if errors.Is(w.Err, spotcheck.ErrNoOrigin) {
		return ""
	}
	var parts []string
	if w.Failed > 0 && w.Err != nil {
		parts = append(parts, fmt.Sprintf("%d record(s) of the spot-check were not written here (%s); a later move tries them again, and nothing the drain did waits on them",
			w.Failed, termsafe.Sanitize(fsutil.RedactHome(w.Err.Error()))))
	}
	if n := len(w.Unkeepable); n > 0 {
		parts = append(parts, fmt.Sprintf("%d record(s) of the drain's state cannot be kept by the spot-check's store (%s)", n, termsafe.Sanitize(w.Unkeepable[0])))
	}
	return strings.Join(parts, "; ")
}
