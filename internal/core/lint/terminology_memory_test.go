package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestTerminologyMemoryRowCoversBothHalves holds the crosswalk's Memory row to
// the canonical vocabulary (iss-2610251558471369): the glossary gives "memory"
// as memory-note's alias, so the row leads with memory notes (recalled on a
// matching prompt, kept on the machine until a person promotes one into the
// project), then names the library's store of cited pages consulted on demand,
// and claims no component abcd does not ship, such as a "writing curator".
func TestTerminologyMemoryRowCoversBothHalves(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootFromPackage, "docs", "reference", "terminology.md"))
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "| **Memory** |") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("docs/reference/terminology.md has no Memory row")
	}
	cells := strings.Split(row, " | ")
	position := cells[len(cells)-1]

	if strings.Contains(strings.ToLower(position), "curator") {
		t.Error("the Memory row names a curator, a component abcd does not ship: the host session distils the pages and `abcd library ingest` writes them")
	}
	notes := strings.Index(position, "memory notes")
	pages := strings.Index(position, "`.abcd/memory/`")
	if notes < 0 || pages < 0 || notes > pages {
		t.Errorf("the Memory row must lead with memory notes (the glossary's sense of the word) and then name the library's page store; notes at %d, pages at %d", notes, pages)
	}
	for _, want := range []string{
		"`abcd memory add`",
		"`" + abcdhome.Display("memory.json") + "`",
		"`.abcd/memory.json`",
		"promote",
		"when a prompt matches",
		"`abcd library ingest`",
		"`abcd library ask`",
		"on demand",
	} {
		if !strings.Contains(position, want) {
			t.Errorf("the Memory row's position does not say %q", want)
		}
	}
}
