---
id: itd-2609020625405251
slug: a-scope-condition-is-dispositioned-from
spec_id: spc-2609020626046252
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-181, itd-177, itd-180, itd-185]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A scope condition is dispositioned from a reading run, keyed to the condition's identity and joined to the item that occasioned it

Typed links: `builds_on` [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are.md) (scope-condition disposition at verdict ingest), [itd-177](../shipped/itd-177-an-intent-s-claims-are-typed-and-its.md) (condition identity), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as.md) (superseding dispositions), [itd-185](../shipped/itd-185-one-ingest-verb-validates-every-cold.md) (the ingest that mints items); `refines` [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are.md) (a second writer into the same surface).

## Press Release

> **A detection can change the standing of an assumption, on the record.** `abcd intent condition <itd-N> <cond-id> --disposition <survived|narrowed|falsified|untested> --occasioned-by <rdi-N|itd-N> --grounds "<why>"` writes one scope-condition disposition against a shipped intent, keyed to the condition's stamped identity, joined to the reading item, or the shipped intent, that occasioned it, with the narrowing stated where the value is narrowed. Until now only the fidelity verdict could disposition a condition; now a reading or a delivery can occasion one, and the record shows which did.

> "When the detection pass tells me a condition I assumed does not hold in the tree, I want to record that against the condition, not against the sentence, and I want the item that told me to be named," said an AI/agent researcher who runs the detection pass over their own shipped state.

## Why This Matters

The cold-reading design has, at Step 6 in Iteration 2, each scope condition dispositioned by the researcher against what the detection pass returned, and it makes the disposition warm: never passed to a reading, populated at Step 6, and in Iteration 1 exercised by the fidelity verdict only. [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are.md) delivered that surface: the enum, the identity key, the narrowing rule, and one writer, `abcd intent audit ingest`, which takes the auditor's verdict. There is no path from a reading run to a condition. A detection item names a tension and the constraint in play, and the researcher's response to it is a disposition on the item; but the condition the item bears on stays as the fidelity verdict left it, or unstamped, and the record cannot show that a reading changed an assumption's standing.

The join matters for the closing run. The design's purpose-durability reading turns on a tension rejected with a named purpose, over a state deliberately unchanged, returning again; a condition that was narrowed because of a detection is the clearest form of a state deliberately changed.

## What's In Scope

- **`abcd intent condition`** as a second writer into the same disposition surface the verdict ingest writes, keyed to the condition identity, refusing a value outside the enum, a narrowing on any value but narrowed, a narrowed value without a narrowing, an occasion that does not resolve to a reading item or to a shipped intent, a ground below the substance floor, and an intent not in `shipped/`.
- **How the two writers coexist.** The section holds one dated block per write. A verdict ingest writes one block covering every condition, as it does today; this verb writes one block covering one condition and naming its occasion. A condition's standing disposition is the one in the latest block that names it, and the render says which block it came from.
- **The occasion recorded** with the disposition, so a condition's standing carries the item, or the delivery, that changed it.
- **Exclusion from readings** unchanged: dispositions render under the heading the assembler already withholds.

## What's Out of Scope

- The reading marking conditions itself. The reading names tensions; the researcher marks, on the rule itd-181 adopted. This is a registered interpretation, because the design framework's letter reads the other way: Section 7.1 has the reading mark each condition survived, narrowed, falsified or untested, and section 14 has each scope condition dispositioned under Iteration 2's cold column. The ground for reading both as the reading naming the condition and the researcher marking it is that the readings companion's ratified registrative body (section 4.2) carries tension, constraint in play and why it is a tension, with no field for a mark; that the companion (section 9) says readings do not disposition their own items; and that the build sheet's W8 makes dispositions warm and never passed to a reading. The join from the researcher's mark to the item is by citation: Where the constraint in play is a scope condition, the detection item cites the condition's identity, and the spec states how the verb reads that citation.
- Dispositioning a condition on a draft or planned intent. Conditions are dispositioned on shipped intents, where there is a delivered state to disposition against.

## Mechanism

We expect a writer keyed to condition identity and joined to a reading item to make a reading's effect on the frame's assumptions countable because the identity survives rewording and the join names the cause, so the closing run can ask which conditions a reading changed. It fails if the verdict ingest and this verb write incompatible shapes, which sharing one validator and one block grammar prevents.

## Scope Conditions

