---
id: itd-112
slug: bare-abcd-opens-with-a-generated-banner
spec_id: spc-41
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-133]
related_intents: [itd-102, itd-134]
severity: minor
impact: additive
---

# A bare abcd on an interactive terminal opens with a generated banner: the flag hoist in half-block colour on its painted panel, version beside it, tagline beneath — identity baked in at build time, full colour ladder, never a byte on a machine-consumed stream

Typed links: `builds_on` itd-133 — the livery grids and palette are the
banner's art source; `refines` itd-102 — the text renders from the canonical
identity block, baked into the binary at build time behind a drift gate.
itd-134 carries the managed-repo generator half of the original capture and
declares its `refines` edge itself. (Slug renamed from the capture slug
2026-08-22 after the split; the retired slug is banned per retire-the-name.)

## Press Release

Alice opens a terminal and types `abcd`. Before the status board they already
know, the tool now greets them: the a-b-c-d signal-flag hoist in colour —
half-block pixels on their painted panel, true flag geometry — with
`abcd v0.6.0` beside it, the tagline underneath, and two highlighted
next-action commands. It reads correctly on a light terminal and a dark one,
in tmux, and over SSH. When Bob pipes `abcd` into a file, greps a CI log, or
reads a hook's context injection, there is no banner and no escape byte —
machine surfaces stay exactly as they were. When Carol builds from source,
their unstamped build says `abcd (dev build)` and never pretends to be a
release. The banner is generated, not hand-drawn: the art comes from the
livery grids, the words from the canonical identity block baked in at build
time, and a drift gate holds both.

## Why This Matters

A bare `abcd` today opens straight into the status board — correct, but
faceless. The identity block (itd-102) keeps the words canonical and the
livery package (itd-133) holds the marks; nothing yet composes them into
the first thing a person actually sees. The banner is that composition:
identity rendered, never hand-drawn, topping the existing output without
replacing it — and the colour ladder it builds becomes the exported,
banner-independent primitive later styled surfaces (itd-110 first) consume
instead of reinventing.

## Reference Designs (observed 2026-08-15, maintainer's terminal)

Two live exemplars from the maintainer's private toolchain set the bar
(described generically; private tool names stay out of the record):

- **Wordmark style**: a block-ASCII wordmark with a per-letter colour
  gradient, followed by a plain monochrome pitch paragraph and standard
  CLI help. Identity lives in the wordmark alone; everything below stays
  quiet.
- **Object style**: a small pictorial colour object (~6 lines, ~18 columns)
  in the left column; the right column carries `name vX.Y.Z` in a warm
  accent, a two-line tagline in muted grey, then closing hint lines with
  runnable commands highlighted inline. Object as brand, tagline as
  orientation, hints as the next action.
- Common properties preserved: the banner reads correctly with colour
  stripped (never blank), and the bare invocation stays useful — the banner
  tops the existing status output, it does not replace it. The exemplars'
  ~50-column budget is deliberately relaxed here (interview ruling below):
  the real tagline is 66 characters and truncating it lost to laying it
  full-width beneath the strip.

## Decisions (grilled 2026-08-21, interview rulings 2026-08-22)

- **Scope split (2026-08-21):** itd-112 ships abcd's own banner and the
  terminal colour stack only. The managed-repo generator is itd-134.
- **Trigger (2026-08-21):** bare invocation on an interactive TTY (stdout)
  only. Non-TTY, CI, hooks, `--json`, quiet modes, and every subcommand
  stay banner-free — the plugin/model surface never receives banner bytes.
- **Identity source (2026-08-22):** title and tagline are baked into the
  binary at build time — a generator writes them from the identity block
  into a Go constant with a drift test, and the generated file is
  registered as an itd-102 positioning surface. The banner never reads the
  cwd's identity block (an installed abcd in a foreign repo must not wear
  that repo's tagline); runtime text therefore stays trusted.
- **Strip form (2026-08-22):** half-block pixels (two pixel rows per text
  row, 23 columns × 3 rows) painted over the livery panel colour — the
  terminal analogue of the SVG panel — which resolves both half-block
  transparency and light-theme erasure of the white pixels. The five-row
  strip is padded to an even height at render time; canonical grids are
  never mutated.
- **Layout (2026-08-22):** strip left with `abcd vX.Y.Z` beside it; tagline
  and next-action hints full-width beneath; total width ≤66 columns, fixed
  (no terminal-width detection, no new dependency).
