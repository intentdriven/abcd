---
id: itd-2611180754469036
slug: abcd-guard-check-help-mirrored-in-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610141706042163]
origin: extracted-from-record
production_mode: hand-written
---

# abcd guard check --help (mirrored in the generated docs/reference/cli/commands.md) says an allow still does not see a parameter expansion with no substitution ($VAR, ${VAR:-git}) wherever it stands, because the guard sees the variable, not its expansion; since the carried-value reading the guard does read a value the same line assigns and uses: P=pkill; $P make and X=rm-payload; sh -c "$X" both block. The help under-states the guard (it fails safe).

## Press Release

> _Seeded by promotion from iss-2610141706042163. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610141706042163`: abcd guard check --help (mirrored in the generated docs/reference/cli/commands.md) says an allow still does not see a parameter expansion with no substitution ($VAR, ${VAR:-git}) wherever it stands, because the guard sees the variable, not its expansion; since the carried-value reading the guard does read a value the same line assigns and uses: P=pkill; $P make and X=rm-payload; sh -c "$X" both block. The help under-states the guard (it fails safe).. Read that issue record for the source observation.

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
