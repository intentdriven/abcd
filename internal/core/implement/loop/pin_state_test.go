package loop

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// pinnedRoutes are route records carrying only a pin, one per pin a route
// can record; each is part of version 14, the version that added both sides.
func pinnedRoutes() map[string]runner.RouteRecord {
	return map[string]runner.RouteRecord{
		"provider pin": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{ProviderPin: &oracle.Pin{Order: []string{"deepinfra"}, DataCollection: "deny",
				Quantizations: []string{"fp8"}, CanonicalSlug: "qwen/qwen3-coder"}}},
		"pin config hash": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{PinConfigSHA256: strings.Repeat("b", 64)}},
		"local pin": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{LocalPin: &oracle.LocalPin{Runtime: "ollama", RuntimeVersion: "0.12.6",
				WeightsSHA256: strings.Repeat("c", 64), Quantisation: "Q4_K_M", ContextLength: 32768, KVCacheType: "f16"}}},
		"local served": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{LocalServed: &oracle.LocalServed{Model: "qwen3-coder", ContextLength: 32768}}},
		"local route": {Asked: runner.OpenCode, Ran: runner.OpenCode, Sides: runner.Sides{Local: true}},
	}
}

// TestAPinnedRouteIsVersion14s: a route carrying a pin and nothing else of
// both sides round-trips at version 14, and a file of version 13 or lower
// carrying it is not one its version wrote (v0.15.1 wrote version 13
// without the pin fields) and is refused, naming the version.
func TestAPinnedRouteIsVersion14s(t *testing.T) {
	root, id, path := sidedRun(t)
	base, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, r := range pinnedRoutes() {
		var route map[string]any
		raw, _ := json.Marshal(r)
		_ = json.Unmarshal(raw, &route)
		for _, version := range []int{SchemaVersion, schemaVersionUnmetered, schemaVersionUnrefused} {
			if err := os.WriteFile(path, base, 0o600); err != nil {
				t.Fatal(err)
			}
			editState(t, root, id, func(doc map[string]any) {
				doc["schema_version"] = version
				doc["lanes"].([]any)[0].(map[string]any)["receipts"].([]any)[0].(map[string]any)["route"] = route
			})
			st, err := ReadState(root, id)
			if version == SchemaVersion {
				if err != nil {
					t.Fatalf("%s at version %d: %v", name, version, err)
				}
				got := st.Lanes[0].Receipts[0].Route
				if got == nil || got.PinConfigSHA256 != r.PinConfigSHA256 || (r.ProviderPin != nil) != (got.ProviderPin != nil) ||
					(r.LocalPin != nil) != (got.LocalPin != nil) || (r.LocalServed != nil) != (got.LocalServed != nil) || r.Local != got.Local {
					t.Fatalf("%s read back = %+v", name, got)
				}
				continue
			}
			rf := mustRefusal(t, err)
			if !strings.Contains(rf.Reason, "schema version") || !strings.Contains(rf.Reason, "both sides") {
				t.Errorf("%s at version %d is refused naming the version: %+v", name, version, rf)
			}
		}
	}
}
