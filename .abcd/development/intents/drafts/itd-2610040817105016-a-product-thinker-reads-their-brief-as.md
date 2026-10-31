---
id: itd-2610040817105016
slug: a-product-thinker-reads-their-brief-as
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-61]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every brief section shows what realises it, and a brief change names what it touches

## Press Release

> A product thinker reads their brief as the target design, in the present tense, and beside each section abcd shows which intents realise it: shipped, planned or still drafts. When they change a section, abcd names exactly which intents to re-check, proposes whether the change needs a new intent, an issue, or a design decision because it contradicts work already planned or built, and records nothing until they confirm.

_Filed by the facilitator on 2026-10-04 at the product thinker's request ("we should file what we've decided so that we can revist after the lab experiemnt"), from their working hypothesis and the state-of-the-art pass that followed it. Under test in a downstream lab; revisited after it._

## Why This Matters

_Facilitator-written._

The product thinker thinks about the product as it will be: the brief is their target design, in the present tense. A downstream project found that this reads as if the product already works (99 present-tense claims against 9 "staged" markers), and its owner called it the biggest issue with abcd. The product thinker's hypothesis keeps the present tense and moves "is it built yet?" out of the prose into links abcd keeps deterministically: each section lists what realises it, and a change to a section names exactly what to re-check, with a model proposing and the product thinker confirming what the change means. The state-of-the-art pass (research note 2026-10-04-brief-to-record-traceability-sota) found this to be settled practice, missing in abcd only as mechanics: stable section identifiers, fingerprinted links and a coverage report.

Typed links: refines itd-61 (a brief edit's implied intents drawn out: this draft makes its derivation deterministic); evidence and the hypothesis are recorded on itd-61, itd-142 and itd-143.

## Decisions

1. 2026-10-04, the product thinker, working hypothesis (verbatim on itd-61): the brief is the target design in the present tense; abcd lists what realises each section and assesses each brief change for a new intent or issue, or a contradiction owing a design decision.
2. 2026-10-04, the product thinker: file it now and revisit after the downstream lab.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [traceability-governance SOTA note](../../research/notes/2026-10-23-traceability-governance-sota.md). The 2026-10-04 note's shape (stable section identifier, `realises` link with a fingerprint, coverage report, status in the links not the prose) stands; this pass researched the gap.

- **Identifiers are a field, never the heading.** Pandoc's auto identifiers are derived from heading text and suffixed `-1`, `-2` on duplicates in document order, so a rename or a move changes them; the stable form is an explicit `{#identifier}`. sphinx-needs hashes the title only when no `id` is given; StrictDoc keeps the UID free-form and adds a machine identifier that matches nodes "even if they are moved, renamed".
- **A suspect link is cleared per parent, by an act.** Doorstop stores the parent's fingerprint in the link ("used by Doorstop to detect when a parent item is changed") and `doorstop clear <child> <parent>` clears the named parent only; Jama clears by hand when the change "has no impact or if you make the necessary changes". Neither expires a suspect link by time.
- **Fingerprint normative text only.** Doorstop's fingerprint covers the item UID, `text`, `ref` and `references`, the UIDs in `links` and any extended attributes opted in, and excludes `header`, `level` and the link fingerprints (Doorstop item reference); Jama fires only on administrator-chosen fields. That scope is the field's answer to suspicion fatigue: layout and reflow fire nothing by construction, while a typo inside normative text still moves the fingerprint. abcd's one fingerprint primitive decides on purpose whether linked ids are in scope, as Doorstop's does.
- **The evidence that it pays:** 71 subjects on real maintenance tasks were 24 per cent faster and 50 per cent more correct with trace links (Mäder and Egyed, 2016). The 2023 practitioner survey on costs could not be retrieved this pass.
- **The read-only report shape:** Spec Kit's `analyze` is "STRICTLY READ-ONLY", one table of id, category, severity, location, summary and recommendation, with a coverage-gap pass and a 50-row cap. The coverage report's first row is the uncovered set.
- **Path 2**: native, shaped to Doorstop's link-plus-fingerprint and OpenFastTrace's coverage states, with two seams: one fingerprint primitive shared with itd-78's edges and itd-61's trigger, and an explicit marker per section the rendering never derives from text. No Go dependency.

The independent fit-challenge runs at plan time.

## Open Questions

- What a section is: a chapter, a heading or a paragraph (the research starts at heading).
- One brief with a status overlay, or the vision plus separate change files.
- Who clears a suspect link: the product thinker (owns what), or the facilitator for plumbing chapters.
- How plumbing chapters are realised: by a step of a feature's spec, by a spec of their own, or by a trace list per section.
- What the downstream lab shows about the present tense once the coverage is visible.
- (SOTA 2026-10-23) Who may clear a suspect link and whether the clearing is recorded on the link (Doorstop re-stamps; Jama clears silently): the product thinker for sections they own, the facilitator for plumbing, and in either case a dated act or a bare re-stamp.
- (SOTA 2026-10-23) Whether the section markers are inserted in one consented migration of the brief's files or appear as sections are first linked, given that the brief is the product thinker's text.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
