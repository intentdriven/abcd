---
id: itd-2611111008070880
slug: gitutil-runwithindex-and-gitutil
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610312044545639]
origin: extracted-from-record
production_mode: hand-written
---

# The auto-plan's checkout digest runs its git commands with no deadline: gitutil.RunWithIndex and gitutil.StreamWithIndex (internal/gitutil/index.go) and, above all, checkoutDigest's `add -A` through pickGitIndex (internal/core/implement/loop/autoplan.go about line 207, pickcommit.go about line 167), which runs the person's configured content filters. Unlike the bounded runner in internal/gitutil/bounded.go, a stalled filter or git holds the digest, and the implement step taking it, with no limit.

## Press Release

> _Seeded by promotion from iss-2610312044545639. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610312044545639`: The auto-plan's checkout digest runs its git commands with no deadline: gitutil.RunWithIndex and gitutil.StreamWithIndex (internal/gitutil/index.go) and, above all, checkoutDigest's `add -A` through pickGitIndex (internal/core/implement/loop/autoplan.go about line 207, pickcommit.go about line 167), which runs the person's configured content filters. Unlike the bounded runner in internal/gitutil/bounded.go, a stalled filter or git holds the digest, and the implement step taking it, with no limit.. Read that issue record for the source observation.

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
