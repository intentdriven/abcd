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
	// Unbacked is why a live reservation was not renewed for want of live
	// work behind it ("" when work stands behind it): it lapses on its lease.
	Unbacked string
}

// LiveWork answers whether live work stands behind a reservation whose
// holding worktree is on branch: ok when it does, otherwise why not, as a
// clause. The heartbeat answers it from the forge (a pull request from the
// branch that is open, armed or queued); the core never asks a forge itself.
type LiveWork func(branch string) (ok bool, why string)

// RenewHeld is the heartbeat's renewal (iss-2611220942554289): every one of
// the person's own reservations of an issue or an intent held from any
// worktree of the checkout at root, as last fetched (View), each renewed when
// it is live, half through its lease and backed by live work, with its own
// term (RenewalTerm), through the renewal from any worktree
// (Request.AnyWorktree, Request.Live). Live work is what live answers for the
// branch of the worktree holding it; one it does not answer for (abandoned,
// or a lane mid-work whose commits have no pull request yet), one on no
// branch, and every one when live is nil is listed and not renewed, with why
// (HeldRenewal.Unbacked), so it lapses on its lease. A lane's own loop renews
// its reservations itself. So a run renews the reservations its pull
// requests stand on from one place, and the worktree that took each keeps it. A reservation held from another clone or
// computer, another person's, a released one and a release subject are left
// out; a lapsed one is listed, never revived. Like RenewDue it renews only a
// tip the holding worktree itself last pushed (pushedHere), answering
// errNotPushedHere for any other, and holds this copy's renewal lock over the
// pass. The error is a view that could not be read, or ErrRenewalBusy; each
// renewal's own failure is in its HeldRenewal. They come in record order.
func RenewHeld(root string, live LiveWork, o Options) ([]HeldRenewal, error) {
	r, err := os.OpenRoot(root)
	if err == nil {
		defer r.Close()
		err = mkStateDir(r, stateRelDir)
	}
	if err != nil {
		return renewHeld(root, live, o)
	}
	var out []HeldRenewal
	ran := false
	err = fsutil.WithFileLockIn(r, renewLockRel, renewLockWait, func() error {
		ran = true
		var verr error
		out, verr = renewHeld(root, live, o)
		return verr
	})
	switch {
	case ran:
		return out, err
	case errors.Is(err, fsutil.ErrLockContention):
		return nil, fmt.Errorf("%w for over %s", ErrRenewalBusy, renewLockWait)
	}
	return renewHeld(root, live, o)
}

// renewHeld is one pass of RenewHeld, under its lock.
func renewHeld(root string, live LiveWork, o Options) ([]HeldRenewal, error) {
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
		if res.State == StateLive {
			h.Unbacked = unbacked(at, live)
		}
		if res.State == StateLive && h.Unbacked == "" && HalfThrough(res, now) {
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

// unbacked is why no live work stands behind a reservation held from the
// worktrees at (each worktree's branch asked of live), "" when one of them is
// on a branch live answers for.
func unbacked(at []string, live LiveWork) string {
	if live == nil {
		return "nothing was asked whether work stands behind it"
	}
	why := "the worktree holding it is on no branch, so no pull request stands behind it"
	for _, dir := range at {
		b := thisBranch(dir)
		if b == "" {
			continue
		}
		ok, w := live(b)
		if ok {
			return ""
		}
		if w == "" {
			w = "no live work stands behind its branch " + b
		}
		why = w
	}
	return why
}
