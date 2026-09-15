package intent

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// Check names, in the fixed order Ready always reports them.
const (
	CheckBucket             = "bucket"
	CheckAcceptanceCriteria = "acceptance_criteria"
	CheckMechanismClaim     = "mechanism_claim"
	CheckScopeConditions    = "scope_conditions"
	CheckSpecLink           = "spec_link"
	CheckSpecBody           = "spec_body"
	CheckGrounds            = "grounds"
)

// ReadyCheck is one finding of the implement-readiness gate.
type ReadyCheck struct {
	Name string `json:"name"` // bucket | acceptance_criteria | mechanism_claim | scope_conditions | spec_link | spec_body | grounds
	OK   bool   `json:"ok"`
	// Advisory marks a check that REPORTS and never gates: its verdict and its
	// remedy are shown, and Ready ignores it. The two claim checks and the
	// grounds check are advisory until the rethink of the reading work settles
	// what a human is asked for at this gate (iss-2609091009111294); the four
	// structural checks are not.
	Advisory bool   `json:"advisory,omitempty"`
	Detail   string `json:"detail"`           // why it passed or failed
	Remedy   string `json:"remedy,omitempty"` // the exact next command/action when !OK
}

// ReadyResult is the structured readiness verdict for one intent: may this
// intent be implemented now? Every check is always evaluated and reported, so a
// surface presents the full picture rather than the first failure.
type ReadyResult struct {
	IntentID string `json:"intent_id"`
	Path     string `json:"path"`   // repo-relative intent path
	Bucket   string `json:"bucket"` // directory-as-truth state
	// SpecID is the spec this gate judged: the intent's own spec_id, or — when
	// the intent owns more than one spec and the spec_id names a closed one — the
	// open spec that realises the remainder (adr-2609151513118583).
	SpecID string       `json:"spec_id"`
	Ready  bool         `json:"ready"`
	Checks []ReadyCheck `json:"checks"` // always exactly 7, fixed order
	// Conditions is the record's scope conditions with their minted identities —
	// the observable surface the identity criteria assert against. Empty for a
	// record whose conditions are absent, or recorded as the nullity token.
	Conditions []ScopeCondition `json:"conditions"`
}

// Ready reports whether an intent is ready to implement: planned
// (directory-as-truth), carrying enumerable Acceptance Criteria, linked
// bidirectionally to a spec, and that spec's body written past its minted stub.
// It is a read-only reporter — the machine-checkable form of the run protocol's
// "is this item ready?" question — and never mutates the store. The mechanism,
// scope-condition and grounds checks are evaluated and reported beside the four
// structural ones but are advisory: a failing one never withholds readiness.
//
// "Not ready" is a result, never an error: error is reserved for structural
// faults (malformed id, unknown intent, unreadable record, store load failure),
// so a surface can map result vs error to distinct exit codes.
func Ready(repoRoot, intentID string) (ReadyResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return ReadyResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return ReadyResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return ReadyResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return ReadyResult{}, err
	}
	data, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
	if err != nil {
		return ReadyResult{}, err
	}
	content := string(data)
	acCount := countAcceptanceCriteria(content)
	claims := ParseClaims(content)

	res := ReadyResult{
		IntentID: it.ID,
		Path:     it.Path,
		Bucket:   it.Bucket,
		SpecID:   it.SpecID,

		Conditions: claims.Conditions,
	}
	res.Checks = append(res.Checks, bucketCheck(it, acCount, content))
	res.Checks = append(res.Checks, acCheck(acCount))
	res.Checks = append(res.Checks, mechanismCheck(it, claims))
	res.Checks = append(res.Checks, scopeConditionsCheck(it, claims))
	linkOK, linked := specLinkCheck(it, store)
	res.Checks = append(res.Checks, linkOK)
	// The spec this gate JUDGED, which for an intent owning more than one spec is
	// the open one rather than the spec_id the record was planned with — so a
	// surface that prints "when done, `abcd spec close <SpecID>`" names the spec
	// the reader is about to finish (adr-2609151513118583).
	if linked.ID != "" {
		res.SpecID = linked.ID
	}
	bodyCheck, err := specBodyCheck(repoRoot, it, linked, linkOK.OK)
	if err != nil {
		return ReadyResult{}, err
	}
	res.Checks = append(res.Checks, bodyCheck)
	res.Checks = append(res.Checks, groundsCheck(it, content))

	res.Ready = true
	for _, c := range res.Checks {
		if !c.OK && !c.Advisory {
			res.Ready = false
			break
		}
	}
	return res, nil
}

