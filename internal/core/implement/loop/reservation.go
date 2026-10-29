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
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The run record's stages for the run's reservation.
const (
	StageReservationRenewed     = "reservation_renewed"
	StageReservationRenewFailed = "reservation_renew_failed"
	// StageReservationReleased is a step's release of an issue run's
	// reservation when its lane is handed back.
	StageReservationReleased = "reservation_released"
	// StageReservationReleaseFailed is a step's release of an issue run's
	// reservation that could not push the tombstone: the drain routing the
	// hand-back tries it again.
	StageReservationReleaseFailed = "reservation_release_failed"
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
//
// A reservation that has lapsed is revived, judged for overlap like a new
// one (claim's crowding), but, once every lane is past its implement stage
// and no spec step is pending, against the same person's other copy only on
// the files the run's lanes' commits changed (revivalChangedFiles;
// iss-2610261837062463, dcn-2610291352512058, dcn-2610291925508286), so that
// copy holding a file the lanes never touched stops nothing; the renewal's
// note names each such reservation it went ahead past. While a lane may hold
// uncommitted work, or where those files cannot be read, the whole footprint
// is judged, as before.
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
	req := claim.Request{Record: st.Key, Renew: true, Lease: renewalTerm(*own), Stage: renewalStage(*st)}
	// At its landing, an issue lane's renewal narrows its footprint to the
	// files its branch changed (narrow.go); a narrowing that was due and
	// could not be made is said in the renewal's note.
	var declined string
	req.Narrow, _, declined = narrowedFootprint(repoRoot, *st, *own)
	lapsed := own.State == claim.StateExpired
	if lapsed {
		req.OwnOverlapOn = revivalChangedFiles(repoRoot, *st)
	}
	res, err := reserveRenewal(repoRoot, req, claimOptions(now))
	if err != nil {
		return renewFailed(st, now, err)
	}
	why := "half its lease having passed"
	if lapsed {
		why = "its lease having lapsed"
	}
	note := fmt.Sprintf("renewed the reservation on %s, %s, until %s", st.Key, why, stamp(res.ExpiresAt))
	if len(res.OwnOverlap) > 0 {
		past := make([]string, len(res.OwnOverlap))
		for i, o := range res.OwnOverlap {
			past[i] = o.Path + " (" + o.Record + ")"
		}
		note += "; revived past your other copy's reservation of " + strings.Join(past, ", ") +
			", which no commit of this run's lanes touched"
	}
	switch {
	case res.Narrowed:
		note += fmt.Sprintf("; its footprint narrowed from %s to the files its branch changed: %s",
			strings.Join(own.Footprint, ", "), strings.Join(res.Footprint, ", "))
	case len(req.Narrow) > 0:
		declined = "the narrowing to the files its branch changed could not be written into the reservation"
	}
	if declined != "" {
		note += "; reservation kept at its full footprint: " + declined
	}
	st.Record = append(st.Record, Entry{At: now, Stage: StageReservationRenewed, Note: note})
	st.UpdatedAt = now
	return ""
}

// reserveRenewal is claim.Reserve for a renewal of a run's reservation, a
// seam a test replaces to read the request each renewal makes.
var reserveRenewal = claim.Reserve

