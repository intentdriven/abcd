package intent

// autoplan.go is the record-only half of `abcd build <itd-N> --auto-plan`
// (itd-2609201916151817 decision 6; spc-2609202134338445 step 11): the
// substitute for the product thinker's sign-off that adr-2610310235000654
// records, read from the draft alone. The planning path runs without a human
// only on a draft whose decisions are all recorded, so every row here judges
// what the record already carries: the criteria, the decisions (no open
// question), the claim sections, the grounds and the record's provenance.
// The rows check that the record is complete, not who wrote it. The two adversarial reviews are the loop's (they need
// agents), and the hold and the blockers are the same rows the build reads.
// Every row reads; none writes.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// The auto-plan check names, in the order AutoPlanChecks reports them. The
// open questions, the claim sections, the hold and the blockers are the
// build's own rows under the build's own names.
const (
	AutoPlanCheckDraft         = "draft"
	AutoPlanCheckPlannable     = "plannable"
	AutoPlanCheckCriteria      = "acceptance_criteria"
	AutoPlanCheckOpenQuestions = StartCheckOpenQuestions
	AutoPlanCheckClaimSections = StartCheckClaimSections
	AutoPlanCheckGrounds       = "grounds"
	AutoPlanCheckProvenance    = "criteria_provenance"
	AutoPlanCheckHold          = StartCheckHold
	AutoPlanCheckBlocked       = StartCheckBlocked
)

// ErrDraftChanged is Plan's refusal of a draft whose bytes are not the ones
// PlanOptions.ExpectDigest names, or of a record no longer in drafts/.
var ErrDraftChanged = errors.New("intent: the draft is not the one the caller judged")

// AutoPlanResult is every record-only auto-plan check for one intent.
type AutoPlanResult struct {
	IntentID string
	// Path is the record's repo-relative path and Bucket its folder.
	Path   string
	Bucket string
	// Digest is "sha256:<hex>" of the record's bytes the rows judged: the
	// draft the reviews read and the plan must still find.
	Digest string
	Rows   []StartRow
}

// OK reports whether every row passes.
func (r AutoPlanResult) OK() bool {
	for _, row := range r.Rows {
		if !row.OK {
			return false
		}
	}
	return true
}

// DraftDigest is the digest AutoPlanResult.Digest carries for content.
func DraftDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// PlannedPathOf is the path plan moves the draft at draftRel to, slash-separated.
func PlannedPathOf(draftRel string) string {
	return filepath.ToSlash(filepath.Join(IntentsRelDir, BucketPlanned, filepath.Base(filepath.FromSlash(draftRel))))
}