// bucketCheck reports the lifecycle-state gate: only a planned intent may be
// implemented. For a draft the remedy names the exact route to planned —
// through the maintainer's sign-off, via the planning interview when the
// Acceptance Criteria are not yet enumerable. Terminal buckets carry no remedy:
// there is nothing to fix, the answer is simply no.
func bucketCheck(it Intent, acCount int, content string) ReadyCheck {
	c := ReadyCheck{Name: CheckBucket}
	switch it.Bucket {
	case BucketPlanned:
		c.OK = true
		c.Detail = fmt.Sprintf("%s is planned", it.ID)
	case BucketDrafts:
		c.Detail = fmt.Sprintf("%s is a draft — an intent that is not planned cannot be implemented", it.ID)
		if acCount > 0 {
			c.Remedy = fmt.Sprintf("confirm the Acceptance Criteria with the maintainer, then run `abcd intent plan %s`", it.ID)
		} else {
			c.Remedy = fmt.Sprintf("run the planning interview (/abcd:intent) to write and confirm Acceptance Criteria, then run `abcd intent plan %s`", it.ID)
		}
	case BucketShipped:
		c.Detail = fmt.Sprintf("%s is already shipped — nothing left to implement", it.ID)
	case BucketDisciplines:
		c.Detail = fmt.Sprintf("%s is a discipline — it imposes gates on other work and is never implemented via a spec of its own", it.ID)
	case BucketSuperseded:
		c.Detail = fmt.Sprintf("%s is superseded — implement the successor instead", it.ID)
		if by := frontmatter.Fields(strings.Split(content, "\n"))["superseded_by"].Value; !frontmatter.IsNull(by) {
			c.Detail += " (superseded_by: " + by + ")"
		}
	default:
		c.Detail = fmt.Sprintf("%s is in unknown bucket %q", it.ID, it.Bucket)
	}
	return c
}

// acCheck reports the itd-1 discipline through the same parser Plan and the
// fidelity review use (countAcceptanceCriteria), so the three gates can never
// disagree about what counts as a criterion.
func acCheck(acCount int) ReadyCheck {
	c := ReadyCheck{Name: CheckAcceptanceCriteria}
	if acCount > 0 {
		c.OK = true
		c.Detail = fmt.Sprintf("%d top-level bullet(s) in '## Acceptance Criteria'", acCount)
	} else {
		c.Detail = "no top-level bullets in '## Acceptance Criteria' (itd-1 discipline)"
		c.Remedy = "add at least one Given-When-Then bullet — the planning interview walks this with the maintainer"
	}
	return c
}

// mechanismCheck reports the mechanism claim: prompted and nullable, so an
// absent section passes (the claim was never carried) and the exact nullity
// token passes as a claim considered and declined. A heading with nothing under
// it is neither, and is the section's one fault.
func mechanismCheck(it Intent, claims Claims) ReadyCheck {
	c := ReadyCheck{Name: CheckMechanismClaim, OK: true, Advisory: true}
	if detail, exempt := claimCheckExemption(it); exempt {
		c.Detail = detail
		return c
	}
	if claims.MechanismPrompt {
		c.Detail = "the '## Mechanism' prompt is unanswered — no claim recorded (prompted, not required)"
		return c
	}
	switch claims.Mechanism {
	case ClaimNullity:
		c.Detail = "mechanism claim declined (nullity recorded)"
	case ClaimStated:
		c.Detail = "mechanism claim stated"
	case ClaimEmpty:
		c.OK = false
		c.Detail = "'## Mechanism' is present but empty — neither a claim nor a recorded decline"
		c.Remedy = "write the falsifiable claim (\"we expect X because Y\") under '## Mechanism', or record the exact token `" + NullityToken + "` alone on its line to decline it"
	default:
		c.Detail = "no '## Mechanism' section — the mechanism claim is prompted, not required"
	}
	return c
}