// renewalStage is the stage a renewal of st's reservation names, as its
// Abcd-Stage trailer (spc-2610161044461448 step 18; dcn-2610161040394970):
// the stage the run's current lane is in, spelled as the loop spells it,
// which claim.LaneStages lists; "" for a run with no lane left to perform
// (done, handed back, held or discarded), whose renewal names none.
func renewalStage(st State) string {
	i := st.current()
	if i < 0 {
		return ""
	}
	if s := string(st.Lanes[i].Stage); claim.ValidLaneStage(s) {
		return s
	}
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

// fenceAct is the landing step a fence guards, as its refusals name it: the
// arming (fenceArm), or the push before it (fencePush).
type fenceAct struct {
	// not is what the lane does not do, after "so this lane does not".
	not string
	// none is what is left undone when origin did not answer, after "so",
	// or after "no release hold could be judged and".
	none string
	// stays is what a refusal over another person's hold, or a release
	// hold, leaves as it was.
	stays string
}

var (
	fenceArm  = fenceAct{not: "arm its merge", none: "the merge is not armed", stays: "the pull request stays open and unarmed"}
	fencePush = fenceAct{not: "push its branch", none: "nothing is pushed", stays: "nothing is pushed and no pull request is opened"}
)

// landFence is the land fence, at the top of the arming (fence, fenceArm),
// renewing the run's reservation with term, the fenceLease. It returns the
// renewed expiry, zero for a run that holds no reservation, and the notice
// naming each release dated ahead that the hold did not count
// (claim.MergeHold.Notice), "" when there is none.
func landFence(c Context, lane Lane, term time.Duration) (time.Time, string, error) {
	return fence(c, lane, term, fenceArm)
}

// pushFence is the fence before the landing's push (spc-2610161044461448
// step 7; dcn-2610161040374343): the land fence's check, made before the push
// as well as at the arming, so a reservation that lapsed and moved to another
// while the lane worked never becomes a duplicate pull request, and a release
// another session is cutting holds the push as `abcd land`'s own release hold
// does (C7). It renews a reservation that holds with the tip's own term
// (renewalTerm), so the push never cuts a lease; a hold that is now another
// person's, or released as finished on origin's default branch, cannot
// change, and a drain hands the lane back at once (stuck.go).
func pushFence(c Context, lane Lane) (time.Time, string, error) {
	term := lease.Default
	if c.State.ReservedAt != nil {
		if snap, err := claim.View(c.RepoRoot, c.Now); err == nil {
			if r, ok := snap.Lookup(c.State.Key); ok && r.Copy == claim.CopyThis {
				term = renewalTerm(r)
			}
		}
	}
	return fence(c, lane, term, fencePush)
}

// fence is the land fence for act. It fetches the reservations from origin,
// then refuses, doing nothing of act, unless the run's own reservation is
// this person's from this copy and live (renewing it with term,
// renewAtFence), and while a release another session is cutting holds every
// merge (releaseHold, C7). A run that holds no reservation fetches them all
// the same, for the hold (claim.FetchMergeHold, the judgement `abcd land`
// makes too). A fetch that fails is a wait by design either way
// (dcn-2610170116582725; fenceUnread). It returns the renewed expiry, zero
// for a run that holds no reservation, and the notice naming each release
// dated ahead that the hold did not count, "" when there is none.
func fence(c Context, lane Lane, term time.Duration, act fenceAct) (time.Time, string, error) {
	if c.State.ReservedAt == nil {
		m, err := claim.FetchMergeHold(c.RepoRoot, c.Now, claimOptions(c.Now))
		var fe *claim.FetchError
		switch {
		case errors.As(err, &fe):
			return time.Time{}, "", fenceUnread(lane,
				"could not read the reservations from "+claim.Remote+": "+fe.Reason+", so no release hold could be judged and "+act.none,
				fe.Reason, "git fetch the reservations")
		case err != nil:
			return time.Time{}, "", refuse(string(StageLand), "", lane.ID, "the reservations could not be read at the land fence: "+err.Error(),
				"settle what it names, then run `abcd implement step` again")
		}
		return time.Time{}, termsafe.Sanitize(m.Notice()), heldMerge(lane, m, act)
	}
	holds, err := renewAtFence(c, lane, term, act)
	if err != nil {
		return time.Time{}, "", err
	}
	notice, err := releaseHold(c, lane, act)
	return holds, notice, err
}

// fenceUnread is the fence's wait on an origin it could not read: never a
// refusal a drain counts toward a stuck lane (stuck.go), whatever git says,
// since the drain's own read of origin at every move is what gives up on an
// origin that stays unread. A lost network also waits on the shared outage
// probe, as the push's own failure does (netFailure); what names the fetch.
func fenceUnread(lane Lane, reason, gitReason, what string) *Refusal {
	r := contend(string(StageLand), "", lane.ID, reason, "run abcd implement step again once origin answers")
	r.originUnread = true
	if implement.IsNetworkFailure(gitReason) {
		r.outage = &outageReport{service: implement.ServiceNetwork, kind: implement.OutageTool, lane: lane.ID, what: what}
	}
	return r
}

// releaseHold refuses act, as contention, while the reservations as the
// fence's renewal just fetched them show a release being cut that is not this
// copy's own (claim.MergeHolds, the judgement `abcd land` makes too, which
// keeps this copy's first sighting of a release dated ahead). It returns the
// notice naming each release dated ahead that the hold did not count.
func releaseHold(c Context, lane Lane, act fenceAct) (string, error) {
	m, err := claim.MergeHolds(c.RepoRoot, c.Now, claimOptions(c.Now))
	if err != nil {
		return "", refuse(string(StageLand), "", lane.ID, "the release holds could not be read at the land fence: "+err.Error(),
			"settle what it names, then run `abcd implement step` again")
	}
	return termsafe.Sanitize(m.Notice()), heldMerge(lane, m, act)
}

// heldMerge is the merge hold's refusal at the fence for act: every other
// session's lane, the same person's other copy included, leaves its pull
// request open (or its branch unpushed) for the cutting session to merge or
// hold. It names each release, its holder and until when.
func heldMerge(lane Lane, m claim.MergeHold, act fenceAct) error {
	if !m.Held() {
		return nil
	}
	r := contend(string(StageLand), "", lane.ID,
		termsafe.Sanitize(m.Reason()+", so this lane does not "+act.not),
		termsafe.Sanitize(act.stays+"; "+m.Waits()+", then run `abcd implement step` again"))
	r.releaseHeld = true
	return r
}

// renewAtFence renews the run's own reservation at the fence for act: it
// fetches the reservations and refuses, doing nothing of act, unless the tip
// is this person's from this copy and live; when it holds, it renews it with
// term.
func renewAtFence(c Context, lane Lane, term time.Duration, act fenceAct) (time.Time, error) {
	id := c.State.Key
	res, err := reserveRenewal(c.RepoRoot, claim.Request{Record: id, Renew: true, Live: true, Lease: term, Stage: renewalStage(c.State)},
		claimOptions(c.Now))
	if err == nil {
		return res.ExpiresAt, nil
	}
	stage := string(StageLand)
	again := "run `abcd implement step` again"
	reserveAgain := "reserve it again with abcd build " + id
	not := ", so this lane does not " + act.not
	var fe *claim.FetchError
	var nh *claim.NotHeld
	switch {
	case errors.As(err, &fe):
		return time.Time{}, fenceUnread(lane,
			"could not read the reservation on "+id+" from "+claim.Remote+": "+fe.Reason+", so "+act.none,
			fe.Reason, "git fetch the reservation on "+id)
	case errors.As(err, &nh):
		switch nh.Why {
		case claim.NotHeldAnotherPerson:
			r := contend(stage, "", lane.ID,
				fmt.Sprintf("the reservation on %s is now %s's, until %s%s", id, nh.Holder, stamp(nh.Expiry), not),
				act.stays+"; take other work, or settle the record with "+nh.Holder+", then "+again)
			r.final = "the reservation on " + id + " is now " + nh.Holder + "'s"
			return time.Time{}, r
		case claim.NotHeldAnotherCopy:
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" moved to "+nh.YourCopy+not+" here",
				"carry the lane on from that copy, or take the reservation back here (`abcd reserve "+id+" --take-over --footprint <path>`), then "+again)
		case claim.NotHeldReleased:
			if nh.Derived {
				r := contend(stage, "", lane.ID,
					"the reservation on "+id+" was released: "+id+" is finished on "+claim.Remote+"'s default branch"+not,
					"confirm the record is finished on the default branch, and hand the lane back or close its pull request")
				r.final = id + " is finished on " + claim.Remote + "'s default branch"
				return time.Time{}, r
			}
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" was released at "+stamp(nh.ReleasedAt)+not, reserveAgain)
		case claim.NotHeldAbsent:
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" is gone from "+claim.Remote+not, reserveAgain)
		case claim.NotHeldUnreadable:
			return time.Time{}, contend(stage, "", lane.ID,
				"the reservation on "+id+" is unreadable ("+nh.Unreadable+")"+not,
				"it can be taken over once its hold ends at "+stamp(nh.Expiry)+" and five minutes more; then "+reserveAgain)
		case claim.NotHeldLapsed:
			return time.Time{}, contend(stage, "", lane.ID,
				"your reservation on "+id+" lapsed at "+stamp(nh.Expiry)+not,
				again+": a step renews this copy's lapsed reservation before its stage")
		}
	case errors.Is(err, claim.ErrBackOff):
		return time.Time{}, contend(stage, "", lane.ID,
			"the reservation on "+id+" could not be renewed: "+err.Error()+"; so this lane does not "+act.not,
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
	clause, failed := releaseReservation(c.RepoRoot, c.State.Key, "", c.Now)
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
//
// A kind other than "" publishes a hand-back (spc-2610161044461448 step 3):
// the tombstone names the kind and the blob id of the issue's record on
// origin's default branch, so no drain on any account takes the issue again
// until a person changes that record.
func releaseReservation(repoRoot, id, kind string, now time.Time) (clause string, failed bool) {
	return releaseReservationAt(repoRoot, id, "", kind, now)
}

// releaseReservationAt is releaseReservation of the one reservation commit
// tip names ("" for whichever the tip holds): one renewed or reserved again
// since is left as it stands.
func releaseReservationAt(repoRoot, id, tip, kind string, now time.Time) (clause string, failed bool) {
	out := releaseAt(repoRoot, id, tip, kind, "", now)
	return out.clause, out.failed
}

// released is what one release came to: the clause the caller writes, whether
// it failed (a fetch or a push that did not reach origin, or a refusal the
// tip gave for a reason of its own: the reservation still holds, so the
// release is due again), and, for a release asked to publish a hand-back,
// why the hand-back is not on origin and is still due ("" when it was
// published, or is no longer due: the record changed, origin holds no one
// record of it, the tombstone would pass the cap, or the tip already marks
// one). A default branch that could not be read keeps it due. published is
// true only when this release's push put the hand-back on origin.
type released struct {
	clause    string
	failed    bool
	pending   string
	published bool
}

// releaseAt is the one release the loop makes: claim's release of id, fenced
// on tip when it names one, publishing kind when it is not "", judged over
// blob when it is not "" (claim.Request.HandBackBlob). The clause quotes a
// refusal and a holder, both derived from the remote, so it is returned
// terminal-safe and home-redacted for every caller.
func releaseAt(repoRoot, id, tip, kind, blob string, now time.Time) (out released) {
	defer func() {
		out.clause = termsafe.Sanitize(fsutil.RedactHome(out.clause))
		out.pending = termsafe.Sanitize(fsutil.RedactHome(out.pending))
	}()
	res, err := claimReserve(repoRoot, claim.Request{Record: id, Release: true, Tip: tip, HandBack: kind, HandBackBlob: blob}, claimOptions(now))
	var nh *claim.NotHeld
	switch {
	case err == nil && res.Act == claim.ActAlreadyReleased && kind != "" && res.HandBackUnpublished != "":
		out = released{clause: "the reservation on " + id + " was already released; " + res.HandBackUnpublished}
		if res.HandBackUnread {
			out.pending = out.clause
		}
		return out
	case err == nil && res.Act == claim.ActAlreadyReleased:
		return released{clause: "the reservation on " + id + " was already released"}
	case err == nil && kind != "" && res.HandBackUnpublished != "":
		out = released{clause: "released the reservation on " + id + " with a tombstone (" + shortSHA(res.Commit) + "); " + res.HandBackUnpublished}
		if res.HandBackUnread {
			out.pending = out.clause
		}
		return out
	case err == nil && kind != "":
		return released{published: true, clause: "released the reservation on " + id + " with a tombstone (" + shortSHA(res.Commit) + ") marking it handed back as " + kind +
			", so no drain takes it again until its record changes on " + claim.Remote + "'s default branch"}
	case err == nil && res.HandBackKept != nil:
		return released{clause: "released the reservation on " + id + " with a tombstone (" + shortSHA(res.Commit) + "); " + res.HandBackKept.KeptNote(id)}
	case err == nil:
		return released{clause: "released the reservation on " + id + " with a tombstone (" + shortSHA(res.Commit) + ")"}
	case errors.As(err, &nh) && nh.Why == claim.NotHeldAnotherPerson:
		out = released{clause: "left the reservation on " + id + " as it stands: it is " + nh.Holder + "'s"}
	case errors.As(err, &nh) && nh.Why == claim.NotHeldAnotherCopy:
		out = released{clause: "left the reservation on " + id + " as it stands: " + nh.YourCopy + " holds it"}
	case errors.As(err, &nh) && nh.Why == claim.NotHeldMoved:
		out = released{clause: "left the reservation on " + id + " as it stands: it was renewed or reserved again since the drain took it"}
	case errors.Is(err, claim.ErrRefused) && !isPushFailure(err):
		// A refusal of the tip's own (a message it may not write, a tip it
		// cannot read) leaves the reservation holding: it is never logged as
		// released, and the drain routing a hand-back tries it again.
		out = released{clause: "could not release the reservation on " + id + ": " + err.Error(), failed: true}
	default:
		out = released{clause: "could not push the tombstone over the reservation on " + id + ": " + err.Error(), failed: true}
	}
	if kind != "" {
		out.pending = out.clause
	}
	return out
}

// isPushFailure reports whether err is a reservation act that failed reaching
// origin, a fetch or a push, rather than a refusal judged on the tip.
func isPushFailure(err error) bool {
	var fe *claim.FetchError
	var pe *claim.PushError
	return errors.As(err, &fe) || errors.As(err, &pe)
}

// reserved is what a run's reservation act came to: when it was made, the
// run record's line for it, whether it was fresh (a reserve or a take-over)
// rather than a renewal of one this copy already held, so only a fresh one is
// released when the run is not created, and why it could not ask the forge
// whether origin is a fork ("" when it asked, or asked nothing). warnings
// are the act's warnings of a footprint path naming a directory
// (claim.DirectoryFootprint.Warning), which the start passes on.
type reserved struct {
	at            *time.Time
	note          string
	fresh         bool
	forkUnchecked string
	forkAllowed   string
	warnings      []string
}

// reserveKey reserves a run's key on origin for this copy with req's
// footprint (and its take-over and lease), and returns the run record's line
// for it. The caller has found a remote named origin. Another's hold is the
// back-off, contention at the reservation check; any other refusal stops the
// start.
func reserveKey(repoRoot, key string, req claim.Request, now time.Time) (reserved, error) {
	req.Record = key
	res, err := claim.Reserve(repoRoot, req, claimOptions(now))
	return reservedNote(res, err, now)
}

// reserveForJudgement reserves id on origin for the judgement the drain is
// about to ask over it, with footprint (and --take-over), and returns the
// drain's hold: the reservation the act made. Every push it makes is the
// drain's, a renewal of this person's reservation that had lapsed included,
// so the drain records it and releases it; the one exception is a renewal of
// a reservation this person still held live from this copy (made by hand,
// which the move's view could not show it), which gives no hold (nil): it is
// never the drain's to release. Another's hold is contention, as for
// reserveKey.
//
// A drain move passes over every issue with a live run here before it
// reserves, so the act reaches an issue whose lane in this checkout holds it
// live only when a concurrent `abcd build` started that run after the move
// read the runs. For that case the request names the lane's stage: when the
// act renews the lane's own live hold, the renewal keeps the stage another
// account reads rather than clearing it (spc-2610161044461448 step 18,
// dcn-2610170809478944); a fresh or lapsed hold names none. Under
// --take-over the request names no stage at all: a take-over is never a
// renewal that may carry one, and claim refuses a stage beside it.
func reserveForJudgement(repoRoot, id string, footprint []string, takeOver bool, now time.Time) (*DrainHold, error) {
	stage := ""
	if runs, err := Runs(repoRoot); err == nil && !takeOver {
		if run, ok := liveRun(runs, id, true); ok {
			stage = renewalStage(run)
		}
	}
	res, err := claim.Reserve(repoRoot, claim.Request{Record: id, Footprint: footprint, TakeOver: takeOver, Drain: true, Stage: stage}, claimOptions(now))
	if _, err := reservedNote(res, err, now); err != nil {
		return nil, err
	}
	if res.Act == claim.ActRenewed && res.RenewedLive {
		return nil, nil
	}
	return &DrainHold{Commit: res.Commit, ClaimedAt: res.ClaimedAt, ExpiresAt: res.ExpiresAt,
		ForkUnchecked: res.ForkUnchecked, ForkAllowed: res.ForkAllowed}, nil
}

// reservedNote is reserveKey's reading of a reservation act's outcome: the
// run record's line for it, or the refusal. A fork check the act could not
// make is named in the line, as reserve names it, so the run record says so.
func reservedNote(res claim.Result, err error, now time.Time) (reserved, error) {
	if err != nil {
		if errors.Is(err, claim.ErrBackOff) {
			return reserved{}, contend(StageReservation, CheckReservation, "", err.Error(),
				"take other work; `abcd peers` lists the reservations on "+claim.Remote)
		}
		if ce := (*claim.CapError)(nil); errors.As(err, &ce) {
			r := refuse(StageReservation, "", "", err.Error(), "wait for one of your pull requests to merge, which ends its reservation, then run the verb again")
			r.capped = slices.Clone(ce.Held)
			return reserved{}, r
		}
		return reserved{}, refuse(StageReservation, "", "", err.Error(), "settle what it names, then run the verb again")
	}
	verb := map[claim.Act]string{claim.ActReserved: "reserved", claim.ActRenewed: "renewed the reservation on",
		claim.ActTakenOver: "took over the reservation on"}[res.Act]
	note := fmt.Sprintf("%s %s on %s for this copy until %s, footprint: %s", verb, res.Record, claim.Remote,
		stamp(res.ExpiresAt), strings.Join(res.Footprint, ", "))
	// A person's own reservation over a hand-back goes ahead, and says so
	// (spc-2610161044461448 step 3); a drain's backed off above.
	if res.HandedBack != nil {
		note += "; " + res.HandedBack.Note(res.Record)
	}
	if res.ForkUnchecked != "" {
		note += "; " + claim.ForkUncheckedLine(res.ForkUnchecked)
	}
	if res.ForkAllowed != "" {
		note += "; " + claim.ForkAllowedLine(res.ForkAllowed)
	}
	t := now
	r := reserved{at: &t, note: note, fresh: res.Act != claim.ActRenewed,
		forkUnchecked: res.ForkUnchecked, forkAllowed: res.ForkAllowed}
	for _, d := range res.Directories {
		r.warnings = append(r.warnings, d.Warning())
	}
	return r, nil
}

// reserveStart is a new run's reservation, once its checks pass and before
// it is created: in a checkout with a remote named origin, the key is
// reserved with the footprint the caller read (the drain's, from the issue's
// found_at) or, when it named none, the one the run's record names
// (runFootprint), and --take-over passes through. A checkout with no remote
// named origin has nowhere to reserve, and reserves nothing (nil, "").
func reserveStart(repoRoot string, chk CheckResult, o Options) (reserved, error) {
	if o.noReserve {
		return reserved{}, nil
	}
	if o.held != nil {
		t := o.held.at
		return reserved{at: &t, note: o.held.note, forkUnchecked: o.held.forkUnchecked, forkAllowed: o.held.forkAllowed}, nil
	}
	ok, err := claim.HasOrigin(repoRoot)
	if err != nil || !ok {
		return reserved{}, err
	}
	fp := o.reserve
	if fp == nil {
		if fp, err = runFootprint(repoRoot, chk.Key, chk.Spec, "nothing was started"); err != nil {
			return reserved{}, err
		}
	}
	return reserveKey(repoRoot, chk.Key, claim.Request{Footprint: fp, TakeOver: o.TakeOver, Drain: o.drain}, o.now())
}

// heldReservation is the reservation a drain took on an issue before it
// asked the judgement over it, as the move's view shows it: when it was
// claimed, the run record's line for it, and the fork check the reservation
// could not make or waived (DrainHold). The lane a no opens starts on it with
// no second push; it is not fresh, so a start that then fails leaves it to the
// drain, which releases it.
type heldReservation struct {
	at                         time.Time
	note                       string
	forkUnchecked, forkAllowed string
}

// reserveResumed is a resumed run's reservation: `abcd build` over a run in
// progress reserves its key again, with the footprint its record names, so a
// run started before reservations (or before its checkout had an origin)
// takes one, a reservation `build --release` released is taken again, one
// this copy holds live is renewed, keeping its footprint and term
// (reserveAgain), so a resume never cuts the day the land fence gave a pull
// request waiting on a person nor meets an overlap a race let in, and one
// whose footprint the record has since widened is reserved again with the
// widened footprint, judged for overlap. Only a run whose lane is at its
// landing can hold a narrowed tip, so only there is an issue's tip within its
// found_at renewed as it stands (reserveAgain's landing). A run
// handed back starts nothing further and reserves nothing. The run's state
// records the reservation; a refusal leaves the run unchanged.
func reserveResumed(repoRoot string, res *StartResult, o Options) error {
	if o.noReserve {
		return nil
	}
	ok, err := claim.HasOrigin(repoRoot)
	if err != nil || !ok {
		return err
	}
	st, err := ReadState(repoRoot, res.RunID)
	if err != nil {
		return err
	}
	if st.handedBack() {
		return nil
	}
	fp, err := runFootprint(repoRoot, st.Key, st.Spec, res.RunID+" is unchanged")
	if err != nil {
		return err
	}
	now := o.now()
	i := st.current()
	landing := i >= 0 && st.Lanes[i].Stage == StageLand
	r, err := reserveAgain(repoRoot, st.Key, fp, o.TakeOver, landing, renewalStage(st), now)
	if err != nil {
		return err
	}
	if err := mutate(repoRoot, res.RunID, func(_ *os.Root, st *State) (bool, error) {
		if st.ReservedAt == nil {
			st.ReservedAt = r.at
		}
		st.Record = append(st.Record, Entry{At: now, Stage: StageReservation, Note: r.note})
		st.UpdatedAt = now
		return true, nil
	}); err != nil {
		if r.fresh {
			releaseKey(repoRoot, st.Key, now)
		}
		return err
	}
	res.Reservation, res.ForkUnchecked, res.ForkAllowed = r.note, r.forkUnchecked, r.forkAllowed
	res.ReservationWarnings = r.warnings
	return nil
}

// reserveAgain is a resumed run's reservation act. Over this copy's own live
// reservation, as last fetched, whose footprint covers every path the
// record names now (fp), it is a renewal keeping the tip's footprint and
// term (renewalTerm), so a resume never cuts the fence's day and, as an act
// that widens nothing, is not judged for overlap: an overlap a race let in
// would otherwise refuse the resume while `implement step` renews the very
// same reservation. A record whose footprint widened since the tip was
// written (a spec's packages line that gained a path) is reserved with it,
// so the widening is published and judged for overlap like any reservation,
// with the tip's term, so a widening resume never cuts the fence's day either.
// An issue's tip that lies within its record's footprint, while its lane is
// at its landing (landing), is the narrowing of that landing (narrow.go),
// renewed as it stands, never read as a widening; anywhere short of the
// landing nothing has narrowed it, so a tip narrower than the found_at is a
// found_at widened mid-run, and is widened like an intent's.
// A renewal the fetched tip shows the view was stale for (the reservation
// gone, released by a tombstone, or moved to this person's other copy)
// falls back to the reservation any other resume makes: a reserve with the
// record's footprint, --take-over passed through.
func reserveAgain(repoRoot, key string, fp []string, takeOver, landing bool, stage string, now time.Time) (reserved, error) {
	plain := claim.Request{Footprint: fp, TakeOver: takeOver}
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return reserveKey(repoRoot, key, plain, now)
	}
	r, ok := snap.Lookup(key)
	if !ok || r.Copy != claim.CopyThis || r.State != claim.StateLive {
		return reserveKey(repoRoot, key, plain, now)
	}
	// An issue run's tip within the record's footprint, its lane at the
	// landing, is the narrowing that landing made (narrow.go), never a
	// widening of the record: the resume renews it as it stands, so it
	// neither publishes the whole found_at again nor is refused by an
	// overlap the narrowing let in.
	narrowed := landing && validIssueKey(key) && covers(fp, r.Footprint)
	if !narrowed && !covers(r.Footprint, fp) {
		return widenReservation(repoRoot, key, fp, takeOver, r, now)
	}
	res, err := reserveRenewal(repoRoot, claim.Request{Record: key, Renew: true, Lease: renewalTerm(r), Stage: stage}, claimOptions(now))
	var nh *claim.NotHeld
	if errors.As(err, &nh) && (nh.Why == claim.NotHeldAbsent || nh.Why == claim.NotHeldAnotherCopy ||
		(nh.Why == claim.NotHeldReleased && !nh.Derived)) {
		return reserveKey(repoRoot, key, plain, now)
	}
	return reservedNote(res, err, now)
}

