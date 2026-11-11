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
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// renewInProgress renews the reservation of each lane the drain has in
// progress at half its lease, judged on view (as the move fetched it; read
// from disk when nil), and logs it in the lane's run record. A lane parked at
// land is the merged-lane finish's to renew (finishMerged), and a lane its run
// has handed back, by its own step or as stuck, is renewed by nothing: the
// drain has not routed it yet, as a paused move does not, and its
// reservation, released or left holding by a release origin refused, is
// the routing's (releaseHandedBack). It returns the
// notes of renewals that failed or found the hold another's, for the move's
// next step; a lane whose run cannot be read is left to the routing below,
// which names it. A run a drive let go is renewed by nothing here either: the tip of
// a release that did not reach origin is the take-up's, which compares the record first.
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
// when one is due, so a closed or conflicting pull request's lane is left to lapse.
// It returns what a renewal that did not go through said. The save is the
// reservation's alone, as every renewal's is, so a run another driver wrote at
// another version stays at it.
func renewParkedDue(repoRoot string, now time.Time) []string {
	runs, err := Runs(repoRoot)
	if err != nil {
		return nil
	}
	var snap claim.Snapshot
	var due []State
	read := false
	for _, run := range runs {
		if parkedAtLand(run) < 0 || run.ReservedAt == nil {
			continue
		}
		if !read {
			if snap, err = claim.View(repoRoot, now); err != nil {
				return nil
			}
			read = true
		}
		if _, d, lost := renewalDue(run, snap, now); d || lost != "" {
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
// holds one, and its lane is in progress and not parked at land.
func renewable(run State) bool {
	outcome, _, _ := laneOutcome(run)
	return run.ReservedAt != nil && outcome == DrainLaneInProgress && parkedAtLand(run) < 0
}
