package launch

import "testing"

// TestProseLinesDropsABOMLedFrontmatter: a BOM ahead of the opening rule does
// not turn a document's frontmatter into prose (iss-2608221126066379).
func TestProseLinesDropsABOMLedFrontmatter(t *testing.T) {
	t.Parallel()
	for _, pl := range proseLines([]byte("\ufeff---\ntitle: x\n---\nProse.\n")) {
		if pl.text != "Prose." && pl.text != "" {
			t.Fatalf("line %d read as prose: %q", pl.n, pl.text)
		}
	}
}
