---
id: itd-2609211913453478
slug: one-page-in-the-glossary-maps-abcd-s-record-families-and-how
spec_id: spc-2609212131112235
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-34, itd-2609212103565953]
related_intents: [itd-24, itd-42, itd-172, itd-78]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_adrs: [adr-2609212115255771]
---

# One page maps the record families, and phase, milestone and roadmap are retired

## Press Release

> **One glossary page maps abcd's record families and how they relate, and three words leave the vocabulary.**
>
> "I kept asking which word to use, and every answer named a different document," said a product thinker who had just approved bundles and wondered whether phases still meant anything. "Now there is one page: intent, spec, step, bundle, issue, release, status. Phase and milestone are on it too, marked superseded, with what replaced them. I read it in five minutes."

## Why This Matters

On 2026-09-21 the product thinker asked, mid-interview, whether abcd has an intent that sorts out its vocabulary and whether phases survive bundles. A glossary existed with phase, intent, spec, roadmap, record and ledger defined in prose and adr-9 making the phase the product layer; nothing drew the map, `bundle` had no entry, and the autonomous run had added a third grouping, the batch. The research pass and the discussion that followed retired phase, milestone and roadmap and named their successors (adr-2609212115255771); this page is where a reader learns that.

## Mechanism

We expect one page that every glossary entry points at to stop new second names for one concept, because a name that must earn a row on a map is a name someone has to justify; shown wrong if a new record family or grouping word appears in the next release without a row on the page.

## Scope Conditions

None stated.

## What's In Scope

- **One page**, `glossary/core/record-families.md`: one row per family, intent, spec, step, bundle, issue, release, status (Now / Next / Later), with the definition, what it groups, what groups it, its lifecycle folders and the verb that moves it; every other glossary entry's `not_to_be_confused_with` may name only a family on the page.
- **Superseded terms**: `phase`, `milestone` and `roadmap` marked `status: superseded` with the successor named (dependencies and the status block; the derived release and each intent's criteria; the status block); the phase documents under the roadmap folder carry a retirement line and stay as history.
- **New entries**: `bundle` (itd-34) and `step` (itd-2609212103565953); `batch` defined on the page as the run's internal order, not a term.
- **The lint**: a glossary entry whose `not_to_be_confused_with` names nothing on the page is refused; a record frontmatter key naming a family the page does not define is reported.
- **The brief's mental-model chapter** reads brief → intent → spec (→ steps), with the bundle as a delivery grouping and the derived release as the checkpoint; adr-9 is superseded by adr-2609212115255771.

## What's Out of Scope

- Renaming any family or folder.
- The dependency graph between intents (itd-78).
- A release press release at the phase's old granularity (named in the decision record's consequences, not ruled).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves; the checkpoint is the derived release plus each intent's criteria (adr-2609212115255771).
2. Now / Next / Later is a rendered status, never stored, and the word roadmap goes with the phase documents.
3. The unit below a spec is the step, a section, not a record family; an issue carries no spec by design; the batch is the run's internal order.
4. One term per concept: the page is the map every entry points at.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** the glossary, **when** the page is read, **then** it holds one row per family (intent, spec, step, bundle, issue, release, status) with the definition, what it groups, what groups it, its lifecycle folders and the verb that moves it, and every other entry points at it.
- **Given** `phase`, `milestone` and `roadmap`, **when** their entries are read, **then** each is marked superseded with its successor named, and the phase documents carry a retirement line.
- **Given** `bundle` and `step`, **when** the glossary is read, **then** each has an entry, and `batch` is defined on the page as the run's internal order.
- **Given** an entry whose `not_to_be_confused_with` names nothing on the page, or a record frontmatter key naming a family the page does not define, **when** the record lint runs, **then** the first is refused and the second reported.
- **Given** the brief's mental-model chapter, **when** it is read, **then** it reads brief → intent → spec (→ steps) with the bundle and the derived release named, and adr-9 reads superseded by adr-2609212115255771.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-682701bb01e2 -->
Fidelity review — receipt rcp-682701bb01e2 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:b3006e36f460db1927e2db0503ab0b44f581b25aadd02659b68d50f555152295
Input attestations: diff:.abcd/development/brief/glossary, internal/core/lint/recordfamilies*.go, .abcd/record-lint.json, brief/01-product/03-mental-model.md, .abcd/development/roadmap and adrs/0009 at chore/audit-run-a-1 5b4a43b6 (git ls-tree -r; spc-2609212131112235 closed, itd-2609211913453478 shipped)@sha256:8f1ef1edbee75c5fa4f3c7acf46fd2b0aff390b29511582fd7212eaaaa3e0008;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the page's table holds exactly the seven rows intent, spec, step, bundle, issue, release, status with definition, groups, grouped-by, lifecycle and moved-by columns, and the glossary_family_pointer rule (blocker in record-lint.json) refuses any entry whose not_to_be_confused_with names nothing on the page; the concern is that the rule is satisfied by naming any family row rather than the page itself and admits other terms beside it, looser than the scope's 'may name only a family on the page'
  evidence: .abcd/development/brief/glossary/core/record-families.md:20 — "| Family | What it is | Groups | Grouped by | Lifecycle | Moved by |"
  evidence: .abcd/development/brief/glossary/core/record-families.md:22 — "| **intent** | one user-facing capability, press release first | its specs | a bundle (delivery) | drafts → planned → shipped (superseded; disciplines) | `intent plan`, `spec close` |"
  evidence: .abcd/development/brief/glossary/core/record-families.md:28 — "| **status** | Now / Next / Later, rendered from the shelves, the gate and the build's state | nothing (a view) | nothing | none: computed | nothing |"
  evidence: .abcd/record-lint.json:440 — ""glossary_family_pointer": { "enabled": true, "severity": "blocker","
  evidence: internal/core/lint/recordfamilies.go:196 — "onPage := func(member string) bool {"
  evidence: internal/core/lint/recordfamilies.go:229 — "name the family it is confused with (" + families + "), or the page itself as core/" + page.term +"
