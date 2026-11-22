---
id: itd-2611151011135445
slug: docs-explanation-artefacts-md-19-says-an
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251558404922]
origin: extracted-from-record
production_mode: hand-written
---

# docs/explanation/artefacts.md:19 says an intent is filed as shipped once its acceptance criteria are verifiably met. That holds on the build loop's closing lane, where an unmet or undecided audit fails the round (internal/core/implement/loop/validate.go:129,186-205). A spec closed by hand ships the intent first and the audit follows; an unmet criterion then leaves the intent in shipped/ with an owed check and an issue (internal/core/intent/audit_owed.go:9-21, ruling DQ1c).

## Press Release

> _Seeded by promotion from iss-2610251558404922. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251558404922`: docs/explanation/artefacts.md:19 says an intent is filed as shipped once its acceptance criteria are verifiably met. That holds on the build loop's closing lane, where an unmet or undecided audit fails the round (internal/core/implement/loop/validate.go:129,186-205). A spec closed by hand ships the intent first and the audit follows; an unmet criterion then leaves the intent in shipped/ with an owed check and an issue (internal/core/intent/audit_owed.go:9-21, ruling DQ1c).. Read that issue record for the source observation.

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
