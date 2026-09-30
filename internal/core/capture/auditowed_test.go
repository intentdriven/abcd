package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/gittest"
)

// auditowed_test.go covers the ledger half of ruling DQ1c: an after-merge
// audit that fails or comes back undecided captures ONE issue carrying the
// owed check, a second one links to it, and a passing re-audit resolves it.

const aoShipped = ".abcd/development/intents/shipped/itd-10-alpha.md"

// auditOwedRepo ships itd-10 through the real close, so its Audit Notes carry
// an OWED receipt and the request the host would read, and returns the root
// and the receipt.
func auditOwedRepo(t *testing.T) (string, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".abcd/development/intents/planned/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n# alpha\n\n"+
			"## Scope Conditions\n\n"+intent.NullityToken+"\n\n## Acceptance Criteria\n\n- ok\n\n"+
			"## Grounds\n\n- pursued: we expect the recorded conjecture to outlive the session that had it\n\n## Audit Notes\n")
	r.Write(".abcd/development/specs/open/spc-1-alpha.md", "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")
	r.Write(".abcd/work/issues/open/.gitkeep", "")
	r.Commit("fixture")
	res, err := intent.Reconcile(r.Root(), "spc-1", "", intent.RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID == "" {
		t.Fatal("the close must park a receipt")
	}
	return r.Root(), res.ReceiptID
}

// auditVerdict is a valid one-criterion verdict for rcp, judging ac-1 v and
// echoing the policy the request issued.
func auditVerdict(t *testing.T, root, rcp, v string) []byte {
	t.Helper()
	req, err := os.ReadFile(filepath.Join(root, ".abcd/.work.local/reviews", rcp+".request.md"))
	if err != nil {
		t.Fatal(err)
	}
	rubric := regexp.MustCompile(`(?m)^- rubric_hash: (\S+)$`).FindStringSubmatch(string(req))[1]
	prompt := regexp.MustCompile(`(?m)^- prompt_hash: (\S+)$`).FindStringSubmatch(string(req))[1]
	rollup := map[string]any{"MET": 0, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0}
	rollup[v] = 1
	b, err := json.Marshal(map[string]any{
		"_type":      intent.VerdictType,
		"receipt_id": rcp,
		"verifier":   map[string]any{"id": "intent-auditor", "version": "claude-opus-5-5"},
		"policy":     map[string]any{"rubric_hash": rubric, "prompt_hash": prompt},
		"criteria": []any{map[string]any{
			"criterion_id": "ac-1", "verdict": v, "rationale": "judged against the delivered code",
			"evidence": []any{map[string]any{"ref": "internal/example.go:1", "quote": "package example"}},
		}},
		"acceptance_rollup": rollup,
		"gap_audit":         map[string]any{"honoured": []any{}, "diverged": []any{}, "missing": []any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func openIssues(t *testing.T, root string) []Issue {
	t.Helper()
	l, err := List(ListRequest{RepoRoot: root, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	return l.Issues
}

func TestAnUndecidedAuditCapturesOneIssueAndAPassingReAuditResolvesIt(t *testing.T) {
	root, rcp := auditOwedRepo(t)
	res, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "INCONCLUSIVE"))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	open := openIssues(t, root)
	if len(open) != 1 {
		t.Fatalf("an undecided audit captures exactly one issue, found %d", len(open))
	}
	iss := open[0]
	if res.OwedIssue != iss.ID || res.OwedIssueLinked {
		t.Fatalf("the result must name the issue filed: %+v, issue %s", res, iss.ID)
	}
	if iss.Severity != auditUndecidedSeverity || iss.Category != auditUndecidedCategory || iss.Source != "agent-finding" ||
		!contains(iss.RelatedIntents, "itd-10") || !strings.HasPrefix(iss.Remedy, "re-run the audit: ") ||
		!strings.Contains(iss.Body, "(receipt "+rcp+")") || iss.FoundAt != aoShipped {
		t.Fatalf("the captured issue = %+v", iss)
	}
	// The drain may take it: its remedy is real, so the field rules decide as
	// for any issue, and a minor inconsistency passes the baseline.
	if v := eligibility(iss, drainrule.Baseline(), deferralAnchor{}); v.Outcome != DrainEligible {
		t.Fatalf("the drain must be able to take an undecided audit's issue: %+v", v)
	}

	// The same undecided verdict again links to the open issue.
	res2, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "INCONCLUSIVE"))
	if err != nil {
		t.Fatal(err)
	}
	if !res2.OwedIssueLinked || res2.OwedIssue != iss.ID || len(openIssues(t, root)) != 1 {
		t.Fatalf("a second undecided audit must link, not file a double: %+v", res2)
	}

	// A failed re-run of the same receipt still links to the one open issue.
	res3, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "NOT_MET"))
	if err != nil {
		t.Fatal(err)
	}
	if !res3.OwedIssueLinked || res3.OwedIssue != iss.ID || len(openIssues(t, root)) != 1 {
		t.Fatalf("a failed re-audit of the flagged receipt must link to the open issue: %+v", res3)
	}

	// A passing re-run clears the flag and resolves the issue, with impact fix.
	res4, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "MET"))
	if err != nil {
		t.Fatal(err)
	}
	if res4.FlagCleared != iss.ID || len(openIssues(t, root)) != 0 {
		t.Fatalf("a passing re-audit must resolve %s: %+v", iss.ID, res4)
	}
	resolved, err := List(ListRequest{RepoRoot: root, State: StateResolved})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Issues) != 1 || resolved.Issues[0].ID != iss.ID || resolved.Issues[0].ResolvedBy == nil ||
		resolved.Issues[0].ResolvedBy.Intent != "itd-10" {
		t.Fatalf("the resolution = %+v", resolved.Issues)
	}
	rec, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(resolved.Issues[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`(?m)^impact: "?fix"?$`).Match(rec) {
		t.Fatalf("the resolution must carry impact fix:\n%s", rec)
	}
	body, err := os.ReadFile(filepath.Join(root, aoShipped))
	if err != nil {
		t.Fatalf("the intent must still be shipped: %v", err)
	}
	if strings.Contains(string(body), "Audit owed (receipt") || !strings.Contains(string(body), "Audit owed flag cleared ") {
		t.Fatalf("the flag must be cleared by a dated line:\n%s", body)
	}
}

