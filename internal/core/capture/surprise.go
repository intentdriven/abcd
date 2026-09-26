package capture

// The surprise verb (itd-2609020625400194, spc-2609020626040342).
//
// A surprise is something the researcher did not expect, recorded as its own
// act and its own record: surprises/srp-N.md, keyed by `occasioned_by` to the
// reading item, admission or disposition that occasioned it, with the surprise
// itself as the body (spc-67). No disposition file is opened for writing on this
// path, which is what "never a field on a disposition" means in code.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/readingitem"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// SurpriseRequest records one surprise.
type SurpriseRequest struct {
	RepoRoot   string
	IssuesRoot string
	// OccasionedBy is the record that occasioned it: an rdi-N, adm-N or dsp-N
	// this ledger holds, and nothing else.
	OccasionedBy string
	// Text is the surprise itself, held to the grounds floor.
	Text string
}

// SurpriseResult is the outcome of a successful Surprise.
type SurpriseResult struct {
	ID           string `json:"id"`
	OccasionedBy string `json:"occasioned_by"`
	Path         string `json:"path"`
	Redacted     int    `json:"redacted,omitempty"`
	Degraded     string `json:"redaction_degraded,omitempty"`
}

// occasionFamilies maps issueschema's one declaration of the families a
// surprise may be occasioned by onto the leaf resolver's names.
func occasionFamilies() []readingitem.Family {
	out := make([]readingitem.Family, 0, len(issueschema.SurpriseOccasionFamilies))
	for _, f := range issueschema.SurpriseOccasionFamilies {
		out = append(out, readingitem.Family(f))
	}
	return out
}

// Surprise writes one surprise entry. It refuses, writing nothing, an occasion
// that is not a handle of the three families or that resolves to nothing, and a
// text below the substance floor.
func Surprise(req SurpriseRequest) (SurpriseResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return SurpriseResult{}, err
	}
	occasion := req.OccasionedBy
	if !issueschema.ValidSurpriseOccasion(occasion) {
		return SurpriseResult{}, fmt.Errorf("%w: --occasioned-by %q is not a handle of %s; a surprise is keyed to the record that occasioned it, never to prose (nothing written)",
			ErrMalformedFrontmatter, occasion, occasionFamilyList())
	}
	text, redacted, degraded, err := requireFreeGrounds(repoRoot, "surprise", req.Text)
	if err != nil {
		return SurpriseResult{}, err
	}
	// The occasion is resolved before anything is minted, and again under the
	// lock, where the write is decided.
	if err := resolveSurpriseOccasion(repoRoot, occasion); err != nil {
		return SurpriseResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return SurpriseResult{}, err
	}

	result := SurpriseResult{OccasionedBy: occasion, Redacted: redacted, Degraded: degraded}
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		if err := resolveSurpriseOccasion(repoRoot, occasion); err != nil {
			return err
		}
		id, err := minter.Mint(issueschema.SurpriseFamily)
		if err != nil {
			return err
		}
		fields, fm := surpriseFields(id, occasion)
		if err := validateSurpriseStrict(fm); err != nil {
			return err
		}
		content, err := buildIssueText(fields, text)
		if err != nil {
			return err
		}
		dir := filepath.Join(issuesRoot, issueschema.SurprisesDir)
		if err := safeMkdirLeaf(dir); err != nil {
			return err
		}
		path := filepath.Join(dir, id+".md")
		if err := refuseExistingRecord(path, id); err != nil {
			return err
		}
		if err := writeReadingRecord(ledgerBase(repoRoot, issuesRoot), path, []byte(content)); err != nil {
			return err
		}
		result.ID, result.Path = id, fsutil.RepoRel(repoRoot, path)
		return nil
	})
	if err != nil {
		return SurpriseResult{}, err
	}
	return result, nil
}

// resolveSurpriseOccasion resolves the occasion through the one occasion
// resolver the reading chain shares, refusing one that names nothing.
func resolveSurpriseOccasion(repoRoot, occasion string) error {
	if _, err := readingitem.ResolveOccasion(repoRoot, occasion, occasionFamilies()...); err != nil {
		return fmt.Errorf("--occasioned-by %s does not resolve: %w (nothing written)", occasion, wrapLocatorErr(err))
	}
	return nil
}

// occasionFamilyList renders the admitted families for a refusal.
func occasionFamilyList() string {
	names := make([]string, 0, len(issueschema.SurpriseOccasionFamilies))
	for _, f := range issueschema.SurpriseOccasionFamilies {
		names = append(names, f+"-N")
	}
	return strings.Join(names, ", ")
}

// surpriseFields assembles one surprise's frontmatter in the schema's order.
func surpriseFields(id, occasion string) ([]kv, map[string]any) {
	fields := []kv{
		{"schema_version", 1},
		{"id", id},
		{"occasioned_by", occasion},
	}
	fm := map[string]any{}
	for _, f := range fields {
		fm[f.key] = f.val
	}
	return fields, fm
}

// validateSurpriseStrict holds a surprise to issueschema's declaration of the
// family: its closed key set, every required key present, and the occasion a
// handle of one of the three families.
func validateSurpriseStrict(fm map[string]any) error {
	if err := requireSchemaVersion(fm); err != nil {
		return err
	}
	for k := range fm {
		if !issueschema.SurpriseKnown[k] {
			return fmt.Errorf("%w: unknown property %q on a surprise", ErrMalformedFrontmatter, k)
		}
	}
	for _, key := range issueschema.SurpriseRequired[1:] {
		if err := requireNonBlankString(fm, key); err != nil {
			return err
		}
	}
	if id := asString(fm["id"]); !recordid.ValidSurpriseID(id) {
		return fmt.Errorf("%w: id %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, id, issueschema.SurpriseFamily)
	}
	if occ := asString(fm["occasioned_by"]); !issueschema.ValidSurpriseOccasion(occ) {
		return fmt.Errorf("%w: occasioned_by %q is not a handle of %s", ErrMalformedFrontmatter, occ, occasionFamilyList())
	}
	return nil
}
