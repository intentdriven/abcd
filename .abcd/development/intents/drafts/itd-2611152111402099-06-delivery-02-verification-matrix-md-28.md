---
id: itd-2611152111402099
slug: 06-delivery-02-verification-matrix-md-28
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611040555095597]
origin: extracted-from-record
production_mode: hand-written
---

# 06-delivery/02-verification-matrix.md:28 says that on a private-to-public flip the user is shown the resulting tracked-versus-ignored list. The gitignore does change, but abcd ahoy install --visibility public prints only that it kept abcd's private files out of git and the visibility change; --details adds the files written, and the --json result has no field for such a list.

## Press Release

> _Seeded by promotion from iss-2611040555095597. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611040555095597`: 06-delivery/02-verification-matrix.md:28 says that on a private-to-public flip the user is shown the resulting tracked-versus-ignored list. The gitignore does change, but abcd ahoy install --visibility public prints only that it kept abcd's private files out of git and the visibility change; --details adds the files written, and the --json result has no field for such a list.. Read that issue record for the source observation.

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
