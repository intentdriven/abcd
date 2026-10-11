---
id: spc-39
slug: maya-sees-how-every-decision-connects
intent: itd-137
---
# maya-sees-how-every-decision-connects-one-relationship-chart

## Summary

spc-39 delivers itd-137's two views of the record over time: the
relationship chart at `/record/graph/` — one chart, two build-time
arrangements, ego focus, the card with the date continuum and viewing
history — and the five-lane genealogy at `/record/timeline/` as one static
SVG. The layouts are ported from `build_data.py` (the reference
implementation); the runtime behaviour is lifted from the prototype's CSS
tokens and JavaScript into `site-src/`, not rewritten from memory. On the
itd-140 boundary both pages are **generic-side**: they consume
`record.json` and nothing else.

## Settled constraints

- **Both arrangements are precomputed at build time** into `record.json`,
  deterministic across rebuilds of the same tree: the coil is
  `build_data.py`'s greedy packer; the by-links view is a seeded spring
  embedding with the hub-dampening and radial decompression the script
  records. Ported in-repo, stdlib-only — an external layout library is a
  new dependency under the sign-off gate and is not assumed.
- **Every record is a circle**; colour names only the three durable types;
  issues are light grey, principles and the rest ink; lifecycle is fill;
  size is link count. Arrowheads only on `builds_on`, `supersedes`,
  `implements`; mirrored pairs collapse; body mentions off by default.
- **Dashed stubs, never invented positions**: supersession arcs draw only
  where both ends exist; a target absent from the tree gets a dashed stub
  ending in ×, consistent with the baseline stubs ruling.
- **Amendment traceability** (interview ruling): the card's last-touched
  date links the record's commit history on GitHub.
- **Perfect at 390 px or removed from the phone view**; full behaviour
  under `prefers-reduced-motion` is: no animation, nothing drifts at rest,
  the list view offered in place of the chart.

## Mechanism

### Build-time layouts (in `internal/core/site`, shipped in `record.json`)

- `layout.go` ports the coil: records placed in effective-date order
  (frontmatter date, else first-commit date; ties by type then id), each
  next bubble beside the last, a little further round and as close in as
  placed bubbles allow; month zones recorded as the first record of each
  month; an overlap sanity count must be zero. Radii follow the script's
  formula; positions normalised to the unit disk.
- The by-links arrangement: connected components over typed links only;
  islands of three or more laid out by the seeded spring embedding with
  degree-weighted springs, percentile radial decompression, the largest
  island centred; pairs ringed outside it; unlinked records on the rim in
  date order, spaced by bubble size. Seeds are fixed constants; a golden
  test pins the output for a fixture tree.
- The genealogy SVG is emitted at build: five lanes (releases, decisions,
  intents, specs, issues) over one axis; releases from CHANGELOG headings
  linking their GitHub release; decisions at frontmatter dates; the rest
  at first-commit dates from the same git pass as the card's continuum;
  crowded days become capsules with counts; issues render as a per-day
  histogram (resolved solid, open outlined); supersession arcs and dashed
  ×-stubs as ruled; the first-commit marker is a dashed vertical; every
  mark links `/record/graph/?focus=<id>`.

### Runtime (lifted from the prototype into `site-src/`)

- The prototype's CSS token system (light base, dark redefinition, both
  `prefers-color-scheme` and explicit `data-theme`) and its chart script
  are lifted into `site-src/` assets and adapted only where the hash
  router becomes real URLs: the canvas stage, tap-to-focus ego view with
  ring-and-fade, drag/pan, wheel/pinch/double-click zoom with corner
  buttons and reset, the explicit two-state arrangement control, the
  Filters pop (type chips, mentions toggle, legend), search with hit
  list, full screen with the fixed-overlay fallback, the settling pass
  behind the "Stand by…" overlay, and the corner discipline — controls in
  the corners, the middle belongs to the chart.
- The card: title; type and state pills in the GitHub palette; every date
  of the record on one continuum (frontmatter diamond, in-tree dot, state
  ring, last touched) spoken as one sentence to screen readers; the
  linked-records pull-out phrased from the focused record; the GitHub
  link; the last-touched date links the record's commit history; back and
  forward buttons and Alt+arrows walk the viewing history, a fresh pick
  dropping the forward trail. Anchored bottom-left, growing upward to the
  control row; on phones a full-width bottom sheet with the chart panning
  to keep the focused bubble visible.
- "Browse as a list" beneath the stage is the keyboard path and
  accessible twin, reaching every record and every link the chart shows.

## Acceptance-criteria mapping

- AC 1 (two arrangements by an explicit control, precomputed,
  deterministic, ported from the reference scripts, no new dependency) →
  build-time layouts.
- AC 2 (mirrored pairs collapse; arrowheads only on directed relations;
  mentions off by default) → `record.json` edges + chart rendering.
- AC 3 (the card: pills, continuum, phrased pull-out, GitHub link,
  history-linking last-touched, back/forward) → the card.
- AC 4 (keyboard visitor reaches everything via the list twin) → list
  twin.
- AC 5 (five-lane static SVG; one git pass; capsules; arcs only where
  both ends exist; dashed ×-stubs) → genealogy SVG.
- AC 6 (`prefers-reduced-motion` removes animation; list offered) →
  runtime behaviour.
- AC 7 (390 px bottom sheet, pan-to-visible, no horizontal scroll) →
  mobile rules + static checks + CI screenshots.

## Out of scope

- Tombstone files for retired ADRs (ruled out; dashed stubs only).
- Body-mention edges on by default; a fourth categorical hue.
- Any runtime layout simulation beyond the glide-and-settle pass.
