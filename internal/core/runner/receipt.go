package runner

// receipt.go is what a dispatch leaves in the record: the role's receipt
// block, the same for every route but the route it names (criterion 7), and
// the fallback receipt, one per fallback, which Tally counts per runner and
// per role for the run's summary (criteria 3 and 4).

import (
	"fmt"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// RouteRecord is the route a role asked for and the one that ran it, and,
// when a runner ran it, both sides of the run (itd-166).
type RouteRecord struct {
	// Asked is the route the configuration named: host or a runner.
	Asked string `json:"asked"`
	// Ran is the route that ran the role: host, a runner, or none.
	Ran string `json:"ran"`
	// Model is the model the runner that ran it reported, the canonical
	// served model; "" on the host, whose own receipt reports it, and when
	// the runner reported none.
	Model string `json:"model"`
	// Override is the --route text verbatim when the person's flag routed the
	// role for this invocation, so a run on a chosen model is never mistaken
	// for the configured route; "" otherwise.
	Override string `json:"override,omitempty"`
	// Sides is both sides of the run, present when a runner ran it; its
	// fields are omitted when empty, so a host's or an older record reads as
	// it always has.
	Sides
}

// RoleReceipt is a role run's receipt block. Everything but Route is the
// request's, so two runs of one request differ in their route alone.
type RoleReceipt struct {
	Role  string `json:"role"`
	Brief string `json:"brief"`
	// Contract is the receipt path the contract names.
	Contract string      `json:"contract"`
	Route    RouteRecord `json:"route"`
}

func newRoleReceipt(req Request, asked string) RoleReceipt {
	return RoleReceipt{Role: req.Role, Brief: req.Brief, Contract: req.Receipt, Route: RouteRecord{Asked: asked}}
}

// FallbackReceipt is one fallback: the role, the runner asked for, the reason,
// and the route that ran instead.
type FallbackReceipt struct {
	At     time.Time `json:"at"`
	Role   string    `json:"role"`
	Asked  string    `json:"asked"`
	Reason Reason    `json:"reason"`
	// Detail is abcd's own account of the reason; it never carries the
	// harness's output.
	Detail string `json:"detail"`
	Ran    string `json:"ran"`
	// Spent is what the launch spent when it was killed (ErrKilled), read
	// from its transcript; nil for every other failure, which spent nothing
	// a transcript shows or reported its spend as it ended.
	Spent *Spend `json:"spent,omitempty"`
}

// Spend is what a killed launch spent (iss-2611021751406429). A harness
// killed at its bound writes no closing event, so the cost it would have
// reported there never comes: the token counts and any cost its transcript
// does carry are read from it instead, and a launch whose transcript shows no
// cost is charged as one agent run of its runner's budget, the unit Quota
// counts, so its spend is never dropped from the tally.
type Spend struct {
	Usage *oracle.Usage `json:"usage,omitempty"`
	Cost  *oracle.Cost  `json:"cost,omitempty"`
	// ChargedRuns is the agent runs the launch is charged when its
	// transcript shows no cost: one, else zero.
	ChargedRuns int `json:"charged_runs,omitempty"`
}

// spendOf is the spend a killed launch's partial answer shows.
func spendOf(ans Answer) *Spend {
	s := &Spend{Usage: ans.Usage, Cost: ans.Cost}
	if s.Cost == nil {
		s.ChargedRuns = 1
	}
	return s
}

// Text is the spend as a record's line names it.
func (s Spend) Text(runner string) string {
	var out string
	if u := s.Usage; u != nil {
		out = fmt.Sprintf("%d prompt, %d completion, %d cache-read and %d cache-write tokens",
			u.PromptTokens, u.CompletionTokens, u.CacheReadTokens, u.CacheWriteTokens)
	} else {
		out = "no tokens its transcript shows"
	}
	if s.Cost != nil {
		return out + " at a cost of " + strconv.FormatFloat(s.Cost.Amount, 'f', -1, 64) + " " + s.Cost.Unit
	}
	return out + fmt.Sprintf(", and no cost, so it is charged as %d agent run of the %s runner's budget", s.ChargedRuns, runner)
}

// Counts is the fallback intel a run's summary reports.
type Counts struct {
	Total    int            `json:"total"`
	ByRunner map[string]int `json:"by_runner"`
	ByRole   map[string]int `json:"by_role"`
	// CostUSD is what the killed launches among them spent, as their
	// transcripts show it in USD, and ChargedRuns the agent runs charged for
	// those whose transcripts show no cost (Spend).
	CostUSD     float64 `json:"cost_usd,omitempty"`
	ChargedRuns int     `json:"charged_runs,omitempty"`
}

// Tally counts fallback receipts per runner asked for and per role. A run with
// none reports zero, with empty maps rather than none.
func Tally(rs []FallbackReceipt) Counts {
	c := Counts{ByRunner: map[string]int{}, ByRole: map[string]int{}}
	for _, r := range rs {
		c.Total++
		c.ByRunner[r.Asked]++
		c.ByRole[r.Role]++
		if s := r.Spent; s != nil {
			if s.Cost != nil && s.Cost.Unit == oracle.CostUSD && oracle.ReadableCost(c.CostUSD+s.Cost.Amount) {
				c.CostUSD += s.Cost.Amount
			}
			c.ChargedRuns += s.ChargedRuns
		}
	}
	return c
}
