package claim

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// errNotPushedHere is a renewal's answer for a reservation whose tip is not
// the commit this copy last pushed for it.
var errNotPushedHere = errors.New("its latest commit is not one this copy pushed, so it is not renewed from here; " +
	"renew it with abcd reserve only if you reserved it from this copy (abcd peers names its holder and expiry)")

// pushedRelDir keeps, per record, the last reservation commit this copy
// pushed for it (notePushed): the commit names RenewDue renews from.
const pushedRelDir = stateRelDir + "/pushed"

// renewLockRel is the lock RenewDue holds over its whole pass in a copy, from
// the view it reads to its last push, so two waiters in one copy take turns: the
// second reads the tips the first has just pushed and finds nothing due, where
// unlocked it judged from the tips both read, lost each push to the first, and
// reported every loss as a fault (iss-2610270659564789).
const renewLockRel = stateRelDir + "/renew.lock"

// renewLockWait bounds how long a pass waits for a sibling's to end. A pass is
// usually a few seconds of pushes; a sibling holding the lock past this is slow
// (a stalled network) or stopped, and either way this pass renews nothing and
// answers ErrRenewalBusy, so a due reservation never lapses unsaid.
var renewLockWait = 2 * time.Minute

// ErrRenewalBusy is RenewDue's answer when another pass in the same copy held
// the renewal lock for all of renewLockWait, and this pass renewed nothing.
var ErrRenewalBusy = errors.New("another renewal pass in this copy has held its lock")

// Renewal is one reservation RenewDue renewed, or failed to: Until is the
// renewed lease's end, and Err why it could not be renewed.
type Renewal struct {
	Record string
	Until  time.Time
	Err    error
}

// RenewDue renews this person's own live reservations from this copy that
// are half through their lease, and nothing else: the renewal a holder
// waiting at the load gate gets, so a reservation made by hand does not lapse
// while its preflight waits its turn (iss-2610170926295770). It reads the
// reservations as last fetched (View), which asks the network nothing, and
// each renewal is a renew-only, live-only act (Request.Renew, Request.Live):
// it never reserves a record anew, never takes one over and never revives one
// that has lapsed. A renewal keeps the tip's own term (RenewalTerm). A
// release subject is left alone: its hold is a release cut's, kept by the
// cut. Only a reservation whose tip is the commit this copy itself last
// pushed for it (pushedHere) is renewed; any other tip is left as it is, and
// answered with errNotPushedHere. A tip a drive left to lapse (LeaveToLapse)
// is left as it is too, and answered with nothing: it is no one's work.
// Each pass holds the copy's renewal lock (renewLockRel); a copy whose lock
// cannot be laid renews unlocked rather than not at all.
// The error is a view that could not be read, or ErrRenewalBusy; each
// renewal's own failure is in its Renewal.
func RenewDue(root string, o Options) ([]Renewal, error) {
	r, err := os.OpenRoot(root)
	if err == nil {
		defer r.Close()
		err = mkStateDir(r, stateRelDir)
	}
	if err != nil {
		return renewDue(root, o)
	}
	var out []Renewal
	ran := false
	err = fsutil.WithFileLockIn(r, renewLockRel, renewLockWait, func() error {
		ran = true
		var verr error
		out, verr = renewDue(root, o)
		return verr
	})
	switch {
	case ran:
		return out, err
	case errors.Is(err, fsutil.ErrLockContention):
		return nil, fmt.Errorf("%w for over %s", ErrRenewalBusy, renewLockWait)
	}
	return renewDue(root, o)
}

// renewDue is one renewal pass, RenewDue's body under its lock.
func renewDue(root string, o Options) ([]Renewal, error) {
	now := o.now()
	snap, err := View(root, now)
	if err != nil {
		return nil, err
	}
	var out []Renewal
	for _, r := range snap.Reservations {
		if r.Copy != CopyThis || r.State != StateLive || isRelease(r.Record) || !HalfThrough(r, now) {
			continue
		}
		if pushedHere(root, r.Record) != r.Commit {
			out = append(out, Renewal{Record: r.Record, Err: errNotPushedHere})
			continue
		}
		// A hold a drive left to lapse at this tip is no one's work now, so a
		// renewal that reads no run leaves it to its lease (LeaveToLapse).
		if LeftToLapse(root, r.Record, r.Commit) {
			continue
		}
		res, err := Reserve(root, Request{Record: r.Record, Renew: true, Live: true, Lease: RenewalTerm(r)}, o)
		out = append(out, Renewal{Record: r.Record, Until: res.ExpiresAt, Err: err})
	}
	return out, nil
}

