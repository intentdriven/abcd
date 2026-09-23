---
id: spc-36
slug: abcd-has-a-face-a-block-pixel-duckling-is-the-mascot-the-off
intent: itd-133
---
# One pixel-grid identity: the livery package, its generator, and the drift-gated SVG assets

## Summary

spc-36 delivers itd-133's visual identity as data plus derivation: a new
`internal/livery` package holds the canonical pixel grids for all seven
assets (duckling full and mini; signal-flag logo full, compact, and the 2×2
icon arrangement — two flags per row, naturally square, the app/web icon
candidate, maintainer-requested 2026-08-21; lifeboat full and mini) and
their palette; a Go generator derives the committed SVG assets from
those grids; and a package test regenerates the SVGs on every `go test` run
and fails on any byte difference. No terminal rendering ships here — itd-112
consumes the grids later. No forge page is rewired — `docs/assets/img/
logo.png` remains the forge/web logo.

The package is named **livery** — a vessel's livery is its distinctive
colours and markings — deliberately avoiding a third homonym on "identity"
(the git-author gate and the positioning block already share nothing but the
word), which resolves the intent's third open question.

## Scope

In: `internal/livery` (grids, palette, SVG derivation), a `go:generate`
program writing `docs/assets/img/livery/*.svg` — seven assets × two canvases
(natural, and square for avatar/icon use; maintainer-requested 2026-08-21) ×
two variants (panel, transparent), 28 files — the in-sync test, geometry
tests, a CHANGELOG entry.

Out (per the intent): ANSI/terminal rendering, colour detection, fallbacks
(itd-112); README/web changes; identity-block schema changes.

## Approach

- **Grids as Go source.** Each asset is a `[]string` of single-character
  cells (one colour key per cell) in `internal/livery/grids.go`, with the
  palette as a `map[rune]` of hex colours. Exported read-only accessors give
  itd-112 its future input. A well-formedness test asserts every row of a
  grid has equal width and every cell is palette-known.
- **Generator.** `internal/livery/gen` is a small program invoked via
  `go generate ./internal/livery/...`; it renders each grid to SVG
  (`<rect>` per cell, `shape-rendering="crispEdges"`) in two variants:
  *panel* (rounded dark panel behind the art — legible on any background)
  and *transparent* (dark surfaces only; its `<desc>` says so), each on two
  canvases: natural, and square with the art centered (the side is the larger
  natural dimension, so nothing is cropped or scaled; the `<desc>` labels it
  for avatar and icon use). The compact
  logo's SVG carries a `<desc>` stating its geometry is approximate, and the
  grids' doc comments repeat both labels. Output is deterministic (fixed
  ordering, no timestamps), which is what makes the drift gate byte-exact.
- **Drift gate as a test.** `TestSVGAssetsInSync` renders every variant
  in-memory and byte-compares against the committed files under
  `docs/assets/img/livery/`, failing with the asset name on mismatch. Living
  in `go test ./...`, it runs in `make preflight` and both CI legs with no
  new workflow step.
- **Geometry as a test.** `TestFlagGeometry` asserts the full-size flag
  grids against the ICS patterns: alfa vertically halved white/blue with a
  swallowtail notch, bravo all-red with a swallowtail notch, charlie's five
  horizontal stripes blue-white-red-white-blue, delta's three bands
  yellow-blue-yellow. The same assertions run over the icon arrangement's
  four flags at their 2×2 offsets. The compact grid is exempt by design and
  asserted only to carry its approximate label.

## Acceptance criteria → how satisfied

1. *One canonical definition, no second copy* — the grids exist only in
   `internal/livery/grids.go`; the SVGs are generated from them and the
   in-sync test proves derivation; the scratch prototypes stay untracked and
   unreferenced.
2. *Byte-identical regeneration* — `TestSVGAssetsInSync`, running in
   preflight and CI.
3. *Legible light and dark* — panel variants carry their own background;
   transparent variants are labelled dark-surface-only in their `<desc>`
   and doc comments; no forge page rewiring.
4. *True geometry full-size, labelled-approximate compact* —
   `TestFlagGeometry` plus the compact variant's labels.
5. *Role assignment discoverable* — recorded in `.abcd/work/DECISIONS.md`
   (2026-08-21 entry), linked from the intent's Prior Art and criteria.
6. *Ships via docs/, scratch stays local* — assets land under
   `docs/assets/img/livery/`, already inside the launch payload's `docs/`
   inclusion; the bundler's namespace denial keeps `.abcd/**` out
   structurally; no committed file references the local tier.

## Notes

The SVG palette is the 256-colour-mode palette's hex equivalents (terracotta
`#d97757`, yellow `#f0c052`, red `#cf554d`, blue `#4a6fb5`, water `#2b7a8c`,
gold `#d9a441`, white `#e8e8e8`, mast grey `#9aa0a8`, panel `#1c1f26`); when
itd-112 builds terminal rendering it maps the same palette keys to ANSI, so
the grids stay the single source across both pipelines.
