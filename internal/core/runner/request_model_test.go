package runner

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

// twoModelProvider lists two models under one keyless provider, so one runner
// enabled at the first can be handed the second per request
// (itd-2610240922461281 decision 4).
const twoModelProvider = `"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder","qwen3-small"]}}}`

// perRequestMachine routes the ruthless-reviewer to opencode at the first
// model and enables claude at it too.
const perRequestMachine = `{"roles":{"ruthless-reviewer":{"runner":"opencode"}},` + twoModelProvider +
	`,"runner":{"fallback_host":"claude","claude":{"model":"local/qwen3-coder"},"opencode":{"model":"local/qwen3-coder"}}}`

// TestARequestModelPastItsShapeIsRefused: a request's model is a
// <provider>/<model> route, bounded and plain as a reported model is, or the
// request is refused before anything runs.
func TestARequestModelPastItsShapeIsRefused(t *testing.T) {
	f := newFake(t, "ok", Claude)
	for name, model := range map[string]string{
		"no provider":    "qwen3-coder",
		"empty provider": "/qwen3-coder",
		"empty model":    "local/",
		"control byte":   "local/qwen\x1b3",
		"past the bound": "local/" + strings.Repeat("q", 200),
		"leading flag":   "-local/qwen3-coder",
	} {
		req := f.request("scribe")
		req.Model = model
		if err := req.Check(); err == nil {
			t.Errorf("%s: %q accepted", name, model)
		}
		if _, _, err := newClaude("").Run(context.Background(), req); err == nil {
			t.Errorf("%s: the claude runner ran %q", name, model)
		}
	}
	if f.launched(Claude) {
		t.Error("a refused request model launched the harness")
	}
}

// TestARequestModelOffTheAllowlistIsRefusedBeforeTheRunnerStarts: the model a
// request names is admitted against its provider's allowlist as a configured
// route is; one off it ends the dispatch before any runner, fallback host
// included, is started, with no fallback receipt, since running the role on
// another model would be a run nobody asked for.
func TestARequestModelOffTheAllowlistIsRefusedBeforeTheRunnerStarts(t *testing.T) {
	for name, model := range map[string]string{
		"unlisted model":   "local/llama-9",
		"unknown provider": "elsewhere/qwen3-coder",
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, "ok", Claude, OpenCode)
			h := newHarness(t, mustLoad(t, perRequestMachine, ""), true)
			req := f.request("ruthless-reviewer")
			req.Model = model
			out, err := h.d.Dispatch(context.Background(), req)
			var fl *Failure
			if err == nil || errors.As(err, &fl) {
				t.Fatalf("outcome %+v, err %v; want a refusal that is not a fallback", out, err)
			}
			if !strings.Contains(err.Error(), "allowlist") && !strings.Contains(err.Error(), "provider") {
				t.Errorf("err = %v, want it to name the provider's admission", err)
			}
			if f.launched(Claude) || f.launched(OpenCode) || len(h.receipts) != 0 {
				t.Fatalf("a runner launched or a receipt was written: receipts %v", h.receipts)
			}
		})
	}
	// A role the host runs is refused too: the host is handed the model, and
	// one the person's providers do not admit is no model to hand it.
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, perRequestMachine, ""), true)
	req := f.request("security-reviewer")
	req.Model = "local/llama-9"
	if out, err := h.d.Dispatch(context.Background(), req); err == nil || out.Handoff {
		t.Fatalf("outcome %+v, err %v; want the host handoff refused", out, err)
	}
}

// TestARequestModelReachesTheRunnerAndIsRecordedAsAsked: one runner enabled
// at one model serves a request at another its provider admits; the model
// passed is recorded as the one asked, beside the one the harness reported.
func TestARequestModelReachesTheRunnerAndIsRecordedAsAsked(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, perRequestMachine, ""), true)
	req := f.request("ruthless-reviewer")
	req.Model = "local/qwen3-small"
	out, err := h.d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	argv := f.argv(t, OpenCode)
	if !slices.Contains(argv, "--model=local/qwen3-small") || slices.Contains(argv, "--model=local/qwen3-coder") {
		t.Errorf("opencode argv %q does not carry the request's model alone", argv)
	}
	if r := out.Receipt.Route; r.Ran != OpenCode || r.ModelAsked != "local/qwen3-small" || r.Model != "" {
		t.Errorf("route = %+v, want the request's model asked and none reported", r)
	}
}

// TestEachRunnerPassesTheRequestModel: the claude and opencode runners each
// pass the request's model on their command line over the route's, and
// record it as asked beside the model the harness reported.
func TestEachRunnerPassesTheRequestModel(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	req := f.request("ruthless-reviewer")
	req.Model = "local/qwen3-small"

	ans, _, err := newClaude("local/qwen3-coder").Run(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	argv := f.argv(t, Claude)
	if !slices.Contains(argv, "--model=qwen3-small") || slices.Contains(argv, "--model=qwen3-coder") {
		t.Errorf("claude argv %q does not carry the request's model alone", argv)
	}
	if ans.ModelAsked != "qwen3-small" || ans.Model != "fake-model" {
		t.Errorf("claude answer asked %q reported %q, want qwen3-small beside fake-model", ans.ModelAsked, ans.Model)
	}

	ans, _, err = newOpenCode("local/qwen3-coder").Run(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	argv = f.argv(t, OpenCode)
	if !slices.Contains(argv, "--model=local/qwen3-small") || slices.Contains(argv, "--model=local/qwen3-coder") {
		t.Errorf("opencode argv %q does not carry the request's model alone", argv)
	}
	if ans.ModelAsked != "local/qwen3-small" || ans.Model != "" {
		t.Errorf("opencode answer asked %q reported %q", ans.ModelAsked, ans.Model)
	}
}

// TestARequestWithoutAModelKeepsTheRoutesModel: a request that names no
// model runs at the model its runner's route names, as before.
func TestARequestWithoutAModelKeepsTheRoutesModel(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, perRequestMachine, ""), true)
	out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if argv := f.argv(t, OpenCode); !slices.Contains(argv, "--model=local/qwen3-coder") {
		t.Errorf("opencode argv %q lacks the route's model", argv)
	}
	if out.Receipt.Route.ModelAsked != "local/qwen3-coder" {
		t.Errorf("route = %+v, want the route's model asked", out.Receipt.Route)
	}
}
