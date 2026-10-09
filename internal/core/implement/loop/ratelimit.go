package loop

// ratelimit.go is the rate-limit checkpoint (itd-2609201925079472 criterion 8,
// decision 6, from itd-29; spc-2609301921521360). A runner that answers a role
// with a rate-limit response ran nothing, and the role is not handed to the
// host, because every lane of the run spends the same budget. Instead, under
// the run's lock:
//
//   - the await the response answered is taken off its lane: its agent has
//     stopped, so its slot is free, and the stage hands the work out afresh,
//     to a fresh agent with the same brief, once the run may move again;
//   - every lane with work in flight, the response's own included, is
//     checkpointed to its own branch: the run record names the branch and the
//     commit it holds, which is where the lane's next agent takes it up. The
//     commits an agent made are on the branch already; the loop makes none of
//     its own, so no commit carries text abcd did not compose or judge, and
//     whatever an agent left uncommitted stays in the lane's worktree, which
//     the loop never cleans under it. An agent still out on another lane may
//     hand back its receipt, as in any pause;
//   - the window ends early for the whole run: next_eligible_at is written
//     once, the run's pause from now, as the window clock writes it; a
//     response inside a pause already written leaves its time as it is;
//   - the record names the lane, the role and the runner the response came
//     from.
//
// The call exits as a closed window does, naming the time; before it every
// step is refused as a pause, and at it a window opens. The pause is the
// run's pace in minutes (decision 2): a reset time a harness reports beside
// its response is not read.

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The run record's stages of the checkpoint.
const (
	StageRateLimit  = "rate-limit"
	StageCheckpoint = "checkpoint"
)

// RateLimit is the rate-limit response a call read: the lane and the role
// whose runner answered with it, and the runner.
type RateLimit struct {
	Lane   string `json:"lane"`
	Role   string `json:"role"`
	Runner string `json:"runner"`
	// Detail is abcd's own account of the response; it never carries the
	// harness's text.
	Detail string `json:"detail"`
}

// rateLimited checkpoints the run on a rate-limit response from the runner
// the await on laneID was handed to, and ends its window.
func rateLimited(repoRoot, runID, laneID string, aw Await, fl *runner.Failure, o Options) (StepResult, error) {
	var res StepResult
	rl := RateLimit{Lane: laneID, Role: aw.Role, Runner: fl.Runner, Detail: fl.Detail}
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		now := o.now()
		for i := range st.Lanes {
			if st.Lanes[i].ID != laneID {
				continue
			}
			l := st.Lanes[i]
			var kept []Await
			for _, a := range l.Awaits {
				if !samePath(repoRoot, a.Receipt, aw.Receipt) {
					kept = append(kept, a)
				}
			}
			l.Awaits = kept
			st.Lanes[i] = l
		}
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageRateLimit,
			Note: fmt.Sprintf("the %s's runner %s answered with a rate-limit response (%s); its agent ran nothing to hand back, "+
				"so its slot is freed and the work is handed out afresh when the run moves again", aw.Role, fl.Runner, fl.Detail)})
		for _, i := range st.laneOrder() {
			l := st.Lanes[i]
			if l.ID != laneID && len(l.Awaits) == 0 {
				continue
			}
			st.Record = append(st.Record, Entry{At: now, Lane: l.ID, Stage: StageCheckpoint, Note: checkpointNote(repoRoot, l)})
		}
		if st.NextEligibleAt == nil || !now.Before(*st.NextEligibleAt) {
			pause := BundledPauseMinutes
			if st.Pace != nil {
				pause = st.Pace.PauseMinutes.Value
			}
			until := now.Add(time.Duration(pause) * time.Minute)
			opened := "the run's window"
			if st.WindowStartedAt != nil {
				opened = "the window opened at " + st.WindowStartedAt.UTC().Format(time.RFC3339)
			}
			st.NextEligibleAt = &until
			st.Record = append(st.Record, Entry{At: now, Stage: "pause",
				Note: fmt.Sprintf("%s ends early on %s's rate-limit response, since every lane spends the same budget; "+
					"no stage is taken before %s (a %d-minute pause)", opened, laneID, until.UTC().Format(time.RFC3339), pause)})
		}
		st.UpdatedAt = now
		res = idleResult(*st)
		res.NextEligibleAt = st.NextEligibleAt
		res.RateLimit = &rl
		res.Next = fmt.Sprintf("the %s runner answered %s's %s with a rate-limit response; %s", fl.Runner, laneID, aw.Role,
			pausedMove(*st, *st.NextEligibleAt))
		return true, nil
	})
	return res, err
}

// checkpointNote names where a lane stands on its branch at a rate-limit
// response: the branch and the commit it holds, and the agents it still has
// out. A branch that cannot be read is named as such.
func checkpointNote(repoRoot string, l Lane) string {
	if l.Branch == "" {
		return l.ID + " has no branch yet; it is taken up from its stage, " + string(l.Stage)
	}
	at := "a head that could not be read"
	if tip, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+l.Branch+"^{commit}", "--"); err == nil {
		at = shortSHA(strings.TrimSpace(tip))
	}
	note := fmt.Sprintf("%s checkpointed to %s at %s, at its %s stage", l.ID, l.Branch, at, l.Stage)
	if len(l.Awaits) > 0 {
		roles := make([]string, 0, len(l.Awaits))
		for _, a := range l.Awaits {
			roles = append(roles, a.Role)
		}
		note += "; its " + strings.Join(roles, ", ") + " may still hand back a receipt"
	}
	return note
}
