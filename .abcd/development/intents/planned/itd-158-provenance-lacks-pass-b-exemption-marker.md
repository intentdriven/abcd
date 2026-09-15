---
id: itd-158
slug: provenance-lacks-pass-b-exemption-marker
spec_id: spc-51
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-136
---

# itd-88's fidelity gap audit found a missing press-release claim: Pass B is promised to ship as a declared exemption in _provenance.json, never a silent gap, but no exemption field or marker exists anywhere in the lifeboat package or the Provenance struct — a promise with no implementing code, recorded in itd-88's Audit Notes (receipt rcp-4d07032fc6ab)

## Press Release

> _Seeded by promotion from iss-136. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-136`: itd-88's fidelity gap audit found a missing press-release claim: Pass B is promised to ship as a declared exemption in _provenance.json, never a silent gap, but no exemption field or marker exists anywhere in the lifeboat package or the Provenance struct — a promise with no implementing code, recorded in itd-88's Audit Notes (receipt rcp-4d07032fc6ab). Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a lifeboat package in which Pass B is exempt, **when** its `_provenance.json` is written, **then** the Provenance record carries an explicit Pass-B exemption marker rather than a silent gap.
- **Given** a Provenance record that carries the exemption marker, **when** the consumer reads it, **then** it recognises the record as exempt rather than treating it as an unmarked gap.
- **Given** a Provenance record that carries no exemption marker, **when** the consumer reads it, **then** it is treated exactly as before, as an unexempt record.
- **Given** the Provenance struct in the lifeboat package, **when** it is marshalled, **then** the exemption field is present, closing the promised-but-unimplemented claim recorded in itd-88's Audit Notes.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
