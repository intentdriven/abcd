package loop

// hold.go is what a run does with its other lanes once one is handed back
// (ruling DR6c, 2026-09-29: "FINISH, BUT HOLD THEM: when one piece is handed
// back, pieces in flight finish but nothing merges until the person re-plans;
// the person then decides whether the held pieces land as they are"). A
// sibling whose round passes after the hand-back does not land: it takes the
// stage `held` before the first landing step that reaches past the machine or
// merges — before the push when it has not pushed, before arming when it has
// opened its pull request, and disarmed (`gh pr merge <n> --disable-auto`)
// when it was already armed. The person's word on each held lane is given
// through `implement step --release <lane>` (it lands as it is) or `--discard
// <lane>` (it does not land), once no lane of the run has anything left to do.

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// judgedHead is the head the lane's last round judged, or its head.
func judgedHead(lane Lane) string {
	if n := len(lane.Validation); n > 0 {
		return lane.Validation[n-1].HeadSHA
	}
	return lane.HeadSHA
}

// handedBackCause is the id of the run's handed-back lane.
func handedBackCause(st State) string {
	for _, l := range st.Lanes {
		if l.Stage == StageHandedBack {
			return l.ID
		}
	}
	return ""
}

// holdLane holds a lane at its landing after a sibling's hand-back. An armed
// lane is disarmed through the forge client first; one whose pushed head the
// default branch already holds had landed before the hand-back, and its
// landing runs on to record it.
func holdLane(c Context, lane *Lane) (Outcome, error) {
	before := HoldBeforePush
	if ld := lane.Landing; ld != nil && ld.Pushed != "" {
		before = HoldBeforeArm
		if ld.Merge != "" {
			if ld.Armed {
				if def, err := defaultBranch(c, *lane); err == nil {
					if on, err := gitutil.IsAncestor(c.RepoRoot, ld.Pushed, "refs/remotes/"+Remote+"/"+def); err == nil && on {
						return landStage(c, lane)
					}
				}
				n := strconv.Itoa(lane.PR)
				if _, err := forge(c, *lane, "pr", "merge", n, "--disable-auto"); err != nil {
					r, _ := AsRefusal(err)
					why := err.Error()
					if r != nil {
						why = r.Reason
					}
					return Outcome{}, refuse(string(StageHeld), "", lane.ID,
						"pull request #"+n+" is armed to merge and "+handedBackCause(c.State)+" was handed back, but the forge refused to withdraw the arming: "+why,
						"decide pull request #"+n+" yourself on the forge (disarm or close it), then run `abcd implement step` again")
				}
			}
			cp := *ld
			cp.Merge, cp.Armed = "", false
			lane.Landing = &cp
		}
	}
	cause := handedBackCause(c.State)
	lane.Hold = &Hold{Since: c.Now, Cause: cause, Head: judgedHead(*lane), Before: before}
	note := fmt.Sprintf("%s held after %s's hand-back: its passing round judged %s, and it stopped before its %s; nothing merges until the person decides (%s)",
		lane.ID, cause, shortSHA(lane.Hold.Head), before, heldWayOut(lane.ID))
	return Outcome{Goto: StageHeld, Note: note}, nil
}

// busyLanes names the lanes of the run with work left: an agent out or a stage
// to take. A done, handed-back, held or discarded lane has none.
func busyLanes(st State) []string {
	var out []string
	for _, i := range st.laneOrder() {
		l := st.Lanes[i]
		switch l.Stage {
		case StageDone, StageHandedBack, StageHeld, StageDiscarded:
			continue
		}
		what := string(l.Stage)
		for _, a := range l.Awaits {
			what += ", awaiting the " + a.Role
		}
		out = append(out, l.ID+" ("+what+")")
	}
	return out
}

