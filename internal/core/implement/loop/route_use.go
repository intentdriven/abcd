package loop

// route_use.go holds `implement step --route` to its own rule
// (iss-2610161801358637): a --route is refused, never merged or ignored. A
// step hands at most one role to an agent, so a --route naming a role the
// step will not hand out is refused before anything is written, naming the
// role the step hands out, or that it hands none. The step's move is read
// from the state as move would make it; the one case the state cannot settle
// is a landing ahead of the role, which moves unless the forge or the network
// holds it: the route for that role is then judged after the move, and one
// the move did not use is named in the record and the result.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// StageRoute is the record's stage for a --route the step did not use.
const StageRoute = "route"

// stepHand is the move a step will make, as far as its role goes: the role it
// hands an agent ("" for none), what it does instead (none), and the landing
// that may move first (landing, "" when none may).
type stepHand struct {
	role, lane, none, landing string
}

// predictHand reads, from the state alone, the move advance's move would
// make after the window and the run's completion are judged. It makes move's
// decisions through the functions move makes them with (binaryMoves, mayWait,
// holdsMove and chooseAgent, schedule.go), so the two cannot drift apart.
func predictHand(st State, g *outageGate, now time.Time) stepHand {
	if st.Complete() {
		return stepHand{none: "every lane of " + st.RunID + " is done"}
	}
	if st.NextEligibleAt == nil {
		if _, ok := windowElapsed(st, now); ok {
			return stepHand{none: "the run's window has elapsed, so the step pauses the run"}
		}
	}
	var h stepHand
	for _, w := range binaryMoves(st) {
		l := st.Lanes[w.lane]
		if g.holdsMove(st, w) {
			// The network outage holds it: it waits without running.
			continue
		}
		if mayWait(st, w) {
			// A landing may wait on the forge or the network, and then the
			// step moves on; only the first such is named.
			if h.landing == "" {
				h.landing = l.ID
			}
			continue
		}
		return stepHand{none: fmt.Sprintf("the step performs %s's %s stage, which abcd runs itself", l.ID, l.Stage)}
	}
	h.role, h.lane, h.none = agentHand(st, g)
	return h
}

// agentHand says chooseAgent's choice, the move moveAgent makes: the role and
// its lane, or why it hands none.
func agentHand(st State, g *outageGate) (role, lane, none string) {
	c := chooseAgent(st, g)
	switch c.pick {
	case pickNone:
		if out := st.allAwaits(); len(out) > 0 {
			parts := make([]string, 0, len(out))
			for _, a := range out {
				parts = append(parts, a.Lane+"'s "+a.Await.Role)
			}
			return "", "", "every agent the run has work for is already out (" + strings.Join(parts, ", ") + ")"
		}
		return "", "", "no lane has work for an agent"
	case pickCeiling:
		return "", "", fmt.Sprintf("the run's ceiling of %d agent(s) is reached", st.ceiling())
	case pickOutage:
		return "", "", "the run's outage holds every agent it has work for"
	case pickNoHelper:
		return "", "", "no helper is free for a new lane"
	}
	if c.want.open >= 0 {
		return "", "", fmt.Sprintf("the step opens a lane for spec step %d, whose first stage, the worktree, abcd runs itself", c.want.step)
	}
	return c.want.role, st.Lanes[c.want.lane].ID, ""
}

// checkRoutes refuses each --route (overrides, role to text) the step will
// not use, and returns the role whose route a landing ahead of it leaves to
// the move, or "".
func checkRoutes(st State, overrides map[string]string, g *outageGate, now time.Time) (string, error) {
	h := predictHand(st, g, now)
	roles := make([]string, 0, len(overrides))
	for r := range overrides {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	for _, role := range roles {
		if role == h.role {
			continue
		}
		var does string
		switch {
		case h.role == "" && h.landing != "":
			does = fmt.Sprintf("this step moves %s's landing and hands no role to an agent: %s", h.landing, h.none)
		case h.role == "":
			does = "this step hands no role to an agent: " + h.none
		case h.landing != "":
			does = fmt.Sprintf("this step lands %s, or, if that landing waits, hands out %s's %s", h.landing, h.lane, h.role)
		default:
			does = fmt.Sprintf("this step hands out %s's %s", h.lane, h.role)
		}
		remedy := "step again without that --route; nothing was written"
		if h.role != "" {
			remedy = fmt.Sprintf("step again without that --route, or route the %s (%s); nothing was written", h.role, oracle.RunnerRouteSyntax)
		}
		return "", refuse(StageRoute, "", h.lane, fmt.Sprintf("--route %s routes the %s, and %s", overrides[role], role, does), remedy)
	}
	if h.role != "" && h.landing != "" {
		return h.role, nil
	}
	return "", nil
}

// noteUnusedRoute records, when the move did not hand out unsure's role, that
// its --route was not used, in the record and the result, and reports whether
// it wrote anything.
func noteUnusedRoute(st *State, res *StepResult, overrides map[string]string, unsure string, now time.Time) bool {
	if unsure == "" || (res.handed && res.Awaiting != nil && res.Awaiting.Role == unsure) {
		return false
	}
	text := overrides[unsure]
	did := "handed out no " + unsure
	if res.PerformedStage != "" {
		did = fmt.Sprintf("performed %s's %s stage instead of handing out the %s", res.Lane, res.PerformedStage, unsure)
	}
	st.Record = append(st.Record, Entry{At: now, Lane: res.Lane, Stage: StageRoute,
		Note: fmt.Sprintf("--route %s was not used: a landing ahead of it moved, and the step %s", text, did)})
	st.UpdatedAt = now
	res.RouteUnused = append(res.RouteUnused, text)
	return true
}
