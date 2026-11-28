package loop

// drain_lease.go is the drain's keep-alive of its lane in progress
// (spc-2610161044461448 step 7; iss-2610161015393523; dcn-2610161040374343).
// A lane's reservation is renewed by its own steps and receipts, so a host
// that paused the lane past its lease let another account take the issue
// over. Every drain move renews the lane in progress itself, at half its
// lease, before its window gate, so a move that waits on the pace, on a
// judgement or on the lane renews too: a lease keep-alive independent of the
// lane's progress, as Chubby's and etcd's sessions keep theirs. The fence
// before the landing's push (pushFence) is the other half: a reservation
// that moved anyway pushes nothing.

import (
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// renewInProgress renews the reservation of each lane the drain has in
// progress at half its lease, judged on view (as the move fetched it; read
// from disk when nil), and logs it in the lane's run record. A lane parked at
// land is the merged-lane finish's to renew (finishMerged), and a lane its run
// has handed back, by its own step or as stuck, is not renewed by this pass: the
// drain has not routed it yet, as a paused move does not, and its
// reservation, released or left holding by a release origin refused, is
// the routing's (releaseHandedBack). A lane handed back whose pull request is
// still armed (armingOwed) is in progress until the forge reports no arming
// (laneOutcome), so this pass renews it: its hand-back keeps the reservation. It returns the
// notes of renewals that failed or found the hold another's, for the move's
// next step; a lane whose run cannot be read is left to the routing below,
// which names it. A run a drive let go is not renewed by this pass either: the tip of
// a release that did not reach origin is the take-up's, which compares the record first.
// Nor is a hold a drive left to lapse (leftToLapse), which the take-up reserves again.
func renewInProgress(repoRoot string, st DrainState, view *claim.Snapshot, now time.Time) []string {
	var notes []string
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		run, err := ReadState(repoRoot, l.RunID)
		if err != nil || !renewable(run) || letGoByDrive(run) {
			continue
		}
		if view == nil {
			snap, err := claim.View(repoRoot, now)
			if err != nil {
				continue
			}
			view = &snap
		}
		// A hold a drive left to lapse is no one's work, and lapses at its lease
		// (leftToLapse): the take-up that works it again reserves it again.
		if leftToLapse(repoRoot, run, *view) {
			continue
		}
		if _, due, lost := renewalDue(run, *view, now); !due && lost == "" {
			continue
		}
		note := ""
		if err := mutateReservation(repoRoot, l.RunID, func(st *State) (bool, error) {
			if !renewable(*st) {
				return false, nil
			}
			logged, updated := len(st.Record), st.UpdatedAt
			note = renewReservation(repoRoot, st, now, view)
			return len(st.Record) != logged || !st.UpdatedAt.Equal(updated), nil
		}); err != nil {
			continue
		}
		if note != "" {
			notes = append(notes, note)
		}
	}
	return notes
}

// renewParkedDue renews each lane parked at land whose reservation is at half its
// lease, or read as another's, as a drain's move does (finishMerged), for a wait
// long enough that no move comes: otherwise a peer could redo work that only awaits
// its merge, and the land fence would not arm. The forge is read first, and only
// when one is due, so a closed or conflicting pull request's lane is left to lapse
// unless a load-gate waiter in this copy renews it (gateWaiterRenews).
// It returns what a renewal that did not go through said. The save is the
// reservation's alone, as every renewal's is, so a run another driver wrote at
// another version stays at it. A run whose lane the drive passed over (passed,
// DrainOptions.PassedLanes) is not read: nothing is renewed or recorded for it.
func renewParkedDue(repoRoot string, now time.Time, passed []string) []string {
	runs, err := Runs(repoRoot)
	if err != nil {
		return nil
	}
	var snap claim.Snapshot
	var due []State
	read := false
	for _, run := range runs {
		if slices.Contains(passed, run.RunID) || parkedAtLand(run) < 0 || !run.holdsAny() {
			continue
		}
		if !read {
			if snap, err = claim.View(repoRoot, now); err != nil {
				return nil
			}
			read = true
		}
		if parkedDue(run, snap, now) {
			due = append(due, run)
		}
	}
	if len(due) == 0 {
		return nil
	}
	listed, unread := listParked(repoRoot)
	var notes []string
	for _, run := range due {
		if !readParked(run, run.Lanes[parkedAtLand(run)], listed, unread).held() {
			continue
		}
		note := ""
		if err := mutateReservation(repoRoot, run.RunID, func(st *State) (bool, error) {
			if parkedAtLand(*st) < 0 {
				return false, nil
			}
			logged, updated := len(st.Record), st.UpdatedAt
			note = renewReservation(repoRoot, st, now, &snap)
			return len(st.Record) != logged || !st.UpdatedAt.Equal(updated), nil
		}); err != nil {
			continue
		}
		if note != "" {
			notes = append(notes, note)
		}
	}
	return notes
}

