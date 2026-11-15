package loop

// stuck.go is the stuck lane (spc-2610161044461448 step 5;
// iss-2610161015368630; dcn-2610161040382067). A drain opens one lane at a
// time, so a lane that can never finish would stop the whole drain: every
// later move would answer "drive" that lane. The lane's progress is read from
// evidence that changes instead. Each refused step of an issue run is logged
// on its lane, with its stage, its check, its reason and a count of the same
// refusal in a row, and the drain's next move reads it: a lane that has
// refused the same way StuckRefusals times, or has kept refusing at one stage
// for StuckMinutes, is handed back as stuck, its reservation released with the
// tombstone, and the drain takes up the next issue in the same move. The
// redrive is a dead-letter queue's, as a message moves after its maximum
// receive count (Amazon SQS's maxReceiveCount).
//
// A wait by design is no refusal and is never logged: the landing's wait for
// its preflight receipt (checkWait), every outage wait, and a land fence,
// at the push or the arming, that could not read origin. A lane whose hold
// the loop could not widen to a path another holder holds live (pathHeld,
// widen.go) waits for that hold to end too, and is never counted toward the
// refusals bound; but it is logged with a count of 0, so it counts toward
// the minutes bound, and two lanes each waiting on a path the other holds,
// each renewing its own hold as it steps, are handed back once that bound
// passes rather than wait on each other for ever. A refusal that
// cannot change makes the lane stuck at once: the issue finished at the
// lane's base, or on origin's default branch as the land fence reads it, or
// the reservation the fence or the push checks now another person's. Only an
// issue run logs refusals; an intent's run refuses as it always has, writing
// nothing.

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StageRefused is the run record's stage for a lane's refused step.
const StageRefused = "refused"

// HandBackStuck is the kind a stuck lane is handed back as; a lane's receipt
// cannot name it.
const HandBackStuck = claim.HandBackStuck

// The bundled stuck bounds (dcn-2610161040382067): three identical refusals,
// or sixty minutes refusing at one stage. A repository sets its own under
// `pace` in .abcd/config.json, a machine in ~/.abcd.noindex/config.json.
const (
	BundledStuckRefusals = 3
	BundledStuckMinutes  = 60
	keyStuckRefusals     = "stuck_refusals"
	keyStuckMinutes      = "stuck_minutes"
	maxStuckRefusals     = 64
)

// stuckGap is the longest a lane may go between two refusals and still be
// refusing all along: the drain's look-again time (drainWaitMinutes), as
// long as a host stepping a refusing lane as it is told waits between
// steps. Only the time across which refusals kept coming counts toward
// StuckMinutes, so a refusal after a longer quiet starts the time afresh.
const stuckGap = drainWaitMinutes * time.Minute

// LaneRefused is a lane's run of refused steps: the stage, check and reason
// of the last refusal, how many in a row were that same refusal, since when
// the lane has refused at that stage with no quiet longer than stuckGap, and
// when it last refused. Final names
// why the refusal cannot change, "" when it can. Count is 0 on a wait on
// another holder's path (Refusal.pathHeld), which counts toward the minutes
// bound alone.
type LaneRefused struct {
	Stage  string    `json:"stage"`
	Check  string    `json:"check,omitempty"`
	Reason string    `json:"reason"`
	Count  int       `json:"count"`
	Since  time.Time `json:"since"`
	Last   time.Time `json:"last"`
	Final  string    `json:"final,omitempty"`
}

// stuckBounds are the stuck bounds a drain move reads, each with the layer
// that supplied it.
type stuckBounds struct {
	refusals, minutes PaceValue
}

// resolveStuckBounds reads the stuck bounds through the pace's layers: the
// repository's file, then the machine's, then the bundled default. They are
// read at every drain move, so a changed bound applies from the next.
func resolveStuckBounds(roots layered.Roots) (stuckBounds, error) {
	s, err := layered.Load(layered.Config, roots)
	if err != nil {
		return stuckBounds{}, paceRefusal(err.Error())
	}
	if err := s.Claim(paceNamespace, paceKeys...); err != nil {
		return stuckBounds{}, paceRefusal(err.Error())
	}
	get := func(key string, bundled, hi int, unit string) (PaceValue, error) {
		v, err := layered.Get(s, paceNamespace+"."+key, bundled, func(n int) error {
			if n < 1 || n > hi {
				return fmt.Errorf("want %s from 1 to %d", unit, hi)
			}
			return nil
		})
		if err != nil {
			return PaceValue{}, paceRefusal(err.Error())
		}
		return PaceValue{Value: v.V, Layer: v.Layer.String(), Origin: v.Origin}, nil
	}
	var b stuckBounds
	if b.refusals, err = get(keyStuckRefusals, BundledStuckRefusals, maxStuckRefusals, "a whole number of identical refusals"); err != nil {
		return stuckBounds{}, err
	}
	if b.minutes, err = get(keyStuckMinutes, BundledStuckMinutes, maxPaceMinutes, "whole minutes refusing at one stage"); err != nil {
		return stuckBounds{}, err
	}
	return b, nil
}

