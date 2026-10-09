package runner

// quota.go is what a route reports of the budget left to it, the runner's
// half of the budget check a run takes before it starts (itd-2609201925079472
// criterion 7, spc-2609301921521360).
//
// A route reports a quota only when its runner can say what is left without
// spending a turn to find out. No shipped runner does: the adapters read only
// the event stream a run prints, and no headless quota read is known to this
// build for either harness (whether one exists is owed to a person's check),
// and abcd cannot read a host session's. So each reports none, and the loop
// names it and skips the check out loud; a runner that gains the report
// implements QuotaReporter and is read from then on, with nothing else
// changing.

import (
	"context"
	"fmt"
)

// Quota is what a route reports of the budget left to it.
type Quota struct {
	// RemainingTokens is the tokens the route may still spend before its
	// provider's limit refuses it.
	RemainingTokens int64 `json:"remaining_tokens"`
}

// QuotaReporter is a Runner that can report its remaining quota before a run
// starts.
type QuotaReporter interface {
	Quota(ctx context.Context) (Quota, error)
}

// QuotaFor reports the quota route reports, ok false when it reports none:
// the host, a runner this machine does not enable, or a runner that cannot
// report one. An error is a report that failed, which is not none.
func (c *Config) QuotaFor(ctx context.Context, route string) (Quota, bool, error) {
	if c == nil || route == Host {
		return Quota{}, false, nil
	}
	rc, ok := c.runners[route]
	if !ok {
		return Quota{}, false, nil
	}
	return quotaOf(ctx, c.adapter(rc))
}

// quotaOf reads r's quota when r can report one.
func quotaOf(ctx context.Context, r Runner) (Quota, bool, error) {
	qr, ok := r.(QuotaReporter)
	if !ok {
		return Quota{}, false, nil
	}
	q, err := qr.Quota(ctx)
	if err != nil {
		return Quota{}, false, fmt.Errorf("runner: the %s runner's quota could not be read: %w", r.Name(), err)
	}
	if q.RemainingTokens < 0 {
		return Quota{}, false, fmt.Errorf("runner: the %s runner reports a negative quota (%d tokens)", r.Name(), q.RemainingTokens)
	}
	return q, true, nil
}
