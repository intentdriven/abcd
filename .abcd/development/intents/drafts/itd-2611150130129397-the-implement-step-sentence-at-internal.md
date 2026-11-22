---
id: itd-2611150130129397
slug: the-implement-step-sentence-at-internal
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211452294]
origin: extracted-from-record
production_mode: hand-written
---

# The implement step sentence at internal/core/surface/sentences.go:221-222, rendered as the verb's one-line help and at docs/reference/cli/commands.md:2788, says the step 'refuses a push with no preflight receipt'. Since 13b7bd1aa (ruling DR6d-2) a landing whose head no preflight receipt names waits instead: a contention that holds only that lane, the call moving the others and exiting 3 only when nothing else moves (internal/core/implement/loop/land.go:439-452). The step's own Long help says so (internal/surface/cli/build.go:834-837), so the sentence and the paragraph under it disagree; an agent reading only the sentence treats a waiting lane as refused.

## Press Release

> _Seeded by promotion from iss-2610252211452294. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211452294`: The implement step sentence at internal/core/surface/sentences.go:221-222, rendered as the verb's one-line help and at docs/reference/cli/commands.md:2788, says the step 'refuses a push with no preflight receipt'. Since 13b7bd1aa (ruling DR6d-2) a landing whose head no preflight receipt names waits instead: a contention that holds only that lane, the call moving the others and exiting 3 only when nothing else moves (internal/core/implement/loop/land.go:439-452). The step's own Long help says so (internal/surface/cli/build.go:834-837), so the sentence and the paragraph under it disagree; an agent reading only the sentence treats a waiting lane as refused.. Read that issue record for the source observation.

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
