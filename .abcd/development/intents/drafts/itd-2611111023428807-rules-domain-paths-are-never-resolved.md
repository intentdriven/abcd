---
id: itd-2611111023428807
slug: rules-domain-paths-are-never-resolved
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610300120250873]
origin: extracted-from-record
production_mode: hand-written
---

# A rules domain points at its home with a bare repository path, and nothing checks that the path still resolves. The link check reads markdown links only, and the one test that reads an OPINIONS path checks the users-directory principle alone. So renaming or moving a principle leaves its OPINIONS rule pointing at a file that is gone, the loader keeps injecting the dead pointer into every matching prompt, and every check stays green.

## Press Release

> _Seeded by promotion from iss-2610300120250873. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610300120250873`: A rules domain points at its home with a bare repository path, and nothing checks that the path still resolves. The link check reads markdown links only, and the one test that reads an OPINIONS path checks the users-directory principle alone. So renaming or moving a principle leaves its OPINIONS rule pointing at a file that is gone, the loader keeps injecting the dead pointer into every matching prompt, and every check stays green.. Read that issue record for the source observation.

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
