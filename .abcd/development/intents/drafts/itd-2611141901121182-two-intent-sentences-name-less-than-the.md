---
id: itd-2611141901121182
slug: two-intent-sentences-name-less-than-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211476480]
origin: extracted-from-record
production_mode: hand-written
---

# Two intent sentences name less than the verb writes. intent audit (internal/core/surface/sentences.go:242-243; page line 3061) says 'Writes an OWED stub only for --owed or an id', but --issue-drift leaves a report under .abcd/.work.local/logs/audit/issue-drift-<ts>/ (internal/core/capture/drift.go:68-73,179), and on a provider route audit <itd-N> sends the request and ingests the verdict into the shipped intent's Audit Notes (internal/surface/cli/cli.go:3446-3448, dispatch.go:179-211). intent consistency (sentences.go:248-249; page line 3123) says 'Writes the request locally', but on a provider route it sends each request and ingests the answers, writing the dated review and one capture per finding (internal/surface/cli/intent_consistency.go:105-121). Both sentences predate provider dispatch (51c7a9744). Same class as iss-2610251559136898, which holds five other sentences.

## Press Release

> _Seeded by promotion from iss-2610252211476480. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211476480`: Two intent sentences name less than the verb writes. intent audit (internal/core/surface/sentences.go:242-243; page line 3061) says 'Writes an OWED stub only for --owed or an id', but --issue-drift leaves a report under .abcd/.work.local/logs/audit/issue-drift-<ts>/ (internal/core/capture/drift.go:68-73,179), and on a provider route audit <itd-N> sends the request and ingests the verdict into the shipped intent's Audit Notes (internal/surface/cli/cli.go:3446-3448, dispatch.go:179-211). intent consistency (sentences.go:248-249; page line 3123) says 'Writes the request locally', but on a provider route it sends each request and ingests the answers, writing the dated review and one capture per finding (internal/surface/cli/intent_consistency.go:105-121). Both sentences predate provider dispatch (51c7a9744). Same class as iss-2610251559136898, which holds five other sentences.. Read that issue record for the source observation.

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
