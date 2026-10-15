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

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
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

// fenceLease is the lease the land fence renews with. Under a ruleset mirror
// every landing the loop makes waits on a person, so it takes the longest
// lease, lease.Max: the loop arms a merge only behind a required approval (a
// review, or a code owner's), and otherwise leaves the pull request open for
// a person to merge, and either can wait overnight. A day is also past any
// merge queue's check timeout and its margin (C43). With no mirror at all it
// is the default lease (decision 29 on the intent, C44). Past the fence,
// renewals keep the hold: a step's, a receipt's and every drain move's, each
// at half the lease and keeping its term.
func fenceLease(mirrored bool) time.Duration {
	if mirrored {
		return lease.Max
	}
	return lease.Default
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

// reserveKey reserves a run's key on origin for this copy with req's
// footprint (and its take-over and lease), and returns the run record's line
// for it. The caller has found a remote named origin. Another's hold is the
// back-off, contention at the reservation check; any other refusal stops the
// start. fresh is true when the act made the reservation (a reserve or a
// take-over) rather than renewing one this copy already held, so only a
// fresh one is released when the run is not created.
func reserveKey(repoRoot, key string, req claim.Request, now time.Time) (at *time.Time, note string, fresh bool, err error) {
	req.Record = key
	res, err := claim.Reserve(repoRoot, req, claimOptions(now))
	return reservedNote(res, err, now)
}

// reservedNote is reserveKey's reading of a reservation act's outcome: the
// run record's line for it, or the refusal.
func reservedNote(res claim.Result, err error, now time.Time) (at *time.Time, note string, fresh bool, _ error) {
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

// reserveStart is a new run's reservation, once its checks pass and before
// it is created: in a checkout with a remote named origin, the key is
// reserved with the footprint the caller read (the drain's, from the issue's
// found_at) or, when it named none, the one the run's record names
// (runFootprint), and --take-over passes through. A checkout with no remote
// named origin has nowhere to reserve, and reserves nothing (nil, "").
func reserveStart(repoRoot string, chk CheckResult, o Options) (at *time.Time, note string, fresh bool, err error) {
	if o.noReserve {
		return nil, "", false, nil
	}
	ok, err := claim.HasOrigin(repoRoot)
	if err != nil || !ok {
		return nil, "", false, err
	}
	fp := o.reserve
	if fp == nil {
		if fp, err = runFootprint(repoRoot, chk.Key, chk.Spec, "nothing was started"); err != nil {
			return nil, "", false, err
		}
	}
	return reserveKey(repoRoot, chk.Key, claim.Request{Footprint: fp, TakeOver: o.TakeOver}, o.now())
}

// reserveResumed is a resumed run's reservation: `abcd build` over a run in
// progress reserves its key again, with the footprint its record names, so a
// run started before reservations (or before its checkout had an origin)
// takes one, a reservation `build --release` released is taken again, one
// this copy holds live is renewed, keeping its footprint and term
// (reserveAgain), so a resume never cuts the day the land fence gave a pull
// request waiting on a person nor meets an overlap a race let in, and one
// whose footprint the record has since widened is reserved again with the
// widened footprint, judged for overlap. A run
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
	at, note, fresh, err := reserveAgain(repoRoot, st.Key, fp, o.TakeOver, now)
	if err != nil {
		return err
	}
	if err := mutate(repoRoot, res.RunID, func(_ *os.Root, st *State) (bool, error) {
		if st.ReservedAt == nil {
			st.ReservedAt = at
		}
		st.Record = append(st.Record, Entry{At: now, Stage: StageReservation, Note: note})
		st.UpdatedAt = now
		return true, nil
	}); err != nil {
		if fresh {
			releaseKey(repoRoot, st.Key, now)
		}
		return err
	}
	res.Reservation = note
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
// so the widening is published and judged for overlap like any reservation.
// A renewal the fetched tip shows the view was stale for (the reservation
// gone, released by a tombstone, or moved to this person's other copy)
// falls back to the reservation any other resume makes: a reserve with the
// record's footprint, --take-over passed through.
func reserveAgain(repoRoot, key string, fp []string, takeOver bool, now time.Time) (*time.Time, string, bool, error) {
	plain := claim.Request{Footprint: fp, TakeOver: takeOver}
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return reserveKey(repoRoot, key, plain, now)
	}
	r, ok := snap.Lookup(key)
	if !ok || r.Copy != claim.CopyThis || r.State != claim.StateLive || !covers(r.Footprint, fp) {
		return reserveKey(repoRoot, key, plain, now)
	}
	res, err := claim.Reserve(repoRoot, claim.Request{Record: key, Renew: true, Lease: renewalTerm(r)}, claimOptions(now))
	var nh *claim.NotHeld
	if errors.As(err, &nh) && (nh.Why == claim.NotHeldAbsent || nh.Why == claim.NotHeldAnotherCopy ||
		(nh.Why == claim.NotHeldReleased && !nh.Derived)) {
		return reserveKey(repoRoot, key, plain, now)
	}
	return reservedNote(res, err, now)
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
// cleaned (a trailing slash dropped), in the order written; empty when the
// spec has no such section or it names none.
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
	for _, p := range spec.ReadFootprint(string(data)).Packages {
		if p = path.Clean(strings.TrimSpace(p)); p != "." && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out, nil
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
func releaseOnHandBack(repoRoot string, st *State, now time.Time) {
	if st.ReservedAt == nil || st.Issue() == "" {
		return
	}
	clause, failed := releaseReservation(repoRoot, st.Key, now)
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
// run, like one handed back by an abcd before that step, is released here.
// It returns the route's clause, "" for a run that holds no reservation.
func releaseHandedBack(repoRoot string, run State, now time.Time) string {
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
	clause, failed := releaseReservation(repoRoot, run.Key, now)
	if failed {
		clause += ", so it lapses with its lease"
	}
	return clause
}

// StageReservationLost is the run record's stage for a reservation the view
// shows held by another person or copy.
const StageReservationLost = "reservation_lost"
