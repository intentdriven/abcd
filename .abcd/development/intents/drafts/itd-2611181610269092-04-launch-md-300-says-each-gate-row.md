---
id: itd-2611181610269092
slug: 04-launch-md-300-says-each-gate-row
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611040554433915]
origin: extracted-from-record
production_mode: hand-written
---

# 04-launch.md:300 says each gate row carries a tier, 04-launch.md:385 says two rows report without one (citation baseline, semantic receipts), and lines 305-368 list the secret scan, merge hold and installability smoke as hard-fail gates. abcd launch --dry-run --json gives no tier on five rows: secret+pii-scan, installability-smoke, merge-hold, citation-baseline and semantic-receipts (internal/core/launch/dryrun.go:153,165; internal/core/launch/mergehold.go:112).

## Press Release

> _Seeded by promotion from iss-2611040554433915. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611040554433915`: 04-launch.md:300 says each gate row carries a tier, 04-launch.md:385 says two rows report without one (citation baseline, semantic receipts), and lines 305-368 list the secret scan, merge hold and installability smoke as hard-fail gates. abcd launch --dry-run --json gives no tier on five rows: secret+pii-scan, installability-smoke, merge-hold, citation-baseline and semantic-receipts (internal/core/launch/dryrun.go:153,165; internal/core/launch/mergehold.go:112).. Read that issue record for the source observation.

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
