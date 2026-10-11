---
id: itd-2610111655362931
slug: on-abcd-s-dashboard-the-product-thinker
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610091918433290, itd-2610032150577708]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# On the dashboard, the product thinker and facilitator review memory notes and promote the ones worth keeping

## Press Release

> On abcd's dashboard, the product thinker and the technical facilitator see every memory note agents have written since they last looked, with who wrote it, when and in which session, and keep, discard or promote each one with a click. A promoted note joins the project's record, so every agent and teammate gets it. Only a person promotes, and in a repository with more than two maintainers another maintainer agrees before a promotion lands.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

- **Given** notes written since the last review, **when** the person opens the review, **then** they see every one of those notes with who wrote it, when and in which session, and they keep, discard or promote each.
- **Given** a note the person promotes, **when** the promotion lands, **then** the note joins the project's record, so every agent and teammate gets it; only a person promotes, and in a repository with more than two maintainers at least one other maintainer must agree before it lands.

## Open Questions

1. **Where does abcd learn how many maintainers a repository has?** (technical facilitator) The more-than-two rule needs the count, and abcd records none.
   - (a) The distinct owners in the CODEOWNERS file the forge reads.
   - (b) A `maintainers` list in `.abcd/config.json`, which `ahoy install` asks for and a person edits.
   - (c) The forge's own list of people with maintain or admin rights, asked at promotion (a network call).
   - (d) The required approving-review count in the branch ruleset mirrored under `.abcd/work/rulesets/`.
2. **How is another maintainer's agreement recorded and checked?** (technical facilitator)
   - (a) As the forge's approval of the pull request that carries the promotion, required by the repository's ruleset; abcd checks nothing itself.
   - (b) As an `agreed_by` stamp the other maintainer adds by running `abcd memory agree <mem-id>` in their own checkout, which a `memory_promotion_agreed` record-lint rule requires before the change lands.
   - (c) Both.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-11, product thinker: reviewing memory notes and promoting them "will be part of a later PT / TF dashboard where the PT can click on those they want to promote", and then "Move them to the dashboard". This draft takes the review and promotion criteria, verbatim, from the memory notes intent (itd-2610091918433290), where they were its second and third criteria, accepted on 2026-10-11, and the two open questions on the maintainer count and the agreement from the memory-recall spec (spc-2610111627213138), where they were its fourth and fifth. It builds on the memory notes intent, whose store it reviews, and on the dashboard intent (itd-2610032150577708), where the review happens.
- 2026-10-11, carried from the same intent: only a person promotes, and above two maintainers another must agree (product thinker, recorded in adr-2610091918443997). Before the move, the memory-recall spec had designed the person's-act rule as a guard entry refusing an agent's `abcd memory promote` and a terminal check, after the source ledger's flip (adr-41), and the agreement as a record-lint rule over an `agreed_by` stamp; that design is a starting point for this intent's plan, not a ruling.
