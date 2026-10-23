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
