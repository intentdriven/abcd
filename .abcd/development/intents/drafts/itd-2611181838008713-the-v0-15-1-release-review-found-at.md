---
id: itd-2611181838008713
slug: the-v0-15-1-release-review-found-at
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610171626398856]
origin: extracted-from-record
production_mode: hand-written
---

# The v0.15.1 release review found at CHANGELOG.md:19-72: INCOMPLETE (judgement call). The 0.15.1 section has no Added line for user-facing surfaces that are new since v0.15.0 and present in this binary. (1) `abcd status`, a top-level verb in `abcd --help` with a user-invocable commands/status.md (block: people); commands/status.md does not exist at v0.15.0. (2) `abcd announce` (commands/announce.md). Both come from the planned itd-2609151838312703 and its open spc-2610150728396006. (3) `abcd implement load --race-gate` and the `load.preflights` / ABCD_LOAD_PREFLIGHTS setting (internal/core/implement/load_slot…

## Press Release

> _Seeded by promotion from iss-2610171626398856. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610171626398856`: The v0.15.1 release review found at CHANGELOG.md:19-72: INCOMPLETE (judgement call). The 0.15.1 section has no Added line for user-facing surfaces that are new since v0.15.0 and present in this binary. (1) `abcd status`, a top-level verb in `abcd --help` with a user-invocable commands/status.md (block: people); commands/status.md does not exist at v0.15.0. (2) `abcd announce` (commands/announce.md). Both come from the planned itd-2609151838312703 and its open spc-2610150728396006. (3) `abcd implement load --race-gate` and the `load.preflights` / ABCD_LOAD_PREFLIGHTS setting (internal/core/implement/load_slot…. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
