package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The internals index's row for 01-agents.md once said every agent prompt
// declared JSON inputs and outputs, while the catalog says fourteen of the
// eighteen return JSON and the four repo-workflow reviewer and researcher
// prompts declare none (iss-2610262038201302). The row is held to the
// catalog's own split: it names the fourteen of the eighteen that declare
// JSON and the four repo-workflow reviewer and researcher prompts that do not.
func TestInternalsIndexAgentsRowStatesTheCatalogsJSONSplit(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	read := func(rel string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	catalog := read(".abcd/development/brief/05-internals/01-agents.md")
	index := read(".abcd/development/brief/05-internals/README.md")

	// The catalog's side of the split, so the row cannot be held to a claim
	// the catalog has since dropped.
	for _, want := range []string{
		"Eighteen agent prompts ship",
		"Fourteen of them return JSON",
		"The four repo-workflow reviewer and researcher prompts declare no JSON",
	} {
		if !strings.Contains(strings.Join(strings.Fields(catalog), " "), want) {
			t.Fatalf("01-agents.md no longer says %q; re-derive the index row from the catalog", want)
		}
	}

	var row string
	for _, line := range strings.Split(index, "\n") {
		if strings.HasPrefix(line, "|") && strings.Contains(line, "[`01-agents.md`](01-agents.md)") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("the internals index has no row for 01-agents.md")
	}
	for _, want := range []string{
		"fourteen of the eighteen",
		"declare JSON inputs and outputs",
		"the four repo-workflow reviewer and researcher prompts do not",
	} {
		if !strings.Contains(row, want) {
			t.Errorf("the index row for 01-agents.md does not say %q, as the catalog does:\n%s", want, row)
		}
	}
	if strings.Contains(row, "each with declared JSON") {
		t.Errorf("the index row for 01-agents.md still says every prompt declares JSON:\n%s", row)
	}
}
