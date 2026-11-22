---
id: itd-2611150128501216
slug: docs-reference-readme-md-3-4-presents
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251558595210]
origin: extracted-from-record
production_mode: hand-written
---

# docs/reference/README.md:3-4 presents the reference folder as describing configuration keys, environment variables and file schemas, and no page ever has: the folder holds the generated command reference, the writing style guide and the terminology crosswalk (the wording dates from 157e3ac1f). The proposed fix narrows the line, but the gap stays: keys a person sets by hand, such as load.preflights and reservation.github_username in ~/.abcd.noindex/config.json or rules.force_refresh_every_n in .abcd/config.json (internal/core/rules/inject.go:339-343), are documented only inside command help.

## Press Release

> _Seeded by promotion from iss-2610251558595210. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251558595210`: docs/reference/README.md:3-4 presents the reference folder as describing configuration keys, environment variables and file schemas, and no page ever has: the folder holds the generated command reference, the writing style guide and the terminology crosswalk (the wording dates from 157e3ac1f). The proposed fix narrows the line, but the gap stays: keys a person sets by hand, such as load.preflights and reservation.github_username in ~/.abcd.noindex/config.json or rules.force_refresh_every_n in .abcd/config.json (internal/core/rules/inject.go:339-343), are documented only inside command help.. Read that issue record for the source observation.

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
