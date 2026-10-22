package loop

// escalate.go is the tier escalation of itd-2609170822093401's Decision 1 and
// spc-2609251028149555's AC 10: a lane starts each role at the tier its
// routing row names, and a round whose gates fail moves the next round's tier
// for that role one step up the ladder local, economy, frontier
// (oracle.Escalate). It is a rule, not a router: the signal is the loop's own
// fact that a round did not pass, and the switch is recorded in the run's state
// beside the round, as the round's Routes, with the failing gate output that
// caused it.
//
// The tier of a role's round n+1 is oracle.Escalate of its tier in round n,
// never its base tier moved twice in one step. The role's row is the floor: a
// row raised during the run takes effect at the next round, and the step up
// is taken from the higher of the two. A round that passed moves nothing: a
// fresh round after it (a sync with the default branch) keeps the tier the
// lane reached, so a lane never drops back below the tier its failure took it
// to, and a role new to the lane (the auditor joining the closing lane)
// starts at its own row's tier.
//
// Only a route that resolves from a tier can move. A role the step hands to a
// runner names a runner and its model, a role pointed at a provider names a
// model, host-decides names no tier, and the implementer has no routing row
// (the routing tables cover the agents in the roster); each is recorded with
// the failure and a note saying its route names no tier, and none is ever
// swapped for another route. An escalated tier is re-resolved through
// oracle.ResolveAt, so a connection that would serve it is held to its
// allowlist: a route the allowlist refuses refuses the step, naming the
// provider and the route, and nothing is written.

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// maxGateOutputBytes bounds the failing gate output a round's route records.
const maxGateOutputBytes = 2048

// Tiering is the model-tier routing a step resolves each role's tier through:
// the routing tables (oracle.Load) and the machine's provider connections
// (oracle.APIConfig.Connections). A step with none records no route.
type Tiering struct {
	Table *oracle.Layered
	Conns oracle.Connections
	// Reviewer is the reviewer route oracle.review selects (ReviewRoute.Name),
	// which takes the ruthless and the security review the stage hands to the
	// host; "" when it selects none.
	Reviewer string
}

// StageRouting is the refusal stage of a routing table or provider
// configuration the step cannot read.
const StageRouting = "routing"

// LoadTiering reads the model-tier routing a step resolves each role's tier
// through, at the roots a step starts from: the routing tables (oracle.Load)
// and the machine's provider connections (oracle.LoadAPI), with reviewer the
// reviewer route the step sends its reviews to (nil for none). A fault is
// refused in the loop's refusal shape before anything moves, as a delegating
// verb refuses it: the table decides which tier a round asks for, so it is
// never guessed past. The tables' diagnostics (an orphan row) are the
// caller's to print.
func LoadTiering(r layered.Roots, rr *ReviewRoute) (*Tiering, []string, error) {
	// A reviewer route with no server on this machine takes no review: every
	// one falls back to the host, which is asked for the review's tier.
	reviewer := ""
	if rr != nil && rr.Absent == "" {
		reviewer = rr.Name
	}
	api, err := oracle.LoadAPI(r)
	if err != nil {
		return nil, nil, refuse(StageRouting, "", "", err.Error(),
			"correct the provider configuration the reason names; nothing moved and nothing is written")
	}
	l, err := oracle.Load(r)
	if err != nil {
		return nil, nil, refuse(StageRouting, "", "", err.Error(),
			"correct the routing table the reason names; nothing moved and nothing is written")
	}
	return &Tiering{Table: l, Conns: api.Connections(), Reviewer: reviewer}, l.Diagnostics, nil
}

// roundRouter decides the route a role handed out in a round runs at.
type roundRouter struct {
	tiers *Tiering
	// runner names the runner the step hands role to and the --route text
	// that put it there; "" for the host.
	runner func(role string) (name, override string)
}

// newRoundRouter is the router a step with o hands its stages; nil when the
// step read no routing, so it records nothing.
func newRoundRouter(o Options) *roundRouter {
	if o.Tiers == nil || o.Tiers.Table == nil {
		return nil
	}
	return &roundRouter{tiers: o.Tiers, runner: o.runnerRoute}
}

