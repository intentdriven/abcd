---
id: itd-2610230856403803
slug: a-dependent-build-lane-waits-for-its
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610230856272073]
origin: extracted-from-record
production_mode: hand-written
---

# A build run's dependent lane cannot start until its predecessor has merged into the default branch: readiness reads a predecessor as landed only when its lane is done, its pull request an ancestor of the default branch, so a step that needs another waits a whole preflight and merge-queue cycle (about one to two and a half hours each on 2026-10-22) before it opens. Stacked changes are standard practice for dependent work (Graphite, Sapling and ghstack stacks, and merge queues taking a stack in order), and on 2026-10-22/23 that shape was rolled by hand twice: merge-gate steps 2 and 3, and model-matrix step 2 stacked on the route branch. Building it is more than a fix: the dependent lane's pull request is based on the predecessor's branch, the stack merges in order, and a lane rebases (and re-stamps what it resolved) when the predecessor changes.

## Press Release

> _Seeded by promotion from iss-2610230856272073. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610230856272073`: A build run's dependent lane cannot start until its predecessor has merged into the default branch: readiness reads a predecessor as landed only when its lane is done, its pull request an ancestor of the default branch, so a step that needs another waits a whole preflight and merge-queue cycle (about one to two and a half hours each on 2026-10-22) before it opens. Stacked changes are standard practice for dependent work (Graphite, Sapling and ghstack stacks, and merge queues taking a stack in order), and on 2026-10-22/23 that shape was rolled by hand twice: merge-gate steps 2 and 3, and model-matrix step 2 stacked on the route branch. Building it is more than a fix: the dependent lane's pull request is based on the predecessor's branch, the stack merges in order, and a lane rebases (and re-stamps what it resolved) when the predecessor changes.. Read that issue record for the source observation.

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
