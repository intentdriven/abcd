---
id: itd-2609292107351737
slug: every-development-session-records-what-it-actually-cost-in
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-2608301744251874, iss-2608301856299268, iss-2608220150157508]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A development session's token cost is measured, and set against the size class its spec predicted

Typed links: `related_issues` [iss-2608301744251874](../../../work/issues/open/iss-2608301744251874-development-token-metering-is-parked-as-a-note-and-owes-a-fi.md) (token metering), [iss-2608301856299268](../../../work/issues/open/iss-2608301856299268-a-size-classification-for-specs-and-tasks-is-parked-as-a-not.md) (size classes) and [iss-2608220150157508](../../../work/issues/open/iss-2608220150157508-local-token-usage-accounting-in-the-history-store.md) (token accounting in the history store), the three records this draft plans as one.

## Press Release

> Every development session records what it actually cost in tokens, and every spec says in advance which size class it expects to fall in, so a prediction can be checked against what the work became. When a transcript is captured into the history store, abcd keeps the per-message token counts and the model id the transcript itself carries, locally and as metadata only; a report sums them per session, per spec and per repository; and a spec declares a size class before it is built that the report sets beside the measured cost. "We used to guess a flat cost per task and never learned whether the guess was right," said Maya, an autonomous-development practitioner. "Now the size we predicted sits next to what the work actually took, and a wrong class shows up in the next report."

## Why This Matters

Nothing in the record measures what a development act cost. The superseded
itd-29 estimated token cost forward from a flat per-task constant, and the
draft itd-98 names token cost only inside one experiment's report, so an
estimate can be wrong indefinitely without anything noticing. The one tally the
2026-08-30 cost calibration made had to be assembled by hand from one session's
notifications, and it was lost with that session. The parked notes
(`.abcd/development/research/notes/2026-08-30-development-token-metering-gap.md`
and `2026-08-30-spec-size-classification-gap.md`) name the pair: a size class
predicts, metering measures, and neither is worth much alone. The product
thinker ruled on 2026-09-29 to plan metering and size classes as one intent
(ruling J3) and that session token accounting in the history store joins it
(ruling J4).

## What's In Scope

- **Measurement at capture**: when the history store captures a transcript, it
  keeps the per-message usage the transcript carries (input, output, cache read,
  cache write) and the model id, aggregated with a timestamp in the stored
  transcript's metadata. Metadata only, never bodies (brief invariant 15's
  existing carve-out for metadata).
- **A report** that sums the counts per session, per spec and per repository,
  local-only: never committed, never rendered on any site.
- **Size classes** a spec declares before it is built, checked against the
  measured cost in the same report. The notes' finding binds the scheme: depth
  does not track volume, and the roadmap forbids time estimates, so a class is
  not denominated in days.
- **Prices on explicit ask only**: a user-level price table; a report meeting a
  model with no recorded price offers an explicit refresh and never fetches on
  its own (adr-38). Money figures are labelled as API-equivalent estimates.

## What's Out of Scope

- Budget enforcement: refusing to start a run on a budget is itd-29's
  superseded ground and is not revived here.
- Anything that leaves the machine.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from the three records and the ruling, unconfirmed: the
planning interview walks each one.

- **Given** a transcript whose messages carry usage fields, **when** the history
  store captures it, **then** the stored metadata carries the summed input,
  output, cache-read and cache-write counts and the model id, and no message
  body is added to the metadata.
- **Given** a transcript that carries no usage fields, **when** it is captured,
  **then** its metadata records the counts as unmeasured rather than zero.
- **Given** captured sessions for one repository, **when** the report runs,
  **then** it sums the counts per session, per spec and per repository, and
  writes nothing.
- **Given** a spec that declares a size class, **when** the report covers the
  sessions that built it, **then** it shows the declared class beside the
  measured cost.
- **Given** a report meeting a model with no recorded price, **when** it runs,
  **then** it shows token counts without a money figure, names the refresh verb,
  and performs no network fetch.

## Open Questions

- **What a size class measures**: the notes say depth does not track volume; is
  the class about the number of ways the work can be quietly wrong, the review
  rounds it takes, or tokens alone?
- **How a session is attributed to a spec**: the branch, the run log's lane
  record, or a stamp at capture.
- **Where the price table lives** in the user-level home, and its refresh verb.
- **Whether itd-98's comparison report consumes this measurement** rather than
  measuring on its own.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
