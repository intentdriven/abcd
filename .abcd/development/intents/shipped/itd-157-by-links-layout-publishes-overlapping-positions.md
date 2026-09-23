---
id: itd-157
shipped_in: v0.6.8
slug: by-links-layout-publishes-overlapping-positions
spec_id: spc-50
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608231350127745
impact: fix
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

<!-- abcd-review: INGESTED receipt=rcp-fa9525fd4a02 -->
Fidelity review — receipt rcp-fa9525fd4a02 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:468ddfd94faf3b08ecbb62d2eacc64f700bf59c91e29e6aa0f6ccfe591a97f49
Input attestations: diff:328a6755^1..328a6755 (PR #555, commit 64ac809c), judged against the tree at 0ab2ad02@sha256:89e0509cf85b299ae37028c2b7bcd9c34a5d84bb3a3da97b72c7abd97aa57e40;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: byLinks seeds islands, pairs and a wound rim in the renderer's space, then runs settle() over every node as the last pass; TestByLinksNeverOverlaps asserts zero over corpora of 1..400 and TestSettleClearsAnyOverlap proves the pass clears a seed that does overlap; the real record builds with 0 overlaps
  evidence: internal/core/site/layout.go:748 — "settle(pos, a.Radius, all, false)"
  evidence: internal/core/site/layout.go:641 — "rr := math.Max(middle+widest+coilGap, total/(2*math.Pi))"
  evidence: internal/core/site/layout_test.go:204 — "func TestByLinksNeverOverlaps"
  evidence: internal/core/site/layout_test.go:290 — "func TestSettleClearsAnyOverlap"
- ac-2 — MET: the published count is overlapCount(), the sum of countOverlaps over a.Coil and a.Links taken at CoilRadius scale; TestOverlapGateMeasuresBothArrangements and TestBuildLayoutDoesNotOverlap re-derive the count from each arrangement and assert the published number is their sum
  evidence: internal/core/site/layout.go:225 — "func (a *Arrangements) overlapCount() int"
  evidence: internal/core/site/layout.go:237 — "func countOverlaps(points []Point, radii []float64, scale float64) int"
  evidence: internal/core/site/layout_test.go:217 — "func TestOverlapGateMeasuresBothArrangements"
  evidence: internal/core/site/build_test.go:346 — "if l.Overlaps != coil+links {"
- ac-3 — MET_WITH_CONCERNS: an injected by-links overlap makes overlapCount() non-zero (TestOverlappingByLinksFlagsRed) and the CLI prints the number; concern: no build or check gate refuses on it — site build exits 0 with the count in its summary and site check never reads Layout.Overlaps, so the only red is the test suite over fixture and synthetic corpora, not a user's record (iss-2609231010077584)
  evidence: internal/core/site/layout_test.go:268 — "func TestOverlappingByLinksFlagsRed"
  evidence: internal/core/site/layout_test.go:278 — "if got := a.overlapCount(); got == 0 {"
  evidence: internal/surface/cli/site.go:201 — "layout: %d overlapping bubbles across both arrangements"
  evidence: internal/core/site/check.go:264 — "func Check(req CheckRequest) (CheckResult, error)"
- ac-4 — MET: the coil's forbidden-interval packing is unchanged beyond sharing clearOutward, and TestCoilNeverOverlaps still asserts zero over the same corpora with the shared counter; the suite passes at 0ab2ad02
  evidence: internal/core/site/layout_test.go:44 — "func TestCoilNeverOverlaps"
  evidence: internal/core/site/layout_test.go:48 — "if got := countOverlaps(a.Coil, a.Radius, a.CoilRadius); got != 0 {"
  evidence: internal/core/site/layout.go:330 — "rho := clearOutward(math.Max(0, prho-inwardDip*need), ux, uy, r, placed, pos, pr, scratch)"

Gap audit:
- honoured:
  - every by-links record — islands, pairs and rim — is placed under one packing rule in one coordinate space and the published layout has no overlapping pair
    evidence: internal/core/site/layout.go:748 — "settle(pos, a.Radius, all, false)"
    evidence: internal/core/site/layout_test.go:204 — "func TestByLinksNeverOverlaps"
  - the rim-row radius is derived from the summed arc-widths rather than fixed, and the pair offset from the two radii
    evidence: internal/core/site/layout.go:641 — "rr := math.Max(middle+widest+coilGap, total/(2*math.Pi))"
    evidence: internal/core/site/layout.go:616 — "half := (a.Radius[c[0]] + a.Radius[c[1]] + coilGap) / 2"
  - the count is taken in the renderer's space, closing the unit-disk-versus-pixels trap
    evidence: internal/core/site/layout.go:237 — "func countOverlaps(points []Point, radii []float64, scale float64) int"
    evidence: internal/core/site/layout_test.go:232 — "TestOverlapCountIsTakenInTheRenderersSpace"
  - the arrangement settles inside the stage rather than by inflating the outer radius
    evidence: internal/core/site/layout_test.go:253 — "func TestByLinksStaysOnTheStage"
- diverged:
  - the overlap gate flags a by-links overlap as a red result — delivered as a non-zero printed count and a test-suite assertion; site build and site check both stay green on a non-zero count
    evidence: internal/surface/cli/site.go:201 — "layout: %d overlapping bubbles across both arrangements"
    evidence: internal/core/site/check.go:264 — "func Check(req CheckRequest) (CheckResult, error)"
- missing: (none)