// widenReservation reserves key again with fp, a footprint the tip own does
// not cover, keeping own's term: the widening is published and judged for
// overlap (and against the cap) like any reservation, so a widening never
// cuts the day a fence gave a pull request either.
func widenReservation(repoRoot, key string, fp []string, takeOver bool, own claim.Reservation, now time.Time) (reserved, error) {
	return reserveKey(repoRoot, key, claim.Request{Footprint: fp, TakeOver: takeOver, Lease: renewalTerm(own)}, now)
}

// reserveOnLeavingLand reserves an issue lane's whole found_at again when the
// lane leaves its landing for an earlier stage (a landing's sync sends it to
// a fresh round), before the lane goes on (dcn-2610170947246937, refining
// dcn-2610170919040486): the landing's renewal may have narrowed the
// reservation to the lane's changes, and the fix round that may follow can
// change any file of the found_at. The sync calls it once it knows it will
// merge and before it merges (mergeDefault; dcn-2610171009498218), so a
// refusal leaves the branch and the worktree at the judged head; the move
// calls it for any other way back from the landing (performMove). A tip of this copy's that already covers the found_at, an
// intent's run, and a run holding no reservation take no act; a tip the view
// shows lost is the land fence's to stop. The widening is the one a resume
// makes (widenReservation), judged for overlap and against the cap; a
// refusal of it refuses the step for a person, naming what it met, with
// nothing merged by the step and nothing pushed, and the lane stays at its
// landing. note is the run record's line for a widening made.
func reserveOnLeavingLand(repoRoot string, st State, lane Lane, now time.Time) (note string, err error) {
	if st.ReservedAt == nil || st.Issue() == "" {
		return "", nil
	}
	fp, err := runFootprint(repoRoot, st.Key, st.Spec, "the lane stays at its landing and nothing is pushed")
	if err != nil {
		return "", err
	}
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return "", refuse(string(StageLand), "", lane.ID,
			"the lane leaves its landing for a fresh round, and its reservation on "+st.Key+" cannot be read to reserve its whole found_at again: "+fsutil.RedactHome(err.Error())+"; the lane stays at its landing and nothing is pushed",
			"settle what it names, then run `abcd implement step` again")
	}
	own, ok := snap.Lookup(st.Key)
	if !ok || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) || covers(own.Footprint, fp) {
		return "", nil
	}
	r, err := widenReservation(repoRoot, st.Key, fp, false, own, now)
	if err != nil {
		reason := err.Error()
		if rf, ok := AsRefusal(err); ok {
			reason = rf.Reason
		}
		return "", refuse(string(StageLand), "", lane.ID,
			fmt.Sprintf("the lane leaves its landing for a fresh round, whose fix round may change any file of %s's found_at, but its reservation, narrowed to %s, cannot be widened back to %s: %s; the lane stays at its landing, its branch at the judged head %s, with nothing merged and nothing pushed",
				st.Key, strings.Join(own.Footprint, ", "), strings.Join(fp, ", "), reason, shortSHA(lane.HeadSHA)),
			"wait for the reservation it names to end, or settle it with its holder, then run `abcd implement step` again; or hand the lane back so the issue is built again")
	}
	return r.note + ", widened again as the lane leaves its landing for a fresh round", nil
}