// waitByDesign reports whether r is a wait the loop makes on purpose rather
// than a refusal: the landing's wait for its preflight receipt, an outage's,
// a land fence's that could not read origin, a merge gate's that could not
// read what it asks at the arming, and a lane whose hold could not be widened
// to a path another holder holds live (widen.go), which noteRefusal alone
// still logs, against the minutes bound.
func waitByDesign(r *Refusal) bool {
	return !r.checkWait.IsZero() || r.Outage != nil || r.outage != nil || r.Stage == StageOutage || r.originUnread || r.releaseHeld || r.gateUnread || r.pathHeld
}

// noteRefusal logs err on the lane it names, in an issue run (stuck.go's
// header): the same refusal as the lane's last adds one to its count, another
// at the same stage starts a new count and keeps the time the lane began
// refusing there, and one at another stage starts afresh. A refusal more than
// stuckGap after the last also starts the time afresh: the lane was not
// refusing across the quiet. The record gains a
// line whenever a new count starts or the refusal turns final, so a host that
// polls a refusing lane does not grow the record. It reports whether it
// changed the state, which the step then writes beside its refusal.
//
// A wait on another holder's path (Refusal.pathHeld) is logged with a count
// of 0 that never grows, so it never reaches the refusals bound, but keeps
// the time the lane began waiting at its stage, as any refusal there does,
// so the minutes bound hands back a lane whose wait does not end, while it
// waits on the same hold with looks no more than twice stuckGap apart. The
// record gains a line only as such a wait begins.
func noteRefusal(st *State, err error, now time.Time) bool {
	r, ok := AsRefusal(err)
	if !ok || st.Issue() == "" || r.Lane == "" || (waitByDesign(r) && !r.pathHeld) {
		return false
	}
	i := -1
	for k, l := range st.Lanes {
		if l.ID == r.Lane {
			i = k
		}
	}
	if i < 0 {
		return false
	}
	lane := st.Lanes[i]
	// A lane past its arming waits on its pull request, which the drain reads
	// from the forge; a lane handed back, held or done refuses by design.
	if (lane.Landing != nil && lane.Landing.Merge != "") || lane.Stage == StageHandedBack || lane.Stage == StageHeld ||
		lane.Stage == StageDone || lane.Stage == StageDiscarded {
		return false
	}
	reason := termsafe.Sanitize(fsutil.RedactHome(r.Reason))
	next := LaneRefused{Stage: r.Stage, Check: r.Check, Reason: reason, Count: 1, Since: now, Last: now, Final: r.final}
	logged := true
	p := lane.Refused
	switch {
	case p != nil && p.Stage == r.Stage && r.pathHeld && p.Count == 0:
		// A wait on another holder's path that goes on keeps the time it
		// began while it waits on the same hold and the looks come no more
		// than twice stuckGap apart: a drive's look-again time can reach
		// stuckGap itself, and a wait that started its time afresh at every
		// look would never reach the minutes bound. A wait on another hold,
		// or after a longer quiet, starts afresh.
		if sameHold(p.Reason, reason) && now.Sub(p.Last) <= 2*stuckGap {
			next.Since = p.Since
		}
	case p != nil && p.Stage == r.Stage && now.Sub(p.Last) <= stuckGap:
		next.Since = p.Since
	}
	switch {
	case r.pathHeld:
		next.Count = 0
		logged = p == nil || p.Stage != r.Stage || p.Count != 0 || next.Since != p.Since
	case p != nil && p.Stage == r.Stage:
		if p.Check == r.Check && p.Reason == reason {
			next.Count = p.Count + 1
			logged = p.Final == "" && r.final != ""
			if next.Final == "" {
				next.Final = p.Final
			}
		}
	}
	lane.Refused = &next
	st.Lanes[i] = lane
	if logged {
		note := fmt.Sprintf("%s refused at %s: %s", lane.ID, refusedAt(next), reason)
		if next.Count == 0 {
			note = fmt.Sprintf("%s waits at %s on another holder's hold, counted toward the minutes bound alone: %s", lane.ID, refusedAt(next), reason)
		}
		if next.Final != "" {
			note += "; it cannot change: " + next.Final
		}
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageRefused, Note: note})
	}
	st.UpdatedAt = now
	return true
}

