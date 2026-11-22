---
id: itd-2611181038548303
slug: abcd-development-brief-04-surfaces
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611040555394037]
origin: extracted-from-record
production_mode: hand-written
---

# .abcd/development/brief/04-surfaces/09-reflect.md:53-54 and 71-72 say `abcd reflect <release-tag>` renders each intent's honoured, diverged and missing gap counts, plus verdict and gap distributions and the dates of this tag and the previous one. The text render (internal/surface/cli/reflect.go:341, 368) shows the acceptance rollup per intent and a metrics line of shipped, audited, unaudited and this tag's date. The --json form carries the gaps, the rollup and, where there is a previous tag, its date (internal/core/reflect/seed.go:102-105).

## Press Release

> _Seeded by promotion from iss-2611040555394037. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611040555394037`: .abcd/development/brief/04-surfaces/09-reflect.md:53-54 and 71-72 say `abcd reflect <release-tag>` renders each intent's honoured, diverged and missing gap counts, plus verdict and gap distributions and the dates of this tag and the previous one. The text render (internal/surface/cli/reflect.go:341, 368) shows the acceptance rollup per intent and a metrics line of shipped, audited, unaudited and this tag's date. The --json form carries the gaps, the rollup and, where there is a previous tag, its date (internal/core/reflect/seed.go:102-105).. Read that issue record for the source observation.

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
