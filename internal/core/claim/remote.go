package claim

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Act is what a reservation verb did.
type Act string

const (
	ActReserved        Act = "reserved"
	ActRenewed         Act = "renewed"
	ActTakenOver       Act = "taken_over"
	ActReleased        Act = "released"
	ActAlreadyReleased Act = "already_released"
)

// The copy a reservation was made from, as a result names it: never another
// copy's key.
const (
	CopyThis    = "this"
	CopyAnother = "another"
)

// Replaced is the reservation a take-over replaced.
type Replaced struct {
	// Holder is whose it was; "" for an unreadable tip.
	Holder string
	// Copy is CopyAnother when it was this person's other copy, else "".
	Copy string
	// YourCopy names that other copy, when Copy is CopyAnother: "your
	// worktree <path> on this machine" or "your copy on another computer".
	YourCopy string
	// Commit is the tip replaced.
	Commit string
	// EffectiveExpiry is when it stopped holding (an unreadable tip's hold).
	EffectiveExpiry time.Time
	// Unreadable is why the tip could not be read, "" when it could.
	Unreadable string
}

// Result is a reservation act that reached the remote, or a release that
// found nothing to do.
type Result struct {
	Act    Act
	Record string
	Holder string
	// Copy is CopyThis for every act that pushed, and for a release found
	// already made, whether the tombstone came from this copy.
	Copy      string
	Footprint []string
	ClaimedAt time.Time
	ExpiresAt time.Time
	// Commit is the commit pushed, or the tombstone already there.
	Commit   string
	Replaced *Replaced
	// RenewedLive is true for a renewal (ActRenewed) of a reservation that
	// was still live when the act judged it, false for one that had lapsed:
	// a drain tells a person's own live reservation, which is never its to
	// release, from one it brought back to life with its own push.
	RenewedLive bool
	// DerivedUnchecked is why origin's default branch could not be read, so
	// whether the record is already finished there was not checked; "" when
	// it was read (or origin has none).
	DerivedUnchecked string
	// HandedBack is the hand-back a person's own reservation was made over,
	// its record unchanged on origin's default branch since, which a surface
	// tells the person (HandBack.Note); nil for every other act.
	HandedBack *HandBack
	// HandBackUnpublished is why a release whose tombstone would have marked
	// a hand-back, one asked for or one it kept (HandBackKept), pushed a plain
	// tombstone or nothing instead: origin's default branch could not be
	// read (HandBackUnread), or holds no record of it to name (or several),
	// the record changed since the hand-back was judged, or the tombstone
	// with the hand-back would pass the message cap. "" when the hand-back
	// was published, or none was due.
	HandBackUnpublished string
	// HandBackUnread is true when HandBackUnpublished is because origin's
	// default branch could not be read for the record's blob at all: the
	// hand-back asked for is still due, and a later release may publish it.
	// Every other reason ends it.
	HandBackUnread bool
	// HandBackKept is the hand-back a plain release carried forward: the
	// released reservation was made over that hand-back's tombstone, and its
	// record is unchanged on origin's default branch since, so the tombstone
	// keeps it standing (HandBack.KeptNote); nil for every other act.
	HandBackKept *HandBack
	// ForkUnchecked is why a fresh reservation could not ask the forge whether
	// one of origin's push URLs is a fork (no answer, or a host not
	// github.com by name), so the act went on without the check; "" when it
	// was checked, not asked (a renewal, a release, a local path), found no
	// fork, or found the one the repository allows (ForkAllowed).
	ForkUnchecked string
	// ForkAllowed is the fork, as owner/name, a fresh reservation was let
	// through on because reservation.fork_origin names it, so the check was
	// waived and is said to be (ForkAllowedLine); "" when no fork was named,
	// or none was allowed.
	ForkAllowed string
	// Narrowed is true for a renewal that wrote Request.Narrow as its
	// footprint, the tip's covering it; false for every other act.
	Narrowed bool
	// OwnOverlap is each path this reservation shares with a live reservation
	// the same person holds from another copy, which an act asking for it
	// (Request.WarnOwnOverlap) goes ahead over and is told of
	// (OwnOverlap.Warning); nil when it shares none. Every other act backs off
	// such a path.
	OwnOverlap []OwnOverlap
	// GeneratedShared is each path this reservation shares with another
	// record's live reservation that was no overlap because origin's default
	// branch declares it generated (GeneratedShare.Line), sorted by path and
	// record; nil when it shares none.
	GeneratedShared []GeneratedShare
	// GeneratedUnread is why the overlap judgement recognised no generated
	// file: origin's default branch declares them but its declaration could
	// not be read (GeneratedUnreadLine), so every shared path was judged an
	// overlap; "" when it was read, there is none, or no overlap was judged.
	GeneratedUnread string
	// Withdrew is each merge a release reserved anew withdrew from origin's
	// default branch and recorded (withdraw.go), a partial withdrawal among
	// them; Withdrawals is every withdrawal the commit pushed records.
	Withdrew    []Withdrawal
	Withdrawals []Withdrawal
	// ForgeFailures is each pull request a withdrawal, re-arm or its record
	// failed on, and ForgeUnchecked why the merges armed could not be listed
	// or withdrawn at all; "" when they were, or the act withdraws none.
	ForgeFailures  []ForgeFailure
	ForgeUnchecked string
	// Admitted is the pull request an --admit re-armed and recorded
	// admitted; nil for every other act.
	Admitted *Withdrawal
	// Stranded is each withdrawal of another person's the commit pushed
	// carries, which this copy never re-arms (dcn-2610222219297651).
	Stranded []Withdrawal
	// Settles is each release whose withdrawals this act re-armed or
	// skipped: its own on a release's --release, and any ended release of
	// the same person's it found on its fetch.
	Settles []Settle
	// Directories is each path of the footprint the request named that is a
	// directory git tracks (iss-2610230016544142), with the tracked files
	// beneath it: the act is made all the same, and a surface tells the
	// person to name the files instead (DirectoryFootprint.Warning). Every
	// act that names a footprint is judged, a reserve that renews this copy's
	// own tip (ActRenewed) included; nil for a release, for a Request.Renew
	// renewal, which names no footprint and keeps the tip's, and for a
	// footprint of files alone.
	Directories []DirectoryFootprint
	// Dropped is each path of the footprint of this person's own live tip
	// that a reserve renewing it, or taking it over from their other copy,
	// leaves out (iss-2610290507512835), in the tip's order: the act wrote
	// its own footprint in place of the tip's, so none of them is held now
	// (DroppedLine). nil when it keeps every path, over a lapsed tip, which
	// held nothing, and for every other act.
	Dropped []string
}

