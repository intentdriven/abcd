package loop

// budget.go is the budget check (itd-2609201925079472 criterion 7, from
// itd-29; spc-2609301921521360): before a run is created, each runner a role
// of the run is routed to is asked for its remaining quota, and the run's
// estimate on that runner is compared with it. A run the estimate exceeds is
// refused naming both numbers, and nothing is written. A runner that reports
// no quota, the host included, is named and the check skipped for it out loud,
// in the start's checks and in the run record.
//
// The estimate is the least the spec's size makes the run start: per step to
// build, one implementer and one round of the two reviewers, and the
// fidelity audit once for an intent (an issue has none). Fix rounds and syncs
// are not foreseen, so an estimate under a quota is no promise the quota
// holds; one over it is a run that cannot finish in the window. Quota and
// estimate are counted in agent runs, the unit runner.Quota names.

import (
	"context"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// CheckBudget is the budget check's row among the start's checks, and the
// run record's stage for it.
const CheckBudget = "budget"

// budgetShare is the part of the estimate one route carries.
type budgetShare struct {
	route string
	roles []string
	runs  int
}

// estimateShares is the run's estimate per route, the routes in the order the
// roles first name them: steps implementers and as many rounds of the two
// reviewers, and, for an intent, the auditor once.
func estimateShares(cfg *runner.Config, steps int, audits bool) []budgetShare {
	roles := []struct {
		role string
		runs int
	}{{RoleImplementer, steps}, {RoleRuthless, steps}, {RoleSecurity, steps}}
	if audits {
		roles = append(roles, struct {
			role string
			runs int
		}{RoleAuditor, 1})
	}
	var out []budgetShare
	for _, r := range roles {
		route := runner.Host
		if cfg != nil {
			route = cfg.RouteFor(r.role).Runner
		}
		k := -1
		for i := range out {
			if out[i].route == route {
				k = i
			}
		}
		if k < 0 {
			out = append(out, budgetShare{route: route})
			k = len(out) - 1
		}
		out[k].roles = append(out[k].roles, r.role)
		out[k].runs += r.runs
	}
	return out
}

// quota asks the named runner for its remaining quota: through the Quota seam
// when a test sets it, else the runner configuration.
func (o Options) quota(name string) (runner.Quota, bool, error) {
	if o.Quota != nil {
		return o.Quota(name)
	}
	if o.Runners == nil {
		return runner.Quota{}, false, nil
	}
	return o.Runners.Quota(context.Background(), name)
}

// budgetCheck is the budget row for a run of chk: the estimate on each route
// against the quota its runner reports. It fails only when a runner reports a
// quota the estimate on it exceeds.
func budgetCheck(chk CheckResult, o Options) CheckRow {
	steps := len(chk.steps)
	audits := chk.Intent != ""
	shares := estimateShares(o.Runners, steps, audits)
	total := 0
	for _, s := range shares {
		total += s.runs
	}
	each := "an implementer, a ruthless-reviewer and a security-reviewer"
	if audits {
		each += ", and the intent-auditor once"
	}
	head := fmt.Sprintf("an estimate of %d agent run(s) from %d step(s), each %s", total, steps, each)

	var parts, over []string
	compared := false
	for _, s := range shares {
		who := "the " + s.route + " runner"
		if s.route == runner.Host {
			parts = append(parts, fmt.Sprintf("the host reports no quota (%s: an estimated %d agent run(s)), so the check is skipped for it",
				strings.Join(s.roles, ", "), s.runs))
			continue
		}
		q, ok, err := o.quota(s.route)
		switch {
		case err != nil:
			parts = append(parts, fmt.Sprintf("%s did not report its quota (%s), so the check is skipped for it (%s: an estimated %d agent run(s))",
				who, termsafe.Sanitize(err.Error()), strings.Join(s.roles, ", "), s.runs))
		case !ok:
			parts = append(parts, fmt.Sprintf("%s reports no quota (%s: an estimated %d agent run(s)), so the check is skipped for it",
				who, strings.Join(s.roles, ", "), s.runs))
		default:
			compared = true
			line := fmt.Sprintf("%s reports %d agent run(s) left, and the run is estimated to start %d on it (%s)",
				who, q.Remaining, s.runs, strings.Join(s.roles, ", "))
			parts = append(parts, line)
			if s.runs > q.Remaining {
				over = append(over, line)
			}
		}
	}
	if len(over) > 0 {
		return CheckRow{Name: CheckBudget, OK: false,
			Detail: "the run's estimate exceeds the quota its runner reports: " + strings.Join(over, "; ") + " (" + head + ")",
			Remedy: "start the run once the runner's quota window has reset, or route the roles it names to another runner in " +
				abcdhome.Display("config.json") + "; nothing was written"}
	}
	detail := head + ": " + strings.Join(parts, "; ")
	if !compared {
		detail = "skipped, as no runner of the run reports a quota: " + detail
	}
	return CheckRow{Name: CheckBudget, OK: true, Detail: detail}
}
