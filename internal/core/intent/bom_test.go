package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// TestIntentWritersReadTheBlockTheReaderReads: a BOM ahead of the opening
// delimiter is one frontmatter.Fields reads past, so every intent writer must
// too. Its private walks refused such a record with "no leading frontmatter
// block" while Load read it fine (iss-2608221126066379). The BOM survives the
// write: a writer edits the keys, never the file's first bytes.
func TestIntentWritersReadTheBlockTheReaderReads(t *testing.T) {
	t.Parallel()
	const bom = "\ufeff"
	doc := bom + "---\nid: itd-10\nslug: alpha\nheld: yes\n---\n# alpha\n"

	set, err := setFrontmatterFields(doc, map[string]string{"spec_id": "spc-1", "slug": "beta"})
	if err != nil {
		t.Fatalf("setFrontmatterFields on a BOM-led record: %v", err)
	}
	if !strings.HasPrefix(set, bom+"---\n") {
		t.Fatalf("the write must keep the file's BOM and opening delimiter:\n%q", set)
	}
	fields := frontmatter.Fields(strings.Split(set, "\n"))
	if fields["spec_id"].Value != "spc-1" || fields["slug"].Value != "beta" {
		t.Fatalf("the reader must see both writes inside the block: %+v\n%q", fields, set)
	}

	removed, err := removeFrontmatterField(doc, "held")
	if err != nil {
		t.Fatalf("removeFrontmatterField on a BOM-led record: %v", err)
	}
	if _, ok := frontmatter.Fields(strings.Split(removed, "\n"))["held"]; ok || !strings.HasPrefix(removed, bom) {
		t.Fatalf("the key must be gone and the BOM kept:\n%q", removed)
	}

	if _, err := frontmatterClose(strings.Split(doc, "\n")); err != nil {
		t.Fatalf("frontmatterClose on a BOM-led record: %v", err)
	}
}

// TestAddRelatedIssueWritesABOMLedRecord: the verb a user runs, end to end, on
// a record the reader loads.
func TestAddRelatedIssueWritesABOMLedRecord(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md",
		"\ufeff---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n")
	if _, err := AddRelatedIssue(root, "itd-10", "iss-4"); err != nil {
		t.Fatalf("AddRelatedIssue on a BOM-led record the reader loads: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := frontmatter.Fields(strings.Split(string(after), "\n"))["related_issues"].Value; got != "[iss-4]" {
		t.Fatalf("related_issues = %q, want [iss-4]:\n%q", got, after)
	}
}
