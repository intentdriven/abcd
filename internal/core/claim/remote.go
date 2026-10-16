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
	// tombstone or nothing instead: origin's default branch holds no one
	// record of it to name, the record changed since the hand-back was judged,
	// or the tombstone with the hand-back would pass the message cap. "" when
	// the hand-back was published, or none was due.
	HandBackUnpublished string
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
}

// beforePush, when a test sets it, runs between the judgement and the push:
// the window a second writer's push lands in when two race.
var beforePush func(root string)

// judged is a tip read from the view just fetched, with what decides an act.
type judged struct {
	tip
	present bool
	// derived is true when the record is terminal on the default branch as
	// fetched, so its reservation counts as released whatever its expiry.
	derived bool
	// blob is the record's blob on the default branch as last fetched, "" when
	// it holds the record in no file or several, or none can be read: what a
	// hand-back's release names, and what a standing hand-back is judged by.
	blob string
	// blobUnread is why origin's default branch could not be read for the
	// blob at all (this checkout names none of it, or its tree would not
	// read), "" when it was read, whatever it held.
	blobUnread string
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

// Reserve performs one act against origin: fetch, judge, write one commit,
// then one plain push, which is the compare-and-swap. A release pushes the
// tombstone; every other request reserves, renews or takes over, by what the
// tip it judged allows. A refusal pushes nothing; a failure leaves nothing
// locally but an unreferenced commit git's collection removes.
func Reserve(root string, req Request, o Options) (Result, error) {
	p, err := prepare(root, req, o, originPresent)
	if err != nil {
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
	j, err := judge(p, tree)
	if err != nil {
		return Result{}, err
	}
	res, m, parent, err := decide(p, j)
	if err != nil || res.Act == ActAlreadyReleased {
		if err == nil {
			res.DerivedUnchecked = p.derivedGap
		}
		return res, err
	}
	if !p.req.Release {
		if err := crowding(p, j, res); err != nil {
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
	if !p.req.Release && (res.Act != ActRenewed || p.req.Drain && !res.RenewedLive) {
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
	res.Commit = sha
	res.Copy = CopyThis
	res.DerivedUnchecked = p.derivedGap
	res.ForkUnchecked = forkUnchecked
	res.ForkAllowed = forkAllowed
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
		return Result{}, message{}, "", backOff("%s is finished on %s's default branch, so there is nothing to reserve; take other work; %s",
			p.id, Remote, nothingPushed)
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
		return res, m, j.Commit, nil
	}
	exp := j.verdict.EffectiveExpiry
	lapsed := !now.Before(exp.Add(TakeoverMargin))
	if j.mine {
		if !p.req.TakeOver && !lapsed {
			return Result{}, message{}, "", backOff("you are already building %s on another computer, until %s; pass --take-over to continue here",
				p.id, stamp(exp))
		}
		res.Act = ActTakenOver
		res.Replaced = &Replaced{Holder: j.Holder, Copy: CopyAnother, Commit: j.Commit, EffectiveExpiry: exp}
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
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldAnotherCopy, ErrRefused,
			fmt.Sprintf("%s is held by your copy on another computer; release it there, or take it over here first; %s", p.id, nothingPushed))
	case p.req.Tip != "" && j.Commit != p.req.Tip:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldMoved, ErrRefused,
			fmt.Sprintf("the reservation on %s is no longer commit %s: it was renewed or reserved again since, so it is left as it stands; %s",
				p.id, p.req.Tip[:12], nothingPushed))
	}
	m := p.tombstone(j.Footprint)
	res := Result{Act: ActReleased, Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt,
		ExpiresAt: m.ExpiresAt}
	if p.req.HandBack != "" {
		res.HandBackUnpublished = p.markHandBack(&m, j)
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
// returns why it named none instead: no one record of it there to name, or a
// record other than the one the hand-back was judged over (HandBackBlob).
func (p *prepared) markHandBack(m *message, j judged) string {
	switch {
	case j.blob == "" && j.blobUnread != "":
		return fmt.Sprintf("%s's default branch could not be read (%s), so the tombstone marks no hand-back", Remote, j.blobUnread)
	case j.blob == "":
		return fmt.Sprintf("%s's default branch as last fetched holds no one record of %s, so the tombstone marks no hand-back", Remote, p.id)
	case p.req.HandBackBlob != "" && p.req.HandBackBlob != j.blob:
		return fmt.Sprintf("%s's record on %s's default branch has changed since the hand-back was judged, so the tombstone marks no hand-back: a person's edit re-admits it",
			p.id, Remote)
	}
	m.HandedBack, m.RecordBlob = p.req.HandBack, j.blob
	return ""
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
	if why := p.markHandBack(&m, j); why != "" {
		done.HandBackUnpublished = why
		return done, message{}, "", nil
	}
	return Result{Act: ActReleased, Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt,
		ExpiresAt: m.ExpiresAt}, m, j.Commit, nil
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
			fmt.Sprintf("the reservation on %s is released: %s is finished on %s's default branch; %s", p.id, p.id, Remote, nothingPushed))
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
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldAnotherCopy, ErrBackOff,
			fmt.Sprintf("%s is held by your copy on another computer, until %s; %s", p.id, stamp(exp), nothingPushed))
	case p.req.Live && !j.verdict.Live:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldLapsed, ErrBackOff,
			fmt.Sprintf("your reservation on %s lapsed at %s; %s", p.id, stamp(exp), nothingPushed))
	}
	p.req.Footprint = append([]string(nil), j.Footprint...)
	m := p.reservation()
	return Result{Act: ActRenewed, Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt,
		ExpiresAt: m.ExpiresAt}, m, j.Commit, nil
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
	// The default branch is read whether or not a tip exists, since a
	// finished record is not reserved either way. One the fetch could not
	// read derives nothing: the copy this clone holds may be stale. Its blob
	// of the record is still read as last fetched, when it reads, and a
	// stale branch reads the record as it was when last fetched: a hand-back
	// made over a newer blob than that reads as changed though no one edited
	// it since, and a person's edit this copy has not fetched reads as
	// unchanged until it is (dcn-2610161412283450, refined by
	// dcn-2610161837009957). One with no blob read stands.
	recs, err := defaultRecords(p.root)
	if err != nil && p.derivedGap == "" {
		return judged{}, refuse("could not read whether %s is finished on the default branch: %s; %s",
			p.id, redacted(err), noReservation)
	}
	switch {
	case originDefault(p.root) == "":
		j.blobUnread = "this checkout names no default branch of " + Remote
	case err != nil:
		j.blobUnread = p.derivedGap + "; " + redacted(err)
	}
	j.derived = p.derivedGap == "" && terminal(recs[p.id], p.id)
	j.blob = blobOf(recs[p.id])
	if !ok {
		return j, nil
	}
	if t.Unreadable != "" {
		j.verdict = lease.Judge(lease.Stamp{Unreadable: true, WrittenAt: t.WrittenAt, FirstSeen: t.FirstSeen}, lease.Default, p.now)
		return j, nil
	}
	j.verdict = lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, p.now)
	j.mine = sameHolder(t.Holder, p.holder)
	j.thisCopy = t.Copy == p.copy
	return j, nil
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
	ref := gitutil.DefaultRef(root)
	if !strings.HasPrefix(ref, "refs/remotes/"+Remote+"/") {
		return ""
	}
	return ref
}