// DroppedLine is how a person is told the paths their reserve stopped
// holding (Result.Dropped), the same words wherever it is shown.
func DroppedLine(paths []string) string {
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = echo(p)
	}
	return fmt.Sprintf("the footprint leaves out %s, which the reservation it replaced held, so this reservation no longer holds them; name each again with --footprint to keep it",
		strings.Join(quoted, ", "))
}

// droppedPaths is each path of held that footprint leaves out, in held's
// order: one neither named again nor beneath a directory it names (within);
// nil when it keeps them all.
func droppedPaths(held, footprint []string) []string {
	var out []string
	for _, p := range held {
		if !within([]string{p}, footprint) && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// beforePush, when a test sets it, runs between the judgement and the push:
// the window a second writer's push lands in when two race.
var beforePush func(root string)

// judged is a tip read from the view just fetched, with what decides an act.
type judged struct {
	tip
	present bool
	// derived is true when the record is terminal on the default branch as
	// fetched, or a release subject's tag is on origin as fetched, so its
	// reservation counts as released whatever its expiry.
	derived bool
	// blob is the record's blob on the default branch as last fetched, "" when
	// it holds the record in no file or several, or none can be read: what a
	// hand-back's release names, and what a standing hand-back is judged by.
	blob string
	// blobUnread is why origin's default branch could not be read for the
	// blob at all (this checkout names none of it, or its tree would not
	// read), "" when it was read, whatever it held.
	blobUnread string
	// defaultRef is origin's default branch as the judgement named it, after
	// the act's fetch, "" when this checkout names none: the footprint's
	// directories are read from it under that one name.
	defaultRef string
	verdict    lease.Verdict
	// mine and thisCopy compare a readable tip's holder and copy with the
	// person and copy acting.
	mine, thisCopy bool
}

// released reports whether the tip no longer holds the record: a readable
// tombstone, or a record the default branch already finished, whatever its
// tip says or fails to say.
func (j judged) released() bool {
	return j.present && (j.derived || j.Unreadable == "" && j.message.released())
}

// Reserve performs one act against origin: ask the embargo check (for a
// reservation, renewal or take-over; checkEmbargo), fetch, judge, write one commit, then one plain
// push, which is the compare-and-swap. A release pushes the
// tombstone; every other request reserves, renews or takes over, by what the
// tip it judged allows. A refusal pushes nothing; a failure leaves nothing
// locally but an unreferenced commit git's collection removes.
func Reserve(root string, req Request, o Options) (Result, error) {
	p, err := prepare(root, req, o, func(root string) error { return originPresent(root, "reservations") })
	if err != nil {
		return Result{}, err
	}
	if err := checkEmbargo(p, o); err != nil {
		return Result{}, err
	}
	tree, err := emptyTree(root)
	if err != nil {
		return Result{}, refuse("%s; %s", redacted(err), nothingPushed)
	}
	gap, err := fetch(root, p.now, noReservation, reservations)
	if err != nil {
		return Result{}, err
	}
	p.derivedGap = gap
	// Every act first settles each ended release whose withdrawals are this
	// person's and not yet re-armed (withdrawact.go): its own subject only
	// once tagged, since the act itself settles it otherwise.
	skip := ""
	if isRelease(p.id) {
		if tagged, err := taggedOnOrigin(root, p.id); err == nil && !tagged {
			skip = p.id
		}
	}
	settles, settled := sweepEnded(p.actor(), tree, skip)
	if settled {
		if p.derivedGap, err = fetch(root, p.now, noReservation, reservations); err != nil {
			return Result{}, err
		}
	}
	if isRelease(p.id) && !p.req.Release && !p.req.Renew && p.req.Admit == 0 {
		// A release already tagged has no cut left to hold.
		tagged, err := taggedOnOrigin(root, p.id)
		if err != nil {
			return Result{}, refuse("could not read whether %s is tagged on %s: %s; %s", releaseTag(p.id), Remote, redacted(err), noReservation)
		}
		if tagged {
			return Result{}, refuse("%s is already tagged on %s, so there is no cut to hold; %s", releaseTag(p.id), Remote, nothingPushed)
		}
		// One release is cut at a time: a second would withdraw the merges
		// the first admitted, and neither cut could tell.
		if err := otherLiveRelease(root, p.id, p.now); err != nil {
			return Result{}, err
		}
	}
	j, err := judge(p, tree)
	if err != nil {
		return Result{}, err
	}
	// A release is judged, and this person's withdrawals read, as their own
	// newest push and the release's holder's record it (ownRelease).
	own, err := p.ownRelease(tree, &j)
	if err != nil {
		return Result{}, err
	}
	own.shownOnce(root, p.id, j.Commit)()
	if own.said() {
		settles = append(settles, own)
	}
	res, m, parent, err := decide(p, j)
	if err != nil || res.Act == ActAlreadyReleased {
		if err == nil {
			res.DerivedUnchecked = p.derivedGap
			if isRelease(p.id) && p.req.Release {
				// A release already ended by its tombstone settles what that
				// tombstone left withdrawn.
				if s, ok := settleEnded(p.actor(), tree, p.id, true); ok {
					settles = append(settles, s)
				}
			}
			res.Settles = settles
		}
		return res, err
	}
	// What the act does with the release's withdrawals, before its push.
	settleOwn := false
	var admitting Withdrawal
	admitState := ""
	switch {
	case !isRelease(p.id) || !j.present || j.Unreadable != "":
	case p.req.Release:
		// The cutting copy ends its reservation: the tombstone carries every
		// withdrawal as it stands, and once it has landed every one of this
		// person's is re-armed and a second tombstone records each state.
		m.Withdrawn = j.Withdrawn
		if slices.ContainsFunc(j.Withdrawn, func(w Withdrawal) bool { return w.unsettled() && p.self.Owns(w.By) }) {
			if why, definite := loginMismatch(root, p.self); definite {
				return Result{}, refuse("%s was not released: %s; %s", p.id, why, nothingPushed)
			}
			settleOwn = true
		}
		res.Stranded = stranded(j.Withdrawn, p.self)
	case p.req.Admit != 0:
		// The admission is recorded before the forge re-arms it, so a push
		// that loses leaves nothing armed.
		admitting, _ = withdrawalOf(j.Withdrawn, p.req.Admit)
		if admitState, err = p.admitCheck(admitting); err != nil {
			return Result{}, err
		}
		m.Withdrawn = withWithdrawal(m.Withdrawn, Withdrawal{Number: admitting.Number, ID: admitting.ID, Form: admitting.Form,
			Head: admitting.Head, By: admitting.By, State: WithdrawnAdmitted})
	default:
		// A reservation keeps what its record still has withdrawn: its own
		// admissions too while it is this person's reservation.
		m.Withdrawn = carried(j.Withdrawn, j.mine && !j.message.released() && !j.derived)
		res.Stranded = stranded(m.Withdrawn, p.self)
	}
	if !p.req.Release {
		if err := crowding(p, j, &res); err != nil {
			return Result{}, err
		}
	}
	// The fork check follows the judgement, so only an act that reserves
	// afresh (a reservation or a take-over, never a renewal or a release,
	// however it was asked for) asks the forge, and precedes the commit and
	// the push, so a fork is refused before anything is pushed. A drain's
	// reservation that renews this copy's lapsed one is afresh too: that push
	// makes it the drain's hold, which a lane opens on, so it asks as a fresh
	// one does (spc-2610161044461448 step 10; the review of steps 1 to 18).
	forkUnchecked, forkAllowed := "", ""
	if !p.req.Release && p.req.Admit == 0 && (res.Act != ActRenewed || p.req.Drain && !res.RenewedLive) {
		if forkUnchecked, forkAllowed, err = checkFork(root, p.now, o.home()); err != nil {
			return Result{}, err
		}
	}
	if p.req.Release && m.HandedBack != "" {
		// A hand-back that would take the tombstone past the cap (a reservation
		// another binary accepted, under a shorter label) is dropped, never the
		// release: a refused release would leave the issue held until its lease
		// ran out (spc-2610161044461448, review of steps 1-18).
		if why := handBackUnfit(root, m); why != "" {
			m.HandedBack, m.RecordBlob, res.HandBackKept, res.HandBackUnpublished = "", "", nil, why
			if j.message.released() {
				// Over a tombstone already there, a plain one says nothing new.
				return Result{Act: ActAlreadyReleased, Record: p.id, Holder: j.Holder, Footprint: j.Footprint, ClaimedAt: j.ClaimedAt,
					ExpiresAt: j.ExpiresAt, Commit: j.Commit, HandBackUnpublished: why, DerivedUnchecked: p.derivedGap}, nil
			}
		}
	}
	sha, err := writeCommit(p, m, tree, parent)
	if err != nil {
		return Result{}, err
	}
	if beforePush != nil {
		beforePush(root)
	}
	if perr := push(root, pushArgs(sha, p.id)); perr != nil {
		return Result{}, lostRace(p, tree, j, perr)
	}
	notePushed(root, p.id, sha)
	if !p.req.Release && !p.req.Renew {
		res.Directories = directoryFootprints(root, j.defaultRef, res.Footprint)
	}
	res.Commit = sha
	res.Copy = CopyThis
	res.DerivedUnchecked = p.derivedGap
	res.ForkUnchecked = forkUnchecked
	res.ForkAllowed = forkAllowed
	if settleOwn {
		s, final, ws := p.settleOwnRelease(tree, sha, m.Withdrawn)
		settles = append(settles, s)
		res.Commit, m.Withdrawn = final, ws
	}
	if isRelease(p.id) && !p.req.Release && p.req.Admit == 0 {
		// A release reserved anew withdraws the merges already armed, now
		// that its own push holds every other session's arming, recording
		// them on a renewal of the reservation it just pushed before it
		// touches the forge. A live renewal withdraws again only what it
		// already records as this person's, withdrawn or kept (a withdrawal
		// the forge refused), that the forge has armed: its record is read
		// from this person's own newest push, so each such entry is theirs.
		commit, recorded, made, fails, unchecked := p.withdrawArmed(tree, sha, m, res.Act == ActRenewed && res.RenewedLive)
		res.ForgeFailures, res.ForgeUnchecked, res.Withdrew = fails, unchecked, made
		res.Commit, m = commit, recorded
	}
	if p.req.Admit != 0 {
		commit, err := p.admitAfter(tree, sha, m, admitting, admitState)
		if err != nil {
			return Result{}, err
		}
		res.Commit = commit
		w, _ := withdrawalOf(m.Withdrawn, p.req.Admit)
		res.Admitted = &w
		// An admission recorded already, admitted again, changes no state,
		// so only the first is commented on.
		if admitting.State != WithdrawnAdmitted {
			if f := releaseForge.comment(root, p.id, w, p.holder, false); f != nil {
				res.ForgeFailures = append(res.ForgeFailures, *f)
			}
		}
	}
	res.Withdrawals = m.Withdrawn
	res.Settles = settles
	return res, nil
}

// pushArgs is the one push every reservation act makes: the commit to the
// record's reservation branch on origin (pushTo).
func pushArgs(sha, id string) []string { return pushTo(sha, refName(id)) }

// pushTo is the one push this package makes: commit sha to the full ref on
// origin, plain, with no force, no `+` refspec, no --no-verify and no
// deletion, so the repository's hooks run and the remote takes it only as a
// fast-forward. ref is a full ref a namespace formed (namespace.ref), never a
// value a writer chose.
func pushTo(sha, ref string) []string {
	return []string{"push", "--porcelain", Remote, sha + ":" + ref}
}

// push makes one push pushTo built. A push origin refused comes back as its
// rejection (pushRejection), for the act to explain in its own terms.
func push(root string, args []string) error {
	trace(args...)
	if out, err := gitutil.Net(root, args...); err != nil {
		return pushRejection(out, err)
	}
	return nil
}

// lostRace explains a rejected push. When the branch moved while the push was
// on its way and now holds a reservation this act cannot take, the refusal
// names its holder with the back-off; otherwise the push failure is reported
// as itself.
func lostRace(p *prepared, tree string, before judged, perr error) error {
	pushFailed := &PushError{Reason: redacted(perr)}
	gap, err := fetch(p.root, p.now, noReservation, reservations)
	if err != nil {
		return pushFailed
	}
	p.derivedGap = gap
	j, err := judge(p, tree)
	if err != nil || !j.present || j.Commit == before.Commit {
		return pushFailed
	}
	_, _, _, derr := decide(p, j)
	if derr == nil {
		return pushFailed
	}
	if j.derived {
		return derr
	}
	// A renewal or a release that met another's tip says so in its own
	// terms, which its caller reads.
	var nh *NotHeld
	if errors.As(derr, &nh) {
		return derr
	}
	if errors.Is(derr, ErrBackOff) && j.Unreadable == "" && !j.mine && !j.message.released() {
		return backOff("%s was reserved by %s while this push was on its way, until %s; back off and take other work",
			p.id, j.Holder, stamp(j.verdict.EffectiveExpiry))
	}
	return derr
}

// pushRejection is a failed push's reason: the porcelain status lines that
// name a refused ref ("!" in the first field), which carry git's own word for
// the rejection, then the error with git's stderr, where a remote's or a
// hook's own message arrives.
func pushRejection(stdout string, err error) error {
	var lines []string
	for _, l := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(l, "!") {
			lines = append(lines, strings.Join(strings.Fields(l), " "))
		}
	}
	if len(lines) == 0 {
		return err
	}
	return fmt.Errorf("%s; %w", strings.Join(lines, "; "), err)
}

// decide is the push table: the result an act will report, the message it
// writes and the parent it writes it on, or the refusal.
func decide(p *prepared, j judged) (Result, message, string, error) {
	// A release and a renewal take their own rows before the finished-record
	// back-off decideReserve makes: the holder's tombstone still lands, and a
	// renewal is told the default branch released it, which the land fence
	// reads.
	if p.req.Release {
		return decideRelease(p, j)
	}
	if p.req.Renew {
		return decideRenew(p, j)
	}
	if p.req.Admit != 0 {
		return decideAdmit(p, j)
	}
	res, m, parent, err := decideReserve(p, j)
	// A drain's take-over or renewal of a reservation made over a standing
	// hand-back backs off it, as a drain's reservation over the hand-back's
	// own tombstone does: the tip alone does not show it.
	if err == nil && p.req.Drain && (res.Act == ActTakenOver || res.Act == ActRenewed) {
		if err := drainOverHandBack(p, j); err != nil {
			return Result{}, message{}, "", err
		}
	}
	return res, m, parent, err
}

// decideReserve is a reservation's rows of the push table: reserved afresh,
// renewed, taken over, or backed off.
func decideReserve(p *prepared, j judged) (Result, message, string, error) {
	m := p.reservation()
	res := Result{Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt, ExpiresAt: m.ExpiresAt}
	now := p.now
	switch {
	case j.derived:
		// A record finished on origin's default branch is not reserved, with
		// a tip or without one: the work it names is done (iss-2610160957404827).
		return Result{}, message{}, "", backOff("%s, so there is nothing to reserve; take other work; %s",
			derivedWhy(p.id), nothingPushed)
	case !j.present:
		res.Act = ActReserved
		return res, m, "", nil
	case j.released():
		// A hand-back whose record is unchanged is a person's to settle: a
		// drain backs off it, and a person's own reservation is told of it
		// (spc-2610161044461448 step 3).
		if j.handBackStands() {
			if p.req.Drain {
				return Result{}, message{}, "", j.handBackBackOff(p.id)
			}
			res.HandedBack = &HandBack{Kind: j.HandedBack, Holder: j.Holder, At: j.ReleasedAt}
		}
		res.Act = ActReserved
		return res, m, j.Commit, nil
	case j.Unreadable != "":
		hold := j.verdict.EffectiveExpiry
		if now.Before(hold.Add(TakeoverMargin)) {
			if j.DatedFromFetch {
				return Result{}, message{}, "", backOff("the reservation on %s is unreadable (%s); its commit date cannot be read, so it is held "+
					"from when this copy first fetched it, until %s, and can be taken over after %s",
					p.id, j.Unreadable, stamp(hold), stamp(hold.Add(TakeoverMargin)))
			}
			return Result{}, message{}, "", backOff("the reservation on %s is unreadable (%s); it is held until %s and can be taken over after %s",
				p.id, j.Unreadable, stamp(hold), stamp(hold.Add(TakeoverMargin)))
		}
		res.Act = ActTakenOver
		res.Replaced = &Replaced{Commit: j.Commit, EffectiveExpiry: hold, Unreadable: j.Unreadable}
		return res, m, j.Commit, nil
	case j.mine && j.thisCopy:
		res.Act, res.RenewedLive = ActRenewed, j.verdict.Live
		if j.verdict.Live {
			// A lapsed tip held nothing while it lapsed, so reviving it
			// drops nothing.
			res.Dropped = droppedPaths(j.Footprint, m.Footprint)
		}
		if p.req.Drain && j.verdict.Live {
			// A drain's re-reserve that renews its own live hold, reached
			// when a build started the lane's run during the drain's move, is
			// a lane's renewal, so it carries the lane's stage; one over a
			// lapsed hold is afresh and carries none (spc-2610161044461448
			// step 18).
			m.Stage = p.req.Stage
		}
		return res, m, j.Commit, nil
	}
	exp := j.verdict.EffectiveExpiry
	lapsed := !now.Before(exp.Add(TakeoverMargin))
	if j.mine {
		if !p.req.TakeOver && !lapsed {
			if wt := siblingCopy(p.root, j.Copy); wt != "" {
				return Result{}, message{}, "", backOff("you are already building %s from your worktree %s on this machine, until %s; "+
					"renew it from there, or pass --take-over to continue here", p.id, wt, stamp(exp))
			}
			return Result{}, message{}, "", backOff("you are already building %s on another computer, until %s; pass --take-over to continue here",
				p.id, stamp(exp))
		}
		res.Act = ActTakenOver
		res.Replaced = &Replaced{Holder: j.Holder, Copy: CopyAnother, YourCopy: yourCopyAt(siblingCopy(p.root, j.Copy)),
			Commit: j.Commit, EffectiveExpiry: exp}
		if j.verdict.Live {
			res.Dropped = droppedPaths(j.Footprint, m.Footprint)
		}
		return res, m, j.Commit, nil
	}
	switch {
	case j.verdict.Live:
		return Result{}, message{}, "", backOff("%s is reserved by %s until %s; back off and take other work", p.id, j.Holder, stamp(exp))
	case !lapsed:
		return Result{}, message{}, "", backOff("%s is reserved by %s, lapsed at %s, under five minutes ago by this computer's clock; retry after %s",
			p.id, j.Holder, stamp(exp), stamp(exp.Add(TakeoverMargin)))
	}
	res.Act = ActTakenOver
	res.Replaced = &Replaced{Holder: j.Holder, Commit: j.Commit, EffectiveExpiry: exp}
	return res, m, j.Commit, nil
}

// decideRelease is the release rows of the push table: only this person's
// reservation from this copy is released, by a tombstone child; a tombstone
// already at the tip is reported and nothing is pushed.
func decideRelease(p *prepared, j judged) (Result, message, string, error) {
	switch {
	case !j.present:
		return Result{}, message{}, "", refuse("%s holds no reservation on %s to release; %s", p.id, Remote, nothingPushed)
	case j.Unreadable != "":
		return Result{}, message{}, "", refuse("the reservation on %s is unreadable (%s), so nobody holds it to release; %s",
			p.id, j.Unreadable, nothingPushed)
	case j.message.released():
		return releasedAlready(p, j)
	case !j.mine:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldAnotherPerson, ErrRefused,
			fmt.Sprintf("%s is reserved by %s, not by you; %s", p.id, j.Holder, nothingPushed))
	case !j.thisCopy:
		yours := yourCopyAt(siblingCopy(p.root, j.Copy))
		nh := j.notHeld(p.id, NotHeldAnotherCopy, ErrRefused,
			fmt.Sprintf("%s is held by %s; release it there, or take it over here first; %s", p.id, yours, nothingPushed))
		nh.YourCopy = yours
		return Result{}, message{}, "", nh
	case p.req.Tip != "" && j.Commit != p.req.Tip:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldMoved, ErrRefused,
			fmt.Sprintf("the reservation on %s is no longer commit %s: it was renewed or reserved again since, so it is left as it stands; %s",
				p.id, p.req.Tip[:12], nothingPushed))
	}
	m := p.tombstone(j.Footprint)
	res := Result{Act: ActReleased, Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt,
		ExpiresAt: m.ExpiresAt}
	if p.req.HandBack != "" {
		res.HandBackUnpublished, res.HandBackUnread = p.markHandBack(&m, j)
		return res, m, j.Commit, nil
	}
	// A plain release of a reservation made over a hand-back that still
	// stands keeps it: the hand-back is a person's to settle by changing the
	// record, which reserving and releasing the issue does not do.
	kept, why := keptHandBack(p, j)
	if kept != nil {
		m.HandedBack, m.RecordBlob = kept.HandedBack, kept.RecordBlob
		res.HandBackKept = &HandBack{Kind: kept.HandedBack, Holder: kept.Holder, At: kept.ReleasedAt}
	}
	res.HandBackUnpublished = why
	return res, m, j.Commit, nil
}

