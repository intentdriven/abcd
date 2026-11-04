package loop

// receipt_standing.go keeps a finished agent's work when the step that would
// have collected it was refused (iss-2611031337115764). A routed agent's
// receipt is handed back by the step that started it, once the agent ends, so a
// refusal at that moment (a run state a lane's abcd had raised, a lock, a
// verifier's passing fault) left the agent finished, its commits on the branch
// and its receipt in place, and every later step asking for a fresh agent:
// the work was lost, or paid for twice.
//
// So a receipt standing at a path a lane awaits is read before anything asks
// for an agent, and verified exactly as `implement receipt` would verify it,
// through the same hand-in under the same lock. Every door a fresh agent comes
// through reads it: a step's move (collectStanding, which any await a step
// would otherwise re-tell passes through), the implement stage when no await is
// out (implementStage), and the launch of a runner, which a --route for a held
// role reaches before the step's own move (dispatchAwait), where a --route that
// so starts nothing is named as unused. A step carrying a --route for a role it
// will not hand out stays refused (route_use.go), and the next plain step
// collects.
//
// A receipt that verifies is taken. It may be taken over a runner's claim still
// standing: a refused step could not drop its claim, and the claim names no
// process to ask, whereas a receipt that verifies is the agent's last act, and
// the runner whose dispatch does end after it finds its hand-in answered. One
// that does not verify is left where it stands, the await stays out, and the
// step's answer says why, since a person or an agent can correct it.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// collectStanding is a step's read of its lanes' awaited receipts. It hands in
// the first that stands and verifies, in lane order, and reports true with the
// lane's result: that is the step's one move, and the next step takes the lane
// on. Otherwise it reports false, with what the step should say of each that
// stands and does not verify. A run that cannot be read gives neither, for the
// step's own read to refuse in its own frame.
func collectStanding(repoRoot, runID string, steps Stages, o Options) (StepResult, bool, string) {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return StepResult{}, false, ""
	}
	var notes []string
	for _, i := range st.laneOrder() {
		l := st.Lanes[i]
		if l.Stage == StageDone || l.Stage == StageDiscarded || l.HandBack != nil {
			continue
		}
		for _, a := range l.Awaits {
			if !received(repoRoot, a.Receipt) {
				continue
			}
			route := claimedRoute(a)
			res, err := handIn(repoRoot, runID, a.Receipt, steps, o, route, true)
			if err == nil {
				res.Next = fmt.Sprintf("%s's %s had already written its receipt at %s, which verified, so no fresh agent is needed; %s",
					l.ID, a.Role, a.Receipt, res.Next)
				if route != nil {
					res.Route = route
				}
				return res, true, ""
			}
			rf, refused := AsRefusal(err)
			if refused && rf.Contention {
				// Another invocation holds the run: the step's own write
				// meets the same lock and answers for it.
				return StepResult{}, false, ""
			}
			notes = append(notes, fmt.Sprintf("a receipt stands at %s and does not verify (%s); it is left there: correct it and run `abcd implement receipt %s`",
				a.Receipt, refusalReason(err), a.Receipt))
		}
	}
	return StepResult{}, false, strings.Join(notes, "; ")
}

// takeStanding is implementStage's read of a receipt standing at the path it
// would await, with no await out: on a copy of the lane, so that a refusal
// leaves the lane as it was. It returns the stage's outcome when the receipt
// verifies, and otherwise why it did not.
func takeStanding(c Context, lane *Lane, await *Await) (Outcome, string) {
	trial := *lane
	trial.Receipts, trial.Resolves = slices.Clone(lane.Receipts), slices.Clone(lane.Resolves)
	c.Await = await
	if err := verifyReceipt(c, &trial, await.Receipt); err != nil {
		return Outcome{}, refusalReason(err)
	}
	*lane = trial
	if lane.HandBack != nil {
		hb := *lane.HandBack
		return Outcome{HandBack: &hb}, ""
	}
	lane.Receipt = await.Receipt
	return Outcome{Note: fmt.Sprintf("the %s stage's receipt verified at %s; it stood at the path the stage would await, so no fresh agent was asked for",
		lane.Stage, await.Receipt)}, ""
}

// claimedRoute is the route a runner's claim on the await names, nil when no
// runner claimed it. A collected receipt is stamped with it, so a run a runner
// made is not recorded as the host's; the runner's own report was lost with the
// refused step, so it names no model.
func claimedRoute(a Await) *runner.RouteRecord {
	if a.Running == nil {
		return nil
	}
	return &runner.RouteRecord{Asked: a.Running.Runner, Ran: a.Running.Runner}
}

// refusalReason is a refusal's reason, without its remedy, or an error's text.
func refusalReason(err error) string {
	if rf, ok := AsRefusal(err); ok {
		return rf.Reason
	}
	return err.Error()
}

// laneThatTook is the lane that has already verified the receipt at path, and
// whether there is one: an implementer's receipt, a validator's return with
// its verdict recorded, or a plan review's.
func (s State) laneThatTook(repoRoot, path string) (Lane, bool) {
	same := func(p string) bool { return p != "" && samePath(repoRoot, path, p) }
	for _, l := range s.Lanes {
		if slices.ContainsFunc(l.Receipts, func(r ReceiptRecord) bool { return same(r.Receipt) }) {
			return l, true
		}
		for _, round := range l.Validation {
			if slices.ContainsFunc(round.Validators, func(v ValidatorRun) bool { return v.Verdict != "" && same(v.Return) }) {
				return l, true
			}
		}
		if l.AutoPlan != nil && slices.ContainsFunc(l.AutoPlan.Reviews, func(r PlanReview) bool { return r.Verdict != "" && same(r.Return) }) {
			return l, true
		}
	}
	return Lane{}, false
}