// holdUntil is the end of a hold as a back-off over it names it, which a
// renewal of the hold moves.
var holdUntil = regexp.MustCompile(` until \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

// sameHold reports whether a and b, two waits' reasons, name the same paths
// and the same hold, whatever end each gives the hold.
func sameHold(a, b string) bool {
	return holdUntil.ReplaceAllString(a, "") == holdUntil.ReplaceAllString(b, "")
}

// refusedAt names a refusal's stage and check.
func refusedAt(r LaneRefused) string {
	if r.Check != "" {
		return r.Stage + " (" + r.Check + ")"
	}
	return r.Stage
}

// stuckReason is why the run's lane in progress is stuck by the bounds b,
// "" when it is not: a refusal that cannot change, the same refusal
// b.refusals times in a row, or refusals at one stage coming for b.minutes,
// from the first to the last, so a quiet after the last refusal never counts.
func stuckReason(run State, b stuckBounds) string {
	i := run.current()
	if i < 0 || run.Issue() == "" {
		return ""
	}
	r := run.Lanes[i].Refused
	if r == nil {
		return ""
	}
	at := refusedAt(*r)
	switch mins := int(r.Last.Sub(r.Since) / time.Minute); {
	case r.Final != "":
		return fmt.Sprintf("its step was refused at %s in a way that cannot change (%s): %s", at, r.Final, r.Reason)
	case r.Count == 0 && mins >= b.minutes.Value:
		return fmt.Sprintf("it has waited at %s for %d minutes on another holder's hold on a path its branch changes, past the bound of %d (%s): %s",
			at, mins, b.minutes.Value, describeOrigin(b.minutes), r.Reason)
	case r.Count >= b.refusals.Value:
		return fmt.Sprintf("its step was refused the same way %s in a row at %s, the bound (%s): %s",
			times(r.Count), at, describeOrigin(b.refusals), r.Reason)
	case mins >= b.minutes.Value:
		return fmt.Sprintf("its step has been refused at %s for %d minutes, past the bound of %d (%s); the last refusal: %s",
			at, mins, b.minutes.Value, describeOrigin(b.minutes), r.Reason)
	}
	return ""
}

// refusingNote is the drain's word on a lane in progress that has refused
// and is not yet stuck: how often, at which stage, since when, and the
// bounds that hand it back. "" for a lane that has not refused.
func refusingNote(run State, b stuckBounds) string {
	i := run.current()
	if i < 0 || run.Lanes[i].Refused == nil {
		return ""
	}
	r := run.Lanes[i].Refused
	if r.Count == 0 {
		return fmt.Sprintf("it has waited at %s since %s on another holder's hold on a path its branch changes; the drain hands it back as stuck at %d minutes",
			refusedAt(*r), stamp(r.Since), b.minutes.Value)
	}
	return fmt.Sprintf("its step has been refused the same way %s in a row at %s, refusing there since %s; the drain hands it back as stuck at %s or %d minutes",
		times(r.Count), refusedAt(*r), stamp(r.Since), times(b.refusals.Value), b.minutes.Value)
}

// times is a count of refusals in words.
func times(n int) string {
	if n == 1 {
		return "1 time"
	}
	return fmt.Sprintf("%d times", n)
}

// handBackStuck hands the run's lane in progress back as stuck by the bounds
// b, and releases the issue's reservation with the hand-back's tombstone
// (releaseOnHandBack), under the run's lock; it returns the run as written.
// The lane is judged again under the lock, so a step running at the same time
// that succeeded, or handed the lane back itself, since the drain read it
// leaves the lane as that step wrote it. The hand-back saves the run whole, through
// save (Options.laneSave).
func handBackStuck(repoRoot, runID string, b stuckBounds, now time.Time, save saveFrame) (State, error) {
	var out State
	err := save(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		why := stuckReason(*st, b)
		if i < 0 || st.Lanes[i].HandBack != nil || why == "" {
			out = *st
			return false, nil
		}
		lane := st.Lanes[i]
		hb := HandBack{Kind: HandBackStuck, Reason: why,
			Home: "a person: the issue stays open, and the lane's run " + st.RunID + " keeps its worktree and branch until `abcd implement step --run " +
				st.RunID + " --discard " + lane.ID + "` discards them"}
		handBackLane(st, &lane, hb, "", now)
		releaseOnHandBack(repoRoot, st, *lane.HandBack, now)
		st.Lanes[i] = lane
		st.UpdatedAt = now
		out = *st
		return true, nil
	})
	return out, err
}
