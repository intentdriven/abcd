package loop

// drive.go is the process driver (spc-2609202134338445 piece 3) and the
// runner's loop half (itd-2609201916056194, spc-2609221533057881): the same
// loop the host calls, with the agent a stage awaits started by the loop
// itself through the command-line runner its role is routed to
// (roles.<role>.runner), and its receipt handed back through the same
// Receipt a host calls. Nothing else changes:
//
//   - a role left on the host (the default) is handed to the host exactly as
//     advance hands it, with the same result and the same state, byte for byte
//     (criterion 2);
//   - a routed role is run through internal/core/runner's Dispatcher, whose
//     validator IS the stage's own receipt verifier, run through Receipt under
//     the run's lock, so a runner's answer is judged exactly as a host
//     sub-agent's is and a verified one advances the lane there and then
//     (criterion 1); the verified receipt, or the validator's recorded return,
//     names the route that ran it, and nothing else about it differs
//     (criterion 7);
//   - a runner that is absent, refuses, fails, answers unparsably or writes a
//     receipt the verifier refuses pauses the lane rather than handing the
//     role to the host (iss-2611012047534449, dcn-2611012113432054;
//     runner_pause.go): the await stays out, marked paused with the reason
//     and the runner's first stderr line, one fallback receipt naming none as
//     the route that ran is written into the run's state (Fallbacks) and the
//     record, which counts them per runner and per role (criterion 4), and a
//     later step retries the role on its runner;
//   - a runner that answers with a rate-limit response is the one failure not
//     handed to the host: every route spends the same budget, so the run's
//     window ends early and every lane with work in flight is checkpointed to
//     its branch (ratelimit.go; itd-2609201925079472 criterion 8).
//
// The runner is started outside the run's lock, which is held only for the
// advance before it and the Receipt or the fallback write after it, so a
// status read or another checkout's step is never kept waiting on a model.
//
// A host session always drives this call: the no-host path, where abcd runs
// alone and a host-routed role goes to runner.fallback_host, is the reversal
// of the host-delegated boundary that adr-2610222222326104 records, which
// `abcd drain` builds first (itd-2610222215485265) and `abcd build` adopts
// after it (spc-2609202134338445 step 12).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// StageRunner is the record's stage for a role a runner ran, and StageFallback
// for one a runner did not.
const (
	StageRunner   = "runner"
	StageFallback = "fallback"
)

// Runners is what the process driver starts a routed role through.
type Runners struct {
	// Config is the runner configuration read when the call began
	// (runner.Load); nil drives nothing and leaves every role on the host.
	Config *runner.Config
	// Transcripts is where each runner's transcript lands: abcd's own store
	// (runner.HistoryStore) in production.
	Transcripts runner.TranscriptStore
	// Timeout bounds one runner's run; runner.DefaultTimeout when zero.
	Timeout time.Duration
}

// roleTools are the tools each role the loop starts is granted without a
// prompt when a runner runs it. A reviewer's are its agent definition's
// (agents/<role>.md, which TestRoleToolsFollowTheAgentDefinitions holds this
// table to) plus Write, since its brief tells it to write its return to a
// path; the implementer's are what a lane's work takes; the auditor's are a
// reviewer's, its verdict written the same way.
var roleTools = map[string][]string{
	RoleImplementer: {"Read", "Edit", "Write", "Bash", "Grep", "Glob"},
	RoleRuthless:    {"Read", "Grep", "Glob", "Bash", "Write"},
	RoleSecurity:    {"Read", "Grep", "Glob", "Bash", "Write"},
	RoleAuditor:     {"Read", "Grep", "Glob", "Bash", "Write"},
}

// toolsFor returns the tools a runner grants role; nil for a role the loop
// does not start.
func toolsFor(role string) []string {
	t, ok := roleTools[role]
	if !ok {
		return nil
	}
	return append([]string(nil), t...)
}

