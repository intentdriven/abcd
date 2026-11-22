---
id: itd-2611112051084859
slug: abcd-implement-log-s-long-help-at
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211483734]
origin: extracted-from-record
production_mode: hand-written
---

# abcd implement log's Long help at internal/surface/cli/implement.go:503-508 (page lines 2468-2474) overstates and understates its checks. It says the claim, window and session events are refused, but the load event and the four outage events are refused too (internal/core/implement/log.go:84-89,405-407). It says 'an at or last_productive is an RFC 3339 time, and a *_min or minutes field a number', but those fields are checked only on the events whose rules name them (intervention, decision, stop, agent_end, ceiling_overrun and backoff: log.go:126-141,168-200,457-466); lane_close --field minutes=abc or gate_run --field at=yesterday is written as given.

## Press Release

> _Seeded by promotion from iss-2610252211483734. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211483734`: abcd implement log's Long help at internal/surface/cli/implement.go:503-508 (page lines 2468-2474) overstates and understates its checks. It says the claim, window and session events are refused, but the load event and the four outage events are refused too (internal/core/implement/log.go:84-89,405-407). It says 'an at or last_productive is an RFC 3339 time, and a *_min or minutes field a number', but those fields are checked only on the events whose rules name them (intervention, decision, stop, agent_end, ceiling_overrun and backoff: log.go:126-141,168-200,457-466); lane_close --field minutes=abc or gate_run --field at=yesterday is written as given.. Read that issue record for the source observation.

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
