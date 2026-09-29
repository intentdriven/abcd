package issueschema

import "testing"

// TestParseDispositionReadsPastABOM: a BOM is the file's encoding mark, not
// preamble, so a BOM-led disposition reads as the record it is
// (iss-2608221126066379).
func TestParseDispositionReadsPastABOM(t *testing.T) {
	t.Parallel()
	rec := ParseDisposition("dsp-1", "\ufeff---\nstate: held\nexit_condition: x\n---\nbody\n")
	if rec.State != "held" || !rec.WellFormed {
		t.Fatalf("ParseDisposition = %+v, want state held and well-formed", rec)
	}
}
