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
	"github.com/intentdriven/abcd/internal/core/peers"
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
	// DerivedUnchecked is why origin's default branch could not be read, so
	// whether the record is already finished there was not checked; "" when
	// it was read (or origin has none).
	DerivedUnchecked string
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
	verdict lease.Verdict
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
	p, err := prepare(root, req, o)
	if err != nil {
		return Result{}, err
	}
	if err := originPresent(root); err != nil {
		return Result{}, err
	}
	tree, err := emptyTree(root)
	if err != nil {
		return Result{}, refuse("%s; %s", redacted(err), nothingPushed)
	}
	gap, err := fetch(root, p.now)
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
	sha, err := writeCommit(p, m, tree, parent)
	if err != nil {
		return Result{}, err
	}
	if beforePush != nil {
		beforePush(root)
	}
	push := pushArgs(sha, p.id)
	trace(push...)
	if out, perr := gitutil.Net(root, push...); perr != nil {
		return Result{}, lostRace(p, tree, j, pushRejection(out, perr))
	}
	res.Commit = sha
	res.Copy = CopyThis
	res.DerivedUnchecked = p.derivedGap
	return res, nil
}

// pushArgs is the one push every act makes: the commit to the record's
// reservation branch on origin, plain, with no force, no `+` refspec, no
// --no-verify and no deletion, so the repository's hooks run and the remote
// takes it only as a fast-forward.
func pushArgs(sha, id string) []string {
	return []string{"push", "--porcelain", Remote, sha + ":" + refName(id)}
}

// lostRace explains a rejected push. When the branch moved while the push was
// on its way and now holds a reservation this act cannot take, the refusal
// names its holder with the back-off; otherwise the push failure is reported
// as itself.
func lostRace(p *prepared, tree string, before judged, perr error) error {
	pushFailed := refuse("origin refused the reservation push: %s; %s", redacted(perr), noReservation)
	gap, err := fetch(p.root, p.now)
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
	// A renewal or a release that met another's tip says so in its own
	// terms, which its caller reads.
	var nh *NotHeld
	if errors.As(derr, &nh) {
		return derr
	}
	if errors.Is(derr, ErrBackOff) && j.Unreadable == "" && !j.mine {
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
	if p.req.Release {
		return decideRelease(p, j)
	}
	if p.req.Renew {
		return decideRenew(p, j)
	}
	m := p.reservation()
	res := Result{Record: p.id, Holder: p.holder, Footprint: m.Footprint, ClaimedAt: m.ClaimedAt, ExpiresAt: m.ExpiresAt}
	now := p.now
	switch {
	case !j.present:
		res.Act = ActReserved
		return res, m, "", nil
	case j.released():
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
		res.Act = ActRenewed
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
		copyOf := CopyAnother
		if j.mine && j.thisCopy {
			copyOf = CopyThis
		}
		return Result{Act: ActAlreadyReleased, Record: p.id, Holder: j.Holder, Copy: copyOf, Footprint: j.Footprint,
			ClaimedAt: j.ClaimedAt, ExpiresAt: j.ExpiresAt, Commit: j.Commit}, message{}, "", nil
	case !j.mine:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldAnotherPerson, ErrRefused,
			fmt.Sprintf("%s is reserved by %s, not by you; %s", p.id, j.Holder, nothingPushed))
	case !j.thisCopy:
		return Result{}, message{}, "", j.notHeld(p.id, NotHeldAnotherCopy, ErrRefused,
			fmt.Sprintf("%s is held by your copy on another computer; release it there, or take it over here first; %s", p.id, nothingPushed))
	}
	m := p.tombstone(j.Footprint)
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
	t, ok, err := readTip(p.root, p.id, trackingName(p.id), tree, p.now, true)
	if err != nil {
		return judged{}, refuse("could not read the reservation on %s: %s; %s", p.id, redacted(err), noReservation)
	}
	j := judged{tip: t, present: ok}
	if !ok {
		return j, nil
	}
	// A default branch the fetch could not read derives nothing: the copy
	// this clone holds may be stale, so it is not read.
	if p.derivedGap == "" {
		j.derived, err = terminalOnDefault(p.root, p.id)
	}
	if err != nil {
		return judged{}, refuse("could not read whether %s is finished on the default branch: %s; %s",
			p.id, redacted(err), noReservation)
	}
	if t.Unreadable != "" {
		j.verdict = lease.Judge(lease.Stamp{Unreadable: true, WrittenAt: t.WrittenAt}, lease.Default, p.now)
		return j, nil
	}
	j.verdict = lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt}, lease.Default, p.now)
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

// terminalOnDefault reports whether id is in a terminal folder on origin's
// default branch as last fetched. With no default branch on origin, nothing
// is derived.
func terminalOnDefault(root, id string) (bool, error) {
	ref := originDefault(root)
	if ref == "" {
		return false, nil
	}
	folders, err := peers.FoldersAt(root, ref)
	if err != nil {
		return false, err
	}
	return terminal(folders, id), nil
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

// fetch reads every reservation branch from origin, with pruning, and the
// default branch the derived release reads, in one command, then stamps when
// it did. It is the only read of the remote an act makes before it judges.
//
// The default branch is the one this clone last knew (origin/HEAD's target),
// and that name can be stale: a remote whose main was renamed refuses the whole
// command for want of the one ref. So when the combined fetch fails, the
// reservations are fetched alone, and only the derived release is lost: gap is
// why the default branch could not be read, and the act goes on without it. A
// fetch of the reservations themselves that fails refuses the act.
func fetch(root string, now time.Time) (gap string, err error) {
	args := []string{"fetch", "--prune", "--no-tags", Remote, "+" + RefPrefix + "*:" + TrackingPrefix + "*"}
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
		return "", &FetchError{Reason: redacted(ferr)}
	}
	if err := stampFetch(root, now); err != nil {
		return "", refuse("fetched the reservations, but could not record when: %s; %s", redacted(err), noReservation)
	}
	return gap, nil
}

// FetchError is a failed fetch of the reservations from origin: Reason is
// git's own, home-redacted, which a caller quotes in its own refusal (the land
// fence). It is an ErrRefused refusal: no reservation was made.
type FetchError struct {
	Reason string
}

func (e *FetchError) Error() string {
	return fmt.Sprintf("could not read the reservations from %s: %s; %s", Remote, e.Reason, noReservation)
}
func (e *FetchError) Is(target error) bool { return target == ErrRefused }

// HasOrigin reports whether the checkout at root has a remote named origin,
// the one reservations live on, reading only its configuration.
func HasOrigin(root string) (bool, error) {
	out, err := localGit(root, 64<<10, "remote")
	if err != nil {
		return false, fmt.Errorf("listing this checkout's remotes: %s", redacted(err))
	}
	return slices.Contains(strings.Fields(out), Remote), nil
}

// stampFetch records when this checkout last fetched the reservations, after a
// fetch that succeeded: the view's age is counted from it, and a view with no
// stamp is one nothing here fetched.
//
// A checkout without its local tier keeps no stamp (the tier is never created
// on the way to a reservation); an act has already refused such a checkout,
// and a listing's view then has no age.
func stampFetch(root string, at time.Time) error {
	if tierPresent(root) != nil {
		return nil
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := r.Mkdir(stateRelDir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(r, fetchedAtRel, []byte(stamp(at)+"\n"), stateMode)
}