// markHandBack names the hand-back asked for on m, the tombstone a release
// pushes over j, with the record's blob on origin's default branch, and
// returns why it named none instead: the branch could not be read (unread is
// then true), no record of it there to name (or several), or a record other
// than the one the hand-back was judged over (HandBackBlob).
func (p *prepared) markHandBack(m *message, j judged) (why string, unread bool) {
	switch {
	case j.blob == "" && j.blobUnread != "":
		return fmt.Sprintf("%s's default branch could not be read (%s), so the tombstone marks no hand-back", Remote, j.blobUnread), true
	case j.blob == "":
		return fmt.Sprintf("%s's default branch as last fetched holds no record of %s, or holds it in more than one file, so the tombstone marks no hand-back", Remote, p.id), false
	case p.req.HandBackBlob != "" && p.req.HandBackBlob != j.blob:
		return fmt.Sprintf("%s's record on %s's default branch has changed since the hand-back was judged, so the tombstone marks no hand-back: a person's edit re-admits it",
			p.id, Remote), false
	}
	m.HandedBack, m.RecordBlob = p.req.HandBack, j.blob
	return "", false
}

// releasedAlready is a release's row over a tip already released by a
// tombstone. A release asked to publish a hand-back over a tombstone that
// marks none publishes it as that tombstone's child, so a hand-back whose
// reservation was released another way (by hand, or by a move whose state
// write failed) is still published; with Tip it is fenced on the commit the
// release names, which must be the tombstone or the reservation it released.
// Every other release over a tombstone pushes nothing.
func releasedAlready(p *prepared, j judged) (Result, message, string, error) {
	copyOf := CopyAnother
	if j.mine && j.thisCopy {
		copyOf = CopyThis
	}
	done := Result{Act: ActAlreadyReleased, Record: p.id, Holder: j.Holder, Copy: copyOf, Footprint: j.Footprint,
		ClaimedAt: j.ClaimedAt, ExpiresAt: j.ExpiresAt, Commit: j.Commit}
	if p.req.HandBack == "" || j.HandedBack != "" || j.derived {
		return done, message{}, "", nil
	}
	if p.req.Tip != "" && p.req.Tip != j.Commit {
		parent, err := firstParent(p.root, j.Commit)
		if err != nil {
			return Result{}, message{}, "", refuse("could not read which reservation the tombstone on %s released: %s; %s", p.id, redacted(err), nothingPushed)
		}
		if parent != p.req.Tip {
			return Result{}, message{}, "", j.notHeld(p.id, NotHeldMoved, ErrRefused,
				fmt.Sprintf("the reservation on %s is no longer commit %s: it was released after it was renewed or reserved again, so the hand-back is not published over it; %s",
					p.id, p.req.Tip[:12], nothingPushed))
		}
	}
	m := p.tombstone(j.Footprint)
	if why, unread := p.markHandBack(&m, j); why != "" {
		done.HandBackUnpublished, done.HandBackUnread = why, unread
		return done, message{}, "", nil
	}
	return Result{Act: ActReleased, Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt,
		ExpiresAt: m.ExpiresAt}, m, j.Commit, nil
}

