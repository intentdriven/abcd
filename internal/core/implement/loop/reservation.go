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

// claimReserve is claim.Reserve, a seam a test replaces to feed the release
// clause a refusal no git would write.
var claimReserve = claim.Reserve

// claimOptions are the options a run's reservation act runs under: the step's
// clock, so a test's injected clock dates the commits too.
func claimOptions(now time.Time) claim.Options {
	return claim.Options{Now: func() time.Time { return now }}
}

// renewReservation renews the run's reservation before a stage, when the view
// (as last fetched; snap, or read here when nil) shows the run's own, from
// this copy and not released, half through its lease since its
// Abcd-Claimed-At: it fetches and pushes a renewal that keeps the tip's own
// term (renewalTerm), and logs reservation_renewed in the run record. Under
// half its lease nothing is pushed. A renewal that fails is logged, once
// until one succeeds, and its note returned for the step's answer; the stage
// still runs, since the land fence is where a lost reservation stops the run.
// A view that shows the reservation held by another person or copy is said
// too, and logged once.
func renewReservation(repoRoot string, st *State, now time.Time, snap *claim.Snapshot) string {
	if st.ReservedAt == nil {
		return ""
	}
	if snap == nil {
		s, err := claim.View(repoRoot, now)
		if err != nil {
			return renewFailed(st, now, err)
		}
		snap = &s
	}
	own, due, lost := renewalDue(*st, *snap, now)
	if lost != "" {
		logOnce(st, now, StageReservationLost, lost)
		return lost
	}
	if !due {
		return ""
	}
	res, err := claim.Reserve(repoRoot, claim.Request{Record: st.Key, Renew: true, Lease: renewalTerm(*own)}, claimOptions(now))
	if err != nil {
		return renewFailed(st, now, err)
	}
	st.Record = append(st.Record, Entry{At: now, Stage: StageReservationRenewed,
		Note: fmt.Sprintf("renewed the reservation on %s, half its lease having passed, until %s", st.Key, stamp(res.ExpiresAt))})
	st.UpdatedAt = now
	return ""
}

// renewalDue reads the run's reservation from snap: own is the tip, due
// whether it is this copy's, unreleased and half through its lease, and lost
// the note to give when the view shows it held by another person or copy.
func renewalDue(st State, snap claim.Snapshot, now time.Time) (own *claim.Reservation, due bool, lost string) {
	for i, r := range snap.Reservations {
		if recordid.SameID(r.Record, st.Key) {
			own = &snap.Reservations[i]
		}
	}
	if own == nil || (own.State != claim.StateLive && own.State != claim.StateExpired) {
		return own, false, ""
	}
	if own.Copy != claim.CopyThis {
		return own, false, fmt.Sprintf("the reservation on %s is held by %s from another copy, until %s, so the land fence will not arm this run's merge",
			st.Key, own.Holder, stamp(own.EffectiveExpiry))
	}
	half := own.ClaimedAt.Add(own.EffectiveExpiry.Sub(own.ClaimedAt) / 2)
	return own, !now.Before(half), ""
}

// renewalTerm is the lease a renewal pushes: the tip's own term, never
// shorter than the default nor longer than a day, so a renewal never cuts the
// day a fence gave a pull request waiting on a person.
func renewalTerm(own claim.Reservation) time.Duration {
	return min(max(lease.Default, own.EffectiveExpiry.Sub(own.ClaimedAt)), lease.Max)
}

// renewFailed logs a failed renewal in the run record, once until a renewal
// succeeds, and returns its note.
func renewFailed(st *State, now time.Time, err error) string {
	note := "could not renew the reservation on " + st.Key + ": " + err.Error()
	logOnce(st, now, StageReservationRenewFailed, note)
	return note
}

// logOnce appends a reservation entry at stage unless the run record's last
// reservation entry is already one at stage, so a condition that holds across
// many steps (origin unreachable, the reservation another's) is logged once
// until something else about the reservation is.
func logOnce(st *State, now time.Time, stage, note string) {
	if last := lastReservationEntry(*st); last != nil && last.Stage == stage {
		return
	}
	st.Record = append(st.Record, Entry{At: now, Stage: stage, Note: note})
	st.UpdatedAt = now
}

// lastReservationEntry is the run record's last entry about its reservation,
// nil when there is none.
func lastReservationEntry(st State) *Entry {
	for k := len(st.Record) - 1; k >= 0; k-- {
		switch st.Record[k].Stage {
		case StageReservationRenewed, StageReservationRenewFailed, StageReservationLost, StageReservation:
			return &st.Record[k]
		}
	}
	return nil
}

// stamp renders a time as a reservation carries one: RFC 3339, UTC.
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// fenceLease is the lease the land fence renews with: lease.Max, a day,
// mirror or not, because every landing the loop makes waits on a person. It
// arms a merge only behind a required approval (a review, or a code owner's)
// read from the ruleset mirror, and otherwise, a checkout with no mirror
// included, leaves the pull request open for a person to merge, and either
// can wait overnight. A day is also past any merge queue's check timeout and
// its margin (C43). The product thinker ruled the no-mirror case on
// 2026-10-15 ("Hold the sign a day", amending decision 29 and C44,
// iss-2610152020234609): a two-hour lapse let a colleague's drain redo work
// that was only waiting to merge. Past the fence, renewals keep the hold: a
// step's, a receipt's and every drain move's, each at half the lease and
// keeping its term.
const fenceLease = lease.Max

// landFence is the land fence, at the top of the arming: it fetches the run's
// reservation and refuses, arming nothing, unless the tip is this person's
// from this copy and live; when it holds, it renews it with term, the
// fenceLease. It returns the renewed expiry, zero
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
// fetch or push that failed (failed is then true). It never refuses. The
// clause quotes a refusal and a holder, both derived from the remote, so it
// is returned terminal-safe and home-redacted for every caller.
func releaseReservation(repoRoot, id string, now time.Time) (clause string, failed bool) {
	defer func() { clause = termsafe.Sanitize(fsutil.RedactHome(clause)) }()
	res, err := claimReserve(repoRoot, claim.Request{Record: id, Release: true}, claimOptions(now))
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
	return clause
}

// StageReservationLost is the run record's stage for a reservation the view
// shows held by another person or copy.
const StageReservationLost = "reservation_lost"