// scopeConditionsCheck reports the context claim: mandatory, with the nullity
// token as the explicit "none stated", and never left blank. A stated section
// must enumerate its conditions as top-level bullets, because a bullet is what
// carries an identity — the same rule acCheck already holds the criteria to.
// Each condition must carry exactly one identity, and no two may share one.
func scopeConditionsCheck(it Intent, claims Claims) ReadyCheck {
	c := ReadyCheck{Name: CheckScopeConditions, OK: true, Advisory: true}
	if detail, exempt := claimCheckExemption(it); exempt {
		c.Detail = detail
		return c
	}
	// The structural faults first — BEFORE the claim-state switch. Each is a thing
	// the STAMP refuses, so the gate has to name it rather than fall through to a
	// remedy that cannot run; and judging them after the switch let a first
	// section reading `None stated.` return OK while a second heading below it
	// carried real, unidentified bullets (iss-2608300259321329).
	if claims.ConditionsDuplicated {
		c.OK = false
		c.Detail = "more than one '## Scope Conditions' heading — which one is the section is undecidable"
		c.Remedy = "merge the duplicated '## Scope Conditions' sections into one"
		return c
	}
	if claims.ConditionsFenced {
		c.OK = false
		c.Detail = "'## Scope Conditions' contains a fenced block — a bullet inside it is an example, not a condition"
		c.Remedy = "move the fenced example out of '## Scope Conditions'"
		return c
	}
	if claims.ConditionsCommented {
		c.OK = false
		c.Detail = "'## Scope Conditions' contains an HTML comment — a bullet parked inside one is not a live condition"
		c.Remedy = "delete the commented-out block from '## Scope Conditions', or move it out of the section"
		return c
	}
	if claims.ConditionsPrompt {
		c.OK = false
		c.Detail = "the '## Scope Conditions' prompt is unanswered — the context claim is unrecorded"
		c.Remedy = scopeConditionsRemedy
		return c
	}
	switch claims.ConditionsState {
	case ClaimNullity:
		c.Detail = "scope conditions declined (nullity recorded)"
		return c
	case ClaimEmpty:
		c.OK = false
		c.Detail = "'## Scope Conditions' is present but empty — write the conditions or the nullity token"
		c.Remedy = scopeConditionsRemedy
		return c
	case ClaimAbsent:
		c.OK = false
		c.Detail = "no '## Scope Conditions' section — the context claim is unrecorded"
		c.Remedy = scopeConditionsRemedy
		return c
	}
	if len(claims.Conditions) == 0 {
		c.OK = false
		c.Detail = "'## Scope Conditions' carries prose but no top-level bullet — a condition without a bullet has nothing to identify"
		c.Remedy = scopeConditionsRemedy
		return c
	}
	if bad := MalformedMarkerOrdinals(claims.Conditions); len(bad) > 0 {
		c.OK = false
		c.Detail = fmt.Sprintf("condition(s) %s carry a malformed identity marker", joinInts(bad))
		c.Remedy = "delete the malformed `<!-- cond: … -->` text; a real identity is minted by `abcd intent plan`, never hand-typed"
		return c
	}
	if multi := MultiplyMarkedConditions(claims.Conditions); len(multi) > 0 {
		c.OK = false
		c.Detail = fmt.Sprintf("condition(s) %s carry more than one identity", joinInts(multi))
		c.Remedy = "delete the surplus `<!-- cond: … -->` marker(s), leaving each condition exactly one"
		return c
	}
	if unmarked := UnmarkedConditionOrdinals(claims.Conditions); len(unmarked) > 0 {
		c.OK = false
		c.Detail = fmt.Sprintf("condition(s) %s carry no identity marker", joinInts(unmarked))
		c.Remedy = fmt.Sprintf("run `abcd intent plan %s` — the write-capable verb stamps every unmarked condition; markers are never hand-typed", it.ID)
		return c
	}
	if dupes := DuplicateConditionIDs(claims.Conditions); len(dupes) > 0 {
		c.OK = false
		c.Detail = fmt.Sprintf("identity %s is carried by more than one condition", strings.Join(dupes, ", "))
		c.Remedy = "delete the duplicated marker from the condition that copied it, then re-stamp it with `abcd intent plan`"
		return c
	}
	c.Detail = fmt.Sprintf("%d scope condition(s), each identified", len(claims.Conditions))
	return c
}

