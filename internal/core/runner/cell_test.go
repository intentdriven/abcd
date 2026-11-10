package runner

import (
	"context"
	"strings"
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

// TestProviderDownIsAnOutageWithNothingSpent: a launch that was retrying the
// model service, or whose result named a 5xx status or the provider's
// throttling answer (429), with nothing spent, is the provider down before the
// copy began; one that spent, one refused by the account, and one with nothing
// seen are not.
func TestProviderDownIsAnOutageWithNothingSpent(t *testing.T) {
	for _, tc := range []struct {
		name string
		seen *HarnessSeen
		want bool
	}{
		{"retrying", &HarnessSeen{Retrying: true, Retries: 3}, true},
		{"out of retries", &HarnessSeen{Retries: 5}, true},
		{"a 503", &HarnessSeen{APIStatus: "503"}, true},
		{"a 503 after spending", &HarnessSeen{APIStatus: "503", Spent: true}, false},
		{"retrying after spending", &HarnessSeen{Retrying: true, Spent: true}, false},
		{"a 429", &HarnessSeen{APIStatus: "429"}, true},
		{"a 429 after spending", &HarnessSeen{APIStatus: "429", Spent: true}, false},
		{"a 401", &HarnessSeen{APIStatus: "401"}, false},
		{"nothing seen", nil, false},
	} {
		fl := &Failure{Runner: OpenCode, Reason: ReasonFailed, Seen: tc.seen}
		if got := fl.ProviderDown(); got != tc.want {
			t.Errorf("%s: ProviderDown = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestAnOpenCodeProviderOutageInACellIsTheProviderDown: an OpenRouter cell
// runs through opencode, so its outage is what opencode's run mode shows of
// one: an error event whose APIError names the HTTP status, then (or not) a
// non-zero exit. Dispatched in cell mode, a 5xx or a 429 with nothing spent
// is the provider down before the copy began; one after a costed step had
// begun, an account's 401 and an error naming no status are not.
func TestAnOpenCodeProviderOutageInACellIsTheProviderDown(t *testing.T) {
	for _, tc := range []struct {
		mode, status string
		spent, down  bool
	}{
		{"oc-503", "503", false, true},
		{"oc-503-zero", "503", false, true},
		{"oc-503-spent", "503", true, false},
		{"oc-429", "429", false, true},
		{"oc-429-spent", "429", true, false},
		{"oc-401", "401", false, false},
		{"oc-nostatus", "", false, false},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			f := newFake(t, tc.mode, Claude, OpenCode)
			h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
			h.d.Cell = true
			out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
			if err != nil {
				t.Fatalf("a cell's failure is an outcome, not an error: %v", err)
			}
			fl := out.Failure
			if fl == nil || fl.Runner != OpenCode || out.Fallback != nil || f.launched(Claude) {
				t.Fatalf("the runner's failure is the outcome, with nothing fallen back: %+v", out)
			}
			if fl.Seen == nil {
				t.Fatalf("the failure carries what opencode's error event showed: %+v", fl)
			}
			if fl.Seen.APIStatus != tc.status || fl.Seen.Spent != tc.spent {
				t.Fatalf("seen = %+v, want status %q spent %v", fl.Seen, tc.status, tc.spent)
			}
			if got := fl.ProviderDown(); got != tc.down {
				t.Fatalf("ProviderDown = %v, want %v: %+v", got, tc.down, fl)
			}
			if tc.status != "" && !strings.Contains(fl.Detail, "API status "+tc.status) {
				t.Fatalf("the detail names the status in abcd's words: %q", fl.Detail)
			}
		})
	}
}