// covers reports whether every path of fp lies within held: equal to one of
// its paths or beneath one, segment by segment, the sense in which two
// footprints overlap.
func covers(held, fp []string) bool {
	for _, p := range fp {
		if !slices.ContainsFunc(held, func(h string) bool {
			h, p := path.Clean(h), path.Clean(p)
			return h == p || strings.HasPrefix(p, h+"/")
		}) {
			return false
		}
	}
	return true
}

// runFootprint is the footprint a run's record names to reserve its key
// with: an intent's spec's `## Footprint` packages line, each cleaned, and an
// issue's found_at path, read from whichever status folder holds the issue:
// a resume is not re-judged, and the run's own lane resolves its issue, so an
// issue that has left open/ mid-run still names its footprint. A record that
// names none is refused at the reservation, its reason ending with outcome
// (what was not done).
func runFootprint(repoRoot, key, specID, outcome string) ([]string, error) {
	if validIssueKey(key) {
		lr, err := capture.List(capture.ListRequest{RepoRoot: repoRoot, State: capture.StateAll})
		if err != nil {
			return nil, err
		}
		i := slices.IndexFunc(lr.Issues, func(iss capture.Issue) bool { return recordid.SameID(iss.ID, key) })
		if i < 0 {
			return nil, refuse(StageReservation, "", "", key+" is in no status folder of this checkout's ledger, so it names no found_at path to reserve it with; "+outcome,
				"restore the issue's record, or remove the run's directory if the issue is gone for good")
		}
		if fp, ok := issueFootprint(lr.Issues[i]); ok {
			return fp, nil
		}
		return nil, refuse(StageReservation, "", "", key+" has no found_at path to reserve it with; give it a found_at naming the file the fix changes, then run abcd build again; "+outcome,
			"give the issue's record a found_at naming the file the fix changes, which the build reserves it with")
	}
	fp, err := specPackages(repoRoot, specID)
	var entry *footprintEntryError
	if errors.As(err, &entry) {
		return nil, refuse(StageReservation, "", "", entry.Error()+", then run abcd build again; "+outcome,
			"name each package on the `- packages:` line of "+specID+"'s `## Footprint` section by its path alone; the reservation names those paths for every other participant")
	}
	if err != nil {
		return nil, err
	}
	if len(fp) == 0 {
		return nil, refuse(StageReservation, "", "", noFootprint(specID, key)+", then run abcd build again; "+outcome,
			"name the packages the build will change on the `- packages:` line of "+specID+"'s `## Footprint` section; the reservation names them for every other participant")
	}
	return fp, nil
}

