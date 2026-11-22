---
id: itd-2611112318017321
slug: disembark-probe-plan-and-pack-share-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251559204442]
origin: extracted-from-record
production_mode: hand-written
---

# disembark probe, plan and pack share one --include-ignored help (internal/surface/cli/cli.go:1332; docs/reference/cli/commands.md:1002, 1020, 1070), and it promises '(widens the scan; the report says so)'. probe's coverage report prints 'scope: WIDE' (internal/core/lifeboat/coverage.go:114-116), and pack writes that report into the lifeboat (plan.go:226-230). plan's output, text and --json alike, is the PlanManifest (plan.go:402-410; RenderManifest 434-450), which carries no scope, so abcd disembark plan --include-ignored says nothing of the wider scan.

## Press Release

> _Seeded by promotion from iss-2610251559204442. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251559204442`: disembark probe, plan and pack share one --include-ignored help (internal/surface/cli/cli.go:1332; docs/reference/cli/commands.md:1002, 1020, 1070), and it promises '(widens the scan; the report says so)'. probe's coverage report prints 'scope: WIDE' (internal/core/lifeboat/coverage.go:114-116), and pack writes that report into the lifeboat (plan.go:226-230). plan's output, text and --json alike, is the PlanManifest (plan.go:402-410; RenderManifest 434-450), which carries no scope, so abcd disembark plan --include-ignored says nothing of the wider scan.. Read that issue record for the source observation.

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
