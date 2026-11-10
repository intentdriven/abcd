package runner

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

// A drive's walled launch of the claude harness takes a model of the harness's
// own vendor by its name alone (iss-2611080057001367), when the route is one
// the machine's class map resolves and the machine configures no block for
// that vendor. Everything else keeps the provider allowlist: an unwalled
// launch, a route outside the class map, another vendor's route, opencode,
// the host, and a vendor the machine does configure.

// bundledReviewModel is the bundled class map's middle class (the loop's
// BundledClasses, which this package cannot import).
const bundledReviewModel = "anthropic/claude-opus-5-5"

// claudeOnly is a subscription machine: the claude runner, no provider block,
// and the reviewer routed to it.
const claudeOnly = `{"roles":{"ruthless-reviewer":{"runner":"claude"}},"runner":{"claude":{}}}`

// testClassRoutes stands in for the routes the bundled class map resolves.
var testClassRoutes = []string{"anthropic/claude-fable-5", bundledReviewModel, "anthropic/claude-sonnet-5-5", "anthropic/claude-haiku-4-5"}

// walledOn is machine read and walled onto a drive's profile, whose
// settings are written so a launch on it stands, holding routes as the class
// map's (testClassRoutes when none is named).
func walledOn(t *testing.T, machine string, routes ...string) *Config {
	t.Helper()
	c := mustLoad(t, machine, "")
	p, err := DriveProfile(profileHome(t, "walled"), "")
	if err != nil {
		t.Fatal(err)
	}
	p.Ledger = c.Ledger()
	if _, err := WriteWallSettings(p, selfBinary()); err != nil {
		t.Fatal(err)
	}
	if len(routes) == 0 {
		routes = testClassRoutes
	}
	return c.Walled(p).WithClassRoutes(routes)
}

// TestAWalledHarnessLaunchesAReviewerWithNoProviderBlock: a reviewer request
// at the bundled class's model launches on a drive's walled launch of a
// claude-only machine, the harness handed the model's name, and the route
// records it as asked.
func TestAWalledHarnessLaunchesAReviewerWithNoProviderBlock(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, walledOn(t, claudeOnly), true)
	req := f.request("ruthless-reviewer")
	req.Model = bundledReviewModel
	out, err := h.d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("a walled subscription launch refused the bundled reviewer: %v", err)
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

// TestAnUnwalledHarnessStillNeedsTheProviderBlock: a claude launch that is not
// a drive's walled one is held to the provider allowlist as before, routed or
// as the configured host, and CanLaunch answers the same.
func TestAnUnwalledHarnessStillNeedsTheProviderBlock(t *testing.T) {
	for name, tc := range map[string]struct {
		machine, role string
		host          bool
	}{
		"routed to claude":  {claudeOnly, "ruthless-reviewer", true},
		"claude as host":    {`{"runner":{"fallback_host":"claude","claude":{}}}`, "security-reviewer", false},
		"no host, routed":   {`{"roles":{"ruthless-reviewer":{"runner":"claude"}},"runner":{"fallback_host":"claude","claude":{}}}`, "ruthless-reviewer", false},
		"host session only": {`{"runner":{"claude":{}}}`, "security-reviewer", true},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, "ok", Claude, OpenCode)
			c := mustLoad(t, tc.machine, "")
			h := newHarness(t, c, tc.host)
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
			if err := c.CanLaunch(tc.role, bundledReviewModel, tc.host); err == nil {
				t.Error("CanLaunch admits what the dispatch refuses")
			}
		})
	}
}

// TestAnotherVendorIsRefusedOnTheWalledHarness: the walled launch takes the
// harness's own vendor's models by name and no other's; a route to a vendor
// the machine does not configure is refused as unconfigured.
func TestAnotherVendorIsRefusedOnTheWalledHarness(t *testing.T) {
	f := newFake(t, "ok", Claude)
	c := walledOn(t, claudeOnly)
	h := newHarness(t, c, true)
	req := f.request("ruthless-reviewer")
	req.Model = "x/claude-opus-5-5"
	if out, err := h.d.Dispatch(context.Background(), req); err == nil || !strings.Contains(err.Error(), `provider "x" is not configured`) {
		t.Fatalf("outcome %+v, err %v; want the other vendor refused as unconfigured", out, err)
	}
	if f.launched(Claude) || len(h.receipts) != 0 {
		t.Fatal("a refused model launched or wrote a receipt")
	}
	if err := c.CanLaunch("ruthless-reviewer", "x/claude-opus-5-5", false); err == nil || !strings.Contains(err.Error(), "oracle.api.x") {
		t.Fatalf("CanLaunch = %v, want the refusal naming oracle.api.x", err)
	}
}

