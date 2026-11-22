---
id: itd-2611180919442124
slug: the-implement-receipt-long-help-at
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211466719]
origin: extracted-from-record
production_mode: hand-written
---

# The implement receipt Long help at internal/surface/cli/build.go:1112-1114 calls the implementer receipt's resolves list optional. On an issue run (build of an issue, or a drain lane) the receipt is refused unless resolves names the lane's issue (internal/core/implement/loop/receipt.go:405-407, added in 2cec1e367), and an implementer may hand the lane back with a handback object instead of commits, which ends the lane (receipt.go:69-73,398-400). The help mentions neither; it renders at docs/reference/cli/commands.md:2659-2661.

## Press Release

> _Seeded by promotion from iss-2610252211466719. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211466719`: The implement receipt Long help at internal/surface/cli/build.go:1112-1114 calls the implementer receipt's resolves list optional. On an issue run (build of an issue, or a drain lane) the receipt is refused unless resolves names the lane's issue (internal/core/implement/loop/receipt.go:405-407, added in 2cec1e367), and an implementer may hand the lane back with a handback object instead of commits, which ends the lane (receipt.go:69-73,398-400). The help mentions neither; it renders at docs/reference/cli/commands.md:2659-2661.. Read that issue record for the source observation.

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