- **Colour ladder (2026-08-21/22):** truecolor straight from the livery hex
  palette, then the pinned 256- and 16-colour tables (promoted into livery
  beside the palette with a key-set parity test), then mono. Precedence:
  `--no-color` > `NO_COLOR` (present and non-empty) > `TERM` dumb/unset >
  `COLORTERM` > 256-colour `TERM` > 16. The ladder ships as an exported,
  banner-independent primitive.
- **`--no-color` shape (2026-08-22):** root-local flag — colour exists only
  on the bare invocation, so no inert persistent flag lands on every
  subcommand.
- **Mono form (2026-08-22, revising the 2026-08-21 shade-block ruling's
  geometry):** mono renders the art as five-row shade-block glyphs — the
  prototyped ░▒▓█ form — since half-blocks cannot carry an image without
  colour. Art-always survives; the mono banner is taller than the colour
  one. UTF-8 is assumed for all art rungs.
- **Palette (2026-08-21):** the fixed livery house palette. Per-repo
  accents belong to itd-134.
- **Version (2026-08-22):** stamped builds render `abcd vX.Y.Z` from the
  build-time version; unstamped builds render `abcd (dev build)`; the
  banner never reads the repo to discover a version.
- **Object vs text-logo** (the capture's open question): foreclosed by the
  itd-133 decision — the object is the flag hoist.

## What's In Scope

- The banner composition on bare interactive invocation, topping the
  unchanged status board.
- The identity bake: generator, Go constant, drift test, positioning-surface
  registration.
- The colour ladder and TTY seam as exported primitives (the repo's two
  hand-rolled stdin TTY checks consolidate here; the banner checks stdout).
- The pinned ANSI tables in livery with parity tests.
- The emission-discipline record: one ADR plus one brief invariant
  (decoration only on interactive TTYs; machine-consumed streams
  undecorated; untrusted text always sanitised; trusted-static art may
  carry ANSI).

## What's Out of Scope

- The managed-repo banner generator and per-repo accents (itd-134).
- Styling any other surface (itd-110 consumes the ladder later).
- Terminal-width detection and responsive layout.
- Windows terminals: "any terminal" means the shipped release targets
  (darwin/linux); abcd ships no Windows binary.

## Scope Conditions

None stated.

## Acceptance Criteria

Adopted by the maintainer 2026-08-22 at the planning interview.

- Given an interactive TTY and a bare `abcd` invocation, when Alice runs
  it, then the banner renders above the status board — the true-geometry
  flag hoist in half-block pixels on the painted panel colour,
  `abcd vX.Y.Z` beside it, tagline and next-action hints beneath, within 66
  columns — and the status board's own bytes are unchanged from today's
  output.
- Given a non-TTY stdout, `--json`, a hook invocation, or any subcommand,
  when Bob captures the output, then it contains no banner bytes and no
  ANSI escapes from the banner path — asserted in tests through the
  injected TTY seam.
- Given the colour ladder, when the environment declares truecolor,
  256-colour, 16-colour, `TERM` dumb or unset, `NO_COLOR` present and
  non-empty, or the root-local `--no-color` flag, then the banner renders
  at exactly the rung that precedence selects — and the mono rung still
  renders the art as five-row shade-block glyphs, never blank and never
  coloured; a non-UTF-8 locale receives the text lines only.
- Given a release build, when the banner renders its words, then the title
  and tagline come from a Go constant generated from the canonical identity
  block, a drift test fails on any divergence, and the generated file is
  registered as an itd-102 positioning surface; given Carol's unstamped
  source build, then the version segment renders `(dev build)`.
- Given the livery package, when the banner needs ANSI colour, then
  truecolor derives from the hex palette, the pinned 256- and 16-colour
  tables live beside it with a key-set parity test, and the render-time
  padding of the five-row strip never mutates a canonical grid (the itd-133
  geometry and drift gates stay green).
- Given the emission-discipline ADR and brief invariant, when itd-112
  ships, then both are recorded, the colour ladder and TTY seam are
  exported banner-independent primitives, and itd-110 carries
  `builds_on: [itd-112]` so the second styled surface consumes rather than
  rebuilds them.

## Open Questions

All resolved at the 2026-08-22 interview: the termsafe carve-out routes to
the emission-discipline ADR + brief invariant (see Decisions); Windows is
scoped out by the release matrix; the Unicode floor is ruled (UTF-8 assumed
for art, text-only otherwise).

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