func TestAFailedAuditCapturesAMajorBugWithAFixRemedy(t *testing.T) {
	root, rcp := auditOwedRepo(t)
	if _, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "NOT_MET")); err != nil {
		t.Fatal(err)
	}
	open := openIssues(t, root)
	if len(open) != 1 {
		t.Fatalf("a failed audit captures exactly one issue, found %d", len(open))
	}
	iss := open[0]
	if iss.Severity != auditFailedSeverity || iss.Category != auditFailedCategory ||
		!strings.HasPrefix(iss.Remedy, "fix, then re-run the audit: ") || !strings.Contains(iss.Remedy, "ac-1 NOT_MET") {
		t.Fatalf("the captured issue = %+v", iss)
	}
	// Its remedy is real, so a drain judges it by its fields: a major issue
	// is handed back on severity under the baseline, never on the remedy.
	if v := eligibility(iss, drainrule.Baseline(), deferralAnchor{}); v.Rule == RuleRemedy {
		t.Fatalf("a failed audit's issue must not be ineligible on its remedy: %+v", v)
	}
}

// TestConcurrentFailedIngestsOfOneReceiptFileOneIssue is the reviewer's probe
// of the "link, never double" rule: the open-issue scan and the filing are one
// step under the ledger lock, so four ingests of one NOT_MET verdict racing
// each other file ONE issue and every other one links to it.
func TestConcurrentFailedIngestsOfOneReceiptFileOneIssue(t *testing.T) {
	root, rcp := auditOwedRepo(t)
	verdict := auditVerdict(t, root, rcp, "NOT_MET")
	const n = 4
	results := make([]intent.IngestVerdictResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = intent.IngestVerdictBytes(root, verdict)
		}(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("ingest %d: %v", i, err)
		}
	}
	open := openIssues(t, root)
	if len(open) != 1 {
		t.Fatalf("four concurrent ingests of one failed audit must file ONE issue, found %d", len(open))
	}
	filed := 0
	for i, r := range results {
		if r.OwedIssue != open[0].ID {
			t.Fatalf("ingest %d names %q, want the one open issue %s", i, r.OwedIssue, open[0].ID)
		}
		if !r.OwedIssueLinked {
			filed++
		}
	}
	if filed != 1 {
		t.Fatalf("exactly one ingest files and every other links, got %d filings: %+v", filed, results)
	}
}

