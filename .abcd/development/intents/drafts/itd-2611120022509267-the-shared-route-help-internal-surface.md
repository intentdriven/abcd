---
id: itd-2611120022509267
slug: the-shared-route-help-internal-surface
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251559163141]
origin: extracted-from-record
production_mode: hand-written
---

# The shared --route help (internal/surface/cli/route.go:86-91) says 'each invocation dispatches one', and it renders for disembark principles, press-release and review (docs/reference/cli/commands.md:1033, 1052, 1083). Run without its payload flag, each of those verbs runs its deterministic mode and dispatches no agent (delegatedAgent, route.go:73-78). It then refuses --route with exit 2 (route.go:115-120). Only graveyard, whose --lessons-json is required, always dispatches one.

## Press Release

> _Seeded by promotion from iss-2610251559163141. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251559163141`: The shared --route help (internal/surface/cli/route.go:86-91) says 'each invocation dispatches one', and it renders for disembark principles, press-release and review (docs/reference/cli/commands.md:1033, 1052, 1083). Run without its payload flag, each of those verbs runs its deterministic mode and dispatches no agent (delegatedAgent, route.go:73-78). It then refuses --route with exit 2 (route.go:115-120). Only graveyard, whose --lessons-json is required, always dispatches one.. Read that issue record for the source observation.

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
