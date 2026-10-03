package capture

import (
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DeferRequest carries a release-cut waiver for one open record
// (iss-2609181223260994): the anchor tag the finding is carried past and the
// reason it is.
type DeferRequest struct {
	RepoRoot   string
	IssuesRoot string
	ID         string
	// After is the anchor tag, which must be the checkout's newest release tag:
	// the cut reads the waiver by string equality against its own base, and a
	// waiver naming any other tag is one the cut refuses.
	After string
	// Reason is why the finding is carried past this cut rather than fixed.
	Reason string
}

// DeferResult reports a deferral in the shape a transition reports: the record,
// where it is, and the waiver written. The record stays in open/.
type DeferResult struct {
	ID             string `json:"id"`
	Path           string `json:"path"`
	Status         State  `json:"status"`
	DeferredAfter  string `json:"deferred_after"`
	DeferralReason string `json:"deferral_reason"`
	Redacted       int    `json:"redacted,omitempty"`
	Degraded       string `json:"redaction_degraded,omitempty"`
}

// deferralNow is the clock the body section's date is read from; a test seam.
var deferralNow = time.Now

// deferrableSeverities are the grades the release cut's finding guard blocks
// on, and so the only grades a waiver has anything to waive.
var deferrableSeverities = map[Severity]bool{SeverityMajor: true, SeverityCritical: true}

// Defer writes the release cut's waiver pair — deferred_after and
// deferral_reason — onto an open major or critical record, and appends a dated
// `## Deferral` section to its body, so the waiver is a validated write rather
// than a hand edit of frontmatter (iss-2609181223260994).
//
// It refuses, with nothing written, everything the cut's own reader would not
// honour: a tag that is not the checkout's current anchor, an empty reason, a
// record that is not open, and a grade the guard never blocks on. A record
// already deferred past an earlier anchor is re-deferred: the pair is replaced
// and a new body section is appended, so the history of each deferral stays in
// the record. A record already deferred past the SAME anchor has the pair and
// that cycle's section replaced, so the body keeps one section per cycle.
func Defer(req DeferRequest) (DeferResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return DeferResult{}, err
	}
	if !reIssID.MatchString(req.ID) {
		return DeferResult{}, refused(fmt.Errorf("defer: invalid iss-N identifier: %q; nothing written", req.ID))
	}
	if !reShippedIn.MatchString(req.After) {
		return DeferResult{}, refused(fmt.Errorf("defer: --after %q is not a release tag (want vMAJOR.MINOR.PATCH); nothing written", req.After))
	}
	redReason, redacted, degraded := redactLedgerText(repoRoot, req.Reason)
	reason := grounds.Fold(redReason)
	if reason == "" {
		return DeferResult{}, refused(fmt.Errorf("defer: the reason is empty — a deferral with no stated reason records nothing; nothing written"))
	}
	reason = termsafe.EncodeHiddenRunes(reason)
	// An id the ledger does not hold is refused before the preamble provisions
	// anything (iss-2609302305500526).
	if err := requireHeld(issuesRoot, req.ID); err != nil {
		return DeferResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return DeferResult{}, err
	}

	var result DeferResult
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		src, status, err := findIssue(issuesRoot, req.ID)
		if err != nil {
			return err
		}
		if status != StateOpen {
			return fmt.Errorf("%w: %s is not open (it is in %s) — only an open record blocks a cut; nothing written",
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
		if sev := Severity(asString(fm["severity"])); !deferrableSeverities[sev] {
			return refused(fmt.Errorf("defer: %s is %s — only a major or critical record blocks a cut, so there is nothing to defer; nothing written",
				req.ID, sev))
		}
		// The record is judged first, so a refusal about it names the record the
		// checkout's ledger holds; the anchor is then the checkout's own.
		if err := requireCurrentAnchor(repoRoot, req.After); err != nil {
			return err
		}
		newContent, err := setScalarField(content, "deferred_after", rawScalar(req.After))
		if err != nil {
			return err
		}
		if newContent, err = setScalarField(newContent, "deferral_reason", reason); err != nil {
			return err
		}
		newContent = appendDeferralSection(newContent, deferralNow().UTC().Format("2006-01-02"), req.After, reason)
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
		result = DeferResult{ID: req.ID, Path: fsutil.RepoRel(repoRoot, src), Status: StateOpen,
			DeferredAfter: req.After, DeferralReason: reason, Redacted: redacted, Degraded: degraded}
		return nil
	})
	if err != nil {
		return DeferResult{}, err
	}
	return result, nil
}

// appendDeferralSection writes one dated `## Deferral` section into the record,
// the body half of a deferral's shape: the frontmatter pair is what the cut
// reads, and the section is what a reader of the record sees, one per cycle.
//
// One per cycle is what the section is, so a second deferral past the SAME
// anchor (a corrected reason, or the verb run twice) rewrites that cycle's
// section in place rather than appending a second one (iss-2609251823555125);
// the superseded wording stays in git's history, where every earlier revision of
// a record lives. A deferral past a different anchor appends, so each cycle's
// section stays in the record.
func appendDeferralSection(content, date, after, reason string) string {
	heading, line := "## Deferral "+date, "Deferred past "+after+": "+reason
	if i := sameCycleDeferral(content, after); i >= 0 {
		lines := strings.Split(content, "\n")
		lines[i], lines[i+2] = heading, line
		return strings.Join(lines, "\n")
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + heading + "\n\n" + line + "\n"
}

// sameCycleDeferral returns the line index of the last `## Deferral` heading
// whose section is the one this verb writes for anchor — the heading, a blank
// line, then `Deferred past <anchor>: ` — or -1 when the record carries none. A
// section of any other shape is a hand edit the verb does not own, so it is left
// alone and the new section appended.
func sameCycleDeferral(content, anchor string) int {
	lines := strings.Split(content, "\n")
	for i := len(lines) - 3; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "## Deferral ") && lines[i+1] == "" &&
			strings.HasPrefix(lines[i+2], "Deferred past "+anchor+": ") {
			return i
		}
	}
	return -1
}

// requireCurrentAnchor refuses a tag that is not the checkout's newest release
// tag — the base the cut measures from and compares deferred_after against.
func requireCurrentAnchor(repoRoot, after string) error {
	anchor, found, err := changelog.LatestReleaseTag(repoRoot)
	if err != nil {
		return fmt.Errorf("defer: reading the release tags: %w", err)
	}
	if !found {
		return refused(fmt.Errorf("defer: this checkout has no release tag, so there is no cut to defer past; nothing written"))
	}
	if after != anchor.Tag() {
		return refused(fmt.Errorf(
			"defer: --after %s is not the current anchor %s — the cut honours a deferral past its own anchor only, and one past any other tag has lapsed or never applied; nothing written",
			after, anchor.Tag()))
	}
	return nil
}
