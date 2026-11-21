package record

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBoardChapterCallsTheLedgerRecordsNoFamilyThePageLacks holds the board
// chapter's paragraph on `abcd <record-id>` to the glossary's one map of record
// families (iss-2610262033533504). The map is a closed set: a name with no row
// there is a name nobody has justified. So the sentence on the admission and the
// surprise, records this package dispatches that have no lifecycle folder, may
// call them a family only once the map has a row for each.
func TestBoardChapterCallsTheLedgerRecordsNoFamilyThePageLacks(t *testing.T) {
	brief := filepath.Join("..", "..", "..", ".abcd", "development", "brief")
	chapter := filepath.Join(brief, "04-surfaces", "08-abcd.md")
	page := filepath.Join(brief, "glossary", "core", "record-families.md")

	families := map[string]bool{}
	pageRaw, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("read the record-families page: %v", err)
	}
	for _, l := range strings.Split(string(pageRaw), "\n") {
		cells := strings.Split(strings.TrimSpace(l), "|")
		if len(cells) < 3 || !strings.HasPrefix(strings.TrimSpace(l), "|") {
			continue
		}
		c := strings.TrimSpace(cells[1])
		if strings.HasPrefix(c, "**") && strings.HasSuffix(c, "**") {
			families[strings.ToLower(strings.Trim(c, "*"))] = true
		}
	}
	if !families["issue"] {
		t.Fatalf("%s: no family table read (no issue row)", page)
	}

	raw, err := os.ReadFile(chapter)
	if err != nil {
		t.Fatalf("read the board chapter: %v", err)
	}
	const opener = "**`abcd <record-id>`**"
	_, rest, ok := strings.Cut(string(raw), opener)
	if !ok {
		t.Fatalf("%s has no %q paragraph", chapter, opener)
	}
	para, _, _ := strings.Cut(rest, "\n\n")
	para = strings.Join(strings.Fields(para), " ")

	const subject = "An admission and a surprise"
	_, sentence, ok := strings.Cut(para, subject)
	if !ok {
		t.Fatalf("the record-id paragraph no longer opens a sentence with %q", subject)
	}
	sentence, _, _ = strings.Cut(sentence, ". ")
	if strings.Contains(strings.ToLower(sentence), "famil") && !(families["admission"] && families["surprise"]) {
		t.Errorf("the board chapter calls the admission and the surprise a family (%q), but %s has no row for either; call them ledger records, or add the rows",
			subject+sentence, page)
	}
}
