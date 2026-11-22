---
id: itd-2611112035118466
slug: the-statusline-sentence-writes-nothing
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211502540]
origin: extracted-from-record
production_mode: hand-written
---

# The statusline sentence 'Writes nothing; never refuses.' (internal/core/surface/sentences.go:416-417, shown at docs/reference/cli/commands.md:4829) is false for an operand. The command declares Args: cobra.NoArgs (internal/surface/cli/statusline.go:137), so abcd statusline x is refused with a usage error, and decideStatus returns an error when the working directory cannot be read (statusline.go:221).

## Press Release

> _Seeded by promotion from iss-2610252211502540. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211502540`: The statusline sentence 'Writes nothing; never refuses.' (internal/core/surface/sentences.go:416-417, shown at docs/reference/cli/commands.md:4829) is false for an operand. The command declares Args: cobra.NoArgs (internal/surface/cli/statusline.go:137), so abcd statusline x is refused with a usage error, and decideStatus returns an error when the working directory cannot be read (statusline.go:221).. Read that issue record for the source observation.

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