// renewable reports whether the drain's move renews run's reservation: it
// holds one, the lead's or, in a run with members, any member's (holdsAny),
// and its lane is in progress and not parked at land.
func renewable(run State) bool {
	outcome, _, _ := laneOutcome(run)
	return run.holdsAny() && outcome == DrainLaneInProgress && parkedAtLand(run) < 0
}

// parkedDue reports whether a parked run's reservation is due for a renewal,
// or held by another person or copy, as the view snap shows it: the run's key
// in a run with no members, and in a run with members each issue it still
// holds (renewalDueAny), so a lead handed back alone, a person's again, is no
// longer read (spc-2611201455051897 step 2).
func parkedDue(run State, snap claim.Snapshot, now time.Time) bool {
	if run.hasMembers() {
		return renewalDueAny(run, snap, now)
	}
	_, due, lost := renewalDue(run, snap, now)
	return due || lost != ""
}

// renewForming renews the holds of the bundle st is forming (drain_form.go)
// at half their lease, judged on view, as a lane in progress is renewed: each
// renewal keeps the tip's own term, and the bundle records the commit it
// pushed, which the move's holds take as their own (judgeHolds.moved). A hold
// the view no longer shows at its tip is the drain's no longer (released,
// renewed or reserved again by hand, or taken over): a member's leaves the
// bundle, and the lead's lets the whole bundle go, the other holds released
// on the move's way out. It returns the notes to give and whether the
// bundle changed, so the move writes it. Nothing is read with no bundle
// forming or no view.
func renewForming(repoRoot string, st *DrainState, holds *judgeHolds, view *claim.Snapshot, now time.Time) (notes []string, changed bool) {
	f := st.Forming
	if f == nil || view == nil {
		return nil, false
	}
	// lost is why the drain no longer holds e, "" while it does.
	lost := func(e formingEntry) string {
		r, ok := view.Lookup(e.issue)
		if ok && r.Commit == e.hold.Commit {
			return ""
		}
		return "the reservation the drain took to judge " + e.issue + " is no longer its tip on " + claim.Remote + movedTo(r, ok)
	}
	// renew renews h, the hold on id, when it is due, and returns the
	// renewed hold, nil when none was due or the renewal failed (note).
	renew := func(id string, h *DrainHold) (*DrainHold, string) {
		own, due, _ := issueRenewalDue(id, *view, now)
		if !due {
			return nil, ""
		}
		res, err := reserveRenewal(repoRoot, claim.Request{Record: id, Renew: true, Lease: claim.RenewalTerm(*own)}, claimOptions(now))
		if err != nil {
			return nil, "could not renew the reservation on " + id + ", which the bundle forming on " + f.Lead + " holds: " + err.Error()
		}
		r := *h
		r.Commit, r.ExpiresAt = res.Commit, res.ExpiresAt
		holds.moved(id, h.Commit, r)
		return &r, ""
	}
	if f.Hold != nil {
		if why := lost(formingEntry{issue: f.Lead, hold: f.Hold}); why != "" {
			st.Forming = nil
			return []string{why + "; the bundle forming on " + f.Lead + " is let go, and its other reservations are released"}, true
		}
	}
	kept := f.Members[:0]
	for _, m := range f.Members {
		if m.Hold != nil {
			if why := lost(formingEntry{issue: m.Issue, hold: m.Hold}); why != "" {
				notes = append(notes, why+"; it leaves the bundle forming on "+f.Lead)
				changed = true
				continue
			}
		}
		kept = append(kept, m)
	}
	f.Members = kept
	if f.Hold != nil {
		h, note := renew(f.Lead, f.Hold)
		if note != "" {
			notes = append(notes, note)
		}
		if h != nil {
			f.Hold, changed = h, true
		}
	}
	for i := range f.Members {
		if f.Members[i].Hold == nil {
			continue
		}
		h, note := renew(f.Members[i].Issue, f.Members[i].Hold)
		if note != "" {
			notes = append(notes, note)
		}
		if h != nil {
			f.Members[i].Hold, changed = h, true
		}
	}
	return notes, changed
}
