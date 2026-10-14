package loop

// reservation.go is the run's reservation on the shared remote
// (itd-2609091034175565, spc-2610031753359849 spec 2): the run's key reserved
// as `refs/heads/abcd-reserved/<key>` on origin, held for the copy the run
// lives in, renewed by `implement step` at half its lease, fenced before the
// landing arms a merge, and tombstoned once the run's last lane has landed. A
// run whose state records no reservation (ReservedAt nil: one started before
// reservations, or in a checkout with no remote named origin) is neither
// renewed, fenced nor tombstoned.
//
// Every act goes through internal/core/claim, which fetches, judges and makes
// one plain fast-forward push in abcd's own namespace; nothing here changes
// how the repository works.

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The run record's stages for the run's reservation.
const (
	StageReservationRenewed     = "reservation_renewed"
	StageReservationRenewFailed = "reservation_renew_failed"
)

// StageReservation is the refusal stage, and the run record's stage, of a
// run's start that reserves its key; CheckReservation is the check a drain
// names when it passes over an issue another holds.
const (
	StageReservation = "reservation"
	CheckReservation = "reservation"
)

// fenceMargin is how far past the merge queue's check timeout the land fence
// renews the reservation, so it covers one run of the queue's checks. It does
// not cover the time a pull request waits in the queue before its checks
// start: after the arming the hold rests on later steps renewing it at half
// its lease, as `landMerged` waits.
const fenceMargin = 15 * time.Minute

// claimOptions are the options a run's reservation act runs under: the step's
// clock, so a test's injected clock dates the commits too.
func claimOptions(now time.Time) claim.Options {
	return claim.Options{Now: func() time.Time { return now }}
}

// renewReservation renews the run's reservation before a stage, when the view
// (as last fetched) shows the run's own, from this copy and not released,
// half through its lease since its Abcd-Claimed-At: it fetches and pushes a
// renewal with the default lease, and logs reservation_renewed in the run
// record. Under half its lease nothing is pushed. A renewal that fails is
// logged and its note returned for the step's answer; the stage still runs,
// since the land fence is where a lost reservation stops the run.
func renewReservation(repoRoot string, st *State, now time.Time) string {
	if st.ReservedAt == nil {
		return ""
	}
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return renewFailed(st, now, err)
	}
	var own *claim.Reservation
	for i, r := range snap.Reservations {
		if recordid.SameID(r.Record, st.Key) {
			own = &snap.Reservations[i]
		}
	}
	if own == nil || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
		return ""
	}
	if half := own.ClaimedAt.Add(own.EffectiveExpiry.Sub(own.ClaimedAt) / 2); now.Before(half) {
		return ""
	}
	res, err := claim.Reserve(repoRoot, claim.Request{Record: st.Key, Renew: true, Lease: lease.Default}, claimOptions(now))
	if err != nil {
		return renewFailed(st, now, err)
	}
	st.Record = append(st.Record, Entry{At: now, Stage: StageReservationRenewed,
		Note: fmt.Sprintf("renewed the reservation on %s, half its lease having passed, until %s", st.Key, stamp(res.ExpiresAt))})
	st.UpdatedAt = now
	return ""
}

// renewFailed logs a failed renewal in the run record and returns its note.
func renewFailed(st *State, now time.Time, err error) string {
	note := "could not renew the reservation on " + st.Key + ": " + err.Error()
	st.Record = append(st.Record, Entry{At: now, Stage: StageReservationRenewFailed, Note: note})
	st.UpdatedAt = now
	return note
}

// stamp renders a time as a reservation carries one: RFC 3339, UTC.
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// fenceLease is the lease the land fence renews with. A merge that waits on a
// person's approval (a required approving review, or a code owner's) can
// wait overnight, so it takes the longest lease, lease.Max. Otherwise it is
// the default, or the merge queue's check timeout plus fenceMargin when that
// is longer, clamped to lease.Max; what holds it past that is the renewals
// that follow (a step's at half its lease, and every drain move's for a lane
// parked at land).
func fenceLease(checkTimeout time.Duration, approval bool) time.Duration {
	if approval {
		return lease.Max
	}
	term := lease.Default
	if checkTimeout > 0 && checkTimeout+fenceMargin > term {
		term = checkTimeout + fenceMargin
	}
	return min(term, lease.Max)
}

