---
id: itd-2609150819439571
slug: no-supported-way-to-correct-a-factual
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2609100505146979]
origin: extracted-from-record
production_mode: hand-written
---

# abcd has no supported operation for correcting a factual error inside a durable record, and no documented convention saying what to do instead. The record is deliberately not rewritten, which is right, but "not rewritten" and "wrong" are different states and only the first has a mechanism.

## Press Release

> _Seeded by promotion from iss-2609100505146979. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2609100505146979`: abcd has no supported operation for correcting a factual error inside a durable record, and no documented convention saying what to do instead. The record is deliberately not rewritten, which is right, but "not rewritten" and "wrong" are different states and only the first has a mechanism.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [traceability-governance SOTA note](../../research/notes/2026-10-23-traceability-governance-sota.md). Material for the planning interview ruling M25 ordered (a convention first, a verb only if asked), which opens on: per record family, where does a correction go, what does it carry, and why does the original stay?

- **The RFC erratum is the closest shape.** "Because RFCs do not change after publication, there is an errata system"; an erratum is technical or editorial, reported as Original Text and Corrected Text with a comment, and carries one of Reported, Verified, Rejected or Held for Document Update (RFC Editor; IESG statement of 2021).
- **The triage rule is the convention's boundary.** Verified when the correction "should be available to implementers"; Rejected when it "proposes a significant change", since "Significant clarifications should not be handled as errata reports" (IESG). The convention adds a boundary of its own, since the ISO Directives were not opened this pass: outdated information is not an erratum's business. So: a wrong sentence is an erratum; a changed decision is a supersession; outdated content is not an erratum; a significant clarification is a new record.
- **The original stays, and the notice is linked to it.** Publishers keep an article "extant, exact and unaltered as far as is possible" and correct it by "a notice that will be permanently linked to the article"; a corrigendum is the author's error, an erratum the publisher's (Elsevier). COPE's own guideline could not be retrieved.
- **The reader must see it from the original.** Crossmark exists so a reader gets "the current status of an item of content" and learns "when a substantial change affecting the citation or interpretation has occurred", even in a downloaded copy. An erratum nobody sees from the record it corrects repeats the gap.
- **ADR tooling has no correction primitive.** Nygard: "If a decision is reversed, we will keep the old one around, but mark it as superseded"; MADR's status set has no erratum; log4brains: "an ADR is immutable. Only its status can change". A misquoted sentence would need a whole new ADR, which is too heavy.
- **Git notes** attach text "without touching the objects themselves" but live on a separate ref nothing in abcd reads or shows; rejected.
- **Path 2**: native, shaped to the RFC erratum: an appended, dated entry (a `## Errata` section, or an erratum-marked entry in a shipped intent's already append-only Audit Notes if the interview rules per family) carrying five fields: the quoted original, the corrected text, the kind, why the original stands, and who verified it on which date, by role; a record-lint rule proves the fields; the board, the site and `abcd <record-id>` show "has errata" on the original. The Reported status is dropped at first. A verb follows only if the interview asks, as M25 says.

The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) One `## Errata` section in every durable record family, or an erratum-marked entry inside each family's existing append-only section (Audit Notes for a shipped intent) with a new section only where none exists: ADRs and specs have no such section today, which the source issue calls the sharper half of the gap.
- (SOTA 2026-10-23) Whether an erratum carries a status at all in a repository with one product thinker (verified on write), or the RFC set's Reported and Held return once a correction is disputed.
- (SOTA 2026-10-23) How "has errata" is rendered on the original (board, site, `abcd <record-id>`) so a reader taking records in the wrong order still meets the correction first, as Crossmark and a retraction notice shown before the article do.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
