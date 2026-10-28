package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The drain state's version 2: a routed judge's stored route (a
// DrainJudgement's runner.RouteRecord) can carry a route's pins and say that
// its model is local, which version 1 never wrote. Version 1 is read as
// version 2's strict subset and written back at 2; a version-1 file carrying
// any of those fields is refused naming it; a version above 2 is a newer
// abcd's.

// judgedDrain is writtenDrain with its first judgement answered by a routed
// judge whose route carries sides.
func judgedDrain(sides runner.Sides) DrainState {
	st := writtenDrain()
	st.Judgements[0].Route = &runner.RouteRecord{Asked: "opencode", Ran: "opencode", Model: "qwen3-coder", Sides: sides}
	return st
}

// writeDrainAt writes st with its schema_version set to version, as an abcd
// that wrote that version would have left it, and returns the file's path.
func writeDrainAt(t *testing.T, root string, st DrainState, version int) string {
	t.Helper()
	if err := fsutil.EnsureRealDirAll(root, RunRelDir, dirPerm); err != nil {
		t.Fatal(err)
	}
	st.SchemaVersion = version
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(DrainStateRel))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func storedDrainVersion(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var peek struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		t.Fatal(err)
	}
	return peek.SchemaVersion
}

func TestDrainSchemaVersionIsTwo(t *testing.T) {
	if DrainSchemaVersion != 2 {
		t.Fatalf("DrainSchemaVersion = %d, want 2", DrainSchemaVersion)
	}
}

// TestAVersionOneDrainStateReadsAndIsWrittenBackAtTwo: a version-1 drain
// state carrying none of version 2's fields, the sides version 1 already
// recorded kept, reads as the current version, and the drain's next write
// stores it at 2.
func TestAVersionOneDrainStateReadsAndIsWrittenBackAtTwo(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	path := writeDrainAt(t, root, judgedDrain(runner.Sides{ModelAsked: "qwen3-coder", HarnessVersion: "1.2.3"}), 1)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-1 drain state reads: %v", err)
	}
	if st.SchemaVersion != DrainSchemaVersion {
		t.Errorf("a version-1 drain state is read at version %d, want %d", st.SchemaVersion, DrainSchemaVersion)
	}
	if r := st.Judgements[0].Route; r == nil || r.HarnessVersion != "1.2.3" {
		t.Errorf("the version-1 judge route's sides are kept: %+v", r)
	}
	if err := writeDrain(root, st); err != nil {
		t.Fatal(err)
	}
	if v := storedDrainVersion(t, path); v != 2 {
		t.Errorf("the drain writes a version-1 state back at version %d, want 2", v)
	}
}

// TestAVersionOneDrainStateCarryingAPinIsRefusedNamingIt: version 1 never
// wrote a pin or `local` on the judge's route, so a version-1 file carrying
// one is not a file version 1 wrote; it is refused naming the key and where
// it sits.
func TestAVersionOneDrainStateCarryingAPinIsRefusedNamingIt(t *testing.T) {
	cases := []struct {
		key   string
		sides runner.Sides
	}{
		{"local", runner.Sides{Local: true}},
		{"provider_pin", runner.Sides{ProviderPin: &oracle.Pin{Order: []string{"deepinfra"}, DataCollection: "deny"}}},
		{"pin_config_sha256", runner.Sides{PinConfigSHA256: strings.Repeat("ab", 32)}},
		{"local_pin", runner.Sides{LocalPin: &oracle.LocalPin{Runtime: "ollama", RuntimeVersion: "0.12.6"}}},
		{"local_served", runner.Sides{LocalServed: &oracle.LocalServed{Model: "qwen3-coder"}}},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			root := repo.Root()
			st := judgedDrain(c.sides)
			writeDrainAt(t, root, st, 1)
			_, _, err := readDrain(root)
			rf := mustRefusal(t, err)
			if !strings.Contains(rf.Reason, "schema version 1") || !strings.Contains(rf.Reason, "`"+c.key+"`") ||
				!strings.Contains(rf.Reason, st.Judgements[0].Issue) {
				t.Errorf("a version-1 drain state carrying %s is refused naming it, the judgement and the version: %q", c.key, rf.Reason)
			}
			if !strings.Contains(rf.Remedy, "restore it or remove it") {
				t.Errorf("the remedy is the drain's own: %q", rf.Remedy)
			}
		})
	}
}

// TestAVersionTwoDrainStateCarryingLocalReads: version 2 writes the judge
// route's pins and `local`, so its own file carrying one reads.
func TestAVersionTwoDrainStateCarryingLocalReads(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, judgedDrain(runner.Sides{Local: true}), 2)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-2 drain state carrying local reads: %v", err)
	}
	if r := st.Judgements[0].Route; r == nil || !r.Local {
		t.Errorf("the judge route's local is kept: %+v", r)
	}
}

