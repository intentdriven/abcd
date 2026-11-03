package capture

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// decisionneeded.go is the decision-record marker on the issue (itd-82
// decision 10; spc-2609212015054359 step 6). A drain that hands an issue back
// because it turns on a rule about trust or safety flags it as needing a
// decision record, with the question stated, in its summary; this is the same
// flag written onto the open issue as `decision_needed: <question>`, so a
// person finds the decision it owes without the summary. Nothing is minted:
// the decision record is a person's to write.
//
// It is the one write a drain makes to a record it otherwise never edits,
// chosen knowing that (decision 10), and it is not the classification, which
// the drain re-derives every run (decision 8): the eligibility rule reads the
// marker as a hand-back a person has not yet answered, and a person answers it
// by writing the decision record and taking the field out.

// DecisionNeededRequest writes the decision-record marker onto one open
// record.
type DecisionNeededRequest struct {
	RepoRoot   string
	IssuesRoot string
	ID         string
	// Question is the question the decision record answers, as the hand-back
	// stated it. Whitespace runs, line breaks included, are folded to single
	// spaces, since the field is one line.
	Question string
}

// DecisionNeededResult reports the marker written and any it replaced, so a
// replacement is never silent. Unchanged is true when the record already
// carried this question, and nothing was written. The record stays in open/.
type DecisionNeededResult struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Status    State  `json:"status"`
	Question  string `json:"question"`
	Previous  string `json:"previous,omitempty"`
	Unchanged bool   `json:"unchanged,omitempty"`
	Redacted  int    `json:"redacted,omitempty"`
	Degraded  string `json:"redaction_degraded,omitempty"`
}

// decisionQuestion is the marker's value for a question as stated: redacted
// as every caller text written into a record is, folded to one line, and its
// hidden runes encoded, the value MarkDecisionNeeded writes and
// DecisionNeededHolds compares.
func decisionQuestion(repoRoot, question string) (value string, redacted int, degraded string) {
	red, redacted, degraded := redactLedgerText(repoRoot, question)
	folded := grounds.Fold(red)
	if folded == "" {
		return "", redacted, degraded
	}
	return termsafe.EncodeHiddenRunes(folded), redacted, degraded
}

// MarkDecisionNeeded writes or replaces the `decision_needed:` of an open
// record, and changes nothing else in it: every other line, the body
// included, is kept byte for byte. It refuses, with nothing written, a
// malformed id, an empty question, and a record that is not open. A record
// already carrying this question is left as it is (Unchanged), so a hand-back
// routed again after a killed call writes nothing twice.
func MarkDecisionNeeded(req DecisionNeededRequest) (DecisionNeededResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return DecisionNeededResult{}, err
	}
	if !reIssID.MatchString(req.ID) {
		return DecisionNeededResult{}, refused(fmt.Errorf("decision_needed: invalid iss-N identifier: %q; nothing written", req.ID))
	}
	question, redacted, degraded := decisionQuestion(repoRoot, req.Question)
	if question == "" {
		return DecisionNeededResult{}, refused(fmt.Errorf("decision_needed: the question is empty — state the question the decision record answers, one line; nothing written"))
	}
	if err := requireHeld(issuesRoot, req.ID); err != nil {
		return DecisionNeededResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return DecisionNeededResult{}, err
	}

	var result DecisionNeededResult
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		src, status, err := findIssue(issuesRoot, req.ID)
		if err != nil {
			return err
		}
		if status != StateOpen {
			return fmt.Errorf("%w: %s is not open (it is in %s) — the decision-record marker is written on an open issue; nothing written",
				ErrTransitionConflict, req.ID, status)
		}
		content, checksum, err := readWithChecksum(src)
		if err != nil {
			return err
		}
		fm, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return err
		}
		previous, _ := fm[issueschema.DecisionNeededKey].(string)
		result = DecisionNeededResult{ID: req.ID, Path: fsutil.RepoRel(repoRoot, src), Status: StateOpen,
			Question: question, Redacted: redacted, Degraded: degraded}
		if previous == question {
			result.Unchanged = true
			return nil
		}
		result.Previous = previous
		newContent, err := setScalarField(content, issueschema.DecisionNeededKey, question)
		if err != nil {
			return err
		}
		newFM, _, err := parseFrontmatterAndBody(newContent)
		if err != nil {
			return err
		}
		if err := validateStrict(newFM); err != nil {
			return err
		}
		if err := validateInvariants(newFM, StateOpen, src); err != nil {
			return err
		}
		_, current, err := readWithChecksum(src)
		if err != nil {
			return err
		}
		if current != checksum {
			return fmt.Errorf("%w: %s changed since it was read", ErrChecksumMismatch, src)
		}
		return writeLedgerFile(repoRoot, issuesRoot, src, []byte(newContent))
	})
	if err != nil {
		return DecisionNeededResult{}, err
	}
	return result, nil
}

// DecisionNeededHolds reports whether this checkout's open record of id
// already carries the decision-record marker with question, as
// MarkDecisionNeeded would write it: a hand-back a killed call already marked
// is found, not marked twice. It reads and writes nothing else; a record that
// is not open, or not held, does not hold it.
func DecisionNeededHolds(repoRoot, id, question string) (bool, error) {
	if !reIssID.MatchString(id) {
		return false, nil
	}
	want, _, _ := decisionQuestion(repoRoot, question)
	if want == "" {
		return false, nil
	}
	lr, err := List(ListRequest{RepoRoot: repoRoot, State: StateOpen})
	if err != nil {
		return false, err
	}
	for _, iss := range lr.Issues {
		if iss.ID == id {
			return iss.DecisionNeeded == want, nil
		}
	}
	return false, nil
}