// route is role's route in round n of lane: after is the failed round it
// follows, nil when it follows none. The second result is the resolved route
// when the role runs at a tier, for the brief's routing section.
func (rt *roundRouter) route(lane Lane, n int, role string, after *ValidationRound) (Round, *oracle.Route, error) {
	r := Round{Role: role}
	if after != nil {
		r.GateOutput = gateOutput(*after)
	}
	if rt.runner != nil {
		if name, override := rt.runner(role); name != "" {
			under := ""
			if override != "" {
				under = " under --route " + override
			}
			r.Note = fmt.Sprintf("the %s runs on the %s runner%s, a route that names a runner and its model, not a tier, so it does not move", role, name, under)
			return r, nil, nil
		}
	}
	if _, reviews := reviewerVerdicts[role]; reviews && rt.tiers.Reviewer != "" {
		r.Note = fmt.Sprintf("the %s is sent to the %s reviewer route, which names no tier, so it does not move", role, rt.tiers.Reviewer)
		return r, nil, nil
	}
	if _, ok := oracle.Ceiling(role); !ok {
		r.Note = fmt.Sprintf("the %s has no row in the model-tier routing, which covers the agents in the roster, so its route names no tier and does not move", role)
		return r, nil, nil
	}
	conns := rt.tiers.Conns
	if conns == nil {
		conns = oracle.NoConnections{}
	}
	// A pointed role is judged before its row is resolved: the loop never
	// sends it to that provider, so the provider's refusal of the row's
	// settings is not this step's to make.
	if c, ok := conns.Pointed(role); ok {
		r.Note = fmt.Sprintf("the %s is pointed at %s by oracle.roles.%s, a route that names a model, not a tier, so it does not move", role, c.Name, role)
		return r, nil, nil
	}
	base, err := oracle.Resolve(role, rt.tiers.Table, conns)
	if err != nil {
		return Round{}, nil, refuse(string(StageValidate), "", lane.ID, fmt.Sprintf("the %s's route for round %d is refused: %v", role, n, err),
			"correct the routing the reason names; nothing was written")
	}
	if base.Row.Tier == oracle.HostDecides {
		r.Note = fmt.Sprintf("the %s's route is %s, which names no tier, so it does not move", role, oracle.HostDecides)
		return r, nil, nil
	}
	// The row is the floor: a row raised during the run takes effect at the
	// next round, and a lowered one never takes the lane back down.
	from := base.Row.Tier
	if prev, ok := priorTier(lane, n, role); ok && rank(prev) > rank(from) {
		from = prev
	}
	r.Tier = from
	if after != nil {
		r.Tier = oracle.Escalate(from)
		if r.Tier == from {
			r.Note = fmt.Sprintf("the %s is at %s, the top of the ladder, so it does not move", role, from)
		} else {
			r.EscalatedFrom = from
		}
	}
	if r.Tier == base.Row.Tier {
		return r, &base, nil
	}
	at, err := oracle.ResolveAt(role, rt.tiers.Table, conns, r.Tier)
	if err != nil {
		return Round{}, nil, refuse(string(StageValidate), "", lane.ID,
			fmt.Sprintf("the %s's round %d moves to tier %s after round %d did not pass, and that route is refused: %v", role, n, r.Tier, n-1, err),
			"correct the routing the reason names, so the escalated tier resolves to a route it admits; nothing was written")
	}
	return r, &at, nil
}

// priorTier is the tier role ran at in the latest round of lane before round
// n that recorded one.
func priorTier(lane Lane, n int, role string) (oracle.Tier, bool) {
	for i := min(n-1, len(lane.Validation)) - 1; i >= 0; i-- {
		for _, r := range lane.Validation[i].Routes {
			if r.Role == role && r.Tier != "" {
				return r.Tier, true
			}
		}
	}
	return "", false
}

// rank is a tier's place on the ladder local, economy, frontier.
func rank(t oracle.Tier) int {
	switch t {
	case oracle.Economy:
		return 1
	case oracle.Frontier:
		return 2
	}
	return 0
}