// decideAdmit is an --admit's row of the push table: only the cutting copy,
// this person's live release reservation from this copy, admits, and only a
// pull request the reservation withdrew and has not admitted yet, and that
// this person withdrew. Its renewal
// keeps every withdrawal and marks that one admitted once the forge has
// re-armed it (admitOne); every other tip is refused, and nothing is pushed.
func decideAdmit(p *prepared, j judged) (Result, message, string, error) {
	n := p.req.Admit
	switch {
	case !j.present:
		return Result{}, message{}, "", refuse("%s holds no reservation on %s to admit #%d into; %s", p.id, Remote, n, nothingPushed)
	case j.derived:
		return Result{}, message{}, "", refuse("%s, so the cut has ended and admits nothing; %s", derivedWhy(p.id), nothingPushed)
	case j.Unreadable != "":
		return Result{}, message{}, "", refuse("the reservation on %s is unreadable (%s); %s", p.id, j.Unreadable, nothingPushed)
	case j.message.released():
		return Result{}, message{}, "", refuse("the reservation on %s was released at %s, so the cut has ended and admits nothing; %s",
			p.id, stamp(j.ReleasedAt), nothingPushed)
	case !j.mine:
		if w, ok := withdrawalOf(j.Withdrawn, n); ok && w.unsettled() && p.self.Owns(w.By) {
			return Result{}, message{}, "", refuse("%s is reserved by %s, not by you: only the copy cutting a release admits a pull request into it, so take the release back with --take-over and admit it then; %s",
				p.id, j.Holder, nothingPushed)
		}
		return Result{}, message{}, "", refuse("%s is reserved by %s, not by you: only the copy cutting a release admits a pull request into it; %s",
			p.id, j.Holder, nothingPushed)
	case !j.thisCopy:
		return Result{}, message{}, "", refuse("%s is held by your copy on another computer: only the copy cutting a release admits a pull request into it, so admit it there; %s",
			p.id, nothingPushed)
	case !j.verdict.Live:
		return Result{}, message{}, "", refuse("your reservation on %s lapsed at %s; reserve it again first; %s",
			p.id, stamp(j.verdict.EffectiveExpiry), nothingPushed)
	}
	i := slices.IndexFunc(j.Withdrawn, func(w Withdrawal) bool { return w.Number == n })
	switch {
	case i < 0:
		listed := "none"
		if len(j.Withdrawn) > 0 {
			listed = strings.Join(numbers(j.Withdrawn), ", ")
		}
		return Result{}, message{}, "", refuse("#%d is not among the pull requests %s withdrew (%s), so there is nothing to admit: only a merge the reservation withdrew is admitted, and one armed since stays armed; %s",
			n, p.id, listed, nothingPushed)
	case !j.Withdrawn[i].unsettled():
		return Result{}, message{}, "", refuse("#%d is recorded %s on %s, so there is nothing to admit; %s", n, j.Withdrawn[i].State, p.id, nothingPushed)
	case !p.self.Owns(j.Withdrawn[i].By):
		// Another person's copy never re-arms a withdrawal
		// (dcn-2610222219297651), and an admission arms one under this
		// person's account: only its withdrawer admits it
		// (dcn-2610230833367984).
		by := j.Withdrawn[i].By
		return Result{}, message{}, "", refuse("#%d was withdrawn by %s, not by you, and only its withdrawer admits it, since an admission arms it under the admitter's account: "+
			"%s takes the release back with --take-over and admits it; %s", n, by, by, nothingPushed)
	}
	// An admission already recorded is admitted again: the forge is read
	// once more, and the pull request armed when it is not, so an --admit
	// cut off between its record and its arming is finished by running it
	// again.
	m := p.reservation()
	m.Withdrawn = carried(j.Withdrawn, true)
	return Result{Act: ActAdmitted, Record: p.id, Holder: p.holder, ClaimedAt: m.ClaimedAt, ExpiresAt: m.ExpiresAt, RenewedLive: true},
		m, j.Commit, nil
}

