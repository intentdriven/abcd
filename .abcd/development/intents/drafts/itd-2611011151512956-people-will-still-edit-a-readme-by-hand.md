---
id: itd-2611011151512956
slug: people-will-still-edit-a-readme-by-hand
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611011151475128, itd-61, itd-2610031651058674]
severity: minor
refines: [itd-2610031651058674, itd-2610031348087517]
related_adrs: [adr-2611011153058085]
origin: researcher-authored
production_mode: hand-written
---

# Hand-written text in a rendered surface is caught and offered back to its source

## Press Release

> People will still edit a README by hand. abcd scans every rendered documentation surface for text that has no home: a hand edit to a rendered section, or new prose added beside them. It assesses each piece, says which section of the brief or the how it belongs in, and in the next iteration offers the product thinker to move it there, from where it renders. Nothing hand-written is discarded unasked: a piece is replaced only once it has been moved into its source or the owner has chosen to drop it.

## Why This Matters

_Written by the agent on 2026-11-01 from the product thinker's rulings that
day (dcn-2611011248594647), restated; the planning interview confirms or
rewrites it._

Rendering every documentation surface from its one home (itd-2611011151475128)
is only realistic if a hand edit is never lost: people will rewrite their
README whatever the convention says. The product thinker ruled on 2026-11-01
that the lane runs in both directions, restated:

- **From the sources to the docs**, by rendering, always.
- **From the docs back to the sources**, when a page was adjusted by hand:
  abcd finds the hand-written text, assesses where it belongs, and in a
  subsequent iteration puts it to the product thinker with an offer to move it
  into its source, from which it is then rendered.

The lane serves both surfaces people write in: a hand edit to a page
rendered from the brief is offered back to the brief, and one to a page
rendered from the how (itd-2611011239589823) is offered back to the how.

That second direction is what lets the first be compulsory. A hand edit is a
finding with a destination, not drift to be overwritten.

The catch half is already designed: itd-2610031651058674 reports a README part
that has gone stale or been edited by hand and writes nothing, and the part's
marker already carries a checksum over its body
(`internal/core/ahoy/readme_marker.go`), which tells a hand edit from a stale
render. This intent widens that check from the setup parts to every rendered
section and adds the half no tool has: placing the piece and offering it back.

Links to the record, typed:

- **refines** itd-2610031651058674: its stale-or-edited check, widened from
  the README parts to every rendered section, with an offer back instead of a
  report alone.
- **refines** itd-2610031348087517's handling of a hand-edited part, which is
  reported and left as it is; here it is reported, placed and offered back.
- **builds on** itd-61, whose reconciliation of a human edit to the brief is
  the step a moved piece goes through once it reaches the brief.
- The rule that a render never discards hand-written text unasked, and holds
  only the section a piece waits in, is adr-2611011153058085 (proposed).

## SOTA

`.abcd/development/research/notes/2026-11-01-documentation-sources-sota.md`
(review 1, findings 2, 7 and 8): catching a hand edit by regenerating and
comparing is established practice; where AI tools write documentation back,
they propose a change for review and never commit it; no tool or study was
found that drafts the matching edit to a rendered page's source. The lane
follows the established half (catch by comparison) and builds the missing one
as a proposal.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **When the scan runs.** At a verb the person runs (the render, the lint, the
  release cut) rather than on a schedule, which the person's space rule rules
  out; which of those, and whether the offer is made at once or at the next
  session the person opens.
- **How a piece is placed.** Whether abcd names the section a piece belongs in
  deterministically (by the section it was edited inside) or a host agent
  proposes it, and how a piece that fits no section is offered.
- **A declined offer.** Whether a piece the product thinker neither moves nor
  drops is offered again or held with a reason; its section stays held either
  way (adr-2611011153058085).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
