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

// TestConsistencyIngestRefusesAnUnrenderableFindingBeforeFiling
// (iss-2610261616580903): the consistency pass asks the site renderer about
// every finding it would file before it files any, so when the renderer
// refuses one, nothing is filed and no report is written, rather than the
// refusal stopping the filing after the earlier findings are in the ledger. A
// finding's free text is neutralised before filing, but a summary that opens
// with a pipe still opens the record body as a one-line table, which the
// renderer refuses; the second finding's summary does.
func TestConsistencyIngestRefusesAnUnrenderableFindingBeforeFiling(t *testing.T) {
	root := consistencyLedgerRepo(t)
	second := cxFinding("| itd-10 says one spec | itd-11 says one or more", cxDoubleExplain)
	second["class"] = "scope_leakage"
	payload := consistencyPayloadWith(t, root, []any{cxFinding(cxDoubleSummary, cxDoubleExplain), second})
	_, err := IngestConsistency(root, payload, "2026-09-26", nil)
	if err == nil || !strings.Contains(err.Error(), "consistency finding 2") ||
		!strings.Contains(err.Error(), "site renderer refuses") {
		t.Fatalf("err = %v, want a refusal naming finding 2 and the renderer", err)
	}
	for _, f := range ledgerFiles(t, root) {
		if filepath.Base(f) != lockFilename {
			t.Fatalf("a refused ingest wrote %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".abcd/work/reviews")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused ingest wrote a report (%v)", err)
	}
}
