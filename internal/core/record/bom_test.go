package record

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadRecordHeadAndBodyReadsPastABOM: the body starts where the block the
// fields came from closes, a BOM at line 0 included (iss-2608221126066379).
func TestReadRecordHeadAndBodyReadsPastABOM(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "iss-1-x.md")
	if err := os.WriteFile(p, []byte("\ufeff---\nid: iss-1\n---\nThe body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fields, body := readRecordHeadAndBody(p)
	if fields["id"].Value != "iss-1" || body != "The body.\n" {
		t.Fatalf("readRecordHeadAndBody = (%v, %q), want the id and the body", fields, body)
	}
}
