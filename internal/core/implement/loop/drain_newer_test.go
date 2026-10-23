package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestADrainStateANewerAbcdWroteIsRefusedAsThat: a drain state of a later
// schema version, carrying a field this abcd does not know, is a newer abcd's
// file. It is refused as that before any strict decode, telling the person to
// run the newer abcd and not to remove the file, rather than read as one that
// does not parse, whose remedy would throw the newer run's drain away
// (iss-2610230223231843). The run state already does this (decodeState).
func TestADrainStateANewerAbcdWroteIsRefusedAsThat(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	if err := fsutil.EnsureRealDirAll(root, RunRelDir, dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := writeDrain(root, writtenDrain()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(DrainStateRel))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(good, &doc); err != nil {
		t.Fatal(err)
	}
	doc["schema_version"] = DrainSchemaVersion + 1
	doc["a_field_a_newer_abcd_adds"] = true
	newer, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, newer, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = readDrain(root)
	rf := mustRefusal(t, err)
	if !strings.Contains(rf.Reason, "a newer abcd wrote") || strings.Contains(rf.Reason, "does not parse") {
		t.Errorf("a newer drain state is refused as a newer abcd's, not as unparseable: %q", rf.Reason)
	}
	if strings.Contains(rf.Remedy, "remove it") || !strings.Contains(rf.Remedy, "do not remove") {
		t.Errorf("the remedy keeps the newer drain state rather than removing it: %q", rf.Remedy)
	}

	// The drain's own version is read as before, and a version this abcd never
	// wrote that is not newer still does not parse.
	if err := os.WriteFile(path, good, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readDrain(root); err != nil {
		t.Fatalf("the drain's own state reads: %v", err)
	}
	doc["schema_version"] = 0
	delete(doc, "a_field_a_newer_abcd_adds")
	older, _ := json.Marshal(doc)
	if err := os.WriteFile(path, older, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = readDrain(root)
	if rf := mustRefusal(t, err); !strings.Contains(rf.Reason, "does not parse") {
		t.Errorf("a drain state of version 0 is still refused as not parsing: %q", rf.Reason)
	}
}
