package site

import "testing"

// TestStripFrontmatterReadsPastABOM: a BOM ahead of the opening `---` does not
// publish the block as prose (iss-2608221126066379).
func TestStripFrontmatterReadsPastABOM(t *testing.T) {
	t.Parallel()
	body, consumed := StripFrontmatter("\ufeff---\na: b\n---\nbody\n")
	if body != "\nbody\n" && body != "body\n" {
		t.Fatalf("StripFrontmatter body = %q", body)
	}
	if consumed == 0 {
		t.Fatalf("StripFrontmatter consumed no lines of a BOM-led block")
	}
}
