---
id: itd-2611150522471756
slug: the-ahoy-install-yes-help-at-internal
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251559060083]
origin: extracted-from-record
production_mode: hand-written
---

# The ahoy install --yes help at internal/surface/cli/cli.go:4261 (docs/reference/cli/commands.md:135) says --yes excludes 'the optional git-identity pin, the status line and the model-tier routing tables, which need an answered prompt (run without --yes, or answer every prompt with: yes | abcd ahoy install)'. Under --yes the install also leaves the README question, the drain eligibility record and the removal of a conventions file that only repeats AGENTS.md: optionalGapIDs (internal/core/ahoy/apply.go:2359) lists them, and optionalSkipped returns every pending one under Yes (apply.go:2378-2381). Those three are asked only at a terminal (terminalOnlyGapIDs, apply.go:2366), and piped stdin is not a terminal (stdinPrompter.AtTerminal, cli.go:4766), so the advised yes | abcd ahoy install never answers them. The install's own result already says so (optionalSkipReason, cli.go:4597-4614). The drain-rule (a101c59d1) and conventions-file (c97f6807b) offers joined the list after the help was worded (db74fb520), and so did the README parts (13615ee16).

## Press Release

> _Seeded by promotion from iss-2610251559060083. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251559060083`: The ahoy install --yes help at internal/surface/cli/cli.go:4261 (docs/reference/cli/commands.md:135) says --yes excludes 'the optional git-identity pin, the status line and the model-tier routing tables, which need an answered prompt (run without --yes, or answer every prompt with: yes | abcd ahoy install)'. Under --yes the install also leaves the README question, the drain eligibility record and the removal of a conventions file that only repeats AGENTS.md: optionalGapIDs (internal/core/ahoy/apply.go:2359) lists them, and optionalSkipped returns every pending one under Yes (apply.go:2378-2381). Those three are asked only at a terminal (terminalOnlyGapIDs, apply.go:2366), and piped stdin is not a terminal (stdinPrompter.AtTerminal, cli.go:4766), so the advised yes | abcd ahoy install never answers them. The install's own result already says so (optionalSkipReason, cli.go:4597-4614). The drain-rule (a101c59d1) and conventions-file (c97f6807b) offers joined the list after the help was worded (db74fb520), and so did the README parts (13615ee16).. Read that issue record for the source observation.

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
