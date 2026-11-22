---
id: itd-2611121103148104
slug: the-implement-loop-package-s-plain-test
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610302058287983]
origin: extracted-from-record
production_mode: hand-written
---

# The implement loop package's plain test time on the ubuntu CI runner, run as one process, grew from 466s to 593s across recent runs and then crossed 600s; internal/surface/cli ran at 400 to 550s. The goroutine dumps show the time going to git process spawns in the state and tracked-file reads. CI's plain step now shards both packages as make preflight does and declares a 40-minute package budget (iss-2610302058222905), but the packages keep growing, and the margin narrows again at this rate.

## Press Release

> _Seeded by promotion from iss-2610302058287983. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610302058287983`: The implement loop package's plain test time on the ubuntu CI runner, run as one process, grew from 466s to 593s across recent runs and then crossed 600s; internal/surface/cli ran at 400 to 550s. The goroutine dumps show the time going to git process spawns in the state and tracked-file reads. CI's plain step now shards both packages as make preflight does and declares a 40-minute package budget (iss-2610302058222905), but the packages keep growing, and the margin narrows again at this rate.. Read that issue record for the source observation.

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