// failedBefore is the round round n of lane follows when that round did not
// pass and its findings went to a fresh implementer, nil otherwise.
func failedBefore(lane Lane, n int) *ValidationRound {
	if n < 2 || n-2 >= len(lane.Validation) || lane.Validation[n-2].Fix == "" {
		return nil
	}
	r := lane.Validation[n-2]
	return &r
}

// recordRoute places role's route on round r, replacing an earlier record
// for the role (a step repeated after a kill writes it again).
func recordRoute(r *ValidationRound, route Round) {
	for i := range r.Routes {
		if r.Routes[i].Role == route.Role {
			routes := append([]Round(nil), r.Routes...)
			routes[i] = route
			r.Routes = routes
			return
		}
	}
	r.Routes = append(append([]Round(nil), r.Routes...), route)
}

// routeRound records, on round r of lane, the route role runs at when the
// step reads routing (c.router), and returns the resolved route when it runs
// at a tier, for its brief, and the record's words for it.
func routeRound(c Context, lane Lane, r *ValidationRound, role string, after *ValidationRound) (*oracle.Route, string, error) {
	if c.router == nil {
		return nil, "", nil
	}
	rr, route, err := c.router.route(lane, r.Round, role, after)
	if err != nil {
		return nil, "", err
	}
	recordRoute(r, rr)
	return route, routeNoteOf(rr), nil
}

// routingSection is the brief's routing section for a role asked to run at a
// tier: the request block a delegating verb hands the host
// (oracle.RenderRequestSection), so the tier the round asks for travels with
// the brief. "" when the role's route names no tier.
func routingSection(route *oracle.Route) string {
	if route == nil {
		return ""
	}
	return "## Routing\n\nThis round asks for the model tier below (itd-2609170822093401):\n\n```\n" +
		oracle.RenderRequestSection(route.Request()) + "```\n\n"
}

// roundTier is the tier the round asks a's role to run at, when a is an
// agent of lane's current round with a route that names one; "" otherwise.
func roundTier(lane Lane, a Await) oracle.Tier {
	n := len(lane.Validation)
	if lane.Stage != StageValidate || n == 0 {
		return ""
	}
	for _, r := range lane.Validation[n-1].Routes {
		if r.Role == a.Role {
			return r.Tier
		}
	}
	return ""
}

// routeNoteOf is the record's words for a route that moved, or "".
func routeNoteOf(r Round) string {
	switch {
	case r.EscalatedFrom != "":
		return fmt.Sprintf("; the %s moves up from %s to %s after the round before did not pass", r.Role, r.EscalatedFrom, r.Tier)
	case r.Tier != "":
		return fmt.Sprintf("; the %s is asked for tier %s", r.Role, r.Tier)
	}
	return ""
}

// gateOutput is the failing output of round r: its verdicts as the loop
// recorded them, the returns of the validators that did not pass, and the
// criteria the audit judged not met or could not decide, bounded.
func gateOutput(r ValidationRound) string {
	var b strings.Builder
	fmt.Fprintf(&b, "round %d did not pass (%s)", r.Round, verdictsLine(r))
	for _, v := range r.Validators {
		if v.Pass || v.Verdict == "" {
			continue
		}
		fmt.Fprintf(&b, "; %s %s: %s", v.Role, v.Verdict, v.Return)
		if v.Audit != nil {
			if len(v.Audit.NotMet) > 0 {
				fmt.Fprintf(&b, ", not met: %s", strings.Join(v.Audit.NotMet, ", "))
			}
			if len(v.Audit.Inconclusive) > 0 {
				fmt.Fprintf(&b, ", undecided: %s", strings.Join(v.Audit.Inconclusive, ", "))
			}
		}
	}
	return boundGateOutput(b.String())
}

// boundGateOutput cuts s to maxGateOutputBytes on a rune boundary.
func boundGateOutput(s string) string {
	if len(s) <= maxGateOutputBytes {
		return s
	}
	cut := maxGateOutputBytes - len("...")
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "..."
}
