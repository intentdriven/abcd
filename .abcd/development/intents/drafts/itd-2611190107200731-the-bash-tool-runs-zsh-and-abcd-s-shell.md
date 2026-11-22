---
id: itd-2611190107200731
slug: the-bash-tool-runs-zsh-and-abcd-s-shell
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611040337241546]
origin: extracted-from-record
production_mode: hand-written
---

# The Bash tool runs zsh, and abcd's shell guidance does not say so: a variable before a colon is mangled, PIPESTATUS is empty so a tail's exit is read as the command's, and an interactive rm alias hung a call for two minutes, all on 2026-11-04.

## Press Release

> _Seeded by promotion from iss-2611040337241546. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611040337241546`: The Bash tool runs zsh, and abcd's shell guidance does not say so: a variable before a colon is mangled, PIPESTATUS is empty so a tail's exit is read as the command's, and an interactive rm alias hung a call for two minutes, all on 2026-11-04.. Read that issue record for the source observation.

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
