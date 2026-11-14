package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// TestIntentPageDocumentsTheRedraftsBound is iss-2611040555506702: the
// `intent` command page documents interview.redrafts beside the plain-Terminal
// planning interview's other setting, the person's route to a runner. The
// figures are the reader's own (internal/core/layered), so a changed default
// or bound fails here until the page follows. The page wraps its prose, so the
// section is read with its whitespace folded.
func TestIntentPageDocumentsTheRedraftsBound(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "intent.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	const heading = "### In a plain Terminal"
	start := strings.Index(page, heading)
	if start < 0 {
		t.Fatalf("the intent page has no %q section", heading)
	}
	section := page[start+len(heading):]
	if end := strings.Index(section, "\n#"); end >= 0 {
		section = section[:end]
	}
	folded := strings.Join(strings.Fields(section), " ")
	for _, want := range []string{
		"roles.planning-interviewer.runner",
		"`" + layered.InterviewRedraftsKey + "`",
		"the repository's `.abcd/config.json`",
		fmt.Sprintf("a whole number from 0 to %d, %d when no file sets it", layered.InterviewRedraftsMax, layered.InterviewRedraftsDefault),
		"`~/.abcd.noindex/config.json`",
	} {
		if !strings.Contains(folded, want) {
			t.Errorf("the intent page's %q section lacks %q", heading, want)
		}
	}
}
