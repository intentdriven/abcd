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
// that has lapsed. A renewal keeps the tip's own term (renewalTerm). A
// release subject is left alone: its hold is a release cut's, kept by the
// cut. Only a reservation whose tip is the commit this copy itself last
// pushed for it (pushedHere) is renewed; any other tip is left as it is, and
// answered with errNotPushedHere.
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
		if r.Copy != CopyThis || r.State != StateLive || isRelease(r.Record) || !halfThrough(r, now) {
			continue
		}
		if pushedHere(root, r.Record) != r.Commit {
			out = append(out, Renewal{Record: r.Record, Err: errNotPushedHere})
			continue
		}
		res, err := Reserve(root, Request{Record: r.Record, Renew: true, Live: true, Lease: renewalTerm(r)}, o)
		out = append(out, Renewal{Record: r.Record, Until: res.ExpiresAt, Err: err})
	}
	return out, nil
}

// halfThrough reports whether r is half through its lease at now, counted from
// its claimed-at to its effective expiry: the point a renewal is due.
func halfThrough(r Reservation, now time.Time) bool {
	return !now.Before(r.ClaimedAt.Add(r.EffectiveExpiry.Sub(r.ClaimedAt) / 2))
}

// renewalTerm is the lease a renewal of r pushes: r's own term, never shorter
// than the default nor longer than a day, so a renewal never cuts a longer
// term a fence gave a pull request waiting on a person.
func renewalTerm(r Reservation) time.Duration {
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
	if _, err := reservableID(id); err != nil {
		return ""
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer r.Close()
	data, err := localtier.ReadInRoot(r, pushedRelDir+"/"+id, 128)
	if err != nil {
		return ""
	}
	if sha := strings.TrimSpace(string(data)); gitutil.IsFullSHA(sha) {
		return sha
	}
	return ""
}