// HalfThrough reports whether r is half through its lease at now, counted from
// its claimed-at to its effective expiry: the point a renewal is due. It and
// RenewalTerm are exported so the implement loop's renewals call them rather
// than keep copies that could drift from the waiter's (iss-2610171047088861).
func HalfThrough(r Reservation, now time.Time) bool {
	return !now.Before(HalfThroughAt(r))
}

// HalfThroughAt is the instant r is half through its lease, counted from its
// claimed-at to its effective expiry: the point a renewal is due, from which
// HalfThrough answers. It is exported so the owed-read's next renewal and wake
// time take the same point rather than keep a copy that could drift from it
// (iss-2611141342176969).
func HalfThroughAt(r Reservation) time.Time {
	return r.ClaimedAt.Add(r.EffectiveExpiry.Sub(r.ClaimedAt) / 2)
}

// RenewalTerm is the lease a renewal of r pushes: r's own term, never shorter
// than the default nor longer than a day, so a renewal never cuts a longer
// term a fence gave a pull request waiting on a person.
func RenewalTerm(r Reservation) time.Duration {
	return min(max(lease.Default, r.EffectiveExpiry.Sub(r.ClaimedAt)), lease.Max)
}

// notePushed keeps sha as the commit this copy last pushed for id, best
// effort: a note that cannot be written only leaves that reservation out of
// RenewDue, to be renewed by hand.
func notePushed(root, id, sha string) {
	if _, err := reservableID(id); err != nil || !gitutil.IsFullSHA(sha) {
		return
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	if err := mkStateDir(r, pushedRelDir); err != nil {
		return
	}
	_ = fsutil.WriteFileAtomicInRoot(r, pushedRelDir+"/"+id, []byte(sha+"\n"), stateMode)
}

// pushedHere is the commit this copy last pushed for id (notePushed), "" when
// none is kept or the note cannot be read.
func pushedHere(root, id string) string {
	return readCommitNote(root, pushedRelDir, id)
}

// readCommitNote is the full commit id the note dir/id in root's state holds,
// "" when there is none, it cannot be read, or it holds anything else.
func readCommitNote(root, dir, id string) string {
	if _, err := reservableID(id); err != nil {
		return ""
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer r.Close()
	data, err := localtier.ReadInRoot(r, dir+"/"+id, 128)
	if err != nil {
		return ""
	}
	if sha := strings.TrimSpace(string(data)); gitutil.IsFullSHA(sha) {
		return sha
	}
	return ""
}

// lapseRelDir keeps, per record, the reservation commit a drive left to lapse
// in this copy (LeaveToLapse).
const lapseRelDir = stateRelDir + "/lapse"

// LeaveToLapse records that this copy leaves id's reservation, whose tip is
// commit, to lapse at its lease (itd-2611141610197753 step 3): a drive keeps
// the hold of a lane it is not working, one whose stop could not be applied or
// one it set aside, so no peer takes the issue while that is unsure, and no
// renewal that reads no driver keeps it alive past its lease. RenewDue, and the
// implement loop's renewals of a run (an `abcd implement status` call, a
// load-gate waiter, a drive's waits and a drain's move), leave such a hold
// alone. The record names the tip alone, so any reservation pushed for id since,
// a take-up's or a step's, ends it with nothing to undo. A note that cannot be
// written is the error, and leaves the hold renewed as before.
func LeaveToLapse(root, id, commit string) error {
	if _, err := reservableID(id); err != nil {
		return err
	}
	if !gitutil.IsFullSHA(commit) {
		return fmt.Errorf("%q is not a full commit id", commit)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := mkStateDir(r, lapseRelDir); err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(r, lapseRelDir+"/"+id, []byte(commit+"\n"), stateMode)
}

// LeftToLapse reports whether this copy left id's reservation to lapse at the
// tip commit (LeaveToLapse). A note that cannot be read is none.
func LeftToLapse(root, id, commit string) bool {
	return commit != "" && readCommitNote(root, lapseRelDir, id) == commit
}

// KeepRenewing undoes LeaveToLapse for id: the drive that left it takes the
// lane up again. A note already gone is no error.
func KeepRenewing(root, id string) error {
	if _, err := reservableID(id); err != nil {
		return err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := r.Remove(lapseRelDir + "/" + id); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