// decideRenew is a renewal's row of the push table: this person's own
// reservation from this copy, not released, is renewed as its child with the
// lease asked for and the tip's own footprint; every other tip is a *NotHeld,
// so a renewal never reserves anew and never takes over. A lapsed one is
// renewed unless the request requires a live one.
func decideRenew(p *prepared, j judged) (Result, message, string, error) {
	exp := j.verdict.EffectiveExpiry
	switch {
	case !j.present:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldAbsent, ErrBackOff,
			fmt.Sprintf("%s holds no reservation on %s to renew; %s", p.id, Remote, nothingPushed))
	case j.derived:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldReleased, ErrBackOff,
			fmt.Sprintf("the reservation on %s is released: %s; %s", p.id, derivedWhy(p.id), nothingPushed))
	case j.Unreadable != "":
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldUnreadable, ErrBackOff,
			fmt.Sprintf("the reservation on %s is unreadable (%s); %s", p.id, j.Unreadable, nothingPushed))
	case j.message.released():
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldReleased, ErrBackOff,
			fmt.Sprintf("the reservation on %s was released at %s; %s", p.id, stamp(j.ReleasedAt), nothingPushed))
	case !j.mine:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldAnotherPerson, ErrBackOff,
			fmt.Sprintf("%s is reserved by %s until %s, not by you; %s", p.id, j.Holder, stamp(exp), nothingPushed))
	case !j.thisCopy:
		yours := yourCopyAt(siblingCopy(p.root, j.Copy))
		nh := j.notHeld(p.id, NotHeldAnotherCopy, ErrBackOff, fmt.Sprintf("%s is held by %s, until %s; %s", p.id, yours, stamp(exp), nothingPushed))
		nh.YourCopy = yours
		return Result{}, message{}, "", nh
	case p.req.Live && !j.verdict.Live:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldLapsed, ErrBackOff,
			fmt.Sprintf("your reservation on %s lapsed at %s; %s", p.id, stamp(exp), nothingPushed))
	}
	p.req.Footprint = append([]string(nil), j.Footprint...)
	narrowed := len(p.req.Narrow) > 0 && within(p.req.Narrow, j.Footprint)
	if narrowed {
		p.req.Footprint = append([]string(nil), p.req.Narrow...)
	}
	m := p.reservation()
	// A lane's renewal names its stage; a renewal that names none carries
	// none, so only the tip's stage is ever read (spc-2610161044461448 step 18).
	m.Stage = p.req.Stage
	return Result{Act: ActRenewed, Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt,
		ExpiresAt: m.ExpiresAt, Narrowed: narrowed}, m, j.Commit, nil
}

