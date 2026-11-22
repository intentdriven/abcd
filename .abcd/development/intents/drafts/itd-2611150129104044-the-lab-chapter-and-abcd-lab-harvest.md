---
id: itd-2611150129104044
slug: the-lab-chapter-and-abcd-lab-harvest
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611040554295339]
origin: extracted-from-record
production_mode: hand-written
---

# The lab chapter and `abcd lab harvest --help` say every product finding is listed with the capture line that files it (.abcd/development/brief/04-surfaces/31-lab.md:124). The line built at internal/core/lab/harvest.go:135 has no --remedy, and `abcd capture` refuses it with exit 2. harvest.md says the flag must be added before the command runs; the chapter, the help and the verb's terminal listing do not.

## Press Release

> _Seeded by promotion from iss-2611040554295339. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611040554295339`: The lab chapter and `abcd lab harvest --help` say every product finding is listed with the capture line that files it (.abcd/development/brief/04-surfaces/31-lab.md:124). The line built at internal/core/lab/harvest.go:135 has no --remedy, and `abcd capture` refuses it with exit 2. harvest.md says the flag must be added before the command runs; the chapter, the help and the verb's terminal listing do not.. Read that issue record for the source observation.

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
