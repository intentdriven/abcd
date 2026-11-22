---
id: itd-2611112317424912
slug: spec-close-production-mode-carries-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211524459]
origin: extracted-from-record
production_mode: hand-written
---

# spec close --production-mode carries the shared productionModeFlagHelp, 'how this record's text was produced ... (default: the repo's declared mode, else hand-written)' (internal/surface/cli/cli.go:3350, wired at cli.go:3930, docs/reference/cli/commands.md:4773). On spec close the flag only stamps the follow-on spec that --remainder mints, and without --remainder it is refused with exit 2 (cli.go:3813-3815). The flag help says neither.

## Press Release

> _Seeded by promotion from iss-2610252211524459. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211524459`: spec close --production-mode carries the shared productionModeFlagHelp, 'how this record's text was produced ... (default: the repo's declared mode, else hand-written)' (internal/surface/cli/cli.go:3350, wired at cli.go:3930, docs/reference/cli/commands.md:4773). On spec close the flag only stamps the follow-on spec that --remainder mints, and without --remainder it is refused with exit 2 (cli.go:3813-3815). The flag help says neither.. Read that issue record for the source observation.

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
