---
id: spc-50
slug: by-links-layout-publishes-overlapping-positions
intent: itd-157
---
# by-links-layout-publishes-overlapping-positions

## Summary

Makes the record-graph's by-links arrangement come to rest and makes the overlap
gate measure it. The by-links layout settles islands with a spring layout that
has no collision pass and packs two rim rows onto a circle whose circumference is
smaller than the sum of the record arc-widths, so it publishes overlapping
positions by construction; and the overlap gate counts only the by-date coil, so
the overlap ships green. This spec adds one post-settle collision-resolution pass
covering islands, pairs and rim under a single packing rule, and widens the gate
to measure the by-links layout too.

## Scope

In:

- `internal/core/site/layout.go` — the `byLinks` builder (lines 366–469): a
  post-settle collision pass after island/pair/rim placement, and the rim-row
  packing so the circumference matches the summed arc-widths.
- The overlap counter, today computed only inside `coil()` (layout.go:332–339):
  measure `a.Links` as well, in one coordinate space.
- `internal/core/site/build.go` (the `Overlaps` result field, lines 299–301,
  465) and the human-facing print in `internal/surface/cli/site.go:197`.
- Tests in `layout_test.go` and `build_test.go`.

Out:

- The renderer half is already fixed (iss-2608231243286557 — the collision
  resolver no longer demands 1.5 units of padding); the JS chart emitted from
  `graphpage.go` is unchanged.
- The by-date coil (`coil()`, layout.go:264–360) already settles and is not
  touched beyond sharing the widened overlap counter.

## Approach

**The two causes.** `byLinks` (layout.go:366–469) settles islands via
`springLayout` (called line 418, Fruchterman-Reingold at 511–587) then
`decompress`/scale/place (419–422) with **no node-radius collision pass** — cause
one. The two rim rows (445–468, radii `rimRow0=0.905`/`rimRow1=0.97`) sum each
record's arc-width `w = 2*a.Radius[i] + coilGap` into `total` (456) and map each
onto a circle at radius `rr`, but the summed widths exceed the circumference at
that radius — overpacked by construction, cause two. Prior local patches
(relaxing linked records alone; a rim-row growth rule that pushed the outer
radius from 0.97 to 1.83) were measured and reverted because they changed the
picture without settling it.

**One packing rule.** After island, pair and rim placement (after line 468), run
a single post-settle collision-resolution pass over *all* by-links nodes
together — the coil already demonstrates the forbidden-interval idea
(layout.go:296–317). The pass iteratively separates any pair whose centre
distance is less than the sum of their published radii (plus the same epsilon the
coil uses), nudging both along their separation vector, bounded by an iteration
cap, until no pair overlaps. Because islands, pairs and rim are resolved in the
same pass and the same coordinate space, no region is packed against a promise a
later region breaks. The rim rows are seeded so their circumference is at least
the summed arc-widths (the row radius is derived from `total`, not fixed), so the
pass starts from a near-feasible configuration rather than an over-packed one and
converges without inflating the outer radius the way the reverted growth rule
did.

**Widen the gate, one coordinate space.** The overlap count lives only in `coil()`
(layout.go:332–339). Extract the counting loop into a helper
`countOverlaps(points []Point, radii []float64) int` and call it for both `a.Coil`
and `a.Links`, summing into `a.Overlaps` (Arrangements struct, layout.go:99–137).
The known trap is that node positions are unit-disk-normalised while radii are in
reference pixels (the mismatch that produced spurious overlaps in
iss-2608231322321751); the helper normalises both into one space before
comparing, so the count is real for by-links as it already is for by-date.
`build.go` (299–301, 465) and the CLI print (`site.go:197`) surface the combined
count as they do today.

## How it satisfies each acceptance criterion

- *By-links overlaps after the spring settle are separated; the published layout
  reports zero overlaps* — the post-settle pass over all nodes. Test: over the
  record corpus, build the by-links layout and assert `countOverlaps(a.Links,
  a.Radius) == 0` (a new `TestByLinksNeverOverlaps` mirroring
  `TestCoilNeverOverlaps`, layout_test.go:40).
- *The overlap gate measures the by-links layout, not only the by-date coil* —
  `a.Overlaps` now sums both arrangements. Test: `TestBuildLayoutDoesNotOverlap`
  (build_test.go:322) is extended to assert the count reflects `a.Links`.
- *A by-links layout that still contains overlaps flags red* — because the gate
  now counts `a.Links`, a residual overlap makes `a.Overlaps != 0`, which the
  CLI prints as a non-zero/red result (`site.go:197`). Test: inject an
  overlapping by-links fixture and assert `a.Overlaps > 0`.
- *The by-date arrangement continues to pass* — the coil's forbidden-interval
  packing is unchanged; the shared counter measures it exactly as before. Test:
  `TestCoilNeverOverlaps` stays green.

## Decisions

One packing rule over all node classes, chosen over per-region patches. The two
measured local fixes each moved the picture without settling it; the intent
resolved that a correct fix is a *design* of the arrangement, not a patch. The
single post-settle pass in one coordinate space is that design. The rim-row
radius becomes derived from the summed arc-widths rather than fixed, so the pass
converges without the outer-radius inflation the reverted growth rule caused.
