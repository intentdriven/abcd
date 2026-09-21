package capture

import (
	"errors"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// LinkRequest edits one record's blocked_by edges AFTER capture
// (iss-2609200951237670). Unblock is applied first, then BlockedBy, so the
// same id on both sides is removed and re-added — a documented net no-op. At
// least one of the two must be non-empty.
type LinkRequest struct {
	RepoRoot   string
	IssuesRoot string
	ID         string   // the subject iss-N, in any status folder
	BlockedBy  []string // iss-N ids to append to the record's blocked_by
	Unblock    []string // iss-N ids to remove from it
}

// LinkResult is the outcome of a successful Link: the subject, its
// repo-relative path, and the blocked_by list AS WRITTEN. An emptied list comes
// back as an empty list, never null — the field is the point of the verb.
type LinkResult struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	BlockedBy []string `json:"blocked_by"`
}

// blockedByDocs names where the field is documented. Every refusal about a
// blocked_by target carries it, on capture and on link alike: the shape was
// always documented, and the finding behind this verb was a session that found
// it by running strings on the binary because neither the refusal nor the flag
// help said where to look.
const blockedByDocs = "blocked_by is documented in .abcd/work/issues/README.md under \"Derived priority\" and in commands/capture.md under \"Link\""

// BlockedByFlagHelp is the --blocked-by flag's help on both verbs that take it,
// exported so the front doors render one sentence rather than two copies.
const BlockedByFlagHelp = "comma-separated iss-N ids this issue is blocked by; each must exist in the ledger — " + blockedByDocs

// validateBlockers is the ONE validator for a list of blocked_by targets, run
// by capture's --blocked-by and by link's, so the two verbs cannot come to
// differ about what an edge may name. Before anything is written it checks, in
// this order: the id shape (^iss-[0-9]+$); that no id names the subject (a
// record cannot block itself — for capture the subject is the migrator's
// ForceID, and empty otherwise, since a minted id cannot be named before it
// exists); that every target exists in the ledger, in ANY status folder; and it
// collapses duplicates, order preserved. Blocking on a resolved or wontfix
// target is legal: existence is what the cross-reference claims, and whether a
// blocker still holds anything up is the read-time priority projection's
// question (prioritise reads open/ alone).
//
// The existence probe matters because the record-lint blocker record_schema
// refuses a cross-reference whose target is not in the corpus — a verb that
// minted an unverified link would hand back a record its own gate rejects, and
// the caller would learn of it from the next preflight rather than from the
// command that wrote it. The refusal names where the field is documented.
//
// verb is the command prefix the refusal carries ("capture", "capture link").
func validateBlockers(issuesRoot, verb, subject string, ids []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, dep := range ids {
		if !reIssID.MatchString(dep) {
			return nil, fmt.Errorf("%s: --blocked-by token %q must match iss-N; nothing written (%s)", verb, dep, blockedByDocs)
		}
		if dep == subject {
			return nil, fmt.Errorf("%s: --blocked-by %s names the record itself, and a record cannot block itself; nothing written", verb, dep)
		}
		if seen[dep] {
			continue
		}
		if _, _, err := findIssue(issuesRoot, dep); err != nil {
			if errors.Is(err, ErrUnknownIssueID) {
				return nil, fmt.Errorf("%s: --blocked-by %s not found in the issue ledger; nothing written (%s)", verb, dep, blockedByDocs)
			}
			return nil, fmt.Errorf("%s: --blocked-by %s: %w; nothing written", verb, dep, err)
		}
		seen[dep] = true
		out = append(out, dep)
	}
	return out, nil
}

