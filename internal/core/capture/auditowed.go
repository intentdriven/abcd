package capture

import (
	"errors"
	"fmt"
	"sort"
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
// one through Capture, the one canonical filer. The open-issue scan and the
// filing are ONE step under the ledger lock, so two ingests of one failed
// audit racing each other file one issue and the second links to it: with the
// scan outside the lock, both saw no carrier and both filed.
//
// More than one open carrier (a race before the lock held the scan, or an
// ingest whose intent write failed after its filing) is collapsed: the oldest
// is the one linked, and every other is declined as its duplicate through the
// ledger's own duplicate closure (wontfix with a duplicates link), after the
// lock is released since that verb takes it.
func fileAuditOwed(o intent.AuditOwed) (intent.AuditFiling, error) {
	rr, ir, err := resolveRoots(o.RepoRoot, "")
	if err != nil {
		return intent.AuditFiling{}, err
	}
	var (
		filing intent.AuditFiling
		extras []string
	)
	err = withLedgerLock(rr, ir, func() error {
		carriers, err := openOwedCarriers(o.RepoRoot, o.IntentID, o.ReceiptID)
		if err != nil {
			return err
		}
		if len(carriers) > 0 {
			filing, extras = intent.AuditFiling{IssueID: carriers[0], Linked: true}, carriers[1:]
			return nil
		}
		res, err := captureHeld(auditOwedRequest(o))
		if err != nil {
			return err
		}
		filing = intent.AuditFiling{IssueID: res.ID}
		return nil
	})
	if err != nil {
		return intent.AuditFiling{}, err
	}
	for _, id := range extras {
		if err := declineDuplicateCarrier(o, id, filing.IssueID); err != nil {
			return intent.AuditFiling{}, err
		}
	}
	return filing, nil
}

// auditOwedRequest is the capture request filing o's owed check.
func auditOwedRequest(o intent.AuditOwed) CaptureRequest {
	sev, cat := auditUndecidedSeverity, auditUndecidedCategory
	if o.Failed {
		sev, cat = auditFailedSeverity, auditFailedCategory
	}
	crit := owedList(o)
	return CaptureRequest{
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
	}
}

// clearAuditOwed resolves every open issue carrying the owed check of a
// receipt a passing audit judged: the one the flag names, when the receipt
// carries a flag, and any other carrier of the same receipt a race or a failed
// intent write left open, flag or no flag, so none outlives the check it
// carries. One List of the open ledger finds them. An issue no longer open
// (resolved or declined by hand, or by a concurrent re-run) is left as it
// stands.
func clearAuditOwed(c intent.AuditCleared) error {
	carriers, err := openOwedCarriers(c.RepoRoot, c.IntentID, c.ReceiptID)
	if err != nil {
		return err
	}
	if c.IssueID != "" && !contains(carriers, c.IssueID) {
		open, err := isOpenIssue(c.RepoRoot, c.IssueID)
		if err != nil {
			return err
		}
		if open {
			carriers = append([]string{c.IssueID}, carriers...)
		}
	}
	for _, id := range carriers {
		_, err := Resolve(ResolveRequest{
			RepoRoot: c.RepoRoot,
			ID:       id,
			Resolution: fmt.Sprintf("A re-run of the fidelity audit of %s (receipt %s) judged no criterion NOT_MET or INCONCLUSIVE, "+
				"so the check this record carried is met and the intent's audit-owed flag is cleared.", c.IntentID, c.ReceiptID),
			Impact:   "fix",
			ByIntent: c.IntentID,
			Grounds: "pursued: the passing re-audit is the check this record carried; " +
				"a later audit of the same receipt judging a criterion NOT_MET or INCONCLUSIVE would show it wrong",
		})
		if err != nil && !errors.Is(err, ErrTransitionConflict) {
			return err
		}
	}
	return nil
}

// isOpenIssue reports whether id is in the open ledger.
func isOpenIssue(repoRoot, id string) (bool, error) {
	list, err := List(ListRequest{RepoRoot: repoRoot, State: StateOpen})
	if err != nil {
		return false, err
	}
	for _, iss := range list.Issues {
		if iss.ID == id {
			return true, nil
		}
	}
	return false, nil
}

// openOwedCarriers lists the open issues carrying the owed check of the
// intent's receipt, oldest first. It takes no lock: fileAuditOwed calls it
// under the ledger lock it holds.
func openOwedCarriers(repoRoot, intentID, rcp string) ([]string, error) {
	list, err := List(ListRequest{RepoRoot: repoRoot, State: StateOpen})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, iss := range list.Issues {
		if carriesOwedAudit(iss, intentID, rcp) {
			ids = append(ids, iss.ID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return olderIssID(ids[i], ids[j]) })
	return ids, nil
}

// olderIssID orders two iss ids by age: a longer number is a later mint (the
// timestamp-numeric ids are longer than every ordinal), and ids of one length
// order by their digits.
func olderIssID(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// declineDuplicateCarrier closes an extra carrier of o's owed check as a
// duplicate of keep. One a concurrent ingest already closed is left as it is.
func declineDuplicateCarrier(o intent.AuditOwed, id, keep string) error {
	_, err := Wontfix(WontfixRequest{
		RepoRoot: o.RepoRoot,
		ID:       id,
		Reason: fmt.Sprintf("A duplicate of %s, which carries the same owed check of the fidelity audit of %s (receipt %s): "+
			"one issue carries an owed audit check, and the flag names %s.", keep, o.IntentID, o.ReceiptID, keep),
		Duplicates: []string{keep},
	})
	if errors.Is(err, ErrTransitionConflict) {
		return nil
	}
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
