package changelog

import "testing"

// TestSummariseReadsPastABOM: the changelog's body is the body frontmatter.Fields
// leaves, so a BOM-led record's frontmatter never reaches the changelog as its
// summary (iss-2608221126066379).
func TestSummariseReadsPastABOM(t *testing.T) {
	t.Parallel()
	title, summary := summarise("\ufeff---\nid: iss-1\nslug: the-slug\n---\n\nThe body paragraph.\n", "iss-1")
	if title != "the-slug" || summary != "The body paragraph." {
		t.Fatalf("summarise = (%q, %q), want (\"the-slug\", \"The body paragraph.\")", title, summary)
	}
}
