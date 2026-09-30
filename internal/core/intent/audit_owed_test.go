package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Ruling DQ1c (2026-09-30, the product thinker): an after-merge audit that
// fails or comes back undecided leaves the intent SHIPPED and FLAGGED. The
// flag names the unmet or undecided criteria with the receipt, and one issue
// captured through the ledger's filer carries the owed check; a later passing
// audit clears the flag with a dated line and resolves that issue.

// verdictWith is validVerdict with its one criterion (ac-1) judged v.
func verdictWith(t *testing.T, rcp, v string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(validVerdict(rcp)), &m); err != nil {
		t.Fatal(err)
	}
	c := m["criteria"].([]any)[0].(map[string]any)
	c["verdict"] = v
	m["acceptance_rollup"] = map[string]any{"MET": 0, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0}
	m["acceptance_rollup"].(map[string]any)[v] = 1
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// fakeAuditLedger records what the ingest asked the ledger for.
type fakeAuditLedger struct {
	filed   []AuditOwed
	cleared []AuditCleared
	nextID  string
	open    map[string]string // receipt -> issue id already open
}

func (f *fakeAuditLedger) file(o AuditOwed) (AuditFiling, error) {
	f.filed = append(f.filed, o)
	if id, ok := f.open[o.ReceiptID]; ok {
		return AuditFiling{IssueID: id, Linked: true}, nil
	}
	if f.open == nil {
		f.open = map[string]string{}
	}
	f.open[o.ReceiptID] = f.nextID
	return AuditFiling{IssueID: f.nextID}, nil
}

func (f *fakeAuditLedger) clear(c AuditCleared) error {
	f.cleared = append(f.cleared, c)
	delete(f.open, c.ReceiptID)
	return nil
}

func withFakeAuditLedger(t *testing.T) *fakeAuditLedger {
	t.Helper()
	f := &fakeAuditLedger{nextID: "iss-2609301200000001"}
	prevFile, prevClear := auditOwedFiler, auditOwedClearer
	SetAuditLedger(f.file, f.clear)
	prevNow := auditNow
	auditNow = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() {
		auditOwedFiler, auditOwedClearer = prevFile, prevClear
		auditNow = prevNow
	})
	return f
}

func shippedAlpha(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatalf("the intent must stay in shipped/: %v", err)
	}
	return string(b)
}

func TestAFailedAfterMergeAuditLeavesTheIntentShippedFlaggedAndFiled(t *testing.T) {
	for _, tc := range []struct {
		verdict, remedy string
		failed          bool
	}{
		{"NOT_MET", "fix, then re-run the audit", true},
		{"INCONCLUSIVE", "re-run the audit", false},
	} {
		t.Run(tc.verdict, func(t *testing.T) {
			f := withFakeAuditLedger(t)
			root := t.TempDir()
			rcp := shipOne(t, root)
			res, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, tc.verdict)))
			if err != nil {
				t.Fatalf("ingest: %v", err)
			}
			if len(f.filed) != 1 {
				t.Fatalf("the ingest must ask the ledger for exactly one issue, asked %d times", len(f.filed))
			}
			o := f.filed[0]
			if o.IntentID != "itd-10" || o.ReceiptID != rcp || o.Failed != tc.failed || o.Remedy() != tc.remedy ||
				len(o.Criteria) != 1 || o.Criteria[0].ID != "ac-1" || o.Criteria[0].Verdict != tc.verdict {
				t.Fatalf("owed check handed to the ledger = %+v (remedy %q)", o, o.Remedy())
			}
			if res.OwedIssue != "iss-2609301200000001" || len(res.AuditOwed) != 1 || res.AuditOwed[0] != "ac-1 "+tc.verdict {
				t.Fatalf("the result must name the owed criteria and the issue carrying them: %+v", res)
			}
			s := shippedAlpha(t, root)
			flag := "Audit owed (receipt " + rcp + "): ac-1 " + tc.verdict + ". Remedy: " + tc.remedy + ". Carried by iss-2609301200000001."
			if !strings.Contains(s, flag) {
				t.Fatalf("the Audit Notes must carry the flag %q:\n%s", flag, s)
			}
			if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err == nil {
				t.Fatal("a failed audit must never un-ship the intent")
			}

			// The same failing verdict again is a noop that links to the open
			// issue: no second filing, the record unchanged.
			res2, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, tc.verdict)))
			if err != nil {
				t.Fatalf("second ingest: %v", err)
			}
			if res2.Status != "noop" || !res2.OwedIssueLinked || res2.OwedIssue != "iss-2609301200000001" {
				t.Fatalf("a second failed audit must link to the open issue, got %+v", res2)
			}
			if got := shippedAlpha(t, root); got != s {
				t.Fatalf("a repeated failed audit must not rewrite the record:\n%s", got)
			}
		})
	}
}

