package capture

// The admission verb (itd-2609020625400194, spc-2609020626040342).
//
// At the widening position acceptance IS admission (itd-180's ruling), and an
// admission carries its grounds in a record of its own (itd-189's schema). So
// `capture admit` writes the two as ONE act under the ledger lock: the item's
// `accepted` disposition, carrying the grounds, and the admission record that
// joins the item to its run's candidate set, both carrying the same folded
// text. Where an `accepted` disposition already stands, the admission is written
// alone, and only on the ground that disposition states — so the two records
// can never give two reasons for one act by any path.
//
// The ruled order, characterise first and admit second, is the shared
// disposition writer's gate (requireCharacterised); the admission-alone branch
// meets the same gate, because it admits.

import (
	"fmt"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// AdmitRequest admits one widening item into its run's candidate set.
type AdmitRequest struct {
	RepoRoot   string
	IssuesRoot string
	// Item is the widening proposal admitted (rdi-N).
	Item string
	// Grounds is why it is admitted: free text with no `<token>:` prefix, held to
	// the substance floor every grounds primitive applies.
	Grounds string
}

// AdmitResult is the outcome of a successful Admit.
type AdmitResult struct {
	// Admission is the admission record's id (adm-N) and Path its repo-relative
	// path.
	Admission string `json:"admission"`
	Path      string `json:"path"`
	// Disposition is the item's `accepted` disposition — the one this act wrote,
	// or the one already standing — and DispositionPath its repo-relative path.
	Disposition     string `json:"disposition"`
	DispositionPath string `json:"disposition_path"`
	// DispositionWritten reports whether this act wrote the disposition (true)
	// or found an `accepted` one standing and wrote the admission alone (false).
	DispositionWritten bool   `json:"disposition_written"`
	Item               string `json:"item"`
	Run                string `json:"run"`
	Grounds            string `json:"grounds"`
	Redacted           int    `json:"redacted,omitempty"`
	Degraded           string `json:"redaction_degraded,omitempty"`
}

// Admit records one admission as one act. It refuses, writing nothing: an item
// that is not a widening item; an item already admitted; a standing disposition
// in any state but `accepted`, or an `accepted` one stating another ground; a
// contested or cyclic disposition set; a ground below the substance floor; and
// any admission before a committed comparative run names the item's run.
func Admit(req AdmitRequest) (AdmitResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return AdmitResult{}, err
	}
	if !recordid.ValidReadingItemID(req.Item) {
		return AdmitResult{}, fmt.Errorf("%w: item %q does not match ^%s-[0-9]+$",
			ErrMalformedFrontmatter, req.Item, issueschema.ReadingItemFamily)
	}
	// The ground is settled first, before the ledger is touched at all: a refusal
	// here writes nothing, and says so.
	ground, redacted, degraded, err := requireFreeGrounds(repoRoot, "admit", req.Grounds)
	if err != nil {
		return AdmitResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return AdmitResult{}, err
	}
	// The pre-flight: a request that cannot be an admission refuses before the
	// lock. Everything that decides the write is read again under it.
	head, err := readItemHead(issuesRoot, req.Item)
	if err != nil {
		return AdmitResult{}, err
	}
	if err := requireWidening(head); err != nil {
		return AdmitResult{}, err
	}

	result := AdmitResult{Item: req.Item, Grounds: ground, Redacted: redacted, Degraded: degraded}
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		head, err := readItemHead(issuesRoot, req.Item)
		if err != nil {
			return err
		}
		if err := requireWidening(head); err != nil {
			return err
		}
		result.Run = head.run

		// The one admitted-proposal probe, asked once: the admissions keyed on
		// the (run, proposal) pair, and the dispositions standing over the item.
		fate, err := itemFateIn(issuesRoot, head.run, head.item)
		if err != nil {
			return err
		}
		if fate.Cyclic {
			return fmt.Errorf("%w: every disposition of %s is superseded by another, so none stands — a supersession cycle only a hand edit can repair; nothing written",
				ErrInvariantViolation, head.item)
		}
		if len(fate.Admissions) > 0 {
			return fmt.Errorf("%w: %s is already admitted into %s's candidate set (%s); an admission is written once, and nothing is written",
				ErrInvariantViolation, head.item, head.run, renderList(fate.Admissions))
		}

		var dispPath string
		switch len(fate.Dispositions) {
		case 0:
			// Both records: the disposition through the shared writer, which
			// holds the ordering gate.
			written, err := writeDispositionLocked(repoRoot, issuesRoot, head, DispositionRequest{
				Item: head.item, State: issueschema.DispositionAccepted, Grounds: ground,
			})
			if err != nil {
				return err
			}
			result.Disposition, result.DispositionWritten, dispPath = written.id, true, written.path
		case 1:
			standingID := fate.Dispositions[0]
			path, err := requireStandingAcceptance(issuesRoot, head.item, standingID, ground)
			if err != nil {
				return err
			}
			// The admission-alone branch admits, so it meets the same gate.
			if err := requireCharacterised(repoRoot, head); err != nil {
				return err
			}
			result.Disposition, dispPath = standingID, path
		default:
			return fmt.Errorf("%w: %s has %d standing answers (%s), so which one is in force is a judgement the ledger does not contain, and an admission cannot stand on it; write `supersedes_disposition` into the records that are no longer meant to stand, by hand, until exactly one does (nothing written)",
				ErrInvariantViolation, head.item, len(fate.Dispositions), renderList(fate.Dispositions))
		}
		result.DispositionPath = fsutil.RepoRel(repoRoot, dispPath)

		admID, admPath, err := writeAdmissionLocked(repoRoot, issuesRoot, head, ground)
		if err != nil {
			if !result.DispositionWritten {
				return err
			}
			// Both or neither: the disposition this act wrote is removed, on the
			// shape writePair in core/reading carries. If the removal fails too,
			// both failures are named, because the ledger then holds an
			// acceptance whose admission never landed and the caller has to know.
			if rmErr := removeContained(ledgerBase(repoRoot, issuesRoot), dispPath); rmErr != nil {
				return fmt.Errorf("%w; and removing the disposition %s this act wrote also failed (%v), so it stands without its admission — remove it by hand",
					err, result.Disposition, rmErr)
			}
			return fmt.Errorf("%w; the disposition %s this act wrote was removed, so nothing is written", err, result.Disposition)
		}
		result.Admission, result.Path = admID, fsutil.RepoRel(repoRoot, admPath)
		return nil
	})
	if err != nil {
		return AdmitResult{}, err
	}
	return result, nil
}