// noFootprint says that specID names no footprint to reserve key with: the
// start's refusal at the reservation, and the pick's exclusion.
func noFootprint(specID, key string) string {
	return fmt.Sprintf("%s has no footprint under ## Footprint to reserve %s with; fill its packages line", specID, key)
}

// maxSpecBytes caps a spec read the loop makes for its footprint.
const maxSpecBytes = 1 << 20

// specPackages is the packages specID's `## Footprint` section names, each
// read as written when that names something in the tree and otherwise as its
// leading path alone, a trailing parenthetical note dropped, and cleaned (a
// trailing slash dropped), in the order written; empty when the
// spec has no such section or it names none. An entry that is still prose,
// holding whitespace and naming nothing in the tree, is a
// *footprintEntryError rather than a path that shares nothing with the
// package it means (iss-2610241549334990).
func specPackages(repoRoot, specID string) ([]string, error) {
	store, err := spec.Load(repoRoot)
	if err != nil {
		return nil, err
	}
	sp, ok := store.Lookup(specID)
	if !ok {
		return nil, fmt.Errorf("%s is not in the spec store", specID)
	}
	data, err := fsutil.ReadGuarded(filepath.Join(repoRoot, filepath.FromSlash(sp.Path)), maxSpecBytes)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range spec.ReadFootprint(string(data)).Packages {
		p := strings.TrimSpace(entry)
		if !inTree(repoRoot, p) {
			p = leadingPath(p)
		}
		if strings.ContainsFunc(p, unicode.IsSpace) && !inTree(repoRoot, p) {
			return nil, &footprintEntryError{Spec: specID, Entry: entry}
		}
		if p = path.Clean(p); p != "." && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out, nil
}

// leadingPath is a packages entry less a trailing parenthetical note, one
// set off from the path by whitespace: `internal/core/x (new)` is
// internal/core/x, while `app/(marketing)` is a path and stays whole. An
// entry that is all parenthetical is left as it is.
func leadingPath(entry string) string {
	e := strings.TrimSpace(entry)
	if !strings.HasSuffix(e, ")") {
		return e
	}
	depth := 0
	for i := len(e) - 1; i > 0; i-- {
		switch e[i] {
		case ')':
			depth++
		case '(':
			if depth--; depth == 0 {
				if lead := strings.TrimRightFunc(e[:i], unicode.IsSpace); len(lead) < i {
					return lead
				}
				return e
			}
		}
	}
	return e
}

// inTree reports whether p, a repository-relative path, names something in
// repoRoot's working tree; a path leaving the tree names nothing in it.
func inTree(repoRoot, p string) bool {
	rel := filepath.FromSlash(p)
	if !filepath.IsLocal(rel) {
		return false
	}
	_, err := os.Lstat(filepath.Join(repoRoot, rel))
	return err == nil
}

// footprintEntryError is a spec's packages entry that names no path: still
// prose once its trailing note is dropped, and nothing in the tree by that
// name. The start refuses at the reservation on it, and the pick excludes
// its intent at the check `footprint`.
type footprintEntryError struct{ Spec, Entry string }

func (e *footprintEntryError) Error() string {
	return fmt.Sprintf("%s's packages entry %q under ## Footprint names no path; write it as its path alone, any note in a trailing parenthetical", e.Spec, e.Entry)
}

// releaseKey releases a reservation a start took and then could not create
// its run for, best effort: a failure leaves it to lapse with its lease.
func releaseKey(repoRoot, key string, now time.Time) {
	_, _ = claim.Reserve(repoRoot, claim.Request{Record: key, Release: true}, claimOptions(now))
}

// ReleaseReservation is `abcd build <id> --release`: it pushes a tombstone
// over this person's reservation on id from this copy, and changes nothing in
// any run's state; a later step of the run meets the release at the land
// fence (C36). A key that is not an intent or issue id is refused by name.
func ReleaseReservation(repoRoot, key string, o Options) (claim.Result, error) {
	if row, ok := keyCheck(key); !ok {
		return claim.Result{}, CheckResult{Key: key, Checks: []CheckRow{row}}.refusal()
	}
	return claim.Reserve(repoRoot, claim.Request{Record: key, Release: true}, claimOptions(o.now()))
}

// releaseOnHandBack releases an issue run's reservation as its lane is handed
// back (iss-2610142247197128): the issue is a person's again, so it is not
// left holding others off until it lapses, whether a drain routes the
// hand-back or the run was started by `abcd build <iss-N>`. The run record
// logs the outcome at reservation_released or, when the fetch or the push
// failed, at reservation_release_failed, which the drain routing the
// hand-back tries again (releaseHandedBack). An intent's run keeps its
// reservation: its held lanes may still land.
func releaseOnHandBack(repoRoot string, st *State, hb HandBack, now time.Time) {
	if st.ReservedAt == nil || st.Issue() == "" {
		return
	}
	clause, failed := releaseReservation(repoRoot, st.Key, tombstoneKind(hb), now)
	stage := StageReservationReleased
	if failed {
		clause += ", so it lapses with its lease unless a drain routing the hand-back releases it"
		stage = StageReservationReleaseFailed
	}
	st.Record = append(st.Record, Entry{At: now, Stage: stage, Note: clause})
	st.UpdatedAt = now
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
// again, so it is not left holding others off until it lapses. A step that
// handed the lane back and released it (releaseOnHandBack) logged
// reservation_released, and that record's clause is returned; one whose
// fetch or push failed logged reservation_release_failed instead, and that
// run, like one handed back by an abcd before that step, is released here,
// and a release that succeeds is logged on the run at reservation_released.
// It returns the route's clause, "" for a run that holds no reservation.
func releaseHandedBack(repoRoot string, run State, hb HandBack, now time.Time) string {
	if run.ReservedAt == nil || run.Issue() == "" {
		return ""
	}
	// The step that handed the lane back has already released it, and its
	// record says how (releaseOnHandBack). A failed release is a different
	// stage, so it is never reused: the tombstone is pushed again below.
	for k := len(run.Record) - 1; k >= 0; k-- {
		if run.Record[k].Stage == StageReservationReleased {
			return run.Record[k].Note
		}
	}
	clause, failed := releaseReservation(repoRoot, run.Key, tombstoneKind(hb), now)
	if failed {
		return clause + ", so it lapses with its lease"
	}
	// The retry released it: the run's record says so (iss-2610150616434848),
	// so its last reservation entry is no longer the failure, and a later
	// routing over the run reuses this entry rather than pushing again. Best
	// effort: the release is on origin whether or not the record takes it.
	_ = mutate(repoRoot, run.RunID, func(_ *os.Root, st *State) (bool, error) {
		for _, e := range st.Record {
			if e.Stage == StageReservationReleased {
				return false, nil
			}
		}
		st.Record = append(st.Record, Entry{At: now, Stage: StageReservationReleased, Note: clause})
		st.UpdatedAt = now
		return true, nil
	})
	return clause
}

// StageReservationLost is the run record's stage for a reservation the view
// shows held by another person or copy.
const StageReservationLost = "reservation_lost"