- ac-2 — MET: phase, milestone and roadmap carry status: superseded and each body names its successor with the ADR (dependencies plus shelves rendered as the status block; the derived release plus acceptance criteria; the rendered status block); all ten phase documents and the roadmap README open with the dated retirement line
  evidence: .abcd/development/brief/glossary/core/phase.md:7 — "status: superseded"
  evidence: .abcd/development/brief/glossary/core/phase.md:18 — "> **Superseded on 2026-09-21 (adr-2609212115255771): the sequencing layer is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the phase documents stay as history.**"
  evidence: .abcd/development/brief/glossary/core/milestone.md:18 — "> **Superseded on 2026-09-21 (adr-2609212115255771).** The word never had an entry of its own"
  evidence: .abcd/development/brief/glossary/core/roadmap.md:18 — "the rendered Now / Next / Later status block on the `abcd` board and the site's Status page replaces the roadmap document and the word."
  evidence: .abcd/development/roadmap/phases/phase-0-substrate.md:1 — "> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record."
  evidence: .abcd/development/roadmap/README.md:73 — "> **Retired on 2026-09-21** (adr-2609212115255771)"
- ac-3 — MET: bundle.md and step.md exist as stable entries, and the page defines the batch as the autonomous run's internal order and not a term of the record
  evidence: .abcd/development/brief/glossary/core/bundle.md:2 — "term: bundle"
  evidence: .abcd/development/brief/glossary/core/step.md:2 — "term: step"
  evidence: .abcd/development/brief/glossary/core/record-families.md:32 — "a **batch** is the autonomous run's internal order, derived from dependencies and the pick, and is not a term of the record."
- ac-4 — MET: glossary_family_pointer emits at the configured severity, blocker in this repo's record-lint.json, so it refuses; record_family_key emits at severityWarn and its message says 'reported, not refused'; both are armed in Lint and pinned by their tests, which pass at BASE
  evidence: internal/core/lint/recordfamilies.go:227 — "File: filepath.FromSlash(e.rel), Line: line, RuleID: ruleGlossaryFamilyPointer, Severity: cfg.Severity,"
  evidence: .abcd/record-lint.json:442 — ""severity": "blocker","
  evidence: internal/core/lint/recordfamilies.go:350 — "File: r.rel, Line: r.fields[k].line, RuleID: ruleRecordFamilyKey, Severity: severityWarn,"
  evidence: internal/core/lint/lint.go:563 — "if gfCfg, ok := cfg.Rules[ruleGlossaryFamilyPointer]; ok && gfCfg.Enabled {"
  evidence: internal/core/lint/recordfamilies_test.go:71 — "func TestGlossaryFamilyPointerRefusesAnEntryNamingNothingOnThePage(t *testing.T) {"
  evidence: internal/core/lint/recordfamilies_test.go:128 — "func TestRecordFamilyKeyReportsAKeyNamingAFamilyThePageDoesNotDefine(t *testing.T) {"
- ac-5 — MET: the mental-model chapter opens with brief, intent, spec and the spec's steps, names the bundle as the delivery grouping and the derived release as the checkpoint, cites adr-2609212115255771 as superseding adr-9, and adr-9's frontmatter reads superseded_by that ADR
  evidence: .abcd/development/brief/01-product/03-mental-model.md:3 — "the **brief** (what is this project?), the **intent** (why does this user-facing change matter?) and the **spec** (how do we build it?), with the spec's **steps** below it"
  evidence: .abcd/development/brief/01-product/03-mental-model.md:56 — "**Steps, bundles and the release.**"
  evidence: .abcd/development/decisions/adrs/0009-phase-as-product-layer.md:4 — "status: superseded"
  evidence: .abcd/development/decisions/adrs/0009-phase-as-product-layer.md:7 — "superseded_by: adr-2609212115255771"

Gap audit:
- honoured:
  - one page maps the seven families with their lifecycle and verb
    evidence: .abcd/development/brief/glossary/core/record-families.md:20 — "| Family | What it is | Groups | Grouped by | Lifecycle | Moved by |"
  - phase, milestone and roadmap leave the vocabulary with successors named
    evidence: .abcd/development/brief/glossary/core/phase.md:18 — "Superseded on 2026-09-21 (adr-2609212115255771)"
  - a family word with no row is reported by the lint
    evidence: internal/core/lint/recordfamilies.go:352 — "which the record-families page (" + page.rel + ") does not define"
- diverged:
  - every other entry's not_to_be_confused_with may name only a family on the page
    evidence: internal/core/lint/recordfamilies.go:230 — "beside any other term it names — every entry points at the one map"
- missing: (none)

## Grounds

- pursued: three terms are retired today and the map is where a reader learns what replaced them; we expect no new grouping word to appear without a row; shown wrong if one does in the next release