// requireWidening refuses an item at any position but widening, by name:
// admission is that position's warm act alone.
func requireWidening(head itemHead) error {
	if head.position == issueschema.PositionWidening {
		return nil
	}
	return fmt.Errorf("%w: %s is a %s item, and admission is the %s position's act alone — answer it with `abcd capture disposition` instead (nothing written)",
		ErrInvariantViolation, head.item, head.position, issueschema.PositionWidening)
}

// requireStandingAcceptance reads the one standing disposition of item and
// returns its path when it is an `accepted` disposition stating ground. Any
// other state refuses naming the disposition and its state; another ground
// refuses naming both texts, so the disposition and the admission cannot state
// two reasons for one act.
func requireStandingAcceptance(issuesRoot, item, id, ground string) (string, error) {
	path := filepath.Join(issuesRoot, issueschema.DispositionsDir, item, id+".md")
	content, err := readRecordGuarded(path)
	if err != nil {
		return "", err
	}
	fm, _, err := parseFrontmatterAndBody(content)
	if err != nil {
		return "", fmt.Errorf("%w: the standing disposition %s of %s does not parse: %v (nothing written)",
			ErrMalformedFrontmatter, id, item, err)
	}
	if state := asString(fm["state"]); state != issueschema.DispositionAccepted {
		return "", fmt.Errorf("%w: %s carries the standing disposition %s in the %q state; an admission stands only on `%s`, so supersede %s first if the answer has changed (nothing written)",
			ErrInvariantViolation, item, id, state, issueschema.DispositionAccepted, id)
	}
	if standing := grounds.Fold(asString(fm["disposition_grounds"])); standing != ground {
		return "", fmt.Errorf("%w: %s's standing acceptance %s states the ground %q, and this admission states %q; one act carries one ground, so admit on the standing ground or supersede %s (nothing written)",
			ErrInvariantViolation, item, id, standing, ground, id)
	}
	return path, nil
}

