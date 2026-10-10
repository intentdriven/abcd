package loop

// ratelimit.go is the rate-limit checkpoint (itd-2609201925079472 criterion 8,
// from itd-29; spc-2609301921521360, and spc-2609202134341288's "The pace and
// the fix rounds, per lane"). A runner's rate-limit response (runner
// ReasonRateLimited, which the dispatcher never falls back on) ends the window
// early for the whole run, since every lane spends the same budget:
//
//   - next_eligible_at is written once: a response met while the run is
//     already paused leaves the pause's end as it was;
//   - the lane the response came from is checkpointed to its branch: its
//     agent is gone, so what it left uncommitted is saved aside with its
//     partial receipt and the worktree reset to the branch's last commit, as a
//     restart does (restart.go; ruling of 2026-10-09: a gone agent's
//     uncommitted work is kept for review and never built on), and its await
//     is dropped, so the first step after the pause hands the same work to a
//     fresh agent. A validator edits nothing and may share the worktree with
//     its round, so only its partial return is saved aside;
//   - every other lane with work in flight is checkpointed at its branch's
//     head, which the record names: its agents are not this process's and are
//     not stopped, its worktree is not touched, and each may still hand back
//     its receipt inside the pause, as at a window's end. One that meets the
//     same limit comes back through here and is checkpointed as the first;
//   - the record names the response, the runner and the lane it came from.
//
// The pause is the run's own pause minutes (decision 2: minutes, a
// quota-window signal being a later refinement), not the reset time a harness
// may report.

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// StageRateLimit is the run record's stage for a rate-limit response, and
// StageCheckpoint for a lane it checkpointed.
const (
	StageRateLimit  = "rate-limit"
	StageCheckpoint = "checkpoint"
)

// RateLimit is a runner's rate-limit response as a step reports it: the lane
// and the agent it came from, the runner, the response in abcd's words, and
// every lane it checkpointed.
type RateLimit struct {
	Lane        string       `json:"lane"`
	Role        string       `json:"role"`
	Runner      string       `json:"runner"`
	Response    string       `json:"response"`
	Checkpoints []Checkpoint `json:"checkpoints"`
}

// Checkpoint is one lane with work in flight, as a rate limit left it.
type Checkpoint struct {
	Lane   string `json:"lane"`
	Branch string `json:"branch"`
	// Head is the branch's last commit, "" when git cannot name it.
	Head string `json:"head"`
	// Aside is where the rate-limited agent's uncommitted work and partial
	// receipt were saved, relative to the checkout root; "" for a lane whose
	// agents were not the ones limited.
	Aside string `json:"aside,omitempty"`
	// Out are the roles of the lane's agents still out after the checkpoint,
	// each of which may hand back its receipt inside the pause.
	Out []string `json:"out"`
	// Kept is why the limited agent's await was kept and its worktree left as
	// it was (the save aside was refused); "" when it was not.
	Kept string `json:"kept,omitempty"`
}