func TestAPassingReAuditClearsTheFlagAndResolvesTheIssue(t *testing.T) {
	f := withFakeAuditLedger(t)
	root := t.TempDir()
	rcp := shipOne(t, root)
	if _, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, "NOT_MET"))); err != nil {
		t.Fatal(err)
	}
	res, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, "MET")))
	if err != nil {
		t.Fatalf("passing re-ingest: %v", err)
	}
	if len(f.cleared) != 1 || f.cleared[0].IssueID != "iss-2609301200000001" || f.cleared[0].IntentID != "itd-10" || f.cleared[0].ReceiptID != rcp {
		t.Fatalf("a passing re-audit must resolve the issue that carried the flag, cleared %+v", f.cleared)
	}
	if res.FlagCleared != "iss-2609301200000001" || res.OwedIssue != "" || len(res.AuditOwed) != 0 {
		t.Fatalf("result = %+v", res)
	}
	s := shippedAlpha(t, root)
	if strings.Contains(s, "Audit owed (receipt") {
		t.Fatalf("the flag must be cleared:\n%s", s)
	}
	cleared := "Audit owed flag cleared 2026-09-30 (receipt " + rcp + "): a re-run of the audit judged no criterion NOT_MET or INCONCLUSIVE, and iss-2609301200000001 is resolved."
	if !strings.Contains(s, cleared) {
		t.Fatalf("the clearance must leave a dated Audit Notes line %q:\n%s", cleared, s)
	}
	// Ingesting the passing verdict again changes nothing and resolves nothing.
	if _, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, "MET"))); err != nil {
		t.Fatal(err)
	}
	if len(f.cleared) != 1 || shippedAlpha(t, root) != s {
		t.Fatalf("a repeated passing ingest must be a noop, cleared %d", len(f.cleared))
	}
}

func TestAPassingFirstAuditAsksTheLedgerForNothing(t *testing.T) {
	f := withFakeAuditLedger(t)
	root := t.TempDir()
	rcp := shipOne(t, root)
	if _, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, "MET_WITH_CONCERNS"))); err != nil {
		t.Fatal(err)
	}
	if len(f.filed) != 0 || len(f.cleared) != 0 {
		t.Fatalf("a passing audit files and clears nothing: filed %d, cleared %d", len(f.filed), len(f.cleared))
	}
	if strings.Contains(shippedAlpha(t, root), "Audit owed") {
		t.Fatal("a passing audit carries no flag")
	}
}

func TestAFlaggedReceiptReEmitsItsRequestForTheReRun(t *testing.T) {
	withFakeAuditLedger(t)
	root := t.TempDir()
	rcp := shipOne(t, root)
	if _, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, "INCONCLUSIVE"))); err != nil {
		t.Fatal(err)
	}
	req := filepath.Join(root, reviewsDir, rcp+".request.md")
	if err := os.Remove(req); err != nil {
		t.Fatal(err)
	}
	res, err := ReEmitAudit(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "check_owed" || !res.RequestWritten {
		t.Fatalf("a flagged receipt owes a re-run, so its request is rewritten: %+v", res)
	}
	if _, err := os.Stat(req); err != nil {
		t.Fatalf("request not rewritten: %v", err)
	}
}

func TestALedgerThatCannotFileRefusesTheIngestWithNothingWritten(t *testing.T) {
	withFakeAuditLedger(t)
	SetAuditLedger(func(AuditOwed) (AuditFiling, error) { return AuditFiling{}, os.ErrPermission }, nil)
	root := t.TempDir()
	rcp := shipOne(t, root)
	before := shippedAlpha(t, root)
	if _, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, "NOT_MET"))); err == nil {
		t.Fatal("a failing verdict whose owed check cannot be filed must be refused")
	}
	if shippedAlpha(t, root) != before {
		t.Fatal("a refused ingest writes nothing: the receipt stays OWED")
	}
}

// The owed-review reader names a flagged receipt apart: its verdict is
// ingested, so its review is not owed, but its check is, and its re-emit
// rewrites the request for the re-run.
func TestTheReviewListingNamesAFlaggedReceiptApart(t *testing.T) {
	withFakeAuditLedger(t)
	root := t.TempDir()
	rcp := shipOne(t, root)
	if _, err := IngestVerdict(root, writeVerdict(t, root, verdictWith(t, rcp, "NOT_MET"))); err != nil {
		t.Fatal(err)
	}
	l, err := Reviews(root)
	if err != nil {
		t.Fatal(err)
	}
	if l.Owed != 0 || l.Ingested != 1 || l.AuditOwed != 1 || len(l.Entries) != 1 {
		t.Fatalf("listing = %+v", l)
	}
	e := l.Entries[0]
	if e.State != ReviewIngested || !e.AuditOwed || e.AuditOwedIssue != "iss-2609301200000001" || e.ReEmit != ReEmitCommand("itd-10") {
		t.Fatalf("a flagged receipt must name its owed check, its issue and its re-emit: %+v", e)
	}
}
