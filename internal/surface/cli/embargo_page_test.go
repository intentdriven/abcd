package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbargoPageNamesThePersonsActs is criterion 10 of itd-2610112239022766:
// the `embargo` command page says in plain words what stays the person's act in
// this release (fixing and landing the fixes, requesting a CVE, putting back
// redacted records) and that abcd closes or publishes an advisory only on the
// person's own approval of what it showed them. The page wraps its prose, so
// the section is read with its whitespace folded.
func TestEmbargoPageNamesThePersonsActs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "embargo.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	const heading = "## What stays the person's act"
	start := strings.Index(page, heading)
	if start < 0 {
		t.Fatalf("the embargo page has no %q section", heading)
	}
	section := page[start+len(heading):]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	folded := strings.Join(strings.Fields(section), " ")
	for _, want := range []string{
		"fixing each vulnerability, landing the fixes",
		"requesting a CVE for an advisory",
		"putting back any record details they redacted",
		"restoring each record into the public ledger",
		"abcd closes or publishes an advisory only on the person's own approval of what it showed them",
	} {
		if !strings.Contains(folded, want) {
			t.Errorf("the embargo page's %q section lacks %q", heading, want)
		}
	}
}