// Link appends to, or removes from, one record's blocked_by list — the
// post-hoc form of capture's --blocked-by, for the ordinary case the create-time
// flag cannot serve: the blocker captured after the blocked record, or in
// another lane. The subject may sit in ANY status folder and never moves: a
// resolved record's edges are history, still editable, and the write is the
// same in-place stamp promote uses from any folder (find, checksum-read, rewrite
// the frontmatter, validate the result against the folder it is in, atomic
// write under the ledger lock). The derived-priority reader picks the change up
// with no other change, because it reads the field this rewrites.
//
// Every refusal is raised before anything is written: the targets through
// validateBlockers (capture's own validator), and an --unblock of an id the
// list does not currently hold, which is refused naming the current list —
// removing an edge that is not there is a wrong belief about the record, not a
// no-op.
func Link(req LinkRequest) (LinkResult, error) {
	const verb = "capture link"
	if len(req.BlockedBy) == 0 && len(req.Unblock) == 0 {
		return LinkResult{}, fmt.Errorf("%s: nothing to do — give --blocked-by <iss-N,...> and/or --unblock <iss-N,...>; nothing written", verb)
	}
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return LinkResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return LinkResult{}, err
	}
	if !reIssID.MatchString(req.ID) {
		return LinkResult{}, fmt.Errorf("invalid iss-N identifier: %q", req.ID)
	}
	// The subject is located BEFORE the targets are validated, so an unknown
	// subject is reported as the fault rather than a target it would never have
	// been linked to.
	if _, _, err := findIssue(issuesRoot, req.ID); err != nil {
		return LinkResult{}, err
	}
	add, err := validateBlockers(issuesRoot, verb, req.ID, req.BlockedBy)
	if err != nil {
		return LinkResult{}, err
	}
	var remove []string
	seen := map[string]bool{}
	for _, dep := range req.Unblock {
		if !reIssID.MatchString(dep) {
			return LinkResult{}, fmt.Errorf("%s: --unblock token %q must match iss-N; nothing written", verb, dep)
		}
		if !seen[dep] {
			seen[dep] = true
			remove = append(remove, dep)
		}
	}

	var result LinkResult
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		src, status, err := findIssue(issuesRoot, req.ID)
		if err != nil {
			return err
		}
		content, _, err := readWithChecksum(src)
		if err != nil {
			return err
		}
		fm, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return err
		}
		if err := validateStrict(fm); err != nil {
			return err
		}
		current := asStrList(fm["blocked_by"])
		// Unblock first: each removal must name an edge the record holds NOW,
		// judged against the bytes under the lock.
		for _, dep := range remove {
			if !containsString(current, dep) {
				return fmt.Errorf("%s: --unblock %s is not in %s's blocked_by, which is currently %s; nothing written",
					verb, dep, req.ID, renderIDList(current))
			}
		}
		next := make([]string, 0, len(current)+len(add))
		for _, dep := range current {
			if !containsString(remove, dep) {
				next = append(next, dep)
			}
		}
		// Then block: an id already present collapses rather than duplicating,
		// which is what makes a repeated link idempotent.
		for _, dep := range add {
			if !containsString(next, dep) {
				next = append(next, dep)
			}
		}
		newContent, err := setListField(content, "blocked_by", next)
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
		if err := validateInvariants(newFM, status, src); err != nil {
			return err
		}
		// In place, atomic — the file keeps its status directory. The write
		// happens under the same lock as the re-read, so no checksum window
		// exists between them.
		if err := fsutil.WriteFileAtomicPreserveMode(src, []byte(newContent)); err != nil {
			return err
		}
		result = LinkResult{ID: req.ID, Path: src, BlockedBy: next}
		return nil
	})
	if err != nil {
		return LinkResult{}, err
	}
	// Machine output carries a repo-relative locator, never an absolute
	// developer-identity path (iss-81).
	result.Path = fsutil.RepoRel(repoRoot, result.Path)
	return result, nil
}

// renderIDList spells a blocked_by list the way the record does — `[iss-1,
// iss-2]` — or `(none)` when the record carries no edge, so the refusal that
// names the current list names it in the shape the reader will see in the file.
func renderIDList(ids []string) string {
	if len(ids) == 0 {
		return "(none)"
	}
	return "[" + strings.Join(ids, ", ") + "]"
}