// notHeld is the *NotHeld a judged tip gives an act that needed this
// person's own reservation from this copy.
func (j judged) notHeld(id, why string, kind error, msg string) *NotHeld {
	nh := &NotHeld{Record: id, Why: why, Expiry: j.verdict.EffectiveExpiry, Derived: j.derived && why == NotHeldReleased,
		Unreadable: j.Unreadable, msg: msg, kind: kind}
	if j.Unreadable == "" {
		nh.Holder = j.Holder
		if why == NotHeldReleased && !nh.Derived {
			nh.ReleasedAt = j.ReleasedAt
		}
	}
	return nh
}

// judge reads the record's tip from the view just fetched and judges it: its
// liveness through lease.Judge (an unreadable tip held from its commit date
// for the default lease), whether the default branch already finished the
// record, and whose it is.
func judge(p *prepared, tree string) (judged, error) {
	t, ok, err := readSeen(p.root, p.id, trackingName(p.id), tree, p.now, true)
	if err != nil {
		return judged{}, refuse("could not read the reservation on %s: %s; %s", p.id, redacted(err), noReservation)
	}
	j := judged{tip: t, present: ok}
	if isRelease(p.id) {
		// A release subject names no record on the default branch: its tag
		// ends it, read by the same fetch as its reservation, so it is never
		// stale beside it, and it carries no record blob.
		if !ok {
			return j, nil
		}
		if j.derived, err = taggedOnOrigin(p.root, p.id); err != nil {
			return judged{}, refuse("could not read whether %s is tagged on %s: %s; %s",
				releaseTag(p.id), Remote, redacted(err), noReservation)
		}
	} else {
		// The default branch is read whether or not a tip exists, since a
		// finished record is not reserved either way. One the fetch could not
		// read derives nothing: the copy this clone holds may be stale. Its
		// blob of the record is still read as last fetched, when it reads, and
		// a stale branch reads the record as it was when last fetched: a
		// hand-back made over a newer blob than that reads as changed though
		// no one edited it since, and a person's edit this copy has not
		// fetched reads as unchanged until it is (dcn-2610161412283450,
		// refined by dcn-2610161837009957). One with no blob read stands.
		// One name for the default branch, resolved after the act's fetch,
		// which can create or move origin/HEAD (iss-2610180545134593).
		ref := originDefault(p.root)
		j.defaultRef = ref
		recs, err := defaultRecords(p.root, ref)
		if err != nil && p.derivedGap == "" {
			return judged{}, refuse("could not read whether %s is finished on the default branch: %s; %s",
				p.id, redacted(err), noReservation)
		}
		switch {
		case ref == "":
			j.blobUnread = "this checkout names no default branch of " + Remote
		case err != nil:
			j.blobUnread = p.derivedGap + "; " + redacted(err)
		}
		j.derived = p.derivedGap == "" && terminal(recs[p.id], p.id)
		j.blob = blobOf(recs[p.id])
		if !ok {
			return j, nil
		}
	}
	if t.Unreadable != "" {
		j.verdict = lease.Judge(lease.Stamp{Unreadable: true, WrittenAt: t.WrittenAt, FirstSeen: t.FirstSeen}, lease.Default, p.now)
		return j, nil
	}
	j.verdict = lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, p.now)
	j.mine = p.self.Owns(t.Holder)
	j.thisCopy = t.Copy == p.copy
	return j, nil
}

