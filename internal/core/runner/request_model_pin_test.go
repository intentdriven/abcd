package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// The tests in this file hold a request's model to the pins the machine sets
// for it, as a --route's model is held (override_pin_test.go): a runner handed
// a model per request asks for it with that model's pins, never the route
// model's, so opencode probes a pinned local model's server and records its
// pin, and an unpinned request model carries none of the route model's.

// TestARequestForAPinnedLocalModelIsProbedAndHeld: opencode's machine model is
// unpinned, and a request asks it for the pinned one: the run is probed and
// pointed at the probed server through a sealed configuration, and the record
// carries the local pin.
func TestARequestForAPinnedLocalModelIsProbedAndHeld(t *testing.T) {
	personsOpenCode(t)
	o := newOllamaRunFake(t, true)
	base := o.srv.URL + "/v1"
	machine, name := overridePinMachine(base, "devstral")
	c := mustLoad(t, `{"roles":{"implementer":{"runner":"opencode"}},`+machine[1:], "")
	f := newFake(t, "ok", Claude, OpenCode)
	req := f.request(roleImplementer)
	req.Model = name + "/" + localRunModel
	out, err := newHarness(t, c, true).d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Handoff || out.Answer == nil || out.Fallback != nil {
		t.Fatalf("outcome = %+v", out)
	}
	if n := o.probes.Load(); n != 1 {
		t.Fatalf("probes = %d, want the request's pinned model probed once", n)
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "opencode.config.json"))
	if err != nil {
		t.Fatalf("no sealed configuration: %v", err)
	}
	if string(raw) != wantLocalConfig(name, base) {
		t.Fatalf("sealed configuration =\n%s\nwant\n%s", raw, wantLocalConfig(name, base))
	}
	if a := out.Answer; a.LocalPin == nil || a.LocalPin.WeightsSHA256 != localRunDigest || a.ModelAsked != req.Model {
		t.Fatalf("answer = %+v, want the request model's local pin", a)
	}
}

// TestARequestForAnUnpinnedModelCarriesNoRoutePin: opencode's machine model is
// pinned, and a request asks it for an unpinned one: nothing is probed, no
// sealed configuration is written, and the record carries no pin, since the
// route model's pin is not the request model's.
func TestARequestForAnUnpinnedModelCarriesNoRoutePin(t *testing.T) {
	o := newOllamaRunFake(t, true)
	machine, name := overridePinMachine(o.srv.URL+"/v1", localRunModel)
	c := mustLoad(t, `{"roles":{"implementer":{"runner":"opencode"}},`+machine[1:], "")
	f := newFake(t, "ok", Claude, OpenCode)
	req := f.request(roleImplementer)
	req.Model = name + "/devstral"
	out, err := newHarness(t, c, true).d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer == nil || o.probes.Load() != 0 {
		t.Fatalf("outcome %+v, probes %d: an unpinned request model was probed", out, o.probes.Load())
	}
	if _, err := os.Stat(filepath.Join(f.log, "opencode.config.json")); err == nil {
		t.Fatal("an unpinned request model was run against a sealed configuration")
	}
	if a := out.Answer; a.LocalPin != nil || a.ProviderPin != nil || a.ModelAsked != req.Model {
		t.Fatalf("answer = %+v, want the request's model and no pin", a)
	}
}

// TestClaudeRefusesARequestForAPinnedModel is dcn-2610292236161673 (a):
// claude's own model is unpinned, and a request asks it for the pinned one:
// claude can hold no pin, so the dispatch is refused before anything is
// launched or probed, in the words a --route to a pinned model on claude is
// refused with, and nothing falls back, to the host or to any other runner.
func TestClaudeRefusesARequestForAPinnedModel(t *testing.T) {
	o := newOllamaRunFake(t, true)
	machine, name := overridePinMachine(o.srv.URL+"/v1", "devstral")
	machine = strings.Replace(machine, `"claude":{}`, `"claude":{"model":"`+name+`/devstral"}`, 1)
	c := mustLoad(t, `{"roles":{"implementer":{"runner":"claude"}},`+machine[1:], "")
	f := newFake(t, "ok", Claude, OpenCode)
	req := f.request(roleImplementer)
	req.Model = name + "/" + localRunModel
	h := newHarness(t, c, true)
	out, err := h.d.Dispatch(context.Background(), req)
	var fl *Failure
	if err == nil || errors.As(err, &fl) || out.Handoff || out.Fallback != nil {
		t.Fatalf("outcome = %+v, err %v; want a refusal that hands nothing to the host", out, err)
	}
	for _, want := range []string{"which the machine pins", layered.BoundKey(req.Model), "can hold neither a provider pin nor a local pin", "runner.opencode.model"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want it to say %q", err, want)
		}
	}
	if len(h.receipts) != 0 {
		t.Fatalf("a fallback receipt was written: %v", h.receipts)
	}
	if f.launched(Claude) || f.launched(OpenCode) || o.probes.Load() != 0 {
		t.Fatal("a runner was launched, or the pinned model probed, for a pinned request model")
	}

	// With no host session and claude the configured host, a role on the
	// host is refused the same way, and nothing else runs it.
	noHost := mustLoad(t, strings.Replace(machine, `"runner":{`, `"runner":{"fallback_host":"claude",`, 1), "")
	h = newHarness(t, noHost, false)
	if _, err := h.d.Dispatch(context.Background(), req); err == nil || !strings.Contains(err.Error(), "which the machine pins") {
		t.Fatalf("err = %v, want the configured host claude refused the pinned request model", err)
	}
	if f.launched(Claude) || len(h.receipts) != 0 {
		t.Fatal("claude was launched as the configured host for a pinned request model")
	}
}

// TestARequestModelAndADifferentRouteModelAreRefusedNamingBoth is
// dcn-2610292236161673 (b): a request names a model and the step's --route
// for the role names another, so the dispatch is refused before anything
// launches, naming both, with no fallback; the same model on both runs.
func TestARequestModelAndADifferentRouteModelAreRefusedNamingBoth(t *testing.T) {
	c := mustLoad(t, perRequestMachine, "")
	if err := c.Override(flagRoutes(t, "ruthless-reviewer=runner:opencode@local/qwen3-small")); err != nil {
		t.Fatal(err)
	}
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, c, true)
	req := f.request("ruthless-reviewer")
	req.Model = "local/qwen3-coder"
	out, err := h.d.Dispatch(context.Background(), req)
	var fl *Failure
	if err == nil || errors.As(err, &fl) || out.Handoff || out.Fallback != nil {
		t.Fatalf("outcome = %+v, err %v; want a refusal that hands nothing to the host", out, err)
	}
	for _, want := range []string{"local/qwen3-coder", "local/qwen3-small", "--route"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want it to name %q", err, want)
		}
	}
	if f.launched(Claude) || f.launched(OpenCode) || len(h.receipts) != 0 {
		t.Fatalf("a runner launched or a receipt was written: receipts %v", h.receipts)
	}

	req.Model = "local/qwen3-small"
	out, err = h.d.Dispatch(context.Background(), req)
	if err != nil || out.Answer == nil || out.Answer.ModelAsked != "local/qwen3-small" {
		t.Fatalf("the same model on the request and the --route: outcome %+v, err %v; want it run at that model", out, err)
	}
}