// AutoPlanChecks runs the record-only auto-plan checks for intentID against
// the checkout at repoRoot. A record that may not be planned unattended is a
// result whose OK is false; an error is a malformed id, an unknown intent or a
// fault in reading the checkout.
//
// The rows, in order:
//
//   - draft: the record is in drafts/ (a planned record has nothing to plan).
//   - plannable: nothing `plan` itself refuses on a draft's own fields: a
//     slug in kebab-case, no `spec_id` already set (a half-planned draft),
//     and a planned form under the record's size cap.
//   - acceptance_criteria: `## Acceptance Criteria` holds at least one
//     top-level bullet, by the parser plan itself applies.
//   - open_questions: no open question, the build's row: every decision is
//     recorded.
//   - claim_sections: the mechanism answered or the section absent, and the
//     scope conditions recorded; a condition with no identity marker yet
//     passes, since plan is the verb that stamps one.
//   - grounds: the most recent ground no run's pick wrote is `pursued:`.
//   - criteria_provenance: the record states where its text came from
//     (`origin`) and how it was produced (`production_mode`), each a value
//     of the closed vocabulary.
//   - hold: no `held:` on the record.
//   - blocked: nothing named in `blocked_by` is unsettled.
func AutoPlanChecks(repoRoot, intentID string) (AutoPlanResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return AutoPlanResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return AutoPlanResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return AutoPlanResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	data, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(it.Path)), it.Path)
	if err != nil {
		return AutoPlanResult{}, err
	}
	content := string(data)
	res := AutoPlanResult{IntentID: it.ID, Path: it.Path, Bucket: it.Bucket, Digest: DraftDigest(data)}
	res.Rows = append(res.Rows, autoPlanDraftRow(it))
	plannable, err := autoPlanPlannableRow(it, content)
	if err != nil {
		return AutoPlanResult{}, err
	}
	res.Rows = append(res.Rows, plannable)
	res.Rows = append(res.Rows, autoPlanCriteriaRow(it.ID, content))
	res.Rows = append(res.Rows, startOpenQuestionsRow(it.ID, content))
	claims, err := autoPlanClaimSectionsRow(it, content)
	if err != nil {
		return AutoPlanResult{}, err
	}
	res.Rows = append(res.Rows, claims)
	res.Rows = append(res.Rows, autoPlanGroundsRow(it.ID, content))
	res.Rows = append(res.Rows, autoPlanProvenanceRow(it.ID, content))
	res.Rows = append(res.Rows, startHoldRow(it))
	blocked, err := startBlockedRow(repoRoot, corpus, it.ID, content)
	if err != nil {
		return AutoPlanResult{}, err
	}
	res.Rows = append(res.Rows, blocked)
	return res, nil
}

// autoPlanDraftRow passes a draft and refuses every other folder.
func autoPlanDraftRow(it Intent) StartRow {
	row := StartRow{Name: AutoPlanCheckDraft}
	switch it.Bucket {
	case BucketDrafts:
		row.OK = true
		row.Detail = it.ID + " is a draft"
	case BucketPlanned:
		row.Detail = it.ID + " is already planned, so there is nothing to plan"
		row.Remedy = "build it without --auto-plan: `abcd build " + it.ID + "`"
	default:
		row.Detail = it.ID + " is in " + it.Bucket + "/, and only a draft is planned"
		row.Remedy = "name a draft intent: `abcd intent` lists them"
	}
	return row
}

// autoPlanPlannableRow refuses a draft `plan` would refuse on its own fields,
// so no review is spent on a draft the plan stage could never plan.
func autoPlanPlannableRow(it Intent, content string) (StartRow, error) {
	row := StartRow{Name: AutoPlanCheckPlannable}
	switch {
	case !slugRe.MatchString(it.Slug):
		row.Detail = fmt.Sprintf("%s has slug %q, and plan takes a kebab-case one", it.ID, clip(it.Slug, groundsDetailRunes))
		row.Remedy = "correct the slug in the record's frontmatter, then run --auto-plan again"
		return row, nil
	case !frontmatter.IsNull(it.SpecID):
		row.Detail = fmt.Sprintf("%s is a draft that already carries spec_id %q (half-planned), which plan refuses", it.ID, clip(it.SpecID, groundsDetailRunes))
		row.Remedy = "a draft carries `spec_id: null`: plan it by hand with `abcd intent plan`, which names what to repair, or set it to null if no spec realises it"
		return row, nil
	}
	probe, err := probeMinter().Mint(specFamily)
	if err != nil {
		return StartRow{}, err
	}
	if err := checkDraftFaceSize(content, it, probe, "", "", it.Path); err != nil {
		row.Detail = err.Error()
		row.Remedy = "shorten the record, then run --auto-plan again"
		return row, nil
	}
	row.OK = true
	row.Detail = "plan takes its slug, its unset spec_id and its size"
	return row, nil
}

