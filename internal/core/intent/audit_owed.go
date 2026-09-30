package intent

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// audit_owed.go — what an after-merge fidelity audit that fails or comes back
// undecided leaves behind (ruling DQ1c, 2026-09-30, the product thinker):
//
//	"(a) STAYS SHIPPED, FLAGGED: the intent stays in shipped/, its changelog
//	entry stands, and a 'check still owed' flag names the unmet/undecided
//	criteria until it is fixed and re-checked."
//
// The owed check is carried by ONE issue the ledger captures automatically,
// naming the intent and the remaining criteria, with a real remedy: "fix, then
// re-run the audit" when a criterion is NOT_MET, "re-run the audit" when the
// verdict was only undecided (INCONCLUSIVE). A later audit of the same receipt
// that judges no criterion NOT_MET or INCONCLUSIVE clears the flag with a dated
// Audit Notes line and resolves that issue. Nothing un-ships: the intent never
// leaves shipped/ (lifecycle.go's "never un-ship" and itd-50 stand).
//
// The ledger's package imports this one, so the filer is a seam it registers
// from init (SetAuditLedger), as it registers its lock (SetLedgerLock). The
// ledger's lock comes before this store's in the one lock order, so the ingest
// calls the seam with the intent store's lock RELEASED: it judges the verdict
// under the lock, files or resolves, and then writes the record under the lock
// again, judging it afresh (IngestVerdictBytes).

// OwedCriterion is one criterion an audit judged NOT_MET or INCONCLUSIVE.
type OwedCriterion struct {
	ID      string `json:"criterion_id"`
	Verdict string `json:"verdict"`
}

// AuditOwed is the check an after-merge audit leaves owed on a shipped intent.
type AuditOwed struct {
	RepoRoot   string
	IntentID   string
	IntentPath string // repo-relative, slash-separated
	ReceiptID  string
	Criteria   []OwedCriterion
	// Failed is true when a criterion is NOT_MET, false when every owed one is
	// INCONCLUSIVE.
	Failed bool
}

// Remedy is the flag's remedy: fix, then re-run, for a failed audit; only the
// re-run for an undecided one.
func (o AuditOwed) Remedy() string {
	if o.Failed {
		return "fix, then re-run the audit"
	}
	return "re-run the audit"
}

// AuditFiling is the ledger's answer to an owed check: the issue carrying it,
// and whether that issue was already open (linked) rather than filed now.
type AuditFiling struct {
	IssueID string
	Linked  bool
}

// AuditCleared asks the ledger to resolve the issue a passing re-audit cleared.
type AuditCleared struct {
	RepoRoot  string
	IntentID  string
	ReceiptID string
	IssueID   string
}

var (
	auditOwedFiler   func(AuditOwed) (AuditFiling, error)
	auditOwedClearer func(AuditCleared) error
	// auditNow dates the clearance line; a test pins it.
	auditNow = time.Now
)

// SetAuditLedger registers the ledger's filer and resolver for owed audit
// checks, for the package that owns the ledger to call once, from init. In a
// binary that links no ledger both stay nil: the flag is still written, and
// says that no issue carries it.
func SetAuditLedger(file func(AuditOwed) (AuditFiling, error), clear func(AuditCleared) error) {
	auditOwedFiler, auditOwedClearer = file, clear
}

// owedCriteria is every criterion of v judged NOT_MET or INCONCLUSIVE, in the
// verdict's order, and whether any is NOT_MET.
func owedCriteria(v verdict) ([]OwedCriterion, bool) {
	var out []OwedCriterion
	failed := false
	for _, c := range v.Criteria {
		switch c.Verdict {
		case "NOT_MET":
			failed = true
		case "INCONCLUSIVE":
		default:
			continue
		}
		out = append(out, OwedCriterion{ID: c.CriterionID, Verdict: c.Verdict})
	}
	return out, failed
}

// owedFlagRe reads a flag line back: the receipt, and the issue carrying it.
var owedFlagRe = regexp.MustCompile(`^Audit owed \(receipt (rcp-[0-9a-f]+)\): .*\. Carried by (iss-[0-9]+)\.$`)

// owedFlagLine renders the flag. Criterion ids are shape-validated and the
// verdicts are the closed enum, so nothing here is free text.
func owedFlagLine(o AuditOwed, issueID string) string {
	parts := make([]string, 0, len(o.Criteria))
	for _, c := range o.Criteria {
		parts = append(parts, c.ID+" "+c.Verdict)
	}
	carried := "No issue carries it: no ledger is linked to file one."
	if issueID != "" {
		carried = "Carried by " + issueID + "."
	}
	return fmt.Sprintf("Audit owed (receipt %s): %s. Remedy: %s. %s",
		o.ReceiptID, strings.Join(parts, " · "), o.Remedy(), carried)
}

// clearedLine is the dated Audit Notes line a passing re-audit leaves where the
// flag was.
func clearedLine(rcp, issueID string, now time.Time) string {
	return fmt.Sprintf("Audit owed flag cleared %s (receipt %s): a re-run of the audit judged no criterion NOT_MET or INCONCLUSIVE, and %s is resolved.",
		now.UTC().Format("2006-01-02"), rcp, issueID)
}

// flaggedIssue returns the issue the review block for rcp names on its flag
// line, if the block carries one.
func flaggedIssue(content, rcp string) (string, bool) {
	lines, b, ok := reviewBlockFor(content, rcp)
	if !ok {
		return "", false
	}
	for _, ln := range lines[b.start:b.end] {
		if m := owedFlagRe.FindStringSubmatch(strings.TrimRight(ln, "\r")); m != nil && m[1] == rcp {
			return m[2], true
		}
	}
	return "", false
}

// hasOwedFlag reports whether the review block for rcp carries a flag line,
// with or without an issue.
func hasOwedFlag(content, rcp string) bool {
	lines, b, ok := reviewBlockFor(content, rcp)
	if !ok {
		return false
	}
	prefix := "Audit owed (receipt " + rcp + "): "
	for _, ln := range lines[b.start:b.end] {
		if strings.HasPrefix(ln, prefix) {
			return true
		}
	}
	return false
}

// auditLedgerOutcome is what the ledger answered between the two holds of the
// intent store's lock, handed to the write.
type auditLedgerOutcome struct {
	issue   string
	linked  bool
	cleared string
}