// TestAVersionThreeDrainStateIsRefusedAsNewer: a version above 2 is a newer
// abcd's, refused as that before any strict decode (iss-2610230223231843).
func TestAVersionThreeDrainStateIsRefusedAsNewer(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, judgedDrain(runner.Sides{Local: true}), 3)
	_, _, err := readDrain(root)
	rf := mustRefusal(t, err)
	if !strings.Contains(rf.Reason, "schema version 3, which a newer abcd wrote") || !strings.Contains(rf.Reason, "reads version 2") {
		t.Errorf("a version-3 drain state is refused as a newer abcd's: %q", rf.Reason)
	}
}

// filledJudgeRoute is a judge route carrying every string both sides and the
// pins can hold, each clean.
func filledJudgeRoute() runner.Sides {
	return runner.Sides{ModelAsked: "qwen3-coder", ServedModels: []string{"qwen/qwen3-coder"}, ServedProvider: "deepinfra",
		FinishReason: "stop", HarnessVersion: "1.0.0", PromptSHA256: strings.Repeat("a", 64),
		Cost: &oracle.Cost{Amount: 0.01, Unit: "usd"}, Unpinned: []oracle.Unpinned{{Field: "served.model", Reason: "not_verified"}},
		ProviderPin: &oracle.Pin{Order: []string{"deepinfra"}, DataCollection: "deny", Quantizations: []string{"fp8"},
			CanonicalSlug: "qwen/qwen3-coder"},
		PinConfigSHA256: strings.Repeat("b", 64),
		LocalPin: &oracle.LocalPin{Runtime: "ollama", RuntimeVersion: "0.12.6", WeightsSHA256: strings.Repeat("c", 64),
			WeightsFile: "qwen3-coder.gguf", Quantisation: "Q4_K_M", KVCacheType: "f16"},
		LocalServed: &oracle.LocalServed{Model: "qwen3-coder", RuntimeVersion: "0.12.6", WeightsSHA256: strings.Repeat("c", 64),
			Quantisation: "Q4_K_M"},
		Local: true}
}

// TestAJudgeRouteCarryingTextTheDrainNeverWritesIsRefused: every string a
// version-2 judge route stores, both sides and its pins included, reaches
// `drain --json` as stored, so a bidi rune in any of them is refused when the
// drain state is read, as it is in the judge route's asked, ran and model.
func TestAJudgeRouteCarryingTextTheDrainNeverWritesIsRefused(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, judgedDrain(filledJudgeRoute()), 2)
	if _, _, err := readDrain(root); err != nil {
		t.Fatalf("a version-2 judge route carrying clean text reads: %v", err)
	}
	const bad = "\u202e"
	for name, forge := range map[string]func(*runner.Sides){
		"model_asked":                 func(s *runner.Sides) { s.ModelAsked += bad },
		"served_models":               func(s *runner.Sides) { s.ServedModels[0] += bad },
		"served_provider":             func(s *runner.Sides) { s.ServedProvider += bad },
		"finish_reason":               func(s *runner.Sides) { s.FinishReason += bad },
		"harness_version":             func(s *runner.Sides) { s.HarnessVersion += bad },
		"prompt_sha256":               func(s *runner.Sides) { s.PromptSHA256 += bad },
		"cost.unit":                   func(s *runner.Sides) { s.Cost.Unit += bad },
		"unpinned.reason":             func(s *runner.Sides) { s.Unpinned[0].Reason += bad },
		"provider_pin.order":          func(s *runner.Sides) { s.ProviderPin.Order[0] += bad },
		"provider_pin.canonical_slug": func(s *runner.Sides) { s.ProviderPin.CanonicalSlug += bad },
		"pin_config_sha256":           func(s *runner.Sides) { s.PinConfigSHA256 += bad },
		"local_pin.runtime":           func(s *runner.Sides) { s.LocalPin.Runtime += bad },
		"local_pin.weights_file":      func(s *runner.Sides) { s.LocalPin.WeightsFile += bad },
		"local_served.model":          func(s *runner.Sides) { s.LocalServed.Model += bad },
		"local_served.quantisation":   func(s *runner.Sides) { s.LocalServed.Quantisation += bad },
	} {
		t.Run(name, func(t *testing.T) {
			sides := filledJudgeRoute()
			forge(&sides)
			writeDrainAt(t, root, judgedDrain(sides), 2)
			_, _, err := readDrain(root)
			r := mustRefusal(t, err)
			noRawRune(t, name, r)
			if r.Stage != StageDrain || !strings.Contains(r.Reason, "text the drain never writes") {
				t.Fatalf("a judge route's %s carrying U+202E: refused at the drain as text it never writes: %+v", name, r)
			}
		})
	}
}
