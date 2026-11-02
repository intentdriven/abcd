package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// TestAKilledLaunchKeepsWhatItSpent: a launch killed at its bound writes no
// closing event, so the cost it would have reported there never comes. Its
// fallback receipt carries what its transcript shows instead: claude's tokens,
// summed once per message, with the launch charged as one agent run since no
// cost shows; opencode's tokens and the cost its finished steps carried. The
// run's tally counts both (iss-2611021751406429).
func TestAKilledLaunchKeepsWhatItSpent(t *testing.T) {
	for _, tc := range []struct {
		runner string
		usage  oracle.Usage
		cost   *oracle.Cost
		runs   int
	}{
		{Claude, oracle.Usage{PromptTokens: 107, CompletionTokens: 23, CacheReadTokens: 2100, CacheWriteTokens: 50}, nil, 1},
		{OpenCode, oracle.Usage{PromptTokens: 300, CompletionTokens: 50, CacheReadTokens: 10, CacheWriteTokens: 2},
			&oracle.Cost{Amount: 0.75, Unit: oracle.CostUSD}, 0},
	} {
		t.Run(tc.runner, func(t *testing.T) {
			f := newFake(t, "spend-hang", Claude, OpenCode)
			machine := `{"roles":{"ruthless-reviewer":{"runner":"` + tc.runner + `"}},` + strings.TrimPrefix(routedMachine, "{")
			h := newHarness(t, mustLoad(t, machine, ""), true)
			req := f.request("ruthless-reviewer")
			req.Timeout = time.Second
			out, err := h.d.Dispatch(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if !out.Handoff || len(h.receipts) != 1 {
				t.Fatalf("outcome = %+v, receipts %v; want one fallback to the host", out, h.receipts)
			}
			if out.Failure == nil || !errors.Is(out.Failure, ErrKilled) {
				t.Errorf("the outcome's failure = %v, want the kill", out.Failure)
			}
			s := h.receipts[0].Spent
			if s == nil {
				t.Fatal("the killed launch's fallback receipt carries no spend: it was dropped")
			}
			if s.Usage == nil || *s.Usage != tc.usage {
				t.Errorf("usage = %+v, want %+v", s.Usage, tc.usage)
			}
			if (s.Cost == nil) != (tc.cost == nil) || s.Cost != nil && *s.Cost != *tc.cost {
				t.Errorf("cost = %+v, want %+v", s.Cost, tc.cost)
			}
			if s.ChargedRuns != tc.runs {
				t.Errorf("charged runs = %d, want %d", s.ChargedRuns, tc.runs)
			}
			c := Tally(h.receipts)
			if c.ChargedRuns != tc.runs || (tc.cost != nil && c.CostUSD != tc.cost.Amount) {
				t.Errorf("tally = %+v, want the spend counted", c)
			}
		})
	}
}

// TestAFailureThatIsNoKillCarriesNoSpend: a harness that exits on its own
// reports what it reports; only a kill is read for its spend.
func TestAFailureThatIsNoKillCarriesNoSpend(t *testing.T) {
	f := newFake(t, "exit1", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
	out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if len(h.receipts) != 1 || h.receipts[0].Spent != nil || errors.Is(out.Failure, ErrKilled) {
		t.Fatalf("receipts = %+v, failure %v; want one with no spend and no kill", h.receipts, out.Failure)
	}
}
