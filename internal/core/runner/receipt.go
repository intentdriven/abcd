package runner

// receipt.go is what a dispatch leaves in the record: the role's receipt
// block, the same for every route but the route it names (criterion 7), and
// the fallback receipt, one per fallback, which Tally counts per runner and
// per role for the run's summary (criteria 3 and 4).

import "time"

// RouteRecord is the route a role asked for and the one that ran it.
type RouteRecord struct {
	// Asked is the route the configuration named: host or a runner.
	Asked string `json:"asked"`
	// Ran is the route that ran the role: host, a runner, or none.
	Ran string `json:"ran"`
	// Model is the model the runner that ran it reported; "" on the host,
	// whose own receipt reports it.
	Model string `json:"model"`
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
}

// Counts is the fallback intel a run's summary reports.
type Counts struct {
	Total    int            `json:"total"`
	ByRunner map[string]int `json:"by_runner"`
	ByRole   map[string]int `json:"by_role"`
}

// Tally counts fallback receipts per runner asked for and per role. A run with
// none reports zero, with empty maps rather than none.
func Tally(rs []FallbackReceipt) Counts {
	c := Counts{ByRunner: map[string]int{}, ByRole: map[string]int{}}
	for _, r := range rs {
		c.Total++
		c.ByRunner[r.Asked]++
		c.ByRole[r.Role]++
	}
	return c
}
