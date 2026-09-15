---
id: itd-181
slug: a-shipped-intent-s-scope-conditions-are-dispositioned-by-the
spec_id: spc-59
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# A shipped intent's scope conditions are dispositioned by the fidelity verdict — what was assumed ex ante and what survived are recorded as different things

## Press Release

> **The difference between what an intent assumed and what held is itself a
> finding.** When the fidelity verdict is ingested for a shipped intent,
> every scope condition — by its persistent identity, not its wording —
> receives one of four values: `survived`, `narrowed`, `falsified`,
> `untested`. A narrowing is stated, never implied by silently changed
> text. Later work inherits only what held.

## What's In Scope

- The disposition surface keyed to scope-condition identity, populated at
  verdict ingest; exercised by the fidelity verdict only for now.

## What's Out of Scope

- Passing dispositions to a reading — warm; excluded by the assembler.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a shipped intent with scope conditions, **when** the fidelity
  verdict is ingested, **then** every condition carries one of the four
  values and none is left absent.
- **Given** a narrowed condition, **when** it is recorded, **then** the
  narrowing is stated rather than implied by a changed text.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-fef217884c57 -->
Fidelity review — receipt rcp-fef217884c57 (verifier abcd:intent-auditor claude-fable-5).

Provenance: abcd:intent-auditor@claude-fable-5 · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:8344ad88aa54b9d39ea9bf0615f8c10cb284cad04e2bead2a319c20c3565d340
Input attestations: diff:21461988^1...21461988^2@sha256:b804a02d7b4f218fca5f420d8af9c6b50c7df28a9bb2706367b8cee4382c769a;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: validateConditionDispositions runs inside validateVerdict on every ingest, resolves the record's cond- identities via ParseClaims, refuses out-of-enum values, refuses partial coverage (len(seen) != len(known)), and the dead-letter path records every condition untested; proven by TestIngestAppliesAllFourDispositions, TestIngestRefusesPartialConditionCoverage and TestDeadLetterRecordsConditionsUntested (go test passes). Concern: no shipped intent on this base carries a scope condition (itd-181 itself records 'None stated'), so the path is exercised by tests only, never yet by a live verdict; and rationale/narrowing text lands in the committed record unredacted (iss-2608300924205748).
  evidence: internal/core/intent/audit.go:515 — "if err := validateConditionDispositions(v, intentContent); err != nil {"
  evidence: internal/core/intent/audit.go:573 — "if !dispositionEnum[c.Disposition] {"
  evidence: internal/core/intent/audit.go:592 — "if len(seen) != len(known) {"
  evidence: internal/core/intent/audit.go:613 — "untested := untestedDispositions(content)"
  evidence: internal/core/intent/audit_conditions_test.go:106 — "func TestIngestAppliesAllFourDispositions(t *testing.T) {"
  evidence: internal/core/intent/audit_conditions_test.go:173 — "func TestIngestRefusesPartialConditionCoverage(t *testing.T) {"
  evidence: internal/core/intent/audit_conditions_test.go:415 — "func TestDeadLetterRecordsConditionsUntested(t *testing.T) {"
  evidence: .abcd/development/intents/shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md:35 — "None stated."
  evidence: .abcd/work/issues/open/iss-2608300924205748-audit-ingest-writes-agent-prose-unredacted.md:11 — "applies no privacy scanner to agent-produced prose"
- ac-2 — MET: A narrowed disposition is refused when narrowing is blank (audit.go:581), a narrowing beside any other disposition is refused (audit.go:584), and the accepted narrowing is rendered verbatim as its own 'narrowing:' line keyed to the cond- identity (audit.go:904); the agent rubric states that reworded prose is not a narrowing; pinned by TestIngestNarrowedRequiresNarrowing, TestIngestedBlockRendersNarrowing and TestIngestRefusesNarrowingOnAnUnnarrowedDisposition.
  evidence: internal/core/intent/audit.go:581 — "return fmt.Errorf(\"scope condition %q is narrowed but states no narrowing\", c.ConditionID)"
  evidence: internal/core/intent/audit.go:584 — "is %s but states a narrowing; only a narrowed condition carries one"
  evidence: internal/core/intent/audit.go:904 — "fmt.Fprintf(b, \"  narrowing: %s\\n\", n)"
  evidence: agents/intent-auditor.md:70 — "## How to dispose each scope condition (rubric — apply as harshly)"
  evidence: internal/core/intent/audit_conditions_test.go:154 — "func TestIngestedBlockRendersNarrowing(t *testing.T) {"
  evidence: internal/core/intent/audit_conditions_test.go:261 — "func TestIngestNarrowedRequiresNarrowing(t *testing.T) {"
  evidence: internal/core/intent/audit_conditions_test.go:392 — "func TestIngestRefusesNarrowingOnAnUnnarrowedDisposition(t *testing.T) {"

