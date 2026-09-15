---
id: itd-161
slug: record-lint-cannot-see-a-decision-shaped-document-filed-outs
spec_id: spc-53
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608230752354926
---

# record-lint cannot see a decision-shaped document filed outside the record stores

## Press Release

> _Seeded by promotion from iss-2608230752354926. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608230752354926`: record-lint cannot see a decision-shaped document filed outside the record stores. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a markdown file filed outside the configured record stores that asserts a record id already held by a real record (for example a heading `# ADR-23` with `Status: Accepted` reusing a taken adr id), **when** record-lint runs, **then** the cross-store detector flags the outside-store id claim.
- **Given** the probe file `research/notes/zz-recurrence-probe.md` reusing the taken id ADR-23, **when** record-lint runs, **then** it exits non-zero with a finding, where before the change it exited 0 with zero findings.
- **Given** a legitimate record filed inside its own record store, **when** record-lint runs, **then** the detector does not flag it.
- **Given** a grandfathered undated Phase 0 note, **when** the detector runs, **then** it does not fire on the filename alone, being weighed against the record baseline.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