// writeAdmissionLocked mints and writes one admission record under
// admissions/<run>/. It must be called under the ledger lock, with ground
// already redacted, folded and held to the floor.
func writeAdmissionLocked(repoRoot, issuesRoot string, head itemHead, ground string) (string, string, error) {
	id, err := minter.Mint(issueschema.AdmissionFamily)
	if err != nil {
		return "", "", err
	}
	fields, fm := admissionFields(id, head.run, head.item, ground)
	if err := validateAdmissionStrict(fm); err != nil {
		return "", "", err
	}
	content, err := buildIssueText(fields, "")
	if err != nil {
		return "", "", err
	}
	if err := ensureFamilyDir(issuesRoot, issueschema.AdmissionsDir, head.run); err != nil {
		return "", "", err
	}
	path := filepath.Join(issuesRoot, issueschema.AdmissionsDir, head.run, id+".md")
	if err := refuseExistingRecord(path, id); err != nil {
		return "", "", err
	}
	if err := writeReadingRecord(ledgerBase(repoRoot, issuesRoot), path, []byte(content)); err != nil {
		return "", "", err
	}
	return id, path, nil
}

// admissionFields assembles one admission's frontmatter in the schema's order.
func admissionFields(id, run, proposal, ground string) ([]kv, map[string]any) {
	fields := []kv{
		{"schema_version", 1},
		{"id", id},
		{"run", run},
		{"proposal", proposal},
		{"grounds", ground},
	}
	fm := map[string]any{}
	for _, f := range fields {
		fm[f.key] = f.val
	}
	return fields, fm
}

// validateAdmissionStrict holds an admission to issueschema's one declaration of
// the family: its closed key set, every required key present and non-blank, and
// each handle well-formed.
func validateAdmissionStrict(fm map[string]any) error {
	if err := requireSchemaVersion(fm); err != nil {
		return err
	}
	for k := range fm {
		if !issueschema.AdmissionKnown[k] {
			return fmt.Errorf("%w: unknown property %q on an admission", ErrMalformedFrontmatter, k)
		}
	}
	for _, key := range issueschema.AdmissionRequired[1:] {
		if err := requireNonBlankString(fm, key); err != nil {
			return err
		}
	}
	if id := asString(fm["id"]); !recordid.ValidAdmissionID(id) {
		return fmt.Errorf("%w: id %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, id, issueschema.AdmissionFamily)
	}
	if run := asString(fm["run"]); !recordid.ValidReadingRunID(run) {
		return fmt.Errorf("%w: run %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, run, issueschema.ReadingRunFamily)
	}
	if p := asString(fm["proposal"]); !recordid.ValidReadingItemID(p) {
		return fmt.Errorf("%w: proposal %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, p, issueschema.ReadingItemFamily)
	}
	return nil
}

// requireFreeGrounds settles a free-text ground — one with no `<token>:` prefix,
// the shape the admission's `grounds` and the disposition's
// `disposition_grounds` hold — against the floor every grounds primitive
// applies: redacted first, so no rewritten span reaches a value the floor has
// passed, then folded, then held to grounds.ValidateText, which refuses the
// empty, whitespace, control-character, too-short and vocabulary-only texts.
func requireFreeGrounds(repoRoot, verb, raw string) (string, int, string, error) {
	red, n, degraded := redactLedgerText(repoRoot, raw)
	folded := grounds.Fold(red)
	if err := grounds.ValidateText(folded); err != nil {
		return "", 0, "", fmt.Errorf("%s: %w: %v; nothing written", verb, ErrGroundsRefused, err)
	}
	return termsafe.EncodeHiddenRunes(folded), n, degraded, nil
}
