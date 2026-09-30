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
	"fmt"
	"strings"
	"time"
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
	if st.Pick != nil && hb.Kind == "" {
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
