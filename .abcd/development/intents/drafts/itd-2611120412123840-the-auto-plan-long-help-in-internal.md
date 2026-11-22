---
id: itd-2611120412123840
slug: the-auto-plan-long-help-in-internal
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610312044556915]
origin: extracted-from-record
production_mode: hand-written
---

# The --auto-plan long help in internal/surface/cli/build.go (about line 205) says only that a checkout that changed since the start hands the lane back, while commands/build.md says what the digest covers (HEAD, the index's entries and their flags, the files the ignore rules admit outside the local tier) and what it does not (ignored files, the git directory).

## Press Release

> _Seeded by promotion from iss-2610312044556915. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610312044556915`: The --auto-plan long help in internal/surface/cli/build.go (about line 205) says only that a checkout that changed since the start hands the lane back, while commands/build.md says what the digest covers (HEAD, the index's entries and their flags, the files the ignore rules admit outside the local tier) and what it does not (ignored files, the git directory).. Read that issue record for the source observation.

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