// autoPlanCriteriaRow refuses a draft whose criteria are still the seeded
// placeholder: plan refuses it too, and nothing unattended may write them.
func autoPlanCriteriaRow(id, content string) StartRow {
	row := StartRow{Name: AutoPlanCheckCriteria}
	if n := countAcceptanceCriteria(content); n > 0 {
		row.OK = true
		row.Detail = fmt.Sprintf("%d top-level bullet(s) in '## Acceptance Criteria'", n)
		return row
	}
	row.Detail = id + " has no top-level bullet in '## Acceptance Criteria'; criteria are a person's, and nothing unattended writes them"
	row.Remedy = "write and confirm the criteria in the planning interview (/abcd:intent), then plan by hand or run --auto-plan again"
	return row
}

// autoPlanClaimSectionsRow is the build's claim-sections row judged on the
// draft as plan would leave it: the scope conditions' identities are stamped
// in memory with a probe minter first, since plan is the verb that writes
// them, so a condition with no marker yet is not a fault here.
func autoPlanClaimSectionsRow(it Intent, content string) (StartRow, error) {
	stamped, _, err := stampScopeConditions(content, probeMinter())
	if err != nil {
		return StartRow{}, err
	}
	claims := parseClaims(stamped)
	r := ReadyResult{IntentID: it.ID, Checks: []ReadyCheck{mechanismCheck(it, claims), scopeConditionsCheck(it, claims)}}
	return startClaimSectionsRow(r, stamped), nil
}

// autoPlanGroundsRow refuses a draft whose most recent ground no run's pick
// wrote is not `pursued:`: a conjecture deferred or declined is not one to plan
// unattended, and a record with none says nothing about why it is pursued.
func autoPlanGroundsRow(id, content string) StartRow {
	row := StartRow{Name: AutoPlanCheckGrounds}
	var last *grounds.Grounds
	for _, g := range ParseGrounds(content) {
		if !isRunPick(g) {
			g := g
			last = &g
		}
	}
	switch {
	case last == nil:
		row.Detail = id + " records no grounds outside a run's pick: why it is pursued is unrecorded"
		row.Remedy = groundsRemedy(id)
	case last.Token != grounds.Pursued:
		row.Detail = fmt.Sprintf("%s's most recent ground is %s: %s", id, last.Token, clip(last.Text, groundsDetailRunes))
		row.Remedy = "a conjecture deferred or declined is not planned unattended; record `pursued:` grounds with `abcd intent ready " + id + " --grounds \"pursued: <why>\"` if it is pursued now"
	default:
		row.OK = true
		row.Detail = "most recent ground pursued: " + clip(last.Text, groundsDetailRunes)
	}
	return row
}

// autoPlanProvenanceRow refuses a draft that does not say where its text came
// from and how it was produced: the criteria the plan stamps as signed off
// must carry their provenance.
func autoPlanProvenanceRow(id, content string) StartRow {
	row := StartRow{Name: AutoPlanCheckProvenance}
	fields := frontmatter.Fields(strings.Split(content, "\n"))
	var bad []string
	origin, ok := frontmatter.ScalarString(fields[provenance.KeyOrigin].Value)
	if !ok || origin == "" {
		bad = append(bad, "no `"+provenance.KeyOrigin+"`")
	} else if _, err := provenance.ParseOrigin(origin); err != nil {
		bad = append(bad, "an `"+provenance.KeyOrigin+"` outside the vocabulary")
	}
	mode, ok := frontmatter.ScalarString(fields[provenance.KeyProductionMode].Value)
	if !ok || mode == "" {
		bad = append(bad, "no `"+provenance.KeyProductionMode+"`")
	} else if _, err := provenance.ParseMode(mode); err != nil {
		bad = append(bad, "a `"+provenance.KeyProductionMode+"` outside the vocabulary")
	}
	if len(bad) > 0 {
		row.Detail = id + " carries " + strings.Join(bad, " and ")
		row.Remedy = "state the record's `origin` and `production_mode` in its frontmatter (record-lint names the vocabulary)"
		return row
	}
	row.OK = true
	row.Detail = "origin " + origin + ", production mode " + mode
	return row
}
