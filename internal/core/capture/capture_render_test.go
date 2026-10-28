package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCaptureRefusesABodyTheSiteRendererRefuses (iss-2610261616580903): issue
// text the site renderer refuses, here a placeholder named after a void HTML
// element, is refused at the write with the construct named, and the ledger is
// left without a record; record_schema would otherwise refuse the record only at
// the next gate run. The same text with the placeholder in a code span is filed.
func TestCaptureRefusesABodyTheSiteRendererRefuses(t *testing.T) {
	repo, ir := ledger(t)
	req := CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Severity: SeverityMinor, Category: "bug", Source: "manual-test",
		Slug: "s", FoundDuring: "t", Remedy: "check the fingerprint the receipt names",
		Text: "the receipt names its <input_fingerprint> but never checks it",
	}
	_, err := Capture(req)
	if !errors.Is(err, ErrRequestRefused) {
		t.Fatalf("err = %v; want a request refusal", err)
	}
	for _, want := range []string{"site renderer", "inline HTML", "nothing written"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %q: %v", want, err)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(ir, "open"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Fatalf("a refused capture left %s in the ledger", e.Name())
		}
	}

	req.Text = "the receipt names its `<input_fingerprint>` but never checks it"
	if _, err := Capture(req); err != nil {
		t.Fatalf("the same text with the placeholder in a code span: %v", err)
	}
}