// decidable finds the held lane a --release or --discard names, refusing a
// lane that is not held and a run with any lane still at work.
func decidable(st State, laneID, flag string) (int, error) {
	i := -1
	for k, l := range st.Lanes {
		if l.ID == laneID {
			i = k
		}
	}
	if i < 0 {
		return -1, refuse(string(StageHeld), "", "", fmt.Sprintf("%s has no lane %q", st.RunID, laneID),
			"name a held lane `abcd implement status` lists")
	}
	if l := st.Lanes[i]; l.Stage != StageHeld || l.Hold == nil {
		return -1, refuse(string(StageHeld), "", laneID, fmt.Sprintf("%s is %s, not held, so %s does not apply to it", laneID, l.Stage, flag),
			"name a held lane `abcd implement status` lists; nothing was changed")
	}
	if busy := busyLanes(st); len(busy) > 0 {
		return -1, contend(string(StageHeld), "", laneID, "lanes of "+st.RunID+" are still at work: "+strings.Join(busy, "; "),
			"run `abcd implement step` until they finish or are held, then decide over the held lanes together; nothing was changed")
	}
	return i, nil
}

// Release lands a held lane as it is, on the person's word (`implement step
// --release <lane>`): its stage returns to `land` and its landing resumes at the
// step it stopped before, synced first when a sibling landed since its base.
// It changes nothing when the lane is not held or any lane is still at work.
func Release(repoRoot, runID, laneID string, o Options) (StepResult, error) {
	var res StepResult
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		i, err := decidable(*st, laneID, "--release")
		if err != nil {
			return false, err
		}
		now := o.now()
		lane := st.Lanes[i]
		h := *lane.Hold
		h.Released = true
		lane.Hold = &h
		lane.Stage = StageLand
		st.Lanes[i] = lane
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: "release",
			Note: fmt.Sprintf("the person released %s to land as it is, at %s; its landing resumes at its %s", lane.ID, shortSHA(h.Head), h.Before)})
		st.UpdatedAt = now
		res = laneResult(*st, lane, "", nil)
		return true, nil
	})
	return res, err
}

// Discard does not land a held lane, on the person's word (`implement step
// --discard <lane>`): its pull request is closed if it opened one, its worktree
// is removed from the machine store and its branch deleted, and its stage is
// `discarded`; its step stays unlanded in the spec, so a replanned remainder
// carries it. It changes nothing when the lane is not held or any lane is still
// at work.
func Discard(repoRoot, runID, laneID string, o Options) (StepResult, error) {
	var res StepResult
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		i, err := decidable(*st, laneID, "--discard")
		if err != nil {
			return false, err
		}
		now := o.now()
		lane := st.Lanes[i]
		c := Context{RepoRoot: repoRoot, RunDir: runRel(st.RunID), State: *st, Now: now}
		did := []string{}
		if lane.PR > 0 {
			n := strconv.Itoa(lane.PR)
			if _, err := forge(c, lane, "pr", "close", n); err != nil {
				return false, err
			}
			did = append(did, "closed pull request #"+n)
		}
		if lane.Worktree != "" {
			if err := removeLaneWorktree(c, lane); err != nil {
				return false, err
			}
			did = append(did, "removed its worktree")
		}
		if lane.Branch != "" {
			tip, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+lane.Branch+"^{commit}", "--")
			if err == nil && gitutil.IsFullSHA(tip) {
				if _, err := gitutil.Run(repoRoot, "update-ref", "-d", "refs/heads/"+lane.Branch, tip); err != nil {
					return false, refuse(string(StageHeld), "", lane.ID, "git could not delete "+lane.Branch+": "+fsutil.RedactHome(err.Error()),
						"settle what git reports, then run `abcd implement step --discard "+lane.ID+"` again")
				}
				did = append(did, "deleted its branch "+lane.Branch+" at "+shortSHA(tip))
			}
		}
		lane.Stage = StageDiscarded
		st.Lanes[i] = lane
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: "discard",
			Note: fmt.Sprintf("the person discarded %s (%s); step %d stays unlanded in %s", lane.ID, strings.Join(did, ", "), lane.SpecStep, specOf(*st))})
		st.UpdatedAt = now
		res = laneResult(*st, lane, "", nil)
		return true, nil
	})
	return res, err
}
