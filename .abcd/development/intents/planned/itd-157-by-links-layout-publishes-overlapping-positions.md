---
id: itd-157
slug: by-links-layout-publishes-overlapping-positions
spec_id: spc-50
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608231350127745
---

# The by-links arrangement's non-settling has a LAYOUT half that is still open, and it is a redesign rather than a fix. The renderer half is fixed (the collision resolver no longer demands 1.5 units of padding the published positions never promised, iss-2608231243286557), but the layout itself still publishes overlapping positions, so the chart still never comes to rest on that arrangement — measured 2026-08-23: by date comes to rest, by links does not. Two causes found and NOT fixed, because fixing them changes the picture: the islands are settled by a spring layout with no collision pass at all, and the two rim rows map each record's arc-width onto a circle whose circumference is smaller than the sum of those widths, so the rows are overpacked by construction. Attempted fixes measured: relaxing the linked records alone left 600 overlaps (the rest are on the rim); adding a rim-row growth rule left 367 and pushed the outermost radius from 0.97 to 1.83, visibly changing the picture without settling it. Both reverted. A correct fix places every record — islands, pairs and rim — under one packing rule, which is a design of the arrangement, not a patch to it.

## Press Release

> _Seeded by promotion from iss-2608231350127745. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608231350127745`: The by-links arrangement's non-settling has a LAYOUT half that is still open, and it is a redesign rather than a fix. The renderer half is fixed (the collision resolver no longer demands 1.5 units of padding the published positions never promised, iss-2608231243286557), but the layout itself still publishes overlapping positions, so the chart still never comes to rest on that arrangement — measured 2026-08-23: by date comes to rest, by links does not. Two causes found and NOT fixed, because fixing them changes the picture: the islands are settled by a spring layout with no collision pass at all, and the two rim rows map each record's arc-width onto a circle whose circumference is smaller than the sum of those widths, so the rows are overpacked by construction. Attempted fixes measured: relaxing the linked records alone left 600 overlaps (the rest are on the rim); adding a rim-row growth rule left 367 and pushed the outermost radius from 0.97 to 1.83, visibly changing the picture without settling it. Both reverted. A correct fix places every record — islands, pairs and rim — under one packing rule, which is a design of the arrangement, not a patch to it.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** the by-links arrangement produces node positions that overlap after its spring settle, **when** the post-settle collision-resolution pass runs, **then** the published layout separates the overlapping nodes and reports zero overlaps.
- **Given** the overlap gate, **when** it evaluates a rendered chart, **then** it measures the by-links layout, not only the by-date coil.
- **Given** a by-links layout that still contains overlapping positions, **when** the overlap gate runs, **then** it flags the by-links overlap as a red result.
- **Given** the by-date arrangement, which already comes to rest, **when** the overlap gate runs, **then** it continues to pass.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
