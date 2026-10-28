package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withSummaryLimit sets the summary limit for one test.
func withSummaryLimit(t *testing.T, n int) {
	t.Helper()
	was := maxConsistencySummaryBytes
	maxConsistencySummaryBytes = n
	t.Cleanup(func() { maxConsistencySummaryBytes = was })
}

// TestConsistencyIngestSplitsASummaryPastTheLimit (iss-2610261651489204): a
// report that would pass the summary limit keeps its header and findings table
// in 00-summary.md, which every record's evidence line points into, and puts
// each finding in full in 01-findings.md, so the reviews board can still read
// the summary.
func TestConsistencyIngestSplitsASummaryPastTheLimit(t *testing.T) {
	// The same findings once at the default limit, to learn where the table
	// ends and how long the whole report runs.
	whole := consistencyRepo(t)
	em, err := EmitConsistency(whole.Root(), "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := ingest(t, whole.Root(), findingsPayload(t, whole.Root(), em, contradiction(), briefDrift()), &fakeFiler{})
	if err != nil {
		t.Fatal(err)
	}
	full, _ := os.ReadFile(filepath.Join(whole.Root(), res.ReportPath))
	table := strings.Index(string(full), "\n### 1. ")
	if table < 0 {
		t.Fatalf("the whole report carries no finding in full:\n%s", full)
	}
	// Room for the split summary, with slack for the size check, which bounds
	// every row with the longest ledger cell an id can make.
	limit := table + len(consistencySplitNote) + 200
	if limit >= len(full) {
		t.Fatalf("limit %d does not fall short of the whole report's %d bytes", limit, len(full))
	}
	withSummaryLimit(t, limit)

	r := consistencyRepo(t)
	root := r.Root()
	em, err = EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeFiler{}
	res, err = ingest(t, root, findingsPayload(t, root, em, contradiction(), briefDrift()), f)
	if err != nil {
		t.Fatal(err)
	}
	wantRel := ".abcd/work/reviews/" + cxDate + "-consistency/00-summary.md"
	if res.ReportPath != wantRel || len(f.reports) != 2 || f.reports[0] != wantRel {
		t.Fatalf("report %s, filer evidence %v; want both at %s", res.ReportPath, f.reports, wantRel)
	}
	summary, err := os.ReadFile(filepath.Join(root, wantRel))
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) > limit {
		t.Fatalf("the summary is %d bytes, past the limit of %d", len(summary), limit)
	}
	s := string(summary)
	for _, want := range []string{"- receipt: " + em.ReceiptID, "| 1 | ", "| 2 | ", "iss-901 (filed)", "[01-findings.md](01-findings.md)"} {
		if !strings.Contains(s, want) {
			t.Errorf("summary lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "\n### 1. ") {
		t.Errorf("summary still carries the findings in full:\n%s", s)
	}
	detail, err := os.ReadFile(filepath.Join(root, filepath.Dir(wantRel), "01-findings.md"))
	if err != nil {
		t.Fatalf("no findings file beside the summary: %v", err)
	}
	d := string(detail)
	for _, want := range []string{"\n### 1. ", "\n### 2. ", "\u201c" + cxQuotePlanned + "\u201d", "\u201c" + cxQuoteBrief + "\u201d", "(00-summary.md)"} {
		if !strings.Contains(d, want) {
			t.Errorf("findings file lacks %q:\n%s", want, d)
		}
	}
}

// TestConsistencyIngestRefusesATablePastTheLimitBeforeFiling: when the
// findings table alone would pass the summary limit, the ingest refuses before
// the first finding is filed, so no record cites a report that was never
// written.
func TestConsistencyIngestRefusesATablePastTheLimitBeforeFiling(t *testing.T) {
	withSummaryLimit(t, 200)
	r := consistencyRepo(t)
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeFiler{}
	_, err = ingest(t, root, findingsPayload(t, root, em, contradiction(), briefDrift()), f)
	if err == nil || !strings.Contains(err.Error(), "summary") || !strings.Contains(err.Error(), "nothing was filed") {
		t.Fatalf("ingest err = %v; want a refusal naming the summary limit and that nothing was filed", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("the filer was called %d times; want none", len(f.calls))
	}
	if _, err := os.Stat(filepath.Join(root, ".abcd/work/reviews/"+cxDate+"-consistency")); !os.IsNotExist(err) {
		t.Fatalf("a report directory was written: %v", err)
	}
}
