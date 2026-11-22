package claim

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// HeldRenewal is one of the person's own reservations RenewHeld found held
// from this checkout: Record, the worktree holding it (HeldFrom, "" for this
// one), whether it is live or lapsed (State), whether this pass renewed it,
// the lease's end after the pass (Until), and why a renewal failed (Err).
type HeldRenewal struct {
	Record   string
	HeldFrom string
	State    string
	Renewed  bool
	Until    time.Time
	Err      error
}

// RenewHeld is the heartbeat's renewal (iss-2611220942554289): every one of
// the person's own reservations of an issue or an intent held from any
// worktree of the checkout at root, as last fetched (View), each renewed when
// it is live and half through its lease, with its own term (RenewalTerm),
// through the renewal from any worktree (Request.AnyWorktree, Request.Live).
// So a run renews the reservations its lanes took from one place, and the
// worktree that took each keeps it. A reservation held from another clone or
// computer, another person's, a released one and a release subject are left
// out; a lapsed one is listed, never revived. Like RenewDue it renews only a
// tip the holding worktree itself last pushed (pushedHere), answering
// errNotPushedHere for any other, and holds this copy's renewal lock over the
// pass. The error is a view that could not be read, or ErrRenewalBusy; each
// renewal's own failure is in its HeldRenewal. They come in record order.
func RenewHeld(root string, o Options) ([]HeldRenewal, error) {
	r, err := os.OpenRoot(root)
	if err == nil {
		defer r.Close()
		err = mkStateDir(r, stateRelDir)
	}
	if err != nil {
		return renewHeld(root, o)
	}
	var out []HeldRenewal
	ran := false
	err = fsutil.WithFileLockIn(r, renewLockRel, renewLockWait, func() error {
		ran = true
		var verr error
		out, verr = renewHeld(root, o)
		return verr
	})
	switch {
	case ran:
		return out, err
	case errors.Is(err, fsutil.ErrLockContention):
		return nil, fmt.Errorf("%w for over %s", ErrRenewalBusy, renewLockWait)
	}
	return renewHeld(root, o)
}

// renewHeld is one pass of RenewHeld, under its lock.
func renewHeld(root string, o Options) ([]HeldRenewal, error) {
	now := o.now()
	snap, err := View(root, now)
	if err != nil {
		return nil, err
	}
	self := readerSelf(root, o)
	var out []HeldRenewal
	for _, res := range snap.Reservations {
		if isRelease(res.Record) || !self.Owns(res.Holder) || res.State != StateLive && res.State != StateExpired {
			continue
		}
		at := []string{root}
		h := HeldRenewal{Record: res.Record, State: res.State, Until: res.EffectiveExpiry}
		if res.Copy != CopyThis {
			if at = siblingRoots(root, res.key); len(at) == 0 {
				continue
			}
			h.HeldFrom = yourCopyAt(siblingCopy(root, res.key))
		}
		if res.State == StateLive && HalfThrough(res, now) {
			if !slices.ContainsFunc(at, func(dir string) bool { return pushedHere(dir, res.Record) == res.Commit }) {
				h.Err = errNotPushedHere
			} else {
				got, err := Reserve(root, Request{Record: res.Record, Renew: true, Live: true, AnyWorktree: true, Lease: RenewalTerm(res)}, o)
				if h.Err = err; err == nil {
					h.Renewed, h.Until = true, got.ExpiresAt
				}
			}
		}
		out = append(out, h)
	}
	slices.SortFunc(out, func(a, b HeldRenewal) int { return strings.Compare(a.Record, b.Record) })
	return out, nil
}
