package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// TestAStepsRouteOverrideRunsTheRoleOnItsRunnerAndModel is
// iss-2610161801358637 in the loop: a step's --route hands the implementer to
// a runner with a model of its own although no file routes it, the runner is
// launched asking for that model, and the verified receipt in the run's state
// and the run record carry the --route verbatim.
func TestAStepsRouteOverrideRunsTheRoleOnItsRunnerAndModel(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	env := newDriveEnv(t, "ok", "claude")
	cfg := runnerConfig(t, `{"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["devstral"]}}},`+
		`"runner":{"claude":{}}}`, "")
	text := "implementer=runner:claude@local/devstral"
	routes, err := oracle.ParseRunnerRoutes([]string{text}, []string{RoleImplementer})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Override(routes); err != nil {
		t.Fatal(err)
	}
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.PerformedStage != StageImplement || res.Route == nil || res.Route.Ran != runner.Claude || res.Route.Override != text {
		t.Fatalf("the step ran the role on the --route's runner: %+v route %+v", res, res.Route)
	}
	if argv := env.argv(t, "claude"); !slices.Contains(argv, "--model=devstral") {
		t.Fatalf("the launch asks for the --route's model: %q", argv)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	rs := st.Lanes[0].Receipts
	if len(rs) != 1 || rs[0].Route == nil || rs[0].Route.Override != text {
		t.Fatalf("the verified receipt records the --route: %+v", rs)
	}
	named := false
	for _, e := range st.Record {
		if e.Stage == StageRunner && strings.Contains(e.Note, "--route "+text) {
			named = true
		}
	}
	if !named {
		t.Fatalf("the run record names the --route: %+v", st.Record)
	}
}

// overrideDrive drives the implement stage under a --route, so the verified
// receipt in the run's state carries `override`, and returns the state path.
func overrideDrive(t *testing.T) (root, id string, steps Stages, o Options, path string) {
	t.Helper()
	root, id, steps, o = startToImplement(t)
	newDriveEnv(t, "ok", "claude")
	cfg := runnerConfig(t, `{"runner":{"claude":{}}}`, "")
	routes, err := oracle.ParseRunnerRoutes([]string{"implementer=runner:claude"}, []string{RoleImplementer})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Override(routes); err != nil {
		t.Fatal(err)
	}
	if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}); err != nil {
		t.Fatal(err)
	}
	return root, id, steps, o, filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
}

// TestARouteOverrideIsStateVersion16s: a receipt's `override` is state
// version 16's, beside a route's pins; version 15 (the run's review set) was
// written without it. A file of version 15 or lower carrying `override` and
// nothing else of a later version is not one its version wrote, and is
// refused naming the key; version 15 and lower without it read as the
// current version and are written back at it by the next mutation. A
// version-12 file without it decodes and re-encodes losslessly apart from
// its version.
func TestARouteOverrideIsStateVersion16s(t *testing.T) {
	if schemaVersionUnplanned != 16 {
		t.Fatalf("the --route's record is state version 16's, got %d", schemaVersionUnplanned)
	}
	root, id, steps, o, path := overrideDrive(t)
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(current), `"override"`) {
		t.Fatalf("the verified receipt carries the --route:\n%s", current)
	}
	for _, version := range []int{15, 14, 13, 12, 11, 9} {
		if err := os.WriteFile(path, current, 0o600); err != nil {
			t.Fatal(err)
		}
		editState(t, root, id, func(doc map[string]any) {
			doc["schema_version"] = version
			bareRoutes(doc, true)
		})
		_, err := ReadState(root, id)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, fmt.Sprintf("schema version %d", version)) || !strings.Contains(r.Reason, "`override`") {
			t.Errorf("version %d carrying `override` is refused naming the version and the key: %+v", version, r)
		}
	}
	if err := os.WriteFile(path, current, 0o600); err != nil {
		t.Fatal(err)
	}
	editState(t, root, id, func(doc map[string]any) {
		doc["schema_version"] = 12
		bareRoutes(doc, false)
	})
	st, err := ReadState(root, id)
	if err != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("a version-12 file without `override` reads as the current version: %v %v", st.SchemaVersion, err)
	}
	// The version-12 file decodes and re-encodes losslessly apart from its
	// version: the document written back is the one read with the version
	// raised, and it decodes to the state the version-12 file did.
	old, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
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
	oldDoc["schema_version"] = float64(SchemaVersion)
	if !reflect.DeepEqual(oldDoc, newDoc) {
		t.Fatalf("re-encoding the version-12 file changed more than its version:\nread:    %s\nwritten: %s", old, encoded)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	again, err := ReadState(root, id)
	if err != nil || !reflect.DeepEqual(st, again) {
		t.Fatalf("the re-encoded file decodes to the state the version-12 file did: %v\n%+v\n%+v", err, st, again)
	}
	if err := os.WriteFile(path, old, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := advance(root, id, steps, o); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(written, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["schema_version"] != float64(SchemaVersion) {
		t.Fatalf("the next mutation writes the file back at version %d:\n%s", SchemaVersion, written)
	}
}

// bareRoutes strips every route record in a state document down to what
// version 13 wrote, its asked, ran and model, keeping `override` when keep
// says so, and drops the run's review set: both sides of the run, which
// version 14 added, the review set, which version 15 added, and a route's
// pins, which version 16 added beside `override`, are refused on their own,
// and this leaves `override` the one key under test.
func bareRoutes(doc map[string]any, keep bool) {
	delete(doc, "review")
	bare := func(route map[string]any) {
		for k := range route {
			if k != "asked" && k != "ran" && k != "model" && !(keep && k == "override") {
				delete(route, k)
			}
		}
	}
	for _, l := range doc["lanes"].([]any) {
		lane := l.(map[string]any)
		rs, _ := lane["receipts"].([]any)
		for _, r := range rs {
			if route, ok := r.(map[string]any)["route"].(map[string]any); ok {
				bare(route)
			}
		}
		vs, _ := lane["validation"].([]any)
		for _, v := range vs {
			vrs, _ := v.(map[string]any)["validators"].([]any)
			for _, vr := range vrs {
				if route, ok := vr.(map[string]any)["route"].(map[string]any); ok {
					bare(route)
				}
			}
		}
	}
}