Gap audit:
- honoured:
  - Every scope condition receives one of four values, keyed to its persistent cond- identity rather than its wording
    evidence: internal/core/intent/audit.go:533 — "func validateConditionDispositions(v verdict, intentContent string) error {"
    evidence: internal/core/intent/audit.go:592 — "if len(seen) != len(known) {"
  - A narrowing is stated, never implied by silently changed text
    evidence: internal/core/intent/audit.go:581 — "is narrowed but states no narrowing"
    evidence: internal/core/intent/audit.go:904 — "narrowing: %s"
  - Disposition surface populated at verdict ingest and exercised by the fidelity verdict only for now
    evidence: internal/core/intent/audit.go:515 — "validateConditionDispositions(v, intentContent)"
    evidence: internal/core/intent/audit.go:118 — "ScopeConditions   []verdictCondition `json:\"scope_conditions\"`"
  - Wired on both front doors: CLI reports the split, plugin verb documents the disposition block, agent contract publishes rubric and rules 7-8
    evidence: internal/surface/cli/cli.go:1757 — "scope conditions %d: survived %d · narrowed %d · falsified %d · untested %d"
    evidence: commands/intent.md:235 — "The verdict also disposes the intent's scope conditions, keyed to the `cond-…`"
    evidence: agents/intent-auditor.md:158 — "7. `scope_conditions` covers the intent's conditions EXACTLY"
    evidence: internal/core/intent/audit_contract_test.go:131 — "func TestAuditorDefinitionDocumentsEveryDisposition(t *testing.T) {"
  - Dead letter records every condition untested so the quarantine stays honest in both vocabularies
    evidence: internal/core/intent/audit.go:613 — "untested := untestedDispositions(content)"
    evidence: internal/core/intent/audit_conditions_test.go:415 — "func TestDeadLetterRecordsConditionsUntested(t *testing.T) {"
- diverged:
  - Delivery proved against fixtures only: no shipped intent on this base carries a scope condition, so the staged-rollout test asserts vacuity over the live corpus and skips (rather than fails) when the shipped directory is unreadable
    evidence: internal/core/intent/audit_conditions_test.go:490 — "func TestShippedVerdictsSurviveTheStagedRollout(t *testing.T) {"
    evidence: internal/core/intent/audit_conditions_test.go:494 — "t.Skipf(\"shipped intents unreadable from the package dir: %v\", err)"
    evidence: .abcd/work/issues/open/iss-2608300927241768-itd-181-review-nits.md:11 — "the staged-rollout test skips on an unreadable shipped directory where a fatal would be stricter"
  - Human dead-letter render omits the untested split the JSON result carries; agent definition frontmatter description still omits the disposition surface (captured nits)
    evidence: internal/surface/cli/cli.go:1757 — "if res.Conditions > 0 {"
    evidence: .abcd/work/issues/open/iss-2608300927241768-itd-181-review-nits.md:11 — "the human dead-letter render omits the untested split the JSON reports"
- missing:
  - 'Later work inherits only what held' — nothing consumes a disposition; recording is the whole delivery, a falsified condition blocks nothing and the cold-reading assembler excludes dispositions by design
    evidence: .abcd/development/intents/shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md:22 — "Later work inherits only what held."
    evidence: .abcd/development/specs/closed/spc-59-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md:160 — "Enforcing that a `falsified` condition blocks anything. Recording is this"
    evidence: .abcd/development/specs/closed/spc-59-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md:155 — "Passing dispositions to a cold reading."
  - Agent-produced rationale, narrowing and evidence refs reach the committed record without the redaction primitive (pre-existing convention, inherited by the new fields)
    evidence: .abcd/work/issues/open/iss-2608300924205748-audit-ingest-writes-agent-prose-unredacted.md:11 — "applies no privacy scanner to agent-produced prose"
    evidence: internal/core/intent/audit.go:892 — "func renderDispositions(b *strings.Builder, conds []verdictCondition) {"

## Grounds

- pursued: Pursued now because the difference between what an intent assumed and what held is itself a finding, and it is currently unrecorded: the audit can compare promise to delivery but has nowhere to say that a condition the design leaned on turned out false. It rides with spc-55 because the identity it keys on has no other consumer, and a mechanism with no consumer is scaffolding.