// originPresent refuses a checkout with no remote named origin, naming the
// remotes it has.
func originPresent(root string) error {
	out, err := localGit(root, 64<<10, "remote")
	if err != nil {
		return refuse("could not list this checkout's remotes: %s; %s", redacted(err), nothingPushed)
	}
	names := strings.Fields(out)
	if slices.Contains(names, Remote) {
		return nil
	}
	if len(names) == 0 {
		return refuse("reservations live on the remote named %s, and this checkout has no remote; %s", Remote, nothingPushed)
	}
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = echoName(n)
	}
	return refuse("reservations live on the remote named %s, and this checkout's remotes are: %s; %s",
		Remote, strings.Join(quoted, ", "), nothingPushed)
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
	if len(spaces) == 0 {
		return "", refuse("a fetch names no namespace; %s", outcome)
	}
	args := []string{"fetch", "--prune", "--no-tags", Remote}
	for _, ns := range spaces {
		args = append(args, ns.refspec())
	}
	ref := originDefault(root)
	withDefault := args
	if ref != "" {
		branch := strings.TrimPrefix(ref, "refs/remotes/"+Remote+"/")
		withDefault = append(append([]string(nil), args...), "+refs/heads/"+branch+":"+ref)
	}
	trace(withDefault...)
	_, ferr := gitutil.Net(root, withDefault...)
	if ferr != nil && ref != "" {
		gap = "could not read " + gitutil.ShortRef(ref) + " from " + Remote + ": " + redacted(ferr)
		trace(args...)
		_, ferr = gitutil.Net(root, args...)
	}
	if ferr != nil {
		e := &FetchError{Reason: redacted(ferr)}
		if what := nouns(spaces); what != reservations.noun || outcome != noReservation {
			e.what, e.outcome = what, outcome
		}
		return "", e
	}
	for _, ns := range spaces {
		if err := stampFetch(root, ns, now); err != nil {
			return "", refuse("fetched %s, but could not record when: %s; %s", ns.noun, redacted(err), outcome)
		}
	}
	return gap, nil
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
