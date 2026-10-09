package loop

// budget.go is the budget check a run takes before it starts
// (itd-2609201925079472 criterion 7, decision 6, from itd-29;
// spc-2609301921521360). Each route the run's roles go through is asked for
// the quota it has left (runner.Config.QuotaFor): where one reports it, the
// share of the run's estimate its roles will spend is compared with it, and a
// run that exceeds it is refused naming both numbers before anything is
// written; a route that reports none (the host, whose quota abcd cannot read,
// and every shipped runner today) is named and the check is skipped out loud,
// in the result and in the run record's first lines.
//
// The estimate is itd-29's: about 80000 tokens an agent, an agent for each
// role a lane's first round takes (its implementer, the ruthless and the
// security reviewer), on every step the run opens a lane for, and the
// intent-auditor once on the lane that closes the spec (an issue run takes no
// audit). It is the run's first round, not its worst case: fix rounds are not
// counted, and a run that spends past its quota on them is what the rate-limit
// checkpoint (ratelimit.go) ends the window for.

import (
	"context"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// EstimateTokensPerAgent is the tokens the estimate counts for each agent a
// run starts (itd-29: "tasks × ~80k tokens").
const EstimateTokensPerAgent = 80000

// StageBudget is the refusal stage, and the run record's, of the budget check.
const StageBudget = "budget"

// QuotaReader reads the quota a route reports: ok false when it reports none.
type QuotaReader func(ctx context.Context, route string) (runner.Quota, bool, error)

// BudgetRow is one route the run's roles go through, with its share of the
// estimate and what it reported.
type BudgetRow struct {
	// Route is the host or a runner.
	Route string `json:"route"`
	// Roles are the roles the run hands this route.
	Roles []string `json:"roles"`
	// EstimateTokens is the estimate of what those roles spend over the run.
	EstimateTokens int64 `json:"estimate_tokens"`
	// RemainingTokens is the quota the route reported; null when it reports
	// none and its check was skipped.
	RemainingTokens *int64 `json:"remaining_tokens"`
	// Checked is true when the route reported a quota and it covered the
	// estimate.
	Checked bool `json:"checked"`
	// Detail is the row in words, as the run record carries it.
	Detail string `json:"detail"`
}

// Budget is the budget check a new run took.
type Budget struct {
	// Steps are the spec steps the run opens lanes for.
	Steps int `json:"steps"`
	// TokensPerAgent is the estimate's figure for one agent.
	TokensPerAgent int64       `json:"tokens_per_agent"`
	Rows           []BudgetRow `json:"rows"`
}

// String is the run record's line for the check.
func (b Budget) String() string {
	parts := make([]string, 0, len(b.Rows))
	for _, r := range b.Rows {
		parts = append(parts, r.Detail)
	}
	return fmt.Sprintf("the budget check, at about %d tokens an agent over %d step(s): %s",
		b.TokensPerAgent, b.Steps, strings.Join(parts, "; "))
}

// runRoles are the agents a run starts for each lane's first round, and the
// auditor it starts once for an intent's run.
var runRoles = []string{RoleImplementer, RoleRuthless, RoleSecurity}

// checkBudget estimates what the run will spend on each route its roles go
// through and compares it with the quota each reports, refusing at the budget
// stage a route the estimate exceeds or whose report fails. It writes nothing.
func checkBudget(steps int, audits bool, o Options) (*Budget, error) {
	roles := append([]string(nil), runRoles...)
	if audits {
		roles = append(roles, RoleAuditor)
	}
	var order []string
	byRoute := map[string]*BudgetRow{}
	for _, role := range roles {
		route := runner.Host
		if o.Runners != nil {
			route = o.Runners.RouteFor(role).Runner
		}
		row, ok := byRoute[route]
		if !ok {
			row = &BudgetRow{Route: route}
			byRoute[route] = row
			order = append(order, route)
		}
		row.Roles = append(row.Roles, role)
		agents := int64(steps)
		if role == RoleAuditor {
			agents = 1
		}
		row.EstimateTokens += agents * EstimateTokensPerAgent
	}
	read := o.Quota
	if read == nil {
		read = o.Runners.QuotaFor
	}
	b := &Budget{Steps: steps, TokensPerAgent: EstimateTokensPerAgent}
	for _, route := range order {
		row := *byRoute[route]
		name := routeName(route)
		q, ok, err := read(context.Background(), route)
		if err != nil {
			return nil, refuse(StageBudget, "", "", fmt.Sprintf("%s did not report its quota: %v", name, err),
				"correct what the reason names and start again; no state is written")
		}
		estimate := fmt.Sprintf("%d tokens for the %s", row.EstimateTokens, strings.Join(row.Roles, ", "))
		if !ok {
			row.Detail = fmt.Sprintf("%s reports no quota, so its check is skipped (an estimate of %s)", name, estimate)
			b.Rows = append(b.Rows, row)
			continue
		}
		left := q.RemainingTokens
		row.RemainingTokens = &left
		if row.EstimateTokens > left {
			return nil, refuse(StageBudget, "", "", fmt.Sprintf("%s reports %d tokens of quota left, and the run's estimate is %s "+
				"(%d step(s) at about %d tokens an agent)", name, left, estimate, steps, EstimateTokensPerAgent),
				"start it once the quota has reset, route its roles elsewhere, or split the spec into fewer steps a run; no state is written")
		}
		row.Checked = true
		row.Detail = fmt.Sprintf("%s reports %d tokens of quota left, which covers an estimate of %s", name, left, estimate)
		b.Rows = append(b.Rows, row)
	}
	return b, nil
}

// routeName names a route in a sentence.
func routeName(route string) string {
	if route == runner.Host {
		return "the host"
	}
	return "the " + route + " runner"
}
