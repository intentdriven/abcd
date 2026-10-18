package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The tests in this file are itd-166's loop half: both sides of a run travel
// with the route record the run state already stores on a verified receipt
// and a validator's return, at schema version 14, and the drive passes them
// through from the runner's answer.

// sidedRoute is a runner's route record carrying both sides.
func sidedRoute() runner.RouteRecord {
	return runner.RouteRecord{Asked: runner.OpenCode, Ran: runner.OpenCode, Model: "",
		Sides: runner.Sides{ModelAsked: "local/qwen3-coder", FinishReason: "stop", PromptSHA256: strings.Repeat("a", 64),
			Usage:    &oracle.Usage{PromptTokens: 1600, CompletionTokens: 300},
			Cost:     &oracle.Cost{Amount: 0.002, Unit: oracle.CostUSD},
			Unpinned: []oracle.Unpinned{{Field: oracle.ServedModel, Reason: oracle.NotReported}}}}
}

// sidedRun is a run whose implement stage a runner ran: its verified receipt
// carries a route with both sides. It returns the root, the run and the
// state file's path.
func sidedRun(t *testing.T) (string, string, string) {
	t.Helper()
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "ok", "claude")
	cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`, "")
	if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}); err != nil {
		t.Fatal(err)
	}
	r := sidedRoute()
	editState(t, root, id, func(doc map[string]any) {
		var route map[string]any
		raw, _ := json.Marshal(r)
		_ = json.Unmarshal(raw, &route)
		doc["lanes"].([]any)[0].(map[string]any)["receipts"].([]any)[0].(map[string]any)["route"] = route
	})
	return root, id, filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
}

// TestASidedRouteRoundTripsAtVersion14 is criterion 7's run-state half: a
// route record carrying both sides is read back whole at the current
// version, 14, which is the version that added them.
func TestASidedRouteRoundTripsAtVersion14(t *testing.T) {
	if SchemaVersion != 14 || schemaVersionUnmetered != 13 || schemaVersionUnrefused != 12 {
		t.Fatalf("both sides are version 14's, after a lane's refusal's 13: %d %d %d", SchemaVersion, schemaVersionUnmetered, schemaVersionUnrefused)
	}
	root, id, _ := sidedRun(t)
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	got := st.Lanes[0].Receipts[0].Route
	want := sidedRoute()
	if got == nil || got.ModelAsked != want.ModelAsked || got.Usage == nil || *got.Usage != *want.Usage ||
		got.Cost == nil || *got.Cost != *want.Cost || len(got.Unpinned) != 1 || got.Unpinned[0] != want.Unpinned[0] {
		t.Fatalf("route read back = %+v, want %+v", got, want)
	}
}

// TestAVersion13FileReadsAsNotRecorded is the version-13 rule: version 13 is
// version 14's strict subset, so a version-13 file without either side reads
// as the current version, its routes not recorded, and one of version 13 or
// lower carrying a side is not a file its version wrote and is refused,
// naming the version.
func TestAVersion13FileReadsAsNotRecorded(t *testing.T) {
	root, id, path := sidedRun(t)
	sided, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []int{schemaVersionUnmetered, schemaVersionUnrefused, schemaVersionUnanswered, schemaVersionUnrouted + 1} {
		if err := os.WriteFile(path, sided, 0o600); err != nil {
			t.Fatal(err)
		}
		editState(t, root, id, func(doc map[string]any) { doc["schema_version"] = version })
		_, err := ReadState(root, id)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, fmt.Sprintf("schema version %d", version)) || !strings.Contains(r.Reason, "both sides") {
			t.Errorf("version %d carrying a route's sides is refused naming the version: %+v", version, r)
		}
	}
	if err := os.WriteFile(path, sided, 0o600); err != nil {
		t.Fatal(err)
	}
	editState(t, root, id, func(doc map[string]any) {
		doc["schema_version"] = schemaVersionUnmetered
		doc["lanes"].([]any)[0].(map[string]any)["receipts"].([]any)[0].(map[string]any)["route"] =
			map[string]any{"asked": "claude", "ran": "claude", "model": "fake-model"}
	})
	st, err := ReadState(root, id)
	if err != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("a version-13 file without sides reads as the current version: %v %v", st.SchemaVersion, err)
	}
	if r := st.Lanes[0].Receipts[0].Route; r == nil || r.Usage != nil || r.Unpinned != nil || r.ModelAsked != "" {
		t.Fatalf("a version-13 route reads as not recorded: %+v", r)
	}
}

// TestTheDrivePassesBothSidesThrough: a role the loop runs through a runner
// has the runner's asked model, token counts and cost on the route its
// verified receipt and its result name, the same route on both.
func TestTheDrivePassesBothSidesThrough(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "metered", "claude")
	cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`, "")
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	r := res.Route
	if r == nil || r.HarnessVersion != "2.1.302" || r.Usage == nil || r.Usage.CompletionTokens != 76 ||
		r.Cost == nil || r.Cost.Unit != oracle.CostUSD || len(r.Unpinned) == 0 {
		t.Fatalf("the result's route carries both sides: %+v", r)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	stored := st.Lanes[0].Receipts[0].Route
	if stored == nil || stored.Usage == nil || *stored.Usage != *r.Usage || stored.HarnessVersion != r.HarnessVersion {
		t.Fatalf("the verified receipt's route carries the same sides: %+v", stored)
	}
	if st.SchemaVersion != SchemaVersion {
		t.Fatalf("written at version %d", st.SchemaVersion)
	}
}
