package runner

import (
	"context"
	"testing"
)

// A dispatch inside a model-matrix cell (itd-2610161733316518 decision 4,
// spc-2610161949559050 step 3): the cell is the route under test, so a runner
// that fails, refuses or answers unparsably is the cell's failure and nothing
// else runs the role: no host session, no configured host, no receipt of a
// fallback, since none happened.

// TestACellRunnerFailureIsTheCellsFailureWithNoHostFallback: under every
// failure kind, with a host session and a configured host both there to fall
// back on, a cell's dispatch hands nothing to the host, launches no other
// runner, writes no fallback receipt and returns the runner's failure.
func TestACellRunnerFailureIsTheCellsFailureWithNoHostFallback(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		onPath     []string
		want       Reason
	}{
		{"absent", "ok", nil, ReasonAbsent},
		{"refuses", "refuse", []string{OpenCode}, ReasonRefused},
		{"fails", "exit1", []string{OpenCode}, ReasonFailed},
		{"unparsable", "garbage", []string{OpenCode}, ReasonUnparsable},
		{"invalid answer", "noreceipt", []string{OpenCode}, ReasonInvalid},
	} {
		for _, hostSession := range []bool{true, false} {
			t.Run(tc.name, func(t *testing.T) {
				f := newFake(t, tc.mode, append([]string{Claude}, tc.onPath...)...)
				h := newHarness(t, mustLoad(t, routed(routedMachine), ""), hostSession)
				h.d.Cell = true
				out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
				if err != nil {
					t.Fatalf("a cell's failure is an outcome, not an error: %v", err)
				}
				if out.Handoff || out.Answer != nil || out.Fallback != nil || len(h.receipts) != 0 {
					t.Fatalf("nothing falls back inside a cell: %+v, receipts %v", out, h.receipts)
				}
				if out.Failure == nil || out.Failure.Reason != tc.want || out.Failure.Runner != OpenCode {
					t.Fatalf("the outcome carries the runner's failure: %+v", out.Failure)
				}
				if out.Receipt.Route.Asked != OpenCode || out.Receipt.Route.Ran != none {
					t.Fatalf("the role receipt names the runner asked and nothing as what ran: %+v", out.Receipt.Route)
				}
				if f.launched(Claude) {
					t.Fatal("the configured host was launched inside a cell")
				}
			})
		}
	}
}

// TestACellOnTheHostWithNoHostSessionRunsNothingElse: a host cell needs the
// host session; with none, the configured stand-in host is a different model
// and never runs it, so the dispatch is refused with nothing launched.
func TestACellOnTheHostWithNoHostSessionRunsNothingElse(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routedMachine, ""), false)
	h.d.Cell = true
	if _, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer")); err == nil {
		t.Fatal("a host cell with no host session ran")
	}
	if f.launched(Claude) || f.launched(OpenCode) || len(h.receipts) != 0 {
		t.Fatalf("nothing is launched or recorded: receipts %v", h.receipts)
	}
}

// TestACellRunnerThatAnswersRunsAsRouted: a cell whose runner answers is an
// ordinary routed run.
func TestACellRunnerThatAnswersRunsAsRouted(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), false)
	h.d.Cell = true
	out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer == nil || out.Failure != nil || out.Receipt.Route.Ran != OpenCode || len(h.receipts) != 0 {
		t.Fatalf("the routed runner ran the cell: %+v", out)
	}
}
