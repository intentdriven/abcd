---
id: itd-2611142303208114
slug: abcd-capture-help-and-docs-reference-cli
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251559033353]
origin: extracted-from-record
production_mode: hand-written
---

# abcd capture --help and docs/reference/cli/commands.md:492 render the remedy flag as --remedy abcd capture remedy the proposed fix, one line (required); abcd capture remedy rewrites it later. The usage string at internal/surface/cli/cli.go:5322 backquotes abcd capture remedy. pflag's UnquoteUsage takes the first backquoted span as the flag's value placeholder, so the placeholder reads as a command instead of string. This is the class iss-2608301212428844 resolved for --grounds. TestGroundsFlagUsageRendersAStringPlaceholder (internal/surface/cli/capture_surface_test.go:1220) pins only the three --grounds flags, so --remedy slipped through. It is the only such placeholder on the page.

## Press Release

> _Seeded by promotion from iss-2610251559033353. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251559033353`: abcd capture --help and docs/reference/cli/commands.md:492 render the remedy flag as --remedy abcd capture remedy the proposed fix, one line (required); abcd capture remedy rewrites it later. The usage string at internal/surface/cli/cli.go:5322 backquotes abcd capture remedy. pflag's UnquoteUsage takes the first backquoted span as the flag's value placeholder, so the placeholder reads as a command instead of string. This is the class iss-2608301212428844 resolved for --grounds. TestGroundsFlagUsageRendersAStringPlaceholder (internal/surface/cli/capture_surface_test.go:1220) pins only the three --grounds flags, so --remedy slipped through. It is the only such placeholder on the page.. Read that issue record for the source observation.

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
