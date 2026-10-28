package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// pinnedRoutes are route records carrying only a pin, one per pin a route
// can record, keyed by the route record's JSON key that carries it; each is
// part of version 16, the version after the run's review set.
func pinnedRoutes() map[string]runner.RouteRecord {
	return map[string]runner.RouteRecord{
		"provider_pin": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{ProviderPin: &oracle.Pin{Order: []string{"deepinfra"}, DataCollection: "deny",
				Quantizations: []string{"fp8"}, CanonicalSlug: "qwen/qwen3-coder"}}},
		"pin_config_sha256": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{PinConfigSHA256: strings.Repeat("b", 64)}},
		"local_pin": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{LocalPin: &oracle.LocalPin{Runtime: "ollama", RuntimeVersion: "0.12.6",
				WeightsSHA256: strings.Repeat("c", 64), Quantisation: "Q4_K_M", ContextLength: 32768, KVCacheType: "f16"}}},
		"local_served": {Asked: runner.OpenCode, Ran: runner.OpenCode,
			Sides: runner.Sides{LocalServed: &oracle.LocalServed{Model: "qwen3-coder", ContextLength: 32768}}},
		"local": {Asked: runner.OpenCode, Ran: runner.OpenCode, Sides: runner.Sides{Local: true}},
	}
}

// TestAPinnedRouteIsVersion16s: a route carrying a pin and nothing else of
// both sides round-trips at version 16. A file of version 14 or 15 carrying
// it is not one its version wrote (both came before the pin fields) and is
// refused, naming the version and the key; one of version 13 or lower is
// refused with the sides it carries (v0.15.1 wrote neither).
func TestAPinnedRouteIsVersion16s(t *testing.T) {
	if SchemaVersion != 16 {
		t.Fatalf("a route's pins are state version 16's, got %d", SchemaVersion)
	}
	root, id, path := sidedRun(t)
	base, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for key, r := range pinnedRoutes() {
		var route map[string]any
		raw, _ := json.Marshal(r)
		_ = json.Unmarshal(raw, &route)
		for _, version := range []int{16, 15, 14, 13, 12} {
			if err := os.WriteFile(path, base, 0o600); err != nil {
				t.Fatal(err)
			}
			editState(t, root, id, func(doc map[string]any) {
				doc["schema_version"] = version
				doc["lanes"].([]any)[0].(map[string]any)["receipts"].([]any)[0].(map[string]any)["route"] = route
			})
			st, err := ReadState(root, id)
			switch version {
			case 16:
				if err != nil {
					t.Fatalf("%s at version %d: %v", key, version, err)
				}
				got := st.Lanes[0].Receipts[0].Route
				if got == nil || got.PinConfigSHA256 != r.PinConfigSHA256 || (r.ProviderPin != nil) != (got.ProviderPin != nil) ||
					(r.LocalPin != nil) != (got.LocalPin != nil) || (r.LocalServed != nil) != (got.LocalServed != nil) || r.Local != got.Local {
					t.Fatalf("%s read back = %+v", key, got)
				}
			case 15, 14:
				if err == nil {
					t.Fatalf("%s at version %d is not one version %d wrote, but it read", key, version, version)
				}
				rf := mustRefusal(t, err)
				if !strings.Contains(rf.Reason, fmt.Sprintf("schema version %d", version)) || !strings.Contains(rf.Reason, "`"+key+"`") {
					t.Errorf("%s at version %d is refused naming the version and the key: %+v", key, version, rf)
				}
			default:
				rf := mustRefusal(t, err)
				if !strings.Contains(rf.Reason, fmt.Sprintf("schema version %d", version)) || !strings.Contains(rf.Reason, "both sides") {
					t.Errorf("%s at version %d is refused naming the version: %+v", key, version, rf)
				}
			}
		}
	}
}

// TestAVersion15FileReadsAsVersion16s is the clean version-15 rule: a file
// as the version before pins writes it, both sides and no pin, is version
// 16's strict subset. It reads as the current version, decodes and re-encodes
// losslessly apart from its version, and the run's next mutation writes it
// back at 16.
func TestAVersion15FileReadsAsVersion16s(t *testing.T) {
	if SchemaVersion != 16 {
		t.Fatalf("a route's pins are state version 16's, got %d", SchemaVersion)
	}
	root, id, path := sidedRun(t)
	editState(t, root, id, func(doc map[string]any) { doc["schema_version"] = 15 })
	old, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(root, id)
	if err != nil || st.SchemaVersion != 16 {
		t.Fatalf("a clean version-15 file reads as version 16: %v %v", st.SchemaVersion, err)
	}
	if r := st.Lanes[0].Receipts[0].Route; r == nil || r.Usage == nil || r.ModelAsked != sidedRoute().ModelAsked ||
		r.ProviderPin != nil || r.PinConfigSHA256 != "" || r.LocalPin != nil || r.LocalServed != nil || r.Local {
		t.Fatalf("a version-15 route keeps both sides and carries no pin: %+v", r)
	}
	encoded, err := json.MarshalIndent(st.masked(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var oldDoc, newDoc map[string]any
	if err := json.Unmarshal(old, &oldDoc); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &newDoc); err != nil {
		t.Fatal(err)
	}
	oldDoc["schema_version"] = float64(16)
	if !reflect.DeepEqual(oldDoc, newDoc) {
		t.Fatalf("re-encoding the version-15 file changed more than its version:\nread:    %s\nwritten: %s", old, encoded)
	}
	if err := mutate(root, id, func(*os.Root, *State) (bool, error) { return true, nil }); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var writtenDoc map[string]any
	if err := json.Unmarshal(written, &writtenDoc); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(oldDoc, writtenDoc) {
		t.Fatalf("the next mutation writes the version-15 file back at version 16, nothing else changed:\nread:    %s\nwritten: %s", old, written)
	}
	again, err := ReadState(root, id)
	if err != nil || !reflect.DeepEqual(st, again) {
		t.Fatalf("the file written back decodes to the state the version-15 file did: %v\n%+v\n%+v", err, st, again)
	}
}