// Drive performs the next stage as advance does and, when the stage hands the
// lane to an agent whose role is routed to a runner, starts that agent through
// the runner and hands its receipt back through Receipt. A role on the host
// returns the await for the host to act on, as advance returns it; a runner
// that did not start its role pauses the lane, and the result names the role
// as paused (runner_pause.go). A step that hands no agent out retries one held
// role once on its runner, and a --route for a held role retries it at once.
func Drive(ctx context.Context, repoRoot, runID string, steps Stages, o Options, rs Runners) (StepResult, error) {
	// An auto-plan run's reviews stay on the host (autoplan.go): its lane has
	// no worktree a runner could be confined to, so no role of it is routed,
	// nothing is dispatched and no fallback is recorded; a --route for it is
	// refused before anything is written, as one the step will not use is.
	if st, err := ReadState(repoRoot, runID); err == nil && st.autoPlanning() && rs.Config != nil {
		if len(rs.Config.FlagRoutes()) > 0 {
			return StepResult{}, refuse(StageRoute, "", "", "--route routes a role to a runner, and an auto-plan run's plan reviews run on the host's agents",
				"step again without --route; nothing was written")
		}
		rs.Config = nil
	}
	var routed func(string) bool
	var overrides map[string]string
	if rs.Config != nil {
		routed = func(role string) bool { return rs.Config.RouteFor(role).Runner != runner.Host }
		overrides = rs.Config.FlagRoutes()
		if len(overrides) > 0 {
			// A --route for a held role (runner_pause.go) retries it at
			// once, and is the step's one move.
			if r, took, err := retryRouted(ctx, repoRoot, runID, steps, o, rs, overrides); took {
				return r, err
			}
		}
		o.runnerRoute = func(role string) (string, string) {
			r := rs.Config.RouteFor(role)
			if r.Runner == runner.Host {
				return "", ""
			}
			return r.Runner, r.Override()
		}
	}
	res, err := advanceWith(repoRoot, runID, steps, o, routed, overrides)
	if err != nil || rs.Config == nil {
		return res, err
	}
	if res.Awaiting == nil || !res.handed {
		// The step handed no agent out: a paused role the person has since
		// routed to the host is the host's now; otherwise a held role,
		// paused or with the host with nothing received, is retried once
		// on its runner.
		if r, took, err := releaseToHost(repoRoot, runID, o, rs, res); took || err != nil {
			return r, err
		}
		return retryHeld(ctx, repoRoot, runID, steps, o, rs, res)
	}
	aw := *res.Awaiting
	if rs.Config.RouteFor(aw.Role).Runner == runner.Host {
		return res, nil
	}
	r, err := dispatchAwait(ctx, repoRoot, runID, steps, o, rs, res.Lane, aw, "", res)
	if errors.Is(err, errClaimed) {
		// A peer's step retried the role in the moment between: its runner
		// has it, so the host is handed nothing.
		res.Awaiting = nil
		res.Next = fmt.Sprintf("another `abcd implement step` started the runner for %s's %s; run `abcd implement step` once it has finished", res.Lane, aw.Role)
		return res, nil
	}
	return r, err
}

// LoadRunners reads the runner configuration (runner.Load) at the roots a lane
// starts from, and refuses in the loop's refusal shape on any fault, a model
// route its provider's allowlist does not admit included, before a run is
// created or a runner launched (itd-2609201916056194 criterion 5). The
// configuration's diagnostics are the caller's to print.
func LoadRunners(r layered.Roots) (*runner.Config, error) {
	c, err := runner.Load(r)
	if err != nil {
		return nil, refuse(StageRunner, "", "", err.Error(),
			"correct the runner configuration the reason names; no runner is launched and nothing is written")
	}
	return c, nil
}

// absIn is p read against the checkout root when it is relative.
func absIn(repoRoot, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(repoRoot, filepath.FromSlash(p))
}

// recordFallback appends one fallback receipt to the run's state and a line to
// its record, under the lock. A receipt that is not "" is the return path of a
// review the reviewer route did not return: its await, which the host's agent
// takes now, drops any answer the route was recorded as placing, in this same
// write. A review whose placed return the verifier refused relies on it: the
// return is taken back without the lock, and this is the drop.
func recordFallback(repoRoot, runID, laneID, receipt string, fb runner.FallbackReceipt, o Options) error {
	return mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		now := o.now()
		if receipt != "" {
			st.dropAnswer(repoRoot, receipt)
		}
		fb.At = now
		st.Fallbacks = append(st.Fallbacks, fb)
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageFallback,
			Note: fmt.Sprintf("the %s was routed to %s, which was %s (%s); %s runs it", fb.Role, fb.Asked, fb.Reason, fb.Detail, fb.Ran)})
		st.UpdatedAt = now
		return true, nil
	})
}

// stampRoute names the route that ran the agent on what the verifier recorded
// from its receipt: the implementer's verified receipt, or the validator's
// run whose return it was.
func stampRoute(lane *Lane, verified string, route *runner.RouteRecord) {
	for i := len(lane.Receipts) - 1; i >= 0; i-- {
		if lane.Receipts[i].Receipt == verified && lane.Receipts[i].Route == nil {
			r := *route
			lane.Receipts[i].Route = &r
			break
		}
	}
	if n := len(lane.Validation); n > 0 {
		vs := lane.Validation[n-1].Validators
		for i := range vs {
			if vs[i].Return == verified && vs[i].Route == nil {
				r := *route
				vs[i].Route = &r
			}
		}
	}
}

// routeNote is the record's line for a role a runner ran.
func routeNote(role string, r runner.RouteRecord) string {
	model := r.Model
	if model == "" {
		model = "none reported"
	}
	under := ""
	if r.Override != "" {
		under = ", under --route " + r.Override
	}
	if r.Asked == r.Ran {
		return fmt.Sprintf("the %s ran on the %s runner (model %s%s)", role, r.Ran, model, under)
	}
	return fmt.Sprintf("the %s, routed to %s, ran on the %s runner (model %s%s)", role, r.Asked, r.Ran, model, under)
}