// orphanCarriers captures extra open carriers of rcp's owed check beside the
// one the first failed ingest filed, as a lost race or a failed intent write
// left them. Their ids sort after the filed one, so it stays the oldest.
func orphanCarriers(t *testing.T, root, rcp string, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if _, err := Capture(CaptureRequest{
			RepoRoot: root, ForceID: id,
			Text:     "The after-merge fidelity audit of itd-10 judged a criterion unmet\n\nThe audit of itd-10 (receipt " + rcp + ") judged ac-1 NOT_MET.\n",
			Severity: "major", Category: "bug", Source: "agent-finding",
			FoundDuring: "abcd intent audit ingest", RelatedIntents: []string{"itd-10"},
			Remedy: "fix, then re-run the audit",
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAFailedIngestLinksTheOldestCarrierAndDeclinesTheOthersAsDuplicates(t *testing.T) {
	root, rcp := auditOwedRepo(t)
	first, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "NOT_MET"))
	if err != nil {
		t.Fatal(err)
	}
	orphanCarriers(t, root, rcp, "iss-9912312359590001", "iss-9912312359590002")
	res, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "NOT_MET"))
	if err != nil {
		t.Fatal(err)
	}
	open := openIssues(t, root)
	if len(open) != 1 || open[0].ID != first.OwedIssue || !res.OwedIssueLinked || res.OwedIssue != first.OwedIssue {
		t.Fatalf("the ingest must link the oldest carrier %s and leave it the only open one: %+v, open %+v", first.OwedIssue, res, open)
	}
	wf, err := List(ListRequest{RepoRoot: root, State: StateWontfix})
	if err != nil {
		t.Fatal(err)
	}
	if len(wf.Issues) != 2 {
		t.Fatalf("the two extra carriers must be declined as duplicates, found %d in wontfix/", len(wf.Issues))
	}
	for _, iss := range wf.Issues {
		if !contains(iss.Duplicates, first.OwedIssue) {
			t.Fatalf("%s must carry duplicates: %s, got %+v", iss.ID, first.OwedIssue, iss.Duplicates)
		}
	}
}

func TestAPassingReAuditResolvesEveryOpenCarrier(t *testing.T) {
	root, rcp := auditOwedRepo(t)
	if _, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "NOT_MET")); err != nil {
		t.Fatal(err)
	}
	orphanCarriers(t, root, rcp, "iss-9912312359590001", "iss-9912312359590002")
	if n := len(openIssues(t, root)); n != 3 {
		t.Fatalf("fixture: want 3 open carriers, found %d", n)
	}
	res, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "MET"))
	if err != nil {
		t.Fatal(err)
	}
	if open := openIssues(t, root); len(open) != 0 || res.FlagCleared == "" {
		t.Fatalf("a passing re-audit must resolve every open carrier: %+v, open %+v", res, open)
	}
}

// TestAPassingAuditResolvesAnOrphanCarrierOfAnUnflaggedReceipt: an ingest whose
// intent write failed after its filing leaves an open carrier while the
// receipt carries no flag. A passing audit of that receipt sweeps the open
// carriers of its check all the same, so the orphan does not outlive it.
func TestAPassingAuditResolvesAnOrphanCarrierOfAnUnflaggedReceipt(t *testing.T) {
	root, rcp := auditOwedRepo(t)
	orphanCarriers(t, root, rcp, "iss-9912312359590001")
	res, err := intent.IngestVerdictBytes(root, auditVerdict(t, root, rcp, "MET"))
	if err != nil {
		t.Fatal(err)
	}
	if open := openIssues(t, root); len(open) != 0 {
		t.Fatalf("a passing audit must resolve the orphan carrier of its receipt: %+v, open %+v", res, open)
	}
}
