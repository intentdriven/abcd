package reading

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

// TestPayloadFieldNotesAFailedAugmenterRun: a repository's opt-in augmenter
// whose run fails while the ingest redacts payload text degrades the scanner
// DURING the redactions, and the ingest's note, read after them, says so
// (iss-2608291814575788).
func TestPayloadFieldNotesAFailedAugmenterRun(t *testing.T) {
	augmenttest.Install(t, &augmenttest.Func{F: func(string, string) ([]scanner.Finding, error) {
		return nil, errors.New("run failed")
	}})
	field, note := newPayloadField(t.TempDir())
	if n := note(); n != "" {
		t.Fatalf("noted before any redaction: %q", n)
	}
	field("some payload text")
	if n := note(); !strings.Contains(n, "run failed") {
		t.Fatalf("note = %q, want the failed augmenter run named", n)
	}
}
