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
// revives it and logged in the run record as a step logs it. The waiter may be
// running a lane's newer abcd, so the save is the reservation's alone and the
// run stays at its version (mutateReservation, iss-2611020719494343).

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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
// path. A run that cannot be read is named in the result and not renewed, and
// the others in its worktree still are; a worktree that cannot be listed, and
// a listing git refuses, renew nothing. It never refuses: the waiter's wait and
// its command go on whatever it finds.
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

// runsView reads the reservation view a round of renewRunsAt picks its due
// runs by; a test replaces it to revive a run between that read and the
// run's lock, as another waiter can.
var runsView = claim.View

// renewRunsAt renews the due reservation of each run in progress in root
// that runID names ("" for any) and serves admits (nil for any). Whether one
// is due is first read from the view as last fetched, as a step reads it
// (renewalDue): this copy's, unreleased and half through its lease, a lapsed
// one included, or held by another person or copy, which is logged once as a
// step logs it. A due one is renewed under the checkout's run lock, the view
// read again there, so a renewal another waiter made since the first read is
// seen and one lapse is renewed once; the state is written only when the
// record changed. While another invocation holds the checkout's run lock the
// run is skipped, and the next round renews it.
func renewRunsAt(root, runID string, serves func(State) bool, now time.Time) []RunRenewal {
	runs, out := renewalRuns(root, runID)
	var view *claim.Snapshot
	for _, st := range runs {
		if (runID != "" && st.RunID != runID) || !renewsWhereItWaits(st) || (serves != nil && !serves(st)) {
			continue
		}
		if view == nil {
			snap, err := runsView(root, now)
			if err != nil {
				return out
			}
			view = &snap
		}
		if _, due, lost := renewalDue(st, *view, now); !due && lost == "" {
			continue
		}
		// A renewal due under a stop settles the run first: what a stop lets go
		// is let go, and what it cannot save keeps its lease below. A renewal is
		// never the run's driver (Options.afar), so a run whose file states
		// another version is not settled: it is left to its own driver's check,
		// and the renewal says so. Below, its claim is renewed at the version its
		// file states while an agent may still be in its call, or while that check
		// has left work a save could not take (stopEndsRenewal). Otherwise it is
		// neither renewed nor let go, and lapses at its lease with its work on
		// its branch and in its worktree, none of it saved aside.
		_, left := settleDueStop(root, st.RunID)
		r := RunRenewal{Root: root, RunID: st.RunID, Key: st.Key}
		err := mutateReservation(root, st.RunID, func(st *State) (bool, error) {
			if !renewsWhereItWaits(*st) {
				return false, nil
			}
			logged, updated := len(st.Record), st.UpdatedAt
			// nil: the view is read again under the lock, not the one read
			// before it, which another waiter's renewal may have outdated.
			r.Note = renewReservation(root, st, now, nil)
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
		if left != "" {
			// Said, so a run left unsettled under a stop is never passed over in silence.
			r.Note = joinNext("left unsettled under the stop for its own driver's check: "+left, r.Note)
		}
		if r.Renewed || r.Note != "" {
			out = append(out, r)
		}
	}
	return out
}

// renewalRuns reads the runs in root that a renewal may look at (the one runID
// names, or every one for ""), oldest first as Runs orders them, and a note for
// each it cannot read. One a newer abcd wrote is the case that matters: it is
// not this abcd's to rewrite, so it is said rather than passed over in silence,
// and it holds no other run's renewal back. A checkout whose runs cannot even
// be listed gives nothing, as it always has.
func renewalRuns(root, runID string) ([]State, []RunRenewal) {
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, nil
	}
	defer dir.Close()
	ids, err := runIDs(dir)
	if err != nil {
		return nil, nil
	}
	var runs []State
	var notes []RunRenewal
	for _, id := range ids {
		if runID != "" && id != runID {
			continue
		}
		st, err := readListedState(dir, id)
		if err != nil {
			notes = append(notes, RunRenewal{Root: root, RunID: id, Note: "could not read the run to renew its reservation: " + err.Error()})
			continue
		}
		runs = append(runs, st)
	}
	slices.SortStableFunc(runs, func(a, b State) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return runs, notes
}

// renewsWhereItWaits reports whether a run's reservation is renewed where its
// agent waits: the run holds one and is in progress, and it is not an issue
// run whose lane was handed back, whose reservation the hand-back released
// (or the drain's routing of it releases), nor one parked at land, which the
// drain's move renews once it has read the lane's pull request
// (drain_finish.go), as the drain's own renewal leaves it (renewable).
func renewsWhereItWaits(st State) bool {
	if st.ReservedAt == nil || st.Complete() || parkedAtLand(st) >= 0 {
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
