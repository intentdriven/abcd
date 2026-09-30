package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deliveryAuditOf composes the delivery audit for the planned itd-10 the way the
// implement loop does before the close: over the intent's bytes, against the
// specs that will be closed once spc-1 closes.
func deliveryAuditOf(t *testing.T, root string) (DeliveryAudit, string) {
	t.Helper()
	rel := filepath.Join(plannedDir, "itd-10-alpha.md")
	content, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	a, err := ComposeDeliveryAudit("itd-10", rel, string(content), []string{"spc-1"})
	if err != nil {
		t.Fatal(err)
	}
	return a, string(content)
}

// TestTheDeliveryAuditIsTheOneTheCloseConsumes is ruling AI's seam: the
// fidelity request the loop issues on the closing lane, before its landing,
// carries the receipt the close parks, and the verdict the auditor returns to it
// is accepted by `abcd intent audit ingest` once the close has shipped the
// intent — so the audit runs once, before the landing, and the close consumes it.
func TestTheDeliveryAuditIsTheOneTheCloseConsumes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	a, _ := deliveryAuditOf(t, root)

	req := a.Request("- the whole delivery: 0123456..89abcde\n")
	for _, want := range []string{a.ReceiptID, a.RubricHash, a.PromptHash, "## Delivered", "0123456..89abcde",
		filepath.Join(shippedDir, "itd-10-alpha.md")} {
		if !strings.Contains(req, want) {
			t.Fatalf("the request carries %q:\n%s", want, req)
		}
	}

	verdict := validVerdict(a.ReceiptID)
	verdict = strings.Replace(verdict, placeholderRubricHash, a.RubricHash, 1)
	verdict = strings.Replace(verdict, placeholderPromptHash, a.PromptHash, 1)
	got, err := a.Check([]byte(verdict))
	if err != nil {
		t.Fatalf("the delivery audit accepts a verdict echoing what it issued: %v", err)
	}
	if got.Worst != "MET" || got.Rollup["MET"] != 1 || len(got.NotMet) != 0 {
		t.Fatalf("the loop reads the verdict's rollup: %+v", got)
	}

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID != a.ReceiptID {
		t.Fatalf("the close parks receipt %s, the delivery audit issued %s", res.ReceiptID, a.ReceiptID)
	}
	vp := writeVerdictRaw(t, root, verdict)
	ing, err := IngestVerdict(root, vp)
	if err != nil || ing.Status != "ingested" {
		t.Fatalf("the close consumes the delivery audit's verdict: %+v %v", ing, err)
	}
}

// TestTheDeliveryAuditRefusesWhatItDidNotIssue: a verdict for another receipt,
// or echoing a prompt hash the audit did not issue, is refused, and a not-met
// criterion is read as one.
func TestTheDeliveryAuditRefusesWhatItDidNotIssue(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	a, _ := deliveryAuditOf(t, root)
	issued := func(v string) string {
		v = strings.Replace(v, placeholderRubricHash, a.RubricHash, 1)
		return strings.Replace(v, placeholderPromptHash, a.PromptHash, 1)
	}
	if _, err := a.Check([]byte(issued(validVerdict("rcp-000000000000")))); err == nil {
		t.Fatal("a verdict for another receipt must be refused")
	}
	if _, err := a.Check([]byte(validVerdict(a.ReceiptID))); err == nil || !strings.Contains(err.Error(), "prompt_hash") && !strings.Contains(err.Error(), "rubric_hash") {
		t.Fatalf("a verdict echoing hashes the audit did not issue must be refused naming them: %v", err)
	}
	notMet := strings.Replace(strings.Replace(issued(validVerdict(a.ReceiptID)), `"verdict": "MET"`, `"verdict": "NOT_MET"`, 1),
		`"MET": 1`, `"MET": 0`, 1)
	notMet = strings.Replace(notMet, `"NOT_MET": 0`, `"NOT_MET": 1`, 1)
	got, err := a.Check([]byte(notMet))
	if err != nil {
		t.Fatal(err)
	}
	if got.Worst != "NOT_MET" || len(got.NotMet) != 1 || got.NotMet[0] != "ac-1" {
		t.Fatalf("a not-met criterion is named: %+v", got)
	}
	for name, c := range map[string][2]string{
		"no criteria": {plannedDir + "/itd-10-alpha.md", "---\nid: itd-10\nspec_id: spc-1\n---\n# alpha\n"},
		"not planned": {shippedDir + "/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1")},
		"another id":  {plannedDir + "/itd-10-alpha.md", plannedLinked("itd-11", "alpha", "spc-1")},
		"no spec_id":  {plannedDir + "/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "null")},
	} {
		if _, err := ComposeDeliveryAudit("itd-10", c[0], c[1], []string{"spc-1"}); err == nil {
			t.Fatalf("%s: refused", name)
		}
	}
}
