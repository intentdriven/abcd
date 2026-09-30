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
//     receipt the verifier refuses leaves the lane awaiting, writes one
//     fallback receipt into the run's state (Fallbacks) and the record, and
//     hands the role to the host with the reason (criterion 3), which the run
//     record counts per runner and per role (criterion 4).
//
// The runner is started outside the run's lock, which is held only for the
// advance before it and the Receipt or the fallback write after it, so a
// status read or another checkout's step is never kept waiting on a model.
//
// A host session always drives this call: the no-host path, where abcd runs
// alone and a host-routed role goes to runner.fallback_host, is the process
// driver's reversal of the host-delegated boundary, which the ADR decision 6
// of itd-2609201916151817 owes records before either path ships.

import (
	"context"
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
// the runner and hands its receipt back through Receipt. A step that re-tells
// an await an earlier call began starts nothing. A role on the host, or a runner
// that did not run it, returns the await for the host to act on, as advance
// returns it; the latter also names the fallback it recorded.
func Drive(ctx context.Context, repoRoot, runID string, steps Stages, o Options, rs Runners) (StepResult, error) {
	res, err := advance(repoRoot, runID, steps, o)
	if err != nil || res.Awaiting == nil || !res.handed || rs.Config == nil {
		// Nothing awaits, or the await is one an earlier call began: the
		// host, or the runner that call started, is already on it.
		return res, err
	}
	aw := *res.Awaiting
	route := rs.Config.RouteFor(aw.Role)
	if route.Runner == runner.Host {
		return res, nil
	}
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return res, err
	}
	var worktree string
	for _, l := range st.Lanes {
		if l.ID == res.Lane {
			worktree = l.Worktree
		}
	}
	req := runner.Request{
		Role:      aw.Role,
		Brief:     absIn(repoRoot, aw.Brief),
		Receipt:   absIn(repoRoot, aw.Receipt),
		Dir:       worktree,
		Checkout:  filepath.Clean(repoRoot),
		Tools:     toolsFor(aw.Role),
		SessionID: fmt.Sprintf("%s-%s-%s-%d", runID, res.Lane, aw.Role, o.now().Unix()),
		Timeout:   rs.Timeout,
	}
	var done *StepResult
	d := &runner.Dispatcher{
		Config:      rs.Config,
		HostSession: true,
		Transcripts: rs.Transcripts,
		Validate: func(ran string, _ runner.Request, ans runner.Answer) error {
			r, err := receipt(repoRoot, runID, aw.Receipt, steps, o,
				&runner.RouteRecord{Asked: route.Runner, Ran: ran, Model: ans.Model})
			if err != nil {
				return err
			}
			done = &r
			return nil
		},
		Record: func(fb runner.FallbackReceipt) error { return recordFallback(repoRoot, runID, res.Lane, fb, o) },
		Now:    o.Now,
	}
	out, err := d.Dispatch(ctx, req)
	if err != nil {
		return res, refuse(StageRunner, "", res.Lane, err.Error(),
			"the lane still awaits its receipt: correct what the reason names and step again, or start the agent the step names by hand")
	}
	if out.Handoff {
		res.Fallback = out.Fallback
		if fb := out.Fallback; fb != nil {
			res.Next = fmt.Sprintf("the %s runner did not run the %s (%s; recorded as a fallback), so the host runs it: %s",
				fb.Asked, fb.Role, fb.Reason, res.Next)
		}
		return res, nil
	}
	if done == nil {
		return res, refuse(StageRunner, "", res.Lane, "the runner ran the "+aw.Role+" but no receipt was handed back",
			"the lane still awaits its receipt; step again")
	}
	r := *done
	r.Route = &out.Receipt.Route
	return r, nil
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
// its record, under the lock.
func recordFallback(repoRoot, runID, laneID string, fb runner.FallbackReceipt, o Options) error {
	return mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		now := o.now()
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
	if r.Asked == r.Ran {
		return fmt.Sprintf("the %s ran on the %s runner (model %s)", role, r.Ran, model)
	}
	return fmt.Sprintf("the %s, routed to %s, ran on the %s runner (model %s)", role, r.Asked, r.Ran, model)
}
