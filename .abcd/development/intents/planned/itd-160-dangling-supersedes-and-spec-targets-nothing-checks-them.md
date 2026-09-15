---
id: itd-160
slug: dangling-supersedes-and-spec-targets-nothing-checks-them
spec_id: spc-52
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608220150157498
---

# Eight typed cross-references point at targets absent from the tree — adr-22 supersedes adr-14, adr-15 and adr-17; adr-25 supersedes adr-8; adr-27 supersedes adr-16; adr-28 supersedes adr-18; adr-35 supersedes adr-4 (all retired under retire-the-name); itd-3 names spec_id spc-1 which has no file — and nothing checks supersedes targets today. The 2026-08-21 site investigation counted six; the in-session grep found eight. The planned site build arms the detector via the .abcd/site-baseline.json ratchet, seeded with what the build finds, and the tombstones-or-stubs question (itd-136/itd-137) decides how they render

## Press Release

> _Seeded by promotion from iss-2608220150157498. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608220150157498`: Eight typed cross-references point at targets absent from the tree — adr-22 supersedes adr-14, adr-15 and adr-17; adr-25 supersedes adr-8; adr-27 supersedes adr-16; adr-28 supersedes adr-18; adr-35 supersedes adr-4 (all retired under retire-the-name); itd-3 names spec_id spc-1 which has no file — and nothing checks supersedes targets today. The 2026-08-21 site investigation counted six; the in-session grep found eight. The planned site build arms the detector via the .abcd/site-baseline.json ratchet, seeded with what the build finds, and the tombstones-or-stubs question (itd-136/itd-137) decides how they render. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a record that introduces a new `supersedes` reference naming a record absent from the tree, **when** the site-baseline reference detector runs, **then** it fails as a red gate.
- **Given** a record whose `spec_id` names a `spc-N` that has no file, **when** the reference detector runs, **then** it fails as a red gate.
- **Given** the existing backlog of dangling supersedes and spec-target references, **when** it is seeded into `.abcd/site-baseline.json`, **then** those references are baselined and do not newly fail the gate.
- **Given** a dangling reference that has been baselined, **when** its target is later added to the tree, **then** the detector still passes and the reference no longer counts as dangling.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