- The verb writes against a shipped intent only. A condition on a planned intent has no delivered state and is out of scope. <!-- cond: cond-2609020626040782 -->
- The occasion is a reading item at any position, not only detection, because an entailment item can bear on a context claim as readily as a detection can; or a shipped intent, because a delivery can change a condition's standing as a reading can, which is how the comparative refusal's surviving condition on itd-199 is re-dispositioned when the comparative channel ships. <!-- cond: cond-2609020626040385 -->
- Dispatch from a reading item to the conditions it occasioned lands with the record dispatcher's coverage of reading items, which is planned work; until then the join is read from the intent's side. <!-- cond: cond-2609020626040303 -->

## Acceptance Criteria

- **Given** a shipped intent with a stamped condition and a reading item, **when** the verb runs with `falsified` and a ground, **then** the condition carries that disposition, the occasion and the ground, and the intent's rendered notes show it in its own dated block.
- **Given** the verb with `narrowed` and no narrowing, **when** it runs, **then** it refuses and names the missing narrowing.
- **Given** the verb with `survived` and a narrowing, **when** it runs, **then** it refuses.
- **Given** a value outside the four, **when** it runs, **then** it refuses and names the enum.
- **Given** an occasion that does not resolve, **when** it runs, **then** it refuses.
- **Given** a planned intent, **when** the verb runs, **then** it refuses and names the bucket.
- **Given** a condition already dispositioned by the verdict ingest, **when** the verb writes a second disposition, **then** both blocks stand and the later one is reported as standing.
- **Given** a reading assembled over the intent, **when** the manifest is read, **then** the condition dispositions are asserted excluded and none reaches the bundle.

## Prior Art

- [itd-181](../shipped/itd-181-a-shipped-intent-s-scope-conditions-are.md) and its spec; [itd-177](../shipped/itd-177-an-intent-s-claims-are-typed-and-its.md) (condition identity); [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as.md) (superseding dispositions).
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-dd287c31bbb6 -->
Fidelity review — receipt rcp-dd287c31bbb6 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:4f9e27e505c21858685d219d0835a99f4744a7df7f57843237ff172f3632ce76
Input attestations: diff:tree at 4c09b5c749de3dc3d1a1e0ab85d9dcd58ffcb4d5 (main lineage, itd-2609020625405251 shipped)@-;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: DispositionCondition writes one dated block under Audit Notes carrying the condition marker, the occasion, the value and the ground; TestConditionWritesADatedBlock asserts the block's exact text, and the CLI front door renders the standing with the block it came from
  evidence: internal/core/intent/condition.go:113 — "func DispositionCondition(repoRoot string, req ConditionRequest) (ConditionResult, error) {"
  evidence: internal/core/intent/condition.go:239 — "func conditionBlock(id, value, ground, narrowing, occasion, date string) string {"
  evidence: internal/core/intent/condition_test.go:64 — "func TestConditionWritesADatedBlock(t *testing.T) {"
  evidence: internal/surface/cli/cli.go:2373 — "func newIntentConditionCommand(asJSON *bool) *cobra.Command {"
- ac-2 — MET: a narrowed value with no narrowing is refused with nothing written and the message names the missing narrowing; TestConditionNarrowedRequiresNarrowing covers it
  evidence: internal/core/intent/condition.go:159 — "is narrowed but states no narrowing; say what now holds (nothing written)"
  evidence: internal/core/intent/condition_test.go:125 — "func TestConditionNarrowedRequiresNarrowing(t *testing.T) {"
- ac-3 — MET: a narrowing beside any value but narrowed is refused; TestConditionNarrowingOnlyOnNarrowed covers it
  evidence: internal/core/intent/condition.go:162 — "but states a narrowing; only a narrowed condition carries one (nothing written)"
  evidence: internal/core/intent/condition_test.go:130 — "func TestConditionNarrowingOnlyOnNarrowed(t *testing.T) {"
- ac-4 — MET: a value outside condition.Enum is refused and the message lists the enum; TestConditionRefusesOutOfEnum covers it
  evidence: internal/core/intent/condition.go:143 — "is not one of %s (nothing written)", req.Disposition, strings.Join(condition.Enum, ", ")"
  evidence: internal/core/intent/condition_test.go:139 — "func TestConditionRefusesOutOfEnum(t *testing.T) {"
- ac-5 — MET: the occasion is resolved through readingitem.ResolveOccasion over the item and intent families and an unresolved one is refused with nothing written; TestConditionOccasionMustResolve and TestConditionOccasionIntentMustBeShipped cover it
  evidence: internal/core/intent/condition.go:170 — "occasion %q does not resolve: %v (nothing written)"
  evidence: internal/core/intent/condition_test.go:146 — "func TestConditionOccasionMustResolve(t *testing.T) {"
