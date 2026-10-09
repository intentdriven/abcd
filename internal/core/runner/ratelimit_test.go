package runner

// ratelimit_test.go proves the runner's half of itd-2609201925079472's
// criteria 7 and 8 (spc-2609301921521360): a harness's rate-limit response is
// a failure of its own kind, which a dispatcher that pauses on it hands back
// rather than falling back on, and a route reports its remaining quota only
// when its runner can, which no shipped runner can.

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestARateLimitResponseIsItsOwnFailure: each harness's rate-limit response is
// a rate-limited failure, whether the harness exits non-zero after it or not;
// a limit that warned and let the run finish is an answer.
func TestARateLimitResponseIsItsOwnFailure(t *testing.T) {
	for _, h := range []string{Claude, OpenCode} {
		t.Run(h, func(t *testing.T) {
			f := newFake(t, "ratelimit", h)
			var r Runner = newClaude("")
			if h == OpenCode {
				r = newOpenCode("")
			}
			_, transcript, err := r.Run(context.Background(), f.request("scribe"))
			var fl *Failure
			if !errors.As(err, &fl) || fl.Reason != ReasonRateLimited {
				t.Fatalf("err = %v, want a %s failure", err, ReasonRateLimited)
			}
			if len(transcript) == 0 {
				t.Fatal("the transcript of a rate-limited run is still returned")
			}
			if strings.Contains(fl.Detail, "limit reached") || strings.Contains(fl.Detail, "rate limited") {
				t.Fatalf("the detail is abcd's own, never the harness's text: %q", fl.Detail)
			}
		})
	}
	f := newFake(t, "ratelimit-passed", Claude)
	if _, _, err := newClaude("").Run(context.Background(), f.request("scribe")); err != nil {
		t.Fatalf("a limit that only warned is not a rate-limit response: %v", err)
	}
}

// TestARateLimitedRunnerPausesRatherThanFallingBack is the dispatch half of
// criterion 8: a dispatcher that pauses on a rate limit hands the response
// back with no fallback receipt and no hand-off, since every route spends the
// run's budget; one that does not (the interviews) falls back as on any other
// failure.
func TestARateLimitedRunnerPausesRatherThanFallingBack(t *testing.T) {
	f := newFake(t, "ratelimit", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
	h.d.PauseOnRateLimit = true
	out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if out.RateLimited == nil || out.RateLimited.Runner != OpenCode || out.RateLimited.Reason != ReasonRateLimited {
		t.Fatalf("the rate-limit response is handed back: %+v", out)
	}
	if out.Handoff || out.Fallback != nil || out.Answer != nil || len(h.receipts) != 0 {
		t.Fatalf("a rate limit is no fallback: outcome %+v, receipts %v", out, h.receipts)
	}
	if len(h.store.got) != 1 {
		t.Fatalf("the rate-limited run's transcript is stored: %v", h.store.got)
	}

	h = newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
	out, err = h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Handoff || out.RateLimited != nil || len(h.receipts) != 1 || h.receipts[0].Reason != ReasonRateLimited {
		t.Fatalf("without the pause a rate limit falls back as any failure does: %+v %v", out, h.receipts)
	}
}

// reporter is a runner that reports its quota.
type reporter struct {
	q   Quota
	err error
}

func (reporter) Name() string { return "reporter" }
func (reporter) Run(context.Context, Request) (Answer, []byte, error) {
	return Answer{}, nil, errors.New("not run")
}
func (r reporter) Quota(context.Context) (Quota, error) { return r.q, r.err }

// TestOnlyARunnerThatCanReportsAQuota is criterion 7's runner half: the host,
// a runner this machine does not enable, and each shipped runner report none;
// a runner that implements the report is read, and a negative one refused.
func TestOnlyARunnerThatCanReportsAQuota(t *testing.T) {
	c := mustLoad(t, routedMachine, "")
	for _, route := range []string{Host, Claude, OpenCode, "elsewhere"} {
		if _, ok, err := c.QuotaFor(context.Background(), route); ok || err != nil {
			t.Fatalf("%s reports no quota: ok %v, err %v", route, ok, err)
		}
	}
	var nilConfig *Config
	if _, ok, err := nilConfig.QuotaFor(context.Background(), Host); ok || err != nil {
		t.Fatalf("no configuration reports no quota: %v %v", ok, err)
	}
	q, ok, err := quotaOf(context.Background(), reporter{q: Quota{RemainingTokens: 500000}})
	if err != nil || !ok || q.RemainingTokens != 500000 {
		t.Fatalf("a reporter's quota is read: %+v %v %v", q, ok, err)
	}
	if _, _, err := quotaOf(context.Background(), reporter{q: Quota{RemainingTokens: -1}}); err == nil {
		t.Fatal("a negative quota is refused")
	}
	if _, _, err := quotaOf(context.Background(), reporter{err: errors.New("down")}); err == nil {
		t.Fatal("a report that fails is an error, not none")
	}
}