// derivedWhy says what released a subject outside its own tip: a release's
// tag on origin, or a record finished on origin's default branch.
func derivedWhy(id string) string {
	if isRelease(id) {
		return releaseTag(id) + " is tagged on " + Remote
	}
	return id + " is finished on " + Remote + "'s default branch"
}

// terminalFolders are the folders whose membership finishes a record: an
// issue resolved or wontfix, an intent shipped or superseded.
var terminalFolders = map[string][]string{
	"iss": {"resolved", "wontfix"},
	"itd": {intent.BucketShipped, intent.BucketSuperseded},
}

// originDefault is origin's default branch as this checkout last fetched it,
// a ref under refs/remotes/origin/, or "".
func originDefault(root string) string {
	trace("symbolic-ref", "--quiet", "refs/remotes/"+Remote+"/HEAD")
	return gitutil.TrackedDefaultBranch(root, Remote)
}

// remoteDefault asks origin which branch is its default, for a clone that
// names none (no origin/HEAD, and no conventional branch name fetched), and
// returns that branch's remote-tracking ref, refs/remotes/origin/<branch>. It
// asks with `git ls-remote --symref origin HEAD`, which reads the remote's
// advertisement and writes nothing here: origin/HEAD stays as the repository
// has it, unset. An answer that names no branch, or a branch whose name is
// not a valid ref, is an error, never a guess. A branch literally named HEAD
// names no branch: its remote-tracking ref would be origin/HEAD itself.
func remoteDefault(root string) (string, error) {
	args := []string{"ls-remote", "--symref", Remote, "HEAD"}
	trace(args...)
	out, err := gitutil.Net(root, args...)
	if err != nil {
		return "", errors.New(redacted(err))
	}
	for _, line := range strings.Split(out, "\n") {
		rest, ok := strings.CutPrefix(line, "ref: ")
		if !ok {
			continue
		}
		target, name, ok := strings.Cut(rest, "\t")
		if !ok || name != "HEAD" {
			continue
		}
		branch, ok := strings.CutPrefix(target, "refs/heads/")
		if !ok || branch == "" || branch == "HEAD" {
			return "", errors.New("origin's HEAD does not name a branch")
		}
		if _, err := localGit(root, 4096, "check-ref-format", "refs/heads/"+branch); err != nil {
			return "", errors.New("origin's HEAD names a branch whose name is not a valid ref")
		}
		return "refs/remotes/" + Remote + "/" + branch, nil
	}
	return "", errors.New("origin did not name a default branch")
}

// originPresent refuses a checkout with no remote named origin, naming the
// remotes it has; what names what lives there ("reservations").
func originPresent(root, what string) error {
	out, err := localGit(root, 64<<10, "remote")
	if err != nil {
		return refuse("could not list this checkout's remotes: %s; %s", redacted(err), nothingPushed)
	}
	names := strings.Fields(out)
	if slices.Contains(names, Remote) {
		return nil
	}
	if len(names) == 0 {
		return refuse("%s live on the remote named %s, and this checkout has no remote; %s", what, Remote, nothingPushed)
	}
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = echoName(n)
	}
	return refuse("%s live on the remote named %s, and this checkout's remotes are: %s; %s",
		what, Remote, strings.Join(quoted, ", "), nothingPushed)
}

// echoName renders a remote's name bare when it is a plain name, quoted
// otherwise.
func echoName(n string) string {
	for _, r := range n {
		if !(r == '-' || r == '_' || r == '.' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return echo(n)
		}
	}
	return n
}

