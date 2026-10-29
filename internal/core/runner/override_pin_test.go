package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests in this file hold a --route's model to the pins the machine sets
// for it (iss-2610161801358637 beside iss-2610161734428291): a runner a
// --route hands a role to asks for the --route's model with that model's
// pins, never the machine model's, so opencode probes a pinned local model's
// server and the claude runner refuses a pinned model, exactly as for the
// machine's own route.

// overridePinMachine is a block on this machine at base serving localRunModel,
// which it pins, and devstral, which it does not; opencode's machine model is
// model, and the block's name.
func overridePinMachine(base, model string) (string, string) {
	name := fmt.Sprintf("local%d", localRunSeq.Add(1))
	pins := `"pins":{"` + localRunModel + `":{"runtime":"ollama","runtime_version":"0.14.2","weights_sha256":"` + localRunDigest +
		`","quantisation":"Q4_K_M","context_length":32768,"kv_cache_type":"f16"}}`
	return `{"oracle":{"api":{"` + name + `":{"base_url":"` + base + `","models":["` + localRunModel + `","devstral"],` + pins + `}}},` +
		`"runner":{"claude":{},"opencode":{"model":"` + name + `/` + model + `"}}}`, name
}

// TestARouteToAPinnedLocalModelIsProbedAndHeld: opencode's machine model is
// unpinned, and a --route asks it for the pinned one: the run is probed and
// pointed at the probed server through a sealed configuration, and the record
// carries the local pin, as a machine route to that model does.
func TestARouteToAPinnedLocalModelIsProbedAndHeld(t *testing.T) {
	personsOpenCode(t)
	o := newOllamaRunFake(t, true)
	base := o.srv.URL + "/v1"
	machine, name := overridePinMachine(base, "devstral")
	c := mustLoad(t, machine, "")
	text := "implementer=runner:opencode@" + name + "/" + localRunModel
	if err := c.Override(flagRoutes(t, text)); err != nil {
		t.Fatal(err)
	}
	f := newFake(t, "ok", Claude, OpenCode)
	out, err := newHarness(t, c, true).d.Dispatch(context.Background(), f.request(roleImplementer))
	if err != nil {
		t.Fatal(err)
	}
	if out.Handoff || out.Answer == nil || out.Fallback != nil {
		t.Fatalf("outcome = %+v", out)
	}
	if n := o.probes.Load(); n != 1 {
		t.Fatalf("probes = %d, want the --route's pinned model probed once", n)
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "opencode.config.json"))
	if err != nil {
		t.Fatalf("no sealed configuration: %v", err)
	}
	if string(raw) != wantLocalConfig(name, base) {
		t.Fatalf("sealed configuration =\n%s\nwant\n%s", raw, wantLocalConfig(name, base))
	}
	r := out.Receipt.Route
	if r.LocalPin == nil || r.LocalPin.WeightsSHA256 != localRunDigest || r.LocalServed == nil || r.Override != text {
		t.Fatalf("route = %+v, want the local pin and the --route", r)
	}
	// The machine's runner keeps its own model, unpinned.
	if rc, _ := c.Runner(OpenCode); rc.LocalPin != nil || rc.Model != name+"/devstral" {
		t.Fatalf("the machine's runner changed: %+v", rc)
	}
}

// TestARouteToAnUnpinnedModelCarriesNoMachinePin: opencode's machine model is
// pinned, and a --route asks it for an unpinned one: nothing is probed, no
// sealed configuration is written, and the record carries no pin, since the
// machine model's pin is not the --route model's.
func TestARouteToAnUnpinnedModelCarriesNoMachinePin(t *testing.T) {
	o := newOllamaRunFake(t, true)
	machine, name := overridePinMachine(o.srv.URL+"/v1", localRunModel)
	c := mustLoad(t, machine, "")
	if err := c.Override(flagRoutes(t, "implementer=runner:opencode@"+name+"/devstral")); err != nil {
		t.Fatal(err)
	}
	f := newFake(t, "ok", Claude, OpenCode)
	out, err := newHarness(t, c, true).d.Dispatch(context.Background(), f.request(roleImplementer))
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer == nil || o.probes.Load() != 0 {
		t.Fatalf("outcome %+v, probes %d: an unpinned --route model was probed", out, o.probes.Load())
	}
	if _, err := os.Stat(filepath.Join(f.log, "opencode.config.json")); err == nil {
		t.Fatal("an unpinned --route model was run against a sealed configuration")
	}
	if r := out.Receipt.Route; r.LocalPin != nil || r.ProviderPin != nil || r.ModelAsked != name+"/devstral" {
		t.Fatalf("route = %+v, want the --route's model and no pin", r)
	}
}

// TestClaudeRefusesARouteToAPinnedModel: the claude runner can hold no pin,
// so a --route asking it for a model the machine pins is refused before
// anything is launched or probed and the role falls back to the host, exactly
// as a machine route to that model does.
func TestClaudeRefusesARouteToAPinnedModel(t *testing.T) {
	o := newOllamaRunFake(t, true)
	base := o.srv.URL + "/v1"
	machine, name := overridePinMachine(base, "devstral")
	pinned := name + "/" + localRunModel

	// The person's --route names claude for a model the machine pins: the
	// configuration already shows claude cannot hold it, so the route is
	// refused before anything is written, naming the runner that can.
	viaFlag := mustLoad(t, machine, "")
	err := viaFlag.Override(flagRoutes(t, "implementer=runner:claude@"+pinned))
	if err == nil || !strings.Contains(err.Error(), "runner.opencode.model") || !strings.Contains(err.Error(), "pins") {
		t.Fatalf("a --route to claude for a pinned model: err = %v, want a refusal naming opencode", err)
	}
	// So is one naming claude with no model when claude's own model is pinned.
	claudePinned := strings.Replace(machine, `"claude":{}`, `"claude":{"model":"`+pinned+`"}`, 1)
	if err := mustLoad(t, claudePinned, "").Override(flagRoutes(t, "implementer=runner:claude")); err == nil ||
		!strings.Contains(err.Error(), "runner.opencode.model") {
		t.Fatalf("a --route to claude whose own model is pinned: err = %v, want a refusal naming opencode", err)
	}

	// A machine route nobody asked for on this step keeps its fallback:
	// claude refuses at launch and the role is handed to the host.
	viaMachine := mustLoad(t, `{"roles":{"implementer":{"runner":"claude"}},`+strings.TrimPrefix(claudePinned, "{"), "")
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, viaMachine, true)
	out, err := h.d.Dispatch(context.Background(), f.request(roleImplementer))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Handoff || out.Answer != nil || out.Fallback == nil {
		t.Fatalf("outcome = %+v, want the host handed the role after a fallback", out)
	}
	if fb := out.Fallback; fb.Asked != Claude || fb.Reason != ReasonFailed || !strings.Contains(fb.Detail, "pins") {
		t.Fatalf("fallback = %+v, want claude refusing the pinned model", fb)
	}
	if f.launched(Claude) {
		t.Fatal("claude was launched on a pinned model")
	}
	if o.probes.Load() != 0 {
		t.Fatal("a pinned claude route was probed")
	}
}
