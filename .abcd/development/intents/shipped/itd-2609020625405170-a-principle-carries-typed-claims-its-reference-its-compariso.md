---
id: itd-2609020625405170
slug: a-principle-carries-typed-claims-its-reference-its-compariso
spec_id: spc-2609020626042471
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-181, itd-177, itd-190, itd-183]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A principle carries typed claims, its reference, its comparison and its evidence, its statement is readable cold, and it inherits only what held

Typed links: `builds_on` [itd-181](itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) (scope-condition disposition), [itd-177](itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md) (claim typing), [itd-190](../disciplines/itd-190-the-claim-recording-gradient-an-intent-s-three-claim-kinds-c.md) (the claim recording gradient), [itd-183](itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (field projection); `refines` [itd-181](itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) (the first consumer of a condition's disposition).

## Press Release

> **The knowledge record becomes a read object.** An entry under `principles/` may declare what kind of claim it makes, what it is a claim about, what comparison produced it, and what evidence it rests on, with the evidence naming the records it distils and the scope conditions it inherits. The assembler projects a principle's statement to a reading and withholds its keys and citations, because the statement is knowledge and the citations are genealogy. A principle that rests on a scope condition dispositioned as falsified is reported by the record lint, so what is carried forward is only what held. The keys are forward-only: an existing principle carries none until its author states them, and the lint reports the untyped as untyped rather than as wrong.

> "By the time I distil a principle I want to know which of the assumptions under it survived delivery and which did not," said an AI/agent researcher who packs lifeboats from their own record. "And I want the next reading to be able to test the principle's statement without seeing the ADRs it was distilled from."

## Why This Matters

The cold-reading design schedules the knowledge-record extension for Iteration 2: claim typing, reference entity, comparison and evidence on `principles/` entries. It names `principles/` as a hard case: in Iteration 2 the knowledge record is a read object, so principle statements sit on the cold side, while their citations back to the ADRs they were distilled from are derivational and excluded, which requires a projection rule rather than a path rule. Today the family is denied to the assembler structurally by the `.abcd` segment and appears on neither the include table nor the declared exclusions, so the manifest is silent about it.

[itd-181](itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) shipped the scope-condition disposition so that later work inherits only what held, and its fidelity verdict found that nothing consumes a disposition: a falsified condition blocks nothing. The consumer the design has in mind is the knowledge record. A principle is what a project carries forward, and a principle resting on an assumption that delivery falsified is exactly the inheritance the disposition exists to prevent.

## Decisions flagged for the maintainer

Both were adopted by the maintainer on 2026-09-02 as [adr-2609021016270132](../../decisions/adrs/2609021016270132-the-principles-family-is-a-declared-record-store-whose-entri.md), which declares the family a record store with typed claims, read cold by statement and never by citation.

- **`principles/` is a declared record store.** Until the ADR the family had no frontmatter, no identifiers and no entry in the record stores the schema gate walks. Declaring it is a record-architecture decision on the pattern of adr-30, and the ADR makes it; the intent carries the keys, the projection, the check and the lifeboat contract.
- **The lifeboat contract changes.** `disembark principles` carries the four keys when it distils, which changes the principles payload a packed lifeboat carries; a consumer of that payload is told by the schema version.

## What's In Scope

- **Four frontmatter keys on principle records**, schema first: `claim_type` (criterion, causal or context, the design documents' words for the three claim kinds intents carry; `mechanism`, the shipped intent token for the causal kind, is read as an alias and written back as `causal`, never refused), `reference` (the entity the principle is about, as a record id or a named surface), `comparison` (what was compared to produce it, in one sentence), and `evidence` (the record ids it distils, and the scope-condition identities it inherits). A key considered and declined is written as an explicit null; an absent key is a claim not carried; the two are never collapsed, on the rule the intent gradient already uses.
- **Forward-only population.** An entry carrying none of the four keys is reported by the lint as untyped, at warn; an entry carrying any of them must carry all four, or state the null, at blocker. Nothing backfills an existing entry.
- **The projection rule** on the include table: a principle's statement section is projected to the reading; the four keys and every citation are withheld and their exclusion asserted in the manifest. The table admits the projection at every position but the comparative; which of the three receives it is a preset choice measured by the presets' eval, and is not made here.
- **The inheritance check** in the record lint: a principle whose evidence names a scope condition dispositioned as falsified is reported; one naming a condition dispositioned as narrowed is reported with the narrowing; one naming a condition with no disposition is reported as untested.
- **The read-block eval** gains a case planting a citation in a principle and asserting its absence from every reading.
- **`disembark principles`** carries the four keys when it distils, and its agent contract and changelog move with it.

## What's Out of Scope

- Changing what a principle says. The extension is to its frontmatter and to how it is read.
- Automatic distillation. The keys are declared by whoever writes the principle; the verb that distils carries them, it does not invent them.
- The frame-level revision record, which is its own intent.

## Mechanism

We expect typed evidence on principles to make the knowledge record checkable because a principle's inheritance becomes a query over condition identities and dispositions rather than a reading of prose, and we expect the projection rule to hold the read block for principles for the reason it holds for shipped intents: the cold and warm halves of one file are separated at field granularity. It fails if a principle's statement cannot be separated from its citations by section, which the first typed entry will show.

## Scope Conditions

- The claim vocabulary is the three claim kinds intents carry, in the design documents' words. A fourth claim type is a ruling this intent does not make. <!-- cond: cond-2609020626047525 -->
- Evidence names records in this repository. A principle distilled from a lifeboat cites the packed record ids, which resolve only in the source repository, and the lint reports them as unresolvable rather than as absent. <!-- cond: cond-2609020626048283 -->
- Existing principles are untyped until their authors type them. The untyped report is a count, not a fault, and it is expected to stay non-zero for some time. <!-- cond: cond-2609020626048565 -->

## Acceptance Criteria

- **Given** a principle record carrying one of the four keys and missing another, **when** the record lint runs, **then** it reports the record and the missing key.
- **Given** a principle record carrying none of the four keys, **when** the record lint runs, **then** it reports the record as untyped at warn and nothing else.
- **Given** a principle whose evidence names a scope condition dispositioned as falsified, **when** the record lint runs, **then** it reports the principle and the condition.
- **Given** a principle whose evidence names a scope condition dispositioned as narrowed, **when** the record lint runs, **then** it reports the principle with the stated narrowing.
- **Given** a principle whose evidence names a scope condition with no disposition, **when** the record lint runs, **then** it reports the principle as resting on an untested condition.
- **Given** a reading assembled at a position whose preset admits principles, **when** the manifest is read, **then** each principle appears as a projected item naming its statement field, and the four keys and the citations are asserted excluded.
- **Given** the read-block eval with a citation planted in a principle, **when** it runs, **then** it fails if the citation reaches any reading.
- **Given** `disembark principles`, **when** it distils, **then** each principle it writes carries the four keys.

## Prior Art

- [itd-181](itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md) (scope-condition disposition), [itd-177](itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md) (claim typing), [itd-190](../disciplines/itd-190-the-claim-recording-gradient-an-intent-s-three-claim-kinds-c.md) (the gradient), [itd-183](itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (field projection), the `disembark` family.
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None. The flagged decisions are adopted as adr-2609021016270132.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-df264ca3973f -->
Fidelity review — receipt rcp-df264ca3973f (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:fa48879ed29ae56d764076e0e001f00311ed3e6f6d3b4650c7c258e5dcaeb05d
Input attestations: diff:internal/core/lint/principles.go, internal/core/reading/include.go, internal/core/lifeboat/synthesis_principles.go, agents/principle-distiller.md, evals/coldreading_fixture_test.go at 4001d807 (git ls-tree -r)@sha256:f879b41b426d5002cb1f9e1d3d7d09ffd443edbf57fac42ab268df7dbba77560;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: a typed principle missing one of the four keys draws principle_claims at blocker naming the missing key, and is not reported as untyped; TestPrincipleClaimsNamesTheMissingKey passes at BASE
  evidence: internal/core/lint/principles.go:291 — "func (p principleCheck) judge(repoRoot string, r schemaRecord) ([]Finding, error) {"
  evidence: internal/core/lint/principles_test.go:189 — "func TestPrincipleClaimsNamesTheMissingKey"
- ac-2 — MET: an entry carrying none of the four keys draws exactly one principle_untyped finding at warn with the message naming the four keys, and the schema rule draws nothing; a warn rule does not fail preflight
  evidence: internal/core/lint/principles.go:301 — "p.add(&out, rulePrincipleUntyped, r, 1, "principle "+h+" is untyped (carries none of "+"
  evidence: internal/core/lint/principles_test.go:164 — "func TestUntypedPrincipleIsAWarnAndNothingElse"
  evidence: internal/core/lint/principles_test.go:153 — "func TestUntypedPrincipleHasNoSchemaFinding"
- ac-3 — MET: evidence naming a scope condition whose shipped intent's audit dispositioned it falsified draws principle_falsified at blocker naming both the principle handle and the condition identity
  evidence: internal/core/lint/principles.go:468 — ""as falsified""
  evidence: internal/core/lint/principles_test.go:383 — "func TestFalsifiedConditionIsReported"
- ac-4 — MET: a narrowed condition draws principle_inheritance carrying the disposition's stated narrowing verbatim, and no falsified finding
  evidence: internal/core/lint/principles.go:476 — ""), which was dispositioned as narrowed: "+d.Narrowing)"
  evidence: internal/core/lint/principles_test.go:398 — "func TestNarrowedConditionCarriesTheNarrowing"
- ac-5 — MET: a condition with no disposition, one recorded untested, and one whose audit disposes only another condition each draw the untested inheritance finding naming the condition, while a survived condition is silent
  evidence: internal/core/lint/principles.go:484 — "p.add(out, rulePrincipleInheritance, r, line, "principle "+h+" rests on the untested condition "+m+" ("+"
  evidence: internal/core/lint/principles_test.go:412 — "func TestUndispositionedConditionIsUntested"
- ac-6 — MET: at widening, entailment and detection the manifest carries one parsed item of kind principle whose Field is the statement label, the statement reaches the bundle while the keys, the reasoning and the link target do not, and the manifest asserts the four keys' exclusion by field projection and the citation exclusion; the comparative assembly carries no principle item
  evidence: internal/core/reading/include.go:199 — "KindPrinciple Kind = "principle""
  evidence: internal/core/reading/include.go:536 — "Rule: "The knowledge record is a read object: a principle travels as its statement, " +"
  evidence: internal/core/reading/principle_test.go:95 — "func TestPrincipleProjectsItsStatementOnly"
  evidence: internal/core/reading/principle_test.go:204 — "func TestManifestAssertsPrincipleExclusions"
  evidence: internal/core/reading/principle_test.go:179 — "func TestPrincipleRowExcludesComparative"
- ac-7 — MET: the read-block eval plants the PRINCIPLE-CITATION class in four homes (an evidence key, the reasoning paragraph, a link target in the statement, and an evidence key on a spec) as exhaust at every position, with a relocation control that must leak; the coldreading-tagged eval passes at BASE (go test -tags coldreading ./evals/, 99.7s)
  evidence: evals/coldreading_fixture_test.go:337 — "Name: "PRINCIPLE-CITATION","
  evidence: evals/coldreading_fixture_test.go:507 — "Class: "PRINCIPLE-CITATION","
  evidence: internal/core/reading/principle_test.go:224 — "func TestPrincipleItemCarryingAHandleRefuses"
- ac-8 — MET: the delegated synthesis carries claim_type (canonicalised through lint.CanonicalClaimType), reference, comparison and the filtered evidence into every principle it writes, the deterministic fallback writes the keys with explicit nulls where it invents nothing, and the agent contract names the four keys; the keys test asserts each entry carries all four
  evidence: internal/core/lifeboat/synthesis_principles.go:287 — "ClaimType: claims.ClaimType,"
  evidence: internal/core/lifeboat/synthesis_principles.go:310 — "ct, ok := lint.CanonicalClaimType(*in.ClaimType)"
  evidence: internal/core/lifeboat/synthesis_principles_keys_test.go:51 — "for _, k := range []string{"claim_type", "reference", "comparison", "evidence"} {"
  evidence: internal/core/lifeboat/principles_contract_test.go:86 — "for _, k := range []string{"`claim_type`", "`reference`", "`comparison`", "`evidence`"} {"
  evidence: agents/principle-distiller.md:53 — ""claim_type": "causal","

Gap audit:
- honoured:
  - mechanism is read as an alias for causal and written back as causal, never refused; a fourth claim type is refused
    evidence: internal/core/lint/principles.go:84 — "func CanonicalClaimType(v string) (string, bool) {"
    evidence: internal/core/lint/principles_test.go:263 — "func TestPrincipleClaimsReadsMechanismAsCausal"
    evidence: internal/core/lint/principles_test.go:251 — "func TestPrincipleClaimsRefusesFourthClaimType"
  - an explicit null declines a claim and a blank value is refused as a forgotten key; the two are never collapsed
    evidence: internal/core/lint/principles_test.go:210 — "func TestPrincipleClaimsRefusesEmptyValue"
  - a statement or title that cites a record is refused, so the statement can be separated from its citations by section
    evidence: internal/core/lint/principles_test.go:303 — "func TestPrincipleStatementMayNotCite"
    evidence: internal/core/reading/principle_test.go:285 — "func TestHeadingShapedStatementNeverTravels"
  - forward-only: nothing backfills an existing entry, and the untyped warn does not fail preflight
    evidence: internal/core/lint/principles_test.go:469 — "func TestWarnRuleDoesNotFailPreflight"
- diverged: (none)
- missing: (none)

Scope-condition dispositions:
- cond-2609020626047525 — survived: ClaimTypes is the three kinds with mechanism as the causal alias, and any other value is refused
  evidence: internal/core/lint/principles.go:84 — "func CanonicalClaimType(v string) (string, bool) {"
  evidence: internal/core/lint/principles_test.go:251 — "func TestPrincipleClaimsRefusesFourthClaimType"
- cond-2609020626048283 — survived: evidence that resolves in no record store, and a condition identity no shipped intent carries, are reported as unresolvable with the lifeboat case named, not as absent
  evidence: internal/core/lint/principles.go:442 — ""unresolvable in this repository's record stores; a principle distilled from a lifeboat cites packed "+"
  evidence: internal/core/lint/principles_test.go:440 — "func TestUnresolvableEvidenceIsReportedNotAbsent"
- cond-2609020626048565 — survived: an untyped entry draws one warn finding and nothing else, and a warn rule does not fail preflight, so the count stays a count
  evidence: internal/core/lint/principles_test.go:164 — "func TestUntypedPrincipleIsAWarnAndNothingElse"
  evidence: internal/core/lint/principles_test.go:469 — "func TestWarnRuleDoesNotFailPreflight"

## Grounds

- pursued: we expect typed evidence on principles to turn inheritance into a query over condition identities and dispositions, and a statement projection to keep the knowledge record readable cold; a principle resting on a falsified condition that the lint passes, or a citation reaching a reading, would show it wrong