// groundsCheck reports the record's recorded grounds — the reasoning behind
// what is being pursued, at the conjecture granularity the ADR family's
// Alternatives Considered does not reach (spc-57). It is reported LAST, because
// it is the only check about why the work is being done at all rather than about
// whether the record is well formed.
//
// It REFUSES, forward-only, and the flip is not free: measured at the branch
// tip, 10 of the 66 planned/ records carry an entry, 56 fail this check, and 36
// of those were READY before the promotion and are NOT READY after it. Records
// planned before the argument existed fail it, and that is the finding, not a
// defect: an unrecorded conjecture is exactly what this reports, and each of the
// 36 records its grounds when it is next picked up — the moment the conjecture
// is still known, which is the only moment it can be recorded honestly.
//
// Only a well-formed entry counts. Prose under the heading is prose: putting a
// gate verdict on a sentence somebody wrote for a human is a judgement no parser
// can make, and the substance floor is deliberately the whole of the machine's
// claim.
func groundsCheck(it Intent, content string) ReadyCheck {
	c := ReadyCheck{Name: CheckGrounds, OK: true, Advisory: true}
	if detail, exempt := groundsCheckExemption(it); exempt {
		c.Detail = detail
		return c
	}
	entries := ParseGrounds(content)
	if len(entries) == 0 {
		c.OK = false
		c.Detail = "no recorded grounds — the conjecture behind pursuing " + it.ID + " is unrecorded"
		c.Remedy = groundsRemedy(it.ID)
		return c
	}
	last := entries[len(entries)-1]
	c.Detail = fmt.Sprintf("%d recorded ground(s), most recent %s", len(entries), last.Token)
	return c
}

// groundsRemedy is the one spelling of how a ground is recorded, so the gate and
// the surfaces name the same command and the same closed vocabulary. It asks for
// the expectation AND its falsifier, because that is the difference between a
// conjecture and a restatement of the decision — the part no parser can check.
func groundsRemedy(intentID string) string {
	return "run `abcd intent ready " + intentID +
		" --grounds \"" + string(grounds.Pursued) + ": <what is expected, and what would show it wrong>\"` " +
		"(vocabulary: " + grounds.ProseList() + ") — name the conjecture being acted on, not the route taken"
}

// claimCheckExemption reports the buckets where a claim check has nothing to
// ask for, and the detail that says why. Discipline records carry no claim
// sections at all; a shipped or superseded record is never backfilled (spc-55
// rules retro-fitting out of scope — an absent stamp is information), so
// printing a write-the-section remedy at one would name work nobody may do
// (iss-2608300210588414). In every case the bucket check has already settled
// the verdict.
func claimCheckExemption(it Intent) (string, bool) {
	switch it.Bucket {
	case BucketDisciplines:
		return disciplineClaimExemption, true
	case BucketShipped, BucketSuperseded:
		return "not applicable — a " + it.Bucket + " record's claims are never backfilled", true
	}
	return "", false
}

// groundsCheckExemption reports the buckets where the GROUNDS check has nothing
// to ask for. The buckets are claimCheckExemption's, and for the same reasons,
// but the detail names grounds rather than claims: reusing the claim string made
// the grounds row of a shipped record report "a shipped record's CLAIMS are
// never backfilled", which is a true sentence about a check that is not the one
// reporting it (iss-2608301657350399, the detail-string class of the resolved
// iss-2608300210588414).
func groundsCheckExemption(it Intent) (string, bool) {
	switch it.Bucket {
	case BucketDisciplines:
		return disciplineGroundsExemption, true
	case BucketShipped, BucketSuperseded:
		return "not applicable — a " + it.Bucket + " record's grounds are never backfilled", true
	}
	return "", false
}

// The strings the claim checks and the grounds check share, so an exemption and
// a remedy read identically wherever they are reported.
const (
	disciplineClaimExemption = "not applicable — discipline records carry no claim sections"
	// A discipline record is exempt for a different reason than a terminal one:
	// it has no conjecture of its own to record, rather than a window that has
	// closed. The two exemption strings say the two different things.
	disciplineGroundsExemption = "not applicable — a discipline record carries no conjecture of its own"
	scopeConditionsRemedy      = "write the conditions this claim holds under as top-level bullets under '## Scope Conditions', or record the exact token `None stated.` alone on its line"
)