// landFence is the land fence, at the top of the arming: it fetches the run's
// reservation and refuses, arming nothing, unless the tip is this person's
// from this copy and live; when it holds, it renews it with term, the
// fenceLease of the merge rule. It returns the renewed expiry, zero
// for a run that holds no reservation.
func landFence(c Context, lane Lane, term time.Duration) (time.Time, error) {
	id := c.State.Key
	if c.State.ReservedAt == nil {
		return time.Time{}, nil
	}
	res, err := claim.Reserve(c.RepoRoot, claim.Request{Record: id, Renew: true, Live: true, Lease: term},
		claimOptions(c.Now))
	if err == nil {
		return res.ExpiresAt, nil
	}
	stage := string(StageLand)
	again := "run `abcd implement step` again"
	reserveAgain := "reserve it again with abcd build " + id
	var fe *claim.FetchError
	var nh *claim.NotHeld
	switch {
	case errors.As(err, &fe):
		return time.Time{}, contend(stage, "", lane.ID,
			"could not read the reservation on "+id+" from "+claim.Remote+": "+fe.Reason+", so the merge is not armed",
			"run abcd implement step again once origin answers")
	case errors.As(err, &nh):
		switch nh.Why {
		case claim.NotHeldAnotherPerson:
			return time.Time{}, contend(stage, "", lane.ID,
				fmt.Sprintf("the reservation on %s is now %s's, until %s, so this lane does not arm its merge", id, nh.Holder, stamp(nh.Expiry)),
				"the pull request stays open and unarmed; take other work, or settle the record with "+nh.Holder+", then "+again)
		case claim.NotHeldAnotherCopy:
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" moved to your copy on another computer, so this lane does not arm its merge here",
				"carry the lane on from that copy, or take the reservation back here (`abcd reserve "+id+" --take-over --footprint <path>`), then "+again)
		case claim.NotHeldReleased:
			if nh.Derived {
				return time.Time{}, contend(stage, "", lane.ID,
					"the reservation on "+id+" was released: "+id+" is finished on "+claim.Remote+"'s default branch, so this lane does not arm its merge",
					"confirm the record is finished on the default branch, and hand the lane back or close its pull request")
			}
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" was released at "+stamp(nh.ReleasedAt)+", so this lane does not arm its merge", reserveAgain)
		case claim.NotHeldAbsent:
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" is gone from "+claim.Remote+", so this lane does not arm its merge", reserveAgain)
		case claim.NotHeldUnreadable:
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" is unreadable ("+nh.Unreadable+"), so this lane does not arm its merge",
				"it can be taken over once its hold ends at "+stamp(nh.Expiry)+" and five minutes more; then "+reserveAgain)
		case claim.NotHeldLapsed:
			return time.Time{}, contend(stage, "", lane.ID,
				"your reservation on "+id+" lapsed at "+stamp(nh.Expiry)+", so this lane does not arm its merge",
				again+": a step renews this copy's lapsed reservation before its stage")
		}
	case errors.Is(err, claim.ErrBackOff):
		return time.Time{}, contend(stage, "", lane.ID,
			"the reservation on "+id+" could not be renewed: "+err.Error()+"; so this lane does not arm its merge",
			"take it up with the holder it names, then "+again)
	}
	return time.Time{}, refuse(stage, "", lane.ID, "the reservation on "+id+" could not be renewed at the land fence: "+err.Error(),
		"settle what it names, then "+again)
}

// completesRun reports whether the lane laneID, landing, is the run's last:
// no step pending and every other lane done. Only that lane's landing ends
// the run's hold on its key; an earlier lane leaves the reservation holding
// for its siblings.
func completesRun(st State, laneID string) bool {
	if len(st.Pending) > 0 {
		return false
	}
	for _, l := range st.Lanes {
		if l.ID != laneID && l.Stage != StageDone {
			return false
		}
	}
	return true
}

// tidyReservation is the tidy-up after the merge: once the ancestor check has
// proved the run's last lane landed, the reservation has already stopped
// counting (the record is finished on the default branch), and a tombstone is
// pushed over it while it is still this person's from this copy. Another
// person's or another copy's is left as it stands and named. It returns the
// landing note's clause, "" for a run that holds no reservation or a lane that
// does not complete the run; a failure is named there and never undoes the
// landing.
func tidyReservation(c Context, lane Lane) string {
	if c.State.ReservedAt == nil || !completesRun(c.State, lane.ID) {
		return ""
	}
	clause, failed := releaseReservation(c.RepoRoot, c.State.Key, c.Now)
	if failed {
		clause += "; the landing stands, and the reservation counts as released while " + c.State.Key + " is finished on the default branch"
	}
	return clause
}

