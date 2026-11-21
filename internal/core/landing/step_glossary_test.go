package landing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheStepEntryRegistersTheLandVerbsOwnSteps holds the step glossary entry
// to the land chapter's vocabulary (iss-2610262038278710). The land chapter
// calls its own gate, push and enqueue acts steps ("Every step is idempotent")
// and its JSON document carries a `steps` field, while the same page cites a
// spec's step in the glossary's sense. The entry must carry a Senses table
// whose qualified row names the land verb's steps, so the chapter's second
// sense is registered and kept apart from a spec's step.
func TestTheStepEntryRegistersTheLandVerbsOwnSteps(t *testing.T) {
	const entry = ".abcd/development/brief/glossary/core/step.md"
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("read %s: %v", entry, err)
	}
	text := string(raw)

	start := strings.Index(text, "## Senses")
	if start < 0 {
		t.Fatalf("%s has no \"## Senses\" section registering the land verb's own steps", entry)
	}
	section := text[start+len("## Senses"):]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}

	var landRow string
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "|") && strings.Contains(line, "**a land step**") {
			landRow = line
			break
		}
	}
	if landRow == "" {
		t.Fatalf("%s's Senses table has no row spelled \"**a land step**\" for the land verb's own steps", entry)
	}
	for _, want := range []string{"`gate`", "`push`", "`enqueue`", "`steps`", "38-land.md", "idempotent"} {
		if !strings.Contains(landRow, want) {
			t.Errorf("%s's land-step row does not name %s", entry, want)
		}
	}
	if !strings.Contains(section, "**step**, unqualified") {
		t.Errorf("%s's Senses table does not keep unqualified \"step\" for a spec's step", entry)
	}
}
