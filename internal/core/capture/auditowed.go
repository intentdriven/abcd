package capture

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

// auditowed.go is the ledger half of ruling DQ1c (2026-09-30, the product
// thinker): an after-merge fidelity audit that fails or comes back undecided
// leaves the intent shipped and flagged, and ONE issue captured here carries
// the owed check. A second failed audit of the same receipt links to that
// issue while it is open rather than filing a double, and a passing re-audit
// resolves it. The intent store calls both through the seam registered below
// (intent.SetAuditLedger), with its own lock released, since this ledger's
// lock comes first in the one lock order.

func init() { intent.SetAuditLedger(fileAuditOwed, clearAuditOwed) }

// Severity and category of an owed audit check. A criterion judged NOT_MET is
// a shipped behaviour that does not do what its intent promised: a bug, and
// major, since the changelog already announces it. A criterion judged only
// INCONCLUSIVE is a record that claims shipped with a check that could not
// confirm it: an inconsistency between the record and its evidence, and
// minor, since nothing is known to be wrong yet. Both categories are in the
// drain's fixable set, so the drain's severity rule decides as for any issue.
const (
	auditFailedSeverity    = "major"
	auditFailedCategory    = "bug"
	auditUndecidedSeverity = "minor"
	auditUndecidedCategory = "inconsistency"
)

// fileAuditOwed returns the open issue already carrying o's receipt, or files
// one through Capture, the one canonical filer.
func fileAuditOwed(o intent.AuditOwed) (intent.AuditFiling, error) {
	list, err := List(ListRequest{RepoRoot: o.RepoRoot, State: StateOpen})
	if err != nil {
		return intent.AuditFiling{}, err
	}
	for _, iss := range list.Issues {
		if carriesOwedAudit(iss, o.IntentID, o.ReceiptID) {
			return intent.AuditFiling{IssueID: iss.ID, Linked: true}, nil
		}
	}
	sev, cat := auditUndecidedSeverity, auditUndecidedCategory
	if o.Failed {
		sev, cat = auditFailedSeverity, auditFailedCategory
	}
	crit := owedList(o)
	res, err := Capture(CaptureRequest{
		RepoRoot:       o.RepoRoot,
		Text:           auditOwedText(o, crit),
		Severity:       Severity(sev),
		Category:       Category(cat),
		Source:         "agent-finding",
		FoundDuring:    "abcd intent audit ingest, the after-merge fidelity audit of " + o.IntentID + " (receipt " + o.ReceiptID + ")",
		FoundAt:        o.IntentPath,
		RelatedIntents: []string{o.IntentID},
		Remedy:         auditOwedRemedy(o, crit),
		// The filing-time match compares the owed check's own words, so a
		// record another filer holds for the same shortfall is linked.
		Match:     auditMatchConfig(o.RepoRoot),
		MatchText: auditOwedTitle(o, crit),
	})
	if err != nil {
		return intent.AuditFiling{}, err
	}
	return intent.AuditFiling{IssueID: res.ID}, nil
}

// clearAuditOwed resolves the issue a passing re-audit cleared. An issue no
// longer open (resolved or declined by hand) is left as it stands.
func clearAuditOwed(c intent.AuditCleared) error {
	list, err := List(ListRequest{RepoRoot: c.RepoRoot, State: StateOpen})
	if err != nil {
		return err
	}
	open := false
	for _, iss := range list.Issues {
		if iss.ID == c.IssueID {
			open = true
			break
		}
	}
	if !open {
		return nil
	}
	_, err = Resolve(ResolveRequest{
		RepoRoot: c.RepoRoot,
		ID:       c.IssueID,
		Resolution: fmt.Sprintf("A re-run of the fidelity audit of %s (receipt %s) judged no criterion NOT_MET or INCONCLUSIVE, "+
			"so the check this record carried is met and the intent's audit-owed flag is cleared.", c.IntentID, c.ReceiptID),
		Impact:   "fix",
		ByIntent: c.IntentID,
		Grounds: "pursued: the passing re-audit is the check this record carried; " +
			"a later audit of the same receipt judging a criterion NOT_MET or INCONCLUSIVE would show it wrong",
	})
	return err
}

// carriesOwedAudit reports whether iss is the record carrying the owed check of
// the intent's receipt: it relates to the intent and names the receipt.
func carriesOwedAudit(iss Issue, intentID, rcp string) bool {
	return contains(iss.RelatedIntents, intentID) && strings.Contains(iss.Body, "(receipt "+rcp+")")
}

func owedList(o intent.AuditOwed) string {
	parts := make([]string, 0, len(o.Criteria))
	for _, c := range o.Criteria {
		parts = append(parts, c.ID+" "+c.Verdict)
	}
	return strings.Join(parts, ", ")
}

func auditOwedTitle(o intent.AuditOwed, crit string) string {
	if o.Failed {
		return fmt.Sprintf("The after-merge fidelity audit of %s judged a criterion unmet, so its check is still owed: %s", o.IntentID, crit)
	}
	return fmt.Sprintf("The after-merge fidelity audit of %s could not decide a criterion, so its check is still owed: %s", o.IntentID, crit)
}

// auditOwedText is the filed record's body: the title line the slug is derived
// from, then the receipt, the flag and the way the record closes.
func auditOwedText(o intent.AuditOwed, crit string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", auditOwedTitle(o, crit))
	fmt.Fprintf(&b, "The audit of %s (receipt %s) judged %s. The intent stays in shipped/ and its changelog entry stands; "+
		"its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).\n\n",
		o.IntentID, o.ReceiptID, crit)
	fmt.Fprintf(&b, "Remedy: %s. `abcd intent audit %s` rewrites the review request for the re-run, and "+
		"`abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE "+
		"clears the flag and resolves this record.\n", o.Remedy(), o.IntentID)
	return b.String()
}

// auditOwedRemedy is the record's remedy: real work a drain can take, not the
// automatic filers' placeholder, since the flag says what must be done.
func auditOwedRemedy(o intent.AuditOwed, crit string) string {
	if o.Failed {
		return fmt.Sprintf("fix, then re-run the audit: make %s of %s hold in the delivered code, then re-emit the request with `abcd intent audit %s` "+
			"and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record",
			crit, o.IntentID, o.IntentID)
	}
	return fmt.Sprintf("re-run the audit: re-emit the request with `abcd intent audit %s`, give the auditor the evidence %s lacked, "+
		"and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record",
		o.IntentID, crit)
}

// auditMatchConfig is the filing-time match's configuration, read as the
// capture verb reads it. A configuration the reader refuses files the record
// unmatched rather than refusing the filing, as every filer does.
func auditMatchConfig(repoRoot string) *match.Config {
	roots, _ := layered.RootsFor(repoRoot)
	cfg, err := match.LoadConfig(roots)
	if err != nil {
		return nil
	}
	return &cfg
}
