package loop

// renewruns.go renews a run's reservation wherever its agent waits
// (iss-2610261837062463, dcn-2610291352512058). Only a step, a receipt and a
// drain's move renewed one, so a run whose implementer held its stage for
// hours, its go test and make preflight queued and running under the
// machine's load gate, had its reservation lapse with no step to renew it, and
// the revival judged like a new reservation was then refused by the same
// person's other copy. RenewRuns is the gated waiter's renewal (implement load
// --wait and --race-gate), at once and every five minutes while it waits and
// while its command runs; RenewRunsIn is implement status's, before it
// renders. Each renews a due reservation through the loop's own renewal
// (renewReservation), from the run's own checkout, which is what makes it this
// copy's, and under the run's lock, so a lapsed one is revived as a step
// revives it and logged in the run record as a step logs it.

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// RunRenewal is one renewal of a run's reservation RenewRuns or RenewRunsIn
// made or tried.
type RunRenewal struct {
	// Root is the checkout the run lives in.
	Root string
	// RunID and Key name the run and the record it reserves.
	RunID string
	Key   string
	// Renewed is true for a renewal pushed and logged as reservation_renewed.
	Renewed bool
	// Note is what the run record logged: the renewal's line when Renewed,
	// else why it failed, or that the reservation is another person's or
	// copy's.
	Note string
}

// RenewRuns renews, as at now, the due reservation of each run in progress
// the gated waiter in waiterRoot serves: each that lives in waiterRoot, and
// each that lives in another worktree of the repository (git worktree list)
// with a lane, not yet done, whose worktree is waiterRoot, compared by real
// path. A worktree whose runs cannot be read, and a listing git refuses,
// renew nothing. It never refuses: the waiter's wait and its command go on
// whatever it finds.
func RenewRuns(waiterRoot string, now time.Time) []RunRenewal {
	out := renewRunsAt(waiterRoot, "", nil, now)
	wts, err := gitutil.ListWorktrees(waiterRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return out
	}
	self := realPath(waiterRoot)
	serves := func(st State) bool {
		for _, l := range st.Lanes {
			if l.Stage != StageDone && l.Stage != StageDiscarded && l.Worktree != "" && realPath(l.Worktree) == self {
				return true
			}
		}
		return false
	}
	for _, wt := range wts {
		if wt.Bare || wt.Prunable || wt.Path == "" || realPath(wt.Path) == self {
			continue
		}
		out = append(out, renewRunsAt(wt.Path, "", serves, now)...)
	}
	return out
}

// RenewRunsIn renews, as at now, the due reservation of each run in progress
// that lives in repoRoot, or of the one runID names ("" for every run):
// implement status's renewal of the runs it shows.
func RenewRunsIn(repoRoot, runID string, now time.Time) []RunRenewal {
	return renewRunsAt(repoRoot, runID, nil, now)
}

// renewRunsAt renews the due reservation of each run in progress in root
// that runID names ("" for any) and serves admits (nil for any). Whether one
// is due is read from the view as last fetched, as a step reads it
// (renewalDue): this copy's, unreleased and half through its lease, a lapsed
// one included, or held by another person or copy, which is logged once as a
// step logs it. A due one is renewed under the run's lock, the state written
// only when the record changed; a run whose lock another invocation holds is
// passed over, since that invocation is a step mid-act, which renews it
// itself.
func renewRunsAt(root, runID string, serves func(State) bool, now time.Time) []RunRenewal {
	runs, err := Runs(root)
	if err != nil {
		return nil
	}
	var view *claim.Snapshot
	var out []RunRenewal
	for _, st := range runs {
		if (runID != "" && st.RunID != runID) || !renewsWhereItWaits(st) || (serves != nil && !serves(st)) {
			continue
		}
		if view == nil {
			snap, err := claim.View(root, now)
			if err != nil {
				return out
			}
			view = &snap
		}
		if _, due, lost := renewalDue(st, *view, now); !due && lost == "" {
			continue
		}
		r := RunRenewal{Root: root, RunID: st.RunID, Key: st.Key}
		err := mutate(root, st.RunID, func(_ *os.Root, st *State) (bool, error) {
			if !renewsWhereItWaits(*st) {
				return false, nil
			}
			logged, updated := len(st.Record), st.UpdatedAt
			r.Note = renewReservation(root, st, now, view)
			if n := len(st.Record); n > logged && st.Record[n-1].Stage == StageReservationRenewed {
				r.Renewed, r.Note = true, st.Record[n-1].Note
			}
			return len(st.Record) != logged || !st.UpdatedAt.Equal(updated), nil
		})
		var rf *Refusal
		switch {
		case errors.As(err, &rf) && rf.Contention:
			continue
		case err != nil:
			r.Renewed, r.Note = false, "could not renew the reservation on "+st.Key+": "+err.Error()
		}
		if r.Renewed || r.Note != "" {
			out = append(out, r)
		}
	}
	return out
}

// renewsWhereItWaits reports whether a run's reservation is renewed where its
// agent waits: the run holds one and is in progress, and it is not an issue
// run whose lane was handed back, whose reservation the hand-back released
// (or the drain's routing of it releases).
func renewsWhereItWaits(st State) bool {
	if st.ReservedAt == nil || st.Complete() {
		return false
	}
	if st.Issue() != "" {
		for _, l := range st.Lanes {
			if l.HandBack != nil || l.Stage == StageHandedBack {
				return false
			}
		}
	}
	return true
}

// realPath is p with its symbolic links resolved, or cleaned when they cannot
// be, so a lane's worktree and the waiter's checkout compare as one place
// however each was spelt.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}