// joinInts renders condition ordinals for a finding.
func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}

// specLinkCheck reports the bidirectional intent↔spec link (the same agreement
// Reconcile enforces, here as a report instead of a refusal). It returns the
// linked spec when, and only when, the link holds, so specBodyCheck never reads
// a file the link did not vouch for.
func specLinkCheck(it Intent, store spec.Store) (ReadyCheck, spec.Spec) {
	c := ReadyCheck{Name: CheckSpecLink}
	if frontmatter.IsNull(it.SpecID) {
		if claimer, ok := store.ByIntent(it.ID); ok {
			c.Detail = fmt.Sprintf("spec_id is null but %s claims %s (one-sided link)", claimer.ID, it.ID)
			c.Remedy = fmt.Sprintf("run `abcd intent link %s %s`", it.ID, claimer.ID)
			return c, spec.Spec{}
		}
		c.Detail = "spec_id is null — no spec realises this intent"
		if it.Bucket == BucketDrafts {
			c.Remedy = fmt.Sprintf("planning (`abcd intent plan %s`) mints and links the spec", it.ID)
		} else {
			c.Remedy = fmt.Sprintf("hand-author %s/open/spc-N-<slug>.md with `intent: %s`, then run `abcd intent link`", spec.SpecsRelDir, it.ID)
		}
		return c, spec.Spec{}
	}
	sp, ok := store.Lookup(it.SpecID)
	if !ok {
		c.Detail = fmt.Sprintf("spec_id is %s but no such spec exists in the store", it.SpecID)
		c.Remedy = fmt.Sprintf("restore %s or correct spec_id via `abcd intent link`", it.SpecID)
		return c, spec.Spec{}
	}
	if sp.Intent != it.ID {
		c.Detail = fmt.Sprintf("bidirectional link disagrees: %s names %s, but %s claims %s", it.ID, it.SpecID, sp.ID, sp.Intent)
		c.Remedy = "correct the spec's `intent:` field or the intent's spec_id so both sides agree"
		return c, spec.Spec{}
	}
	c.OK = true
	// An intent owns one or more specs (adr-2609151513118583). The gate reports on
	// the spec that is still being built — the OPEN one — because that is the
	// design record the implementation it is gating builds against; the spec_id
	// names the spec the intent was planned with, which after a partial delivery
	// is the closed one.
	if sp.Status == spec.StatusClosed {
		if open := store.OpenSpecsForIntent(it.ID); len(open) > 0 {
			c.Detail = fmt.Sprintf("linked to %s (bidirectional); %s is closed and %s is the open spec realising %s",
				open[0].ID, sp.ID, open[0].ID, it.ID)
			return c, open[0]
		}
		if it.Bucket == BucketPlanned {
			c.Detail = fmt.Sprintf("linked to %s (bidirectional); note: the spec is closed while the intent is still planned (drift)", sp.ID)
			return c, sp
		}
	}
	c.Detail = fmt.Sprintf("linked to %s (bidirectional)", sp.ID)
	return c, sp
}

// specBodyCheck reports whether the linked spec's body has been written past
// the minted stub — the spec is the design record implementation builds
// against, so an untouched placeholder means there is nothing to build from. A
// read failure on the linked spec is a structural fault, not a finding.
func specBodyCheck(repoRoot string, it Intent, sp spec.Spec, linkOK bool) (ReadyCheck, error) {
	c := ReadyCheck{Name: CheckSpecBody}
	if !linkOK {
		c.Detail = "unchecked — no linked spec"
		return c, nil
	}
	data, err := readRepoFile(filepath.Join(repoRoot, sp.Path), sp.Path)
	if err != nil {
		return ReadyCheck{}, err
	}
	if spec.BodyIsStub(string(data)) {
		c.Detail = fmt.Sprintf("%s still carries the minted _Draft: placeholder — the design record is unwritten", sp.ID)
		c.Remedy = fmt.Sprintf("write the spec body at %s, then re-run `abcd intent ready %s`", sp.Path, it.ID)
		return c, nil
	}
	c.OK = true
	c.Detail = fmt.Sprintf("spec body at %s is written", sp.Path)
	return c, nil
}
