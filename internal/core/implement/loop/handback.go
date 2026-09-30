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
	lane.Awaiting = nil
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

// handedBackRefusal is every later step's answer on a handed-back lane.
func handedBackRefusal(st State, lane Lane) error {
	reason := lane.ID + " was handed back"
	if lane.HandBack != nil {
		reason = handBackSummary(keyOf(st), *lane.HandBack)
	}
	return refuse(string(StageHandedBack), "", lane.ID, reason,
		"the loop starts nothing further for this lane; "+keyOf(st)+" is the person's to replan from the findings the reason names")
}
