package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
