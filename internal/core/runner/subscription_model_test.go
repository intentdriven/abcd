package runner

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

// The subscription harness takes a model by its name alone (iss-2611080057001367):
// a role whose runner is the claude harness is admitted at a <provider>/<model>
// route whose provider the machine does not configure, since the harness runs
// it on the person's own login and spends no provider key. Every other runner,
// and every provider the machine does configure, keeps the allowlist.

// bundledReviewModel is the bundled class map's middle class (the loop's
// BundledClasses, which this package cannot import).
const bundledReviewModel = "anthropic/claude-opus-5-5"

// claudeOnly is a subscription machine: the claude runner, no provider block,
// and the reviewer routed to it.
const claudeOnly = `{"roles":{"ruthless-reviewer":{"runner":"claude"}},"runner":{"claude":{}}}`

// TestASubscriptionHarnessLaunchesAReviewerWithNoProviderBlock: a reviewer
// request at the bundled class's model launches on a claude-only machine, the
// harness handed the model's name, and the route records it as asked.
func TestASubscriptionHarnessLaunchesAReviewerWithNoProviderBlock(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, claudeOnly, ""), true)
	req := f.request("ruthless-reviewer")
	req.Model = bundledReviewModel
	out, err := h.d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("a subscription machine refused the bundled reviewer: %v", err)
	}
	if argv := f.argv(t, Claude); !slices.Contains(argv, "--model=claude-opus-5-5") {
		t.Errorf("claude argv %q does not carry the model's name", argv)
	}
	if r := out.Receipt.Route; r.Ran != Claude || r.ModelAsked != "claude-opus-5-5" || len(h.receipts) != 0 {
		t.Errorf("route = %+v, receipts %v; want claude run at the model asked, with no fallback", r, h.receipts)
	}
	if f.launched(OpenCode) {
		t.Error("opencode launched")
	}
}

// TestASubscriptionFallbackHostIsAdmittedToo: with no host session, a role on
// the host runs on the configured host, and when that is the claude harness
// the same rule admits the model.
func TestASubscriptionFallbackHostIsAdmittedToo(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, `{"runner":{"fallback_host":"claude","claude":{}}}`, ""), false)
	req := f.request("security-reviewer")
	req.Model = bundledReviewModel
	out, err := h.d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Receipt.Route.Ran != Claude || !slices.Contains(f.argv(t, Claude), "--model=claude-opus-5-5") {
		t.Errorf("route = %+v, want claude at the model's name", out.Receipt.Route)
	}
}

// TestOpencodeStillNeedsTheProviderBlock: the rule is the subscription
// harness's alone. opencode spends a provider's key, so a model whose provider
// the machine does not configure is refused for it before anything starts,
// with no fallback, routed or as the configured host; and a role the host
// session runs is still handed only a model the providers admit.
func TestOpencodeStillNeedsTheProviderBlock(t *testing.T) {
	for name, tc := range map[string]struct {
		machine string
		host    bool
		role    string
	}{
		"routed to opencode": {`{"roles":{"ruthless-reviewer":{"runner":"opencode"}},"runner":{"fallback_host":"claude","claude":{},"opencode":{}}}`, false, "ruthless-reviewer"},
		"opencode the host":  {`{"runner":{"fallback_host":"opencode","claude":{},"opencode":{}}}`, false, "security-reviewer"},
		"the host session":   {`{"runner":{"claude":{}}}`, true, "security-reviewer"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, "ok", Claude, OpenCode)
			h := newHarness(t, mustLoad(t, tc.machine, ""), tc.host)
			req := f.request(tc.role)
			req.Model = bundledReviewModel
			out, err := h.d.Dispatch(context.Background(), req)
			var fl *Failure
			if err == nil || errors.As(err, &fl) || out.Handoff {
				t.Fatalf("outcome %+v, err %v; want a refusal that is not a fallback", out, err)
			}
			if !strings.Contains(err.Error(), `provider "anthropic" is not configured`) {
				t.Errorf("err = %v, want it to name the unconfigured provider", err)
			}
			if f.launched(Claude) || f.launched(OpenCode) || len(h.receipts) != 0 {
				t.Fatalf("a runner launched or a receipt was written: receipts %v", h.receipts)
			}
		})
	}
}

// anthropicBlock configures the provider with one model, so its list decides.
const anthropicBlock = `"oracle":{"api":{"anthropic":{"base_url":"https://api.example.invalid/v1","models":["claude-haiku-4-5"]}}}`

