package oracle

import "sort"

// proposed is one agent's bundled row and its contract ceiling.
type proposed struct {
	tier Tier
	// ceiling is the most agents one step of this agent may run at once,
	// itself included. No agent contract under agents/ declares a fan-out
	// ceiling, and every agent in the roster is a single prompt that spawns no
	// sub-agent of its own, so each ceiling is 1 (.abcd/work/DECISIONS.md,
	// 2026-09-25). A contract that grows a ceiling field moves the number
	// there; the roster test keeps this table and agents/ in step either way.
	ceiling int
}

// proposal is abcd's best guess at the tier each agent deserves: frontier for
// the verdicts a human reads and acts on, economy for the rest
// (spc-2609180535002478 § Scope). It is keyed on the agent's name, and
// TestProposalNamesEveryAgentInTheRosterAndNoOther holds it to the agents/
// roster in both directions. Calibrating it from measurements belongs to
// itd-2609221009495079, not to this table.
var proposal = map[string]proposed{
	"intent-auditor":             {Frontier, 1},
	"lifeboat-reviewer":          {Frontier, 1},
	"ruthless-reviewer":          {Frontier, 1},
	"security-reviewer":          {Frontier, 1},
	"release-changelog-composer": {Frontier, 1},
	"cold-reading-comparative":   {Economy, 1},
	"cold-reading-detection":     {Economy, 1},
	"cold-reading-entailment":    {Economy, 1},
	"cold-reading-widening":      {Economy, 1},
	"docs-currency-reviewer":     {Economy, 1},
	"graveyard-interpreter":      {Economy, 1},
	"principle-distiller":        {Economy, 1},
	"press-release-composer":     {Economy, 1},
	"scribe":                     {Economy, 1},
	"sota-researcher":            {Economy, 1},
}

// Roster returns every agent the proposal names, sorted: the runtime roster a
// table's rows are checked against.
func Roster() []string {
	out := make([]string, 0, len(proposal))
	for a := range proposal {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// Proposal returns the bundled table: one row per agent, its tier and its
// fan-out at the contract ceiling. The map is a copy.
func Proposal() Table {
	t := make(Table, len(proposal))
	for a, p := range proposal {
		t[a] = Row{Tier: p.tier, FanOut: p.ceiling}
	}
	return t
}

// Ceiling returns the agent contract's fan-out ceiling; ok is false for a name
// outside the roster.
func Ceiling(agent string) (int, bool) {
	p, ok := proposal[agent]
	return p.ceiling, ok
}

func inRoster(agent string) bool {
	_, ok := proposal[agent]
	return ok
}
