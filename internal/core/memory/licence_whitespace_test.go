package memory

import (
	"strings"
	"testing"
)

// TestLicenceWhitespaceIsRefusedOnBothSourceShapes pins the one licence rule
// across the two source: shapes. The YAML reader keeps interior whitespace in
// a quoted scalar, so `licence: " "` reached validateSourceBlock as a string of
// one space: the single-source branch trimmed and refused it while the
// sources[] entry checked only for the empty string and let it through. Both
// shapes must now judge a licence through the same gate.
func TestLicenceWhitespaceIsRefusedOnBothSourceShapes(t *testing.T) {
	hash := strings.Repeat("a", 64)
	citation := map[string]any{"type": "knowledge", "title": "T", "year": 2026}
	for _, lic := range []string{"", " ", "\t"} {
		single := map[string]any{
			"class": "external_pdf", "citation": citation, "licence": lic,
			"source_hash": hash, "ingested_at": "2026-08-19",
		}
		if err := validateSourceBlock(single); err == nil {
			t.Errorf("single-source shape accepted licence %q", lic)
		}
		multi := map[string]any{
			"classes": []any{"external_pdf"},
			"sources": []any{map[string]any{
				"class": "external_pdf", "citation": citation, "licence": lic,
				"source_hash": hash, "ingested_at": "2026-08-19",
			}},
		}
		if err := validateSourceBlock(multi); err == nil {
			t.Errorf("sources[] shape accepted licence %q", lic)
		}
	}
}