// releaseReservation pushes a tombstone over the run's reservation on id
// while it is still this person's from this copy, and says what came of it:
// released, already released, left as it stands with its holder named, or a
// fetch or push that failed (failed is then true). It never refuses.
func releaseReservation(repoRoot, id string, now time.Time) (clause string, failed bool) {
	res, err := claim.Reserve(repoRoot, claim.Request{Record: id, Release: true}, claimOptions(now))
	var nh *claim.NotHeld
	switch {
	case err == nil && res.Act == claim.ActAlreadyReleased:
		return "the reservation on " + id + " was already released", false
	case err == nil:
		return "released the reservation on " + id + " with a tombstone (" + shortSHA(res.Commit) + ")", false
	case errors.As(err, &nh) && nh.Why == claim.NotHeldAnotherPerson:
		return "left the reservation on " + id + " as it stands: it is " + nh.Holder + "'s", false
	case errors.As(err, &nh) && nh.Why == claim.NotHeldAnotherCopy:
		return "left the reservation on " + id + " as it stands: your copy on another computer holds it", false
	case errors.Is(err, claim.ErrRefused) && !isPushFailure(err):
		return "left the reservation on " + id + " as it stands: " + err.Error(), false
	}
	return "could not push the tombstone over the reservation on " + id + ": " + err.Error(), true
}

// isPushFailure reports whether err is a reservation act that failed reaching
// origin, a fetch or a push, rather than a refusal judged on the tip.
func isPushFailure(err error) bool {
	var fe *claim.FetchError
	var pe *claim.PushError
	return errors.As(err, &fe) || errors.As(err, &pe)
}

// reserveKey reserves a new run's key on origin for this copy with footprint,
// after the run's checks pass and before it is created, and returns the run
// record's line for it. A checkout with no remote named origin has nowhere to
// reserve, and reserves nothing (nil, ""). Another's hold is the back-off,
// contention at the reservation check; any other refusal stops the start.
// fresh is true when the act made the reservation (a reserve or a take-over)
// rather than renewing one this copy already held, so only a fresh one is
// released when the run is not created.
func reserveKey(repoRoot, key string, footprint []string, now time.Time) (at *time.Time, note string, fresh bool, err error) {
	ok, err := claim.HasOrigin(repoRoot)
	if err != nil {
		return nil, "", false, err
	}
	if !ok {
		return nil, "", false, nil
	}
	res, err := claim.Reserve(repoRoot, claim.Request{Record: key, Footprint: footprint}, claimOptions(now))
	if err != nil {
		if errors.Is(err, claim.ErrBackOff) {
			return nil, "", false, contend(StageReservation, CheckReservation, "", err.Error(),
				"take other work; `abcd peers` lists the reservations on "+claim.Remote)
		}
		return nil, "", false, refuse(StageReservation, "", "", err.Error(), "settle what it names, then run the verb again")
	}
	verb := map[claim.Act]string{claim.ActReserved: "reserved", claim.ActRenewed: "renewed the reservation on",
		claim.ActTakenOver: "took over the reservation on"}[res.Act]
	note = fmt.Sprintf("%s %s on %s for this copy until %s, footprint: %s", verb, res.Record, claim.Remote,
		stamp(res.ExpiresAt), strings.Join(res.Footprint, ", "))
	t := now
	return &t, note, res.Act != claim.ActRenewed, nil
}

// releaseKey releases a reservation a start took and then could not create
// its run for, best effort: a failure leaves it to lapse with its lease.
func releaseKey(repoRoot, key string, now time.Time) {
	_, _ = claim.Reserve(repoRoot, claim.Request{Record: key, Release: true}, claimOptions(now))
}

// issueFootprint is the footprint a drain reserves an issue with: the
// repository-relative path its found_at names, a locator suffix dropped; ok
// is false when found_at names none.
func issueFootprint(iss capture.Issue) ([]string, bool) {
	p, ok := capture.FoundAtPath(iss.FoundAt)
	if !ok {
		return nil, false
	}
	p = path.Clean(p)
	if !fsutil.ValidRelPath(p) {
		return nil, false
	}
	return []string{p}, true
}

// noFoundAtHandBack is the drain's hand-back of an issue it cannot reserve:
// its found_at names no path. It is a flag, routed home to the record.
func noFoundAtHandBack(issue string, now time.Time) DrainRoute {
	return DrainRoute{Issue: issue, From: DrainFromReservation, Kind: "no-found-at", Route: RouteHome,
		Home:   "the issue's record: give it a found_at naming the file the fix changes, which the drain reserves it with",
		Reason: issue + " has no found_at path to reserve it with; handed back", Wrote: "nothing: a flag in this summary", At: &now}
}

// releaseHandedBack releases the reservation of an issue run whose lane was
// handed back, as the drain routes the hand-back: the issue is a person's
// again, so it is not left holding others off until it lapses. It returns
// the route's clause, "" for a run that holds no reservation.
func releaseHandedBack(repoRoot string, run State, now time.Time) string {
	if run.ReservedAt == nil || run.Issue() == "" {
		return ""
	}
	clause, failed := releaseReservation(repoRoot, run.Key, now)
	if failed {
		clause += ", so it lapses with its lease"
	}
	return termsafe.Sanitize(fsutil.RedactHome(clause))
}
