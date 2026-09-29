package peers

import "testing"

// TestParseTitleReadsPastABOM: a BOM-led record has the title its family's
// rule gives it, not "malformed" (iss-2608221126066379).
func TestParseTitleReadsPastABOM(t *testing.T) {
	t.Parallel()
	if got := parseTitle([]byte("\ufeff---\nid: iss-1\n---\nThe first body line.\n"), "iss"); got != "The first body line." {
		t.Fatalf("parseTitle = %q, want the first body line", got)
	}
}