// TestAConfiguredProviderStillHoldsClaudeToItsList: where the machine does
// configure the provider, its list decides for the claude harness too, so a
// model off it is refused, and one on it runs.
func TestAConfiguredProviderStillHoldsClaudeToItsList(t *testing.T) {
	machine := `{"roles":{"ruthless-reviewer":{"runner":"claude"}},` + anthropicBlock + `,"runner":{"claude":{}}}`
	f := newFake(t, "ok", Claude)
	h := newHarness(t, mustLoad(t, machine, ""), true)
	req := f.request("ruthless-reviewer")
	req.Model = bundledReviewModel
	if out, err := h.d.Dispatch(context.Background(), req); err == nil || !strings.Contains(err.Error(), "not on anthropic's list") {
		t.Fatalf("outcome %+v, err %v; want the model off the provider's list refused", out, err)
	}
	if f.launched(Claude) || len(h.receipts) != 0 {
		t.Fatal("a refused model launched or wrote a receipt")
	}
	req.Model = "anthropic/claude-haiku-4-5"
	if _, err := h.d.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("a listed model was refused: %v", err)
	}
}

// TestADenylistedModelIsRefusedOnTheSubscriptionHarness: an oracle.denylist
// entry refuses a model with no provider block as it does with one, whether it
// names the model or its vendor.
func TestADenylistedModelIsRefusedOnTheSubscriptionHarness(t *testing.T) {
	for _, entry := range []string{"claude-opus-5-5", "anthropic/claude-opus-5-5", "anthropic/*"} {
		t.Run(entry, func(t *testing.T) {
			machine := `{"roles":{"ruthless-reviewer":{"runner":"claude"}},"oracle":{"denylist":["` + entry + `"]},"runner":{"claude":{}}}`
			f := newFake(t, "ok", Claude)
			h := newHarness(t, mustLoad(t, machine, ""), true)
			req := f.request("ruthless-reviewer")
			req.Model = bundledReviewModel
			if out, err := h.d.Dispatch(context.Background(), req); err == nil || !strings.Contains(err.Error(), "oracle.denylist") {
				t.Fatalf("outcome %+v, err %v; want the denylist's refusal", out, err)
			}
			if f.launched(Claude) || len(h.receipts) != 0 {
				t.Fatal("a denied model launched or wrote a receipt")
			}
		})
	}
}

// TestCanLaunchSharesTheDispatchsAdmission: the check a drive makes at its
// start answers as the dispatch would, and a refusal names the role, the
// model and the configuration to add, and nothing else of the machine.
func TestCanLaunchSharesTheDispatchsAdmission(t *testing.T) {
	ok := mustLoad(t, claudeOnly, "")
	if err := ok.CanLaunch("ruthless-reviewer", bundledReviewModel, false); err != nil {
		t.Fatalf("a subscription machine cannot launch the bundled reviewer: %v", err)
	}
	for name, tc := range map[string]struct {
		machine string
		want    []string
	}{
		"opencode route": {`{"roles":{"ruthless-reviewer":{"runner":"opencode"}},"runner":{"claude":{},"opencode":{}}}`,
			[]string{"oracle.api.anthropic", "roles.ruthless-reviewer.runner"}},
		"off the list": {`{"roles":{"ruthless-reviewer":{"runner":"claude"}},` + anthropicBlock + `,"runner":{"claude":{}}}`,
			[]string{"oracle.api.anthropic", "not on anthropic's list"}},
		"denylisted": {`{"roles":{"ruthless-reviewer":{"runner":"claude"}},"oracle":{"denylist":["anthropic/*"]},"runner":{"claude":{}}}`,
			[]string{"oracle.denylist"}},
		"no landing": {`{"runner":{"claude":{}}}`, []string{"runner.fallback_host", "roles.ruthless-reviewer.runner"}},
	} {
		t.Run(name, func(t *testing.T) {
			err := mustLoad(t, tc.machine, "").CanLaunch("ruthless-reviewer", bundledReviewModel, false)
			if err == nil {
				t.Fatal("launchable, want a refusal")
			}
			for _, w := range append(tc.want, "ruthless-reviewer", bundledReviewModel) {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q does not name %q", err, w)
				}
			}
			if strings.Contains(err.Error(), "/Users/") || strings.Contains(err.Error(), "/home/") || strings.Contains(err.Error(), "/tmp") {
				t.Errorf("refusal %q names an absolute path", err)
			}
		})
	}
}
