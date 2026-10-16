package loop

// drain_finish.go finishes merged lanes (spc-2609212015054359 step 3,
// iss-2610090936460789). A lane's worktree and branch are cleaned up only by
// the landing's last step, run once its pull request has merged, and nothing
// came back to run it: the land stage waits until the merge, the drain read the
// lane as its pull request and moved on, and a run paused by its window clock
// refused the step for hours. So every drain move, before its window clock is
// read, takes the landing's last step for each issue lane parked at land whose
// pushed head is on the default branch: the cleanup does no new work, so the
// pace does not hold it.

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// DrainFinished is one merged lane a drain move took the landing's last step
// for: Done when its worktree and branch were removed and the lane marked
// done, otherwise Note says why the landing refused.
type DrainFinished struct {
	Issue string `json:"issue"`
	RunID string `json:"run_id"`
	PR    int    `json:"pr,omitempty"`
	Done  bool   `json:"done"`
	Note  string `json:"note"`
}

// parkedAtLand is the issue lane of run waiting on its pull request's merge:
// at land, its pull request armed or left open, its head pushed, and not yet
// cleaned up. It reports -1 for any other run.
func parkedAtLand(run State) int {
	if run.Issue() == "" {
		return -1
	}
	i := run.current()
	if i < 0 {
		return -1
	}
	l := run.Lanes[i]
	if l.Stage != StageLand || l.PR <= 0 || l.Landing == nil || l.Landing.Merge == "" || l.Landing.Merged != "" ||
		!gitutil.IsFullSHA(l.Landing.Pushed) {
		return -1
	}
	return i
}

// finishMerged takes the landing's last step for every issue lane in the
// checkout parked at land whose pushed head is on the default branch as the
// remote holds it, whatever the drain's or the run's window clock says. The
// default branch is fetched once, and only when a lane is parked, and the
// forge is asked nothing about a lane whose head is not on it. A lane the
// landing refuses to finish (a branch moved past what landed, a worktree with
// changes) is named, not fatal: the drain's own move goes on.
func finishMerged(repoRoot string, now time.Time) ([]DrainFinished, error) {
	runs, err := Runs(repoRoot)
	if err != nil {
		return nil, err
	}
	var parked []State
	for _, run := range runs {
		if parkedAtLand(run) >= 0 {
			parked = append(parked, run)
		}
	}
	out := []DrainFinished{}
	if len(parked) == 0 {
		return out, nil
	}
	// Every drain move keeps its parked lanes held (spc-2610031753359849): a
	// pull request waiting on a person's merge can wait overnight, and
	// nothing else steps the run, so each parked run's reservation is renewed
	// here at half its lease, logged in the run's record.
	// The view is read once, from disk, outside the run lock, and only a run
	// it shows due (or held by another) is locked and renewed; the renewal's
	// own fetch and push are made for that run alone.
	if snap, err := claim.View(repoRoot, now); err == nil {
		for _, run := range parked {
			if run.ReservedAt == nil {
				continue
			}
			if _, due, lost := renewalDue(run, snap, now); !due && lost == "" {
				continue
			}
			if err := mutate(repoRoot, run.RunID, func(_ *os.Root, st *State) (bool, error) {
				if parkedAtLand(*st) < 0 {
					return false, nil
				}
				logged, updated := len(st.Record), st.UpdatedAt
				renewReservation(repoRoot, st, now, &snap)
				return len(st.Record) != logged || !st.UpdatedAt.Equal(updated), nil
			}); err != nil {
				if _, ok := AsRefusal(err); !ok {
					return nil, err
				}
			}
		}
	}
	refused := func(run State, r *Refusal) DrainFinished {
		return DrainFinished{Issue: run.Issue(), RunID: run.RunID, PR: run.Lanes[parkedAtLand(run)].PR,
			Note: fsutil.RedactHome(r.Reason + "; " + r.Remedy)}
	}
	first := parked[0].Lanes[parkedAtLand(parked[0])]
	def, err := defaultBranch(Context{RepoRoot: repoRoot}, first)
	if err != nil {
		r, ok := AsRefusal(err)
		if !ok {
			return nil, err
		}
		for _, run := range parked {
			out = append(out, refused(run, r))
		}
		return out, nil
	}
	tracking, err := landing.FetchDefault(gitutil.Net, repoRoot, def)
	if err != nil {
		r := refuse(StageDrain, "", "", "git could not fetch "+def+" from "+Remote+" to find the merged lanes: "+errors.Unwrap(err).Error(),
			"the next `abcd drain` looks again")
		for _, run := range parked {
			out = append(out, refused(run, r))
		}
		return out, nil
	}
	for _, run := range parked {
		lane := run.Lanes[parkedAtLand(run)]
		f := DrainFinished{Issue: run.Issue(), RunID: run.RunID, PR: lane.PR}
		on, err := gitutil.IsAncestor(repoRoot, lane.Landing.Pushed, tracking)
		if err != nil {
			// git could not answer, which is not a no: the lane is named, so
			// an unreadable pushed head does not drop it from finished.
			f.Note = fsutil.RedactHome("git could not place " + shortSHA(lane.Landing.Pushed) + " on " + tracking + ": " + err.Error() +
				"; settle what git reports, then run `abcd drain` again")
			out = append(out, f)
			continue
		}
		if !on {
			continue
		}
		err = mutate(repoRoot, run.RunID, func(_ *os.Root, st *State) (bool, error) {
			i := parkedAtLand(*st)
			if i < 0 {
				return false, nil
			}
			w := laneWant(*st, i)
			if w.kind != wantBinary || w.hold {
				return false, nil
			}
			res, err := performMove(repoRoot, st, DefaultStages(), w, now)
			if err != nil {
				return false, err
			}
			if st.Lanes[i].Stage != StageDone {
				// The landing's step completed something short of the cleanup;
				// what it wrote is kept, and the lane is not claimed done.
				f.Note = res.Next
				return true, nil
			}
			f.Done = true
			f.Note = st.Record[len(st.Record)-1].Note
			return true, nil
		})
		if err != nil {
			r, ok := AsRefusal(err)
			if !ok {
				return nil, err
			}
			out = append(out, refused(run, r))
			continue
		}
		if f.Done || f.Note != "" {
			out = append(out, f)
		}
	}
	return out, nil
}

// finishedNote names, before a drain move's next move, the merged lanes it
// finished and those it could not.
func finishedNote(fs []DrainFinished) string {
	var done, not []string
	for _, f := range fs {
		pr := ""
		if f.PR > 0 {
			pr = ", pull request #" + strconv.Itoa(f.PR)
		}
		if f.Done {
			done = append(done, fmt.Sprintf("%s's run %s%s", f.Issue, f.RunID, pr))
		} else {
			not = append(not, fmt.Sprintf("%s's run %s%s (%s)", f.Issue, f.RunID, pr, f.Note))
		}
	}
	var parts []string
	if len(done) > 0 {
		parts = append(parts, fmt.Sprintf("finished %d merged lane(s), each worktree and branch removed and the lane done: %s",
			len(done), strings.Join(done, "; ")))
	}
	if len(not) > 0 {
		parts = append(parts, fmt.Sprintf("could not finish %d merged lane(s): %s", len(not), strings.Join(not, "; ")))
	}
	return strings.Join(parts, "; ")
}