// fetch reads every branch of each namespace from origin, with pruning, and
// the default branch the derived release reads, in one command, then stamps
// each namespace with when it did. It is the only read of the remote an act
// makes before it judges. outcome ends a refusal, saying what the act did not
// do ("no reservation was made").
//
// The default branch is the one this clone last knew (origin/HEAD's target),
// and that name can be stale: a remote whose main was renamed refuses the whole
// command for want of the one ref. So when the combined fetch fails, the
// namespaces are fetched alone, and only the default branch is lost: gap is
// why it could not be read, and the act goes on without it. A fetch of the
// namespaces themselves that fails refuses the act.
//
// The namespaces are stamped in the order given, and a stamp that cannot be
// written refuses the act after the fetch has landed: the namespaces stamped
// before it keep their stamps, each true, and it and those after it keep the
// stamp of their last fetch that recorded one.
func fetch(root string, now time.Time, outcome string, spaces ...namespace) (gap string, err error) {
	return fetchWith(root, now, outcome, originDefault(root), spaces...)
}

// fetchWith is fetch with the default branch's remote-tracking ref named by
// the caller: ref is refs/remotes/origin/<branch>, or "" to fetch the
// namespaces alone.
func fetchWith(root string, now time.Time, outcome, ref string, spaces ...namespace) (gap string, err error) {
	if len(spaces) == 0 {
		return "", refuse("a fetch names no namespace; %s", outcome)
	}
	// A namespace that names each branch (only) may name thousands: those
	// refspecs travel on stdin (`git fetch --stdin`), never on the command
	// line, which the system's argument limit bounds. A globbed refspec stays
	// on the command line.
	var listed []string
	for _, ns := range spaces {
		if ns.only != nil {
			listed = append(listed, ns.refspecs()...)
		}
	}
	args := []string{"fetch", "--prune", "--no-tags", Remote}
	if len(listed) > 0 {
		args = []string{"fetch", "--prune", "--no-tags", "--stdin", Remote}
	}
	bare := len(args)
	for _, ns := range spaces {
		if ns.only == nil {
			args = append(args, ns.refspecs()...)
		}
		args = append(args, ns.also...)
	}
	stdin := ""
	if len(listed) > 0 {
		stdin = strings.Join(listed, "\n") + "\n"
	}
	withDefault := args
	if ref != "" {
		branch := strings.TrimPrefix(ref, "refs/remotes/"+Remote+"/")
		withDefault = append(append([]string(nil), args...), "+refs/heads/"+branch+":"+ref)
	}
	// A fetch naming no refspec at all (a listed namespace origin holds
	// nothing of, and no default branch) is not run: a bare `git fetch
	// origin` would fetch whatever this clone is configured to.
	var ferr error
	if len(listed) > 0 {
		ferr = stdinFetchFloor()
	}
	if ferr == nil {
		if len(withDefault) > bare || len(listed) > 0 {
			trace(withDefault...)
			_, ferr = gitutil.NetInput(root, stdin, withDefault...)
		}
		if ferr != nil && ref != "" {
			gap = "could not read " + gitutil.ShortRef(ref) + " from " + Remote + ": " + redacted(ferr)
			ferr = nil
			if len(args) > bare || len(listed) > 0 {
				trace(args...)
				_, ferr = gitutil.NetInput(root, stdin, args...)
			}
		}
	}
	if ferr != nil {
		// Each namespace keeps that its fetch failed, and why, for every
		// reader to say rather than show the last view as current (C10).
		for _, ns := range spaces {
			noteFetchFailure(root, ns, now, redacted(ferr))
		}
		e := &FetchError{Reason: redacted(ferr)}
		if what := nouns(spaces); what != reservations.noun || outcome != noReservation {
			e.what, e.outcome = what, outcome
		}
		return "", e
	}
	for _, ns := range spaces {
		if err := stampFetch(root, ns, now); err != nil {
			noteFetchFailure(root, ns, now, "fetched, but could not record when: "+redacted(err))
			return "", refuse("fetched %s, but could not record when: %s; %s", ns.noun, redacted(err), outcome)
		}
		clearFetchFailure(root, ns)
	}
	return gap, nil
}

// stdinFetchFloor refuses a fetch that reads its refspecs on standard input
// (`git fetch --stdin`) on a git older than 2.29, which first read them
// there, in fixed words naming the version: nil when git is at the floor.
func stdinFetchFloor() error {
	ok, version := gitutil.GitAtLeast(2, 29)
	switch {
	case ok:
		return nil
	case version == "":
		return errors.New("`git fetch --stdin`, which the status fetch reads its branches on, needs git 2.29 or later, and this git's version could not be read")
	}
	return fmt.Errorf("`git fetch --stdin`, which the status fetch reads its branches on, needs git 2.29 or later, and this git is %s", version)
}

// FetchError is a failed fetch from origin: Reason is git's own,
// home-redacted, which a caller quotes in its own refusal (the land fence). It
// is an ErrRefused refusal: nothing was made. The reservations' fetch names
// them, and every other fetch names what it read.
type FetchError struct {
	Reason string

	// what is what the fetch read and outcome what was not done; "" for the
	// reservations' fetch, whose words are fixed.
	what, outcome string
}

func (e *FetchError) Error() string {
	what, outcome := e.what, e.outcome
	if what == "" {
		what, outcome = reservations.noun, noReservation
	}
	return fmt.Sprintf("could not read %s from %s: %s; %s", what, Remote, e.Reason, outcome)
}
func (e *FetchError) Is(target error) bool { return target == ErrRefused }

// PushError is a reservation push origin refused, for a reason other than a
// tip the act cannot take: Reason is git's own rejection, home-redacted. It is
// an ErrRefused refusal: no reservation was made.
type PushError struct {
	Reason string
}

func (e *PushError) Error() string {
	return fmt.Sprintf("%s refused the reservation push: %s; %s", Remote, e.Reason, noReservation)
}
func (e *PushError) Is(target error) bool { return target == ErrRefused }

// HasOrigin reports whether the checkout at root has a remote named origin,
// the one reservations live on, reading only its configuration.
func HasOrigin(root string) (bool, error) {
	out, err := localGit(root, 64<<10, "remote")
	if err != nil {
		return false, fmt.Errorf("listing this checkout's remotes: %s", redacted(err))
	}
	return slices.Contains(strings.Fields(out), Remote), nil
}

// stampFetch records when this checkout last fetched ns, after a fetch that
// succeeded: the view's age is counted from it, and a view with no stamp is
// one nothing here fetched.
//
// A checkout without its local tier keeps no stamp (the tier is never created
// on the way to a reservation); an act has already refused such a checkout,
// and a listing's view then has no age.
func stampFetch(root string, ns namespace, at time.Time) error {
	if tierPresent(root) != nil {
		return nil
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := mkStateDir(r, ns.stateDir); err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(r, ns.stampRel(), []byte(stamp(at)+"\n"), stateMode)
}