- ac-6 — MET: an intent outside shipped/ is refused and the message names the bucket it is in; TestConditionRefusesUnshippedBucket covers it
  evidence: internal/core/intent/condition.go:122 — "%s is in %s, not shipped; a condition is dispositioned against a delivered state"
  evidence: internal/core/intent/condition_test.go:182 — "func TestConditionRefusesUnshippedBucket(t *testing.T) {"
- ac-7 — MET_WITH_CONCERNS: a verdict block and a later condition block both stand in the record and Standing reports the condition block, so the criterion's direction holds (condition_test.go:100-111, TestVerdictIngestUnchangedBesideConditionBlocks). Concern: standing is folded by source precedence, not by position — a verdict ingested after a reading-occasioned block does not stand unless its rationale names that occasion — which refines the intent's In Scope statement that the latest block names the standing; the refinement is recorded in the spec and the 2026-09-25 ruling in DECISIONS.md
  evidence: internal/core/condition/condition.go:183 — "func Standing(content string) map[string]Disposition {"
  evidence: internal/core/condition/condition.go:167 — "The fold is by source precedence, not by position."
  evidence: internal/core/condition/condition_test.go:113 — "func TestVerdictDoesNotOverrideAReadingOccasionedBlock(t *testing.T) {"
  evidence: internal/core/intent/condition_test.go:329 — "func TestVerdictIngestUnchangedBesideConditionBlocks(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-2609020626046252-a-scope-condition-is-dispositioned-from.md:27 — "a verdict overrides a reading-occasioned block only where its rationale"
- ac-8 — MET: TestConditionBlockNeverReachesTheBundle assembles a reading over an intent carrying a condition block and asserts the block's text is absent from the bundle, the condition's identity marker travels, and the manifest asserts the Audit Notes exclusion
  evidence: internal/core/reading/assemble_test.go:1822 — "func TestConditionBlockNeverReachesTheBundle(t *testing.T) {"
  evidence: internal/core/reading/assemble_test.go:1857 — "the manifest does not assert the Audit Notes exclusion"

Gap audit:
- honoured:
  - a second writer into the same disposition surface, keyed to the condition identity and joined to its occasion
    evidence: internal/core/intent/condition.go:239 — "func conditionBlock(id, value, ground, narrowing, occasion, date string) string {"
  - every refusal exits with nothing written
    evidence: internal/core/intent/condition.go:205 — "did not read back as written; nothing written"
  - the ground is held to the substance floor and neutralised before it is written
    evidence: internal/core/intent/condition_test.go:226 — "func TestConditionGroundsBelowTheFloorRefuse(t *testing.T) {"
  - the render says which block a standing came from
    evidence: internal/surface/cli/cli.go:2430 — "func renderConditionStanding(w io.Writer, standing []intent.StandingEntry) {"
- diverged:
  - the standing disposition is the one in the latest block that names the condition: delivered as source precedence, where a condition block stands over a verdict unless the verdict names its occasion
    evidence: internal/core/condition/condition.go:167 — "The fold is by source precedence, not by position."
- missing: (none)

Scope-condition dispositions:
- cond-2609020626040782 — survived: the verb refuses any intent outside shipped/ and names the bucket, so a condition on a planned intent is never written
  evidence: internal/core/intent/condition.go:122 — "%s is in %s, not shipped"
- cond-2609020626040385 — narrowed: the occasion resolves through the item family across every run directory and through the shipped-intent family, but the intent under disposition is refused as its own occasion, so a shipped intent occasions a condition only on another intent
  narrowing: holds for a reading item at any position and for a shipped intent other than the one whose condition is dispositioned; the intent's own delivery stays the verdict ingest's ground
  evidence: internal/core/intent/condition.go:166 — "readingitem.FamilyItem, readingitem.FamilyIntent)"
  evidence: internal/core/intent/condition.go:173 — "occasion %s is the intent itself; its own delivery is the verdict ingest's ground, not this verb's"
- cond-2609020626040303 — survived: the join is read from the intent's side by ConditionStanding and no dispatcher reaches a reading item at BASE, as the condition assumed
  evidence: internal/core/intent/condition.go:91 — "func ConditionStanding(repoRoot, intentID string) (ConditionStandingView, error) {"
  evidence: .abcd/development/specs/closed/spc-2609020626046252-a-scope-condition-is-dispositioned-from.md:317 — "### Dispatch, from the intent's side only"

## Grounds

- pursued: we expect a second writer keyed to condition identity and joined to a reading item to make a reading's effect on the frame's assumptions countable for the closing run; a condition changed because of a detection that the record cannot trace to that detection would show it wrong