// TestOpencodeStillNeedsTheProviderBlock: opencode spends a provider's key, so
// a model whose provider the machine does not configure is refused for it
// before anything starts, with no fallback, routed or as the configured host.
func TestOpencodeStillNeedsTheProviderBlock(t *testing.T) {
	for name, tc := range map[string]struct {
		machine, role string
	}{
		"routed to opencode": {`{"roles":{"ruthless-reviewer":{"runner":"opencode"}},"runner":{"fallback_host":"claude","claude":{},"opencode":{}}}`, "ruthless-reviewer"},
		"opencode the host":  {`{"runner":{"fallback_host":"opencode","claude":{},"opencode":{}}}`, "security-reviewer"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, "ok", Claude, OpenCode)
			h := newHarness(t, mustLoad(t, tc.machine, ""), false)
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

// TestAConfiguredProviderStillHoldsTheWalledHarnessToItsList: where the
// machine does configure the vendor, its list decides for the walled launch
// too, so a model off it is refused, and one on it runs.
func TestAConfiguredProviderStillHoldsTheWalledHarnessToItsList(t *testing.T) {
	f := newFake(t, "ok", Claude)
	h := newHarness(t, walledOn(t, `{"roles":{"ruthless-reviewer":{"runner":"claude"}},`+anthropicBlock+`,"runner":{"claude":{}}}`), true)
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

// TestADenylistedModelIsRefusedOnTheWalledHarness: an oracle.denylist entry
// refuses a model the walled launch takes by name, matched against the route
// as written, whether it names the model, the route or the vendor, and with a
// bracketed suffix on the model.
func TestADenylistedModelIsRefusedOnTheWalledHarness(t *testing.T) {
	for _, tc := range []struct{ entry, model string }{
		{"claude-opus-5-5", bundledReviewModel},
		{"anthropic/claude-opus-5-5", bundledReviewModel},
		{"anthropic/*", bundledReviewModel},
		{"anthropic/claude-opus-5-5", bundledReviewModel + "[1m]"},
	} {
		t.Run(tc.entry+" "+tc.model, func(t *testing.T) {
			f := newFake(t, "ok", Claude)
			h := newHarness(t, walledOn(t, `{"roles":{"ruthless-reviewer":{"runner":"claude"}},"oracle":{"denylist":["`+tc.entry+`"]},"runner":{"claude":{}}}`,
				append(slices.Clone(testClassRoutes), tc.model)...), true)
			req := f.request("ruthless-reviewer")
			req.Model = tc.model
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
// model and the configuration that would let it run, by why it was refused,
// and nothing else of the machine.
func TestCanLaunchSharesTheDispatchsAdmission(t *testing.T) {
	if err := walledOn(t, claudeOnly).CanLaunch("ruthless-reviewer", bundledReviewModel, false); err != nil {
		t.Fatalf("a walled subscription machine cannot launch the bundled reviewer: %v", err)
	}
	const runnerAdvice = "roles.ruthless-reviewer.runner"
	for name, tc := range map[string]struct {
		cfg           *Config
		want, without []string
	}{
		"opencode route": {mustLoad(t, `{"roles":{"ruthless-reviewer":{"runner":"opencode"}},"runner":{"claude":{},"opencode":{}}}`, ""),
			[]string{"oracle.api.anthropic", runnerAdvice}, []string{"oracle.denylist"}},
		"walled, off the list": {walledOn(t, `{`+anthropicBlock+`,"runner":{"claude":{}}}`),
			[]string{"oracle.api.anthropic", "not on anthropic's list"}, []string{runnerAdvice}},
		"walled, denylisted": {walledOn(t, `{"oracle":{"denylist":["anthropic/*"]},"runner":{"claude":{}}}`),
			[]string{"oracle.denylist", "models.classes"}, []string{"oracle.api.anthropic", runnerAdvice}},
		"no landing": {mustLoad(t, `{"runner":{"claude":{}}}`, ""), []string{"runner.fallback_host", runnerAdvice}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			err := tc.cfg.CanLaunch("ruthless-reviewer", bundledReviewModel, false)
			if err == nil {
				t.Fatal("launchable, want a refusal")
			}
			for _, w := range append(tc.want, "ruthless-reviewer", bundledReviewModel) {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q does not name %q", err, w)
				}
			}
			for _, w := range tc.without {
				if strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q names %q, which does not apply", err, w)
				}
			}
			if strings.Contains(err.Error(), "/Users/") || strings.Contains(err.Error(), "/home/") || strings.Contains(err.Error(), "/tmp") {
				t.Errorf("refusal %q names an absolute path", err)
			}
		})
	}
}

// TestTheWalledHarnessTakesByNameOnlyWhatTheClassMapResolves: the name-only
// admission is bounded by the class map the drive resolved. A route of the
// harness's own vendor outside it falls through to the provider allowlist and
// is refused there, one inside it runs, a machine's own class route stands in
// for the bundled one, and a walled configuration handed no class map admits
// nothing by name. CanLaunch answers the same each time.
func TestTheWalledHarnessTakesByNameOnlyWhatTheClassMapResolves(t *testing.T) {
	const override = "anthropic/claude-opus-6"
	for name, tc := range map[string]struct {
		routes []string
		model  string
		admit  bool
	}{
		"in the class map":         {testClassRoutes, bundledReviewModel, true},
		"outside the class map":    {testClassRoutes, "anthropic/claude-unlisted-1", false},
		"the machine's own class":  {[]string{override, "anthropic/claude-sonnet-5-5"}, override, true},
		"the class it replaced":    {[]string{override, "anthropic/claude-sonnet-5-5"}, bundledReviewModel, false},
		"no class map handed over": {nil, bundledReviewModel, false},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, "ok", Claude)
			c := walledOn(t, claudeOnly, "unused/placeholder")
			c = c.WithClassRoutes(tc.routes)
			h := newHarness(t, c, true)
			req := f.request("ruthless-reviewer")
			req.Model = tc.model
			_, err := h.d.Dispatch(context.Background(), req)
			can := c.CanLaunch("ruthless-reviewer", tc.model, false)
			if tc.admit {
				if err != nil || can != nil {
					t.Fatalf("dispatch %v, CanLaunch %v; want the class map's route admitted", err, can)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), `provider "anthropic" is not configured`) {
				t.Fatalf("dispatch err = %v; want the provider allowlist's refusal", err)
			}
			if can == nil || !strings.Contains(can.Error(), "oracle.api.anthropic") {
				t.Fatalf("CanLaunch = %v; want the refusal naming oracle.api.anthropic", can)
			}
			if f.launched(Claude) || len(h.receipts) != 0 {
				t.Fatal("a refused model launched or wrote a receipt")
			}
		})
	}
}