// rateLimitWindow ends the run's window on the rate-limit response fl, which
// lane laneID's agent awaited at aw met, and checkpoints every lane with work
// in flight.
func rateLimitWindow(repoRoot, runID, laneID string, aw Await, fl *runner.Failure, o Options) (StepResult, error) {
	var res StepResult
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		now := o.now()
		hit := slices.IndexFunc(st.Lanes, func(l Lane) bool { return l.ID == laneID })
		if hit < 0 {
			return false, refuse(StageRateLimit, "", "", fmt.Sprintf("%s has no lane %q to checkpoint", st.RunID, laneID),
				"restore the run's state file")
		}
		rl := &RateLimit{Lane: laneID, Role: aw.Role, Runner: fl.Runner, Response: fl.Detail, Checkpoints: []Checkpoint{}}
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageRateLimit,
			Note: fmt.Sprintf("the %s runner answered %s's %s with a rate-limit response (%s); every lane spends the same budget, so the run's window ends early",
				fl.Runner, laneID, aw.Role, fl.Detail)})

		for _, i := range st.laneOrder() {
			lane := st.Lanes[i]
			if i != hit && len(lane.Awaits) == 0 {
				continue
			}
			cp := Checkpoint{Lane: lane.ID, Branch: lane.Branch, Head: branchHead(repoRoot, lane.Branch)}
			if i == hit {
				checkpointLimited(repoRoot, root, st, &lane, aw, fl, &cp, now)
				st.Lanes[i] = lane
			}
			for _, a := range lane.Awaits {
				cp.Out = append(cp.Out, a.Role)
			}
			if cp.Out == nil {
				cp.Out = []string{}
			}
			st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageCheckpoint, Note: checkpointNote(cp, i == hit, aw.Role)})
			rl.Checkpoints = append(rl.Checkpoints, cp)
		}

		until := st.NextEligibleAt
		if until == nil || !now.Before(*until) {
			pause := BundledPauseMinutes
			if st.Pace != nil {
				pause = st.Pace.PauseMinutes.Value
			}
			end := now.Add(time.Duration(pause) * time.Minute)
			until = &end
			st.NextEligibleAt = until
			opened := "the run's window"
			if st.WindowStartedAt != nil {
				opened = "the window opened at " + st.WindowStartedAt.UTC().Format(time.RFC3339)
			}
			st.Record = append(st.Record, Entry{At: now, Stage: "pause",
				Note: fmt.Sprintf("a rate-limit response on %s ends %s early; no stage is taken before %s (a %d-minute pause)",
					laneID, opened, end.UTC().Format(time.RFC3339), pause)})
		}
		st.UpdatedAt = now

		res = idleResult(*st)
		at := *until
		res.NextEligibleAt = &at
		res.RateLimit = rl
		res.Next = fmt.Sprintf("the %s runner's rate limit on %s's %s ended the run's window early. %s",
			fl.Runner, laneID, aw.Role, pausedMove(*st, at))
		return true, nil
	})
	return res, err
}

// checkpointLimited checkpoints the lane whose agent met the rate limit: its
// uncommitted work (an implementer's) and partial receipt saved aside, its
// worktree reset to the branch's last commit, and its await dropped. A save
// the loop refuses keeps the await and leaves the worktree, and cp says why.
func checkpointLimited(repoRoot string, root *os.Root, st *State, lane *Lane, aw Await, fl *runner.Failure, cp *Checkpoint, now time.Time) {
	k := slices.IndexFunc(lane.Awaits, func(a Await) bool { return samePath(repoRoot, a.Receipt, aw.Receipt) })
	if k < 0 {
		// The await is gone already: its receipt was handed back, or a
		// restart re-told it, before the response was read.
		return
	}
	why := fmt.Sprintf("rate limit: the %s runner answered the %s with one", fl.Runner, aw.Role)
	aside, err := saveAside(repoRoot, root, st.RunID, *lane, lane.Awaits[k], why, aw.Role == RoleImplementer, now)
	if err != nil {
		reason := fsutil.RedactHome(err.Error())
		if r, ok := AsRefusal(err); ok {
			reason = r.Reason
		}
		cp.Kept = reason
		return
	}
	cp.Aside = aside.Path
	if aw.Role == RoleImplementer {
		cp.Head = aside.Head
	}
	lane.Awaits = slices.Delete(slices.Clone(lane.Awaits), k, k+1)
	if len(lane.Awaits) == 0 {
		lane.Awaits = nil
	}
}

// checkpointNote is the run record's line for one lane's checkpoint.
func checkpointNote(cp Checkpoint, limited bool, role string) string {
	head := "an unnamed head"
	if cp.Head != "" {
		head = shortSHA(cp.Head)
	}
	at := fmt.Sprintf("%s is checkpointed at %s, the head of its branch %s", cp.Lane, head, cp.Branch)
	out := ""
	if len(cp.Out) > 0 {
		out = "; still out, and free to hand back its receipt inside the pause: " + strings.Join(cp.Out, ", ")
	}
	switch {
	case !limited:
		return at + out
	case cp.Kept != "":
		return fmt.Sprintf("%s; the %s's await is kept and its worktree left as it was, since its work could not be saved aside (%s): settle it, then `abcd implement step --restart %s` once the pause ends%s",
			at, role, cp.Kept, cp.Lane, out)
	case cp.Aside == "":
		return fmt.Sprintf("%s; the %s's await was already settled%s", at, role, out)
	}
	return fmt.Sprintf("%s; what the %s left (uncommitted work and partial receipt) is saved aside at %s for review, never built on, and a fresh %s takes the work once the pause ends%s",
		at, role, cp.Aside, role, out)
}

// branchHead is the last commit of branch in the checkout, "" when git cannot
// name it.
func branchHead(repoRoot, branch string) string {
	if branch == "" {
		return ""
	}
	sha, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(sha) {
		return ""
	}
	return sha
}
