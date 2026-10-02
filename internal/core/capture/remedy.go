package capture

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// remedy.go is the verb a person runs to write the fix an open issue proposes
// (`abcd capture remedy`). Every new issue carries a remedy (ruling BX3 of
// 2026-09-29), an automatic filer with no fix yet writes
// issueschema.MachineRemedy (ruling H12), and a drain skips such a record
// until a person writes a real one: this is how they write it, and how a
// record filed before the rule gains one.

// RemedyRequest writes a person's remedy onto one open record.
type RemedyRequest struct {
	RepoRoot   string
	IssuesRoot string
	ID         string
	// Remedy is the proposed fix. Whitespace runs, line breaks included, are
	// folded to single spaces, since the field is one line.
	Remedy string
}

// RemedyResult reports the remedy written and the one it replaced, so a
// replacement is never silent. The record stays in open/.
type RemedyResult struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Status   State  `json:"status"`
	Remedy   string `json:"remedy"`
	Previous string `json:"previous,omitempty"`
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
}

// SetRemedy writes or replaces the `remedy:` of an open record. It refuses,
// with nothing written, a malformed id, an empty remedy, the machine value
// (which only an automatic filer writes, and which would leave the record as
// the drain already skips it), and a record that is not open. A record still
// carrying the older `suggested_fix:` keeps it; `remedy:` is read first.
func SetRemedy(req RemedyRequest) (RemedyResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return RemedyResult{}, err
	}
	if !reIssID.MatchString(req.ID) {
		return RemedyResult{}, refused(fmt.Errorf("remedy: invalid iss-N identifier: %q; nothing written", req.ID))
	}
	redRemedy, redacted, degraded := redactLedgerText(repoRoot, req.Remedy)
	remedy := grounds.Fold(redRemedy)
	switch {
	case remedy == "":
		return RemedyResult{}, refused(fmt.Errorf("remedy: the remedy is empty — name the fix the issue proposes, one line; nothing written"))
	case issueschema.IsMachineRemedy(remedy):
		return RemedyResult{}, refused(fmt.Errorf(
			"remedy: %q is the value an automatic filer writes when it has no fix, and a drain skips it — name the fix the issue proposes; nothing written",
			issueschema.MachineRemedy))
	}
	remedy = termsafe.EncodeHiddenRunes(remedy)
	// An id the ledger does not hold is refused before the preamble provisions
	// anything (iss-2609302305500526).
	if err := requireHeld(issuesRoot, req.ID); err != nil {
		return RemedyResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return RemedyResult{}, err
	}

	var result RemedyResult
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		src, status, err := findIssue(issuesRoot, req.ID)
		if err != nil {
			return err
		}
		if status != StateOpen {
			return fmt.Errorf("%w: %s is not open (it is in %s) — a remedy is written on an open issue; nothing written",
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
		previous := issueschema.RemedyOf(fm)
		newContent, err := setScalarField(content, "remedy", remedy)
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
		if err := writeLedgerFile(repoRoot, issuesRoot, src, []byte(newContent)); err != nil {
			return err
		}
		result = RemedyResult{ID: req.ID, Path: fsutil.RepoRel(repoRoot, src), Status: StateOpen,
			Remedy: remedy, Previous: previous, Redacted: redacted, Degraded: degraded}
		return nil
	})
	if err != nil {
		return RemedyResult{}, err
	}
	return result, nil
}
