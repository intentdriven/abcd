---
id: spc-41
slug: bare-abcd-opens-with-a-generated-banner
intent: itd-112
---
# The bare-invocation banner: baked identity, half-block livery art, and the exported colour ladder

## Summary

spc-41 delivers itd-112: a bare `abcd` on an interactive TTY greets with a
generated banner — the true-geometry flag hoist from the livery grids in
half-block pixels on the painted panel colour, `abcd vX.Y.Z` beside it,
tagline and next-action hints beneath — above the unchanged status board.
Machine-consumed streams receive no decoration bytes, per adr-49 and brief
invariant 13. The build ships two new exported primitives (the colour
ladder and the TTY seam) and the identity bake, all drift-gated.

## Scope

In: the banner composition and its wiring at the CLI root; the identity
bake (generator → Go constant, drift test, itd-102 surface registration);
the colour ladder (truecolor/256/16/mono with the ruled precedence) and
stdout TTY seam as exported primitives; pinned 256/16 ANSI tables in
`internal/livery` with parity tests; the root-local `--no-color` flag; a
CHANGELOG entry.

Out (per the intent): the managed-repo generator (itd-134); styling other
surfaces (itd-110 consumes the primitives later); terminal-width
detection; Windows (no shipped binary).

## Approach

- **Identity bake.** A generator (the livery/gen pattern) reads the
  canonical identity block and writes title + tagline into a generated Go
  file; a test regenerates and byte-compares (drift gate), and the
  generated file is registered as a `regexp` positioning surface so
  `abcd identity` sees banner drift. The banner never reads the cwd's
  block — an installed abcd in a foreign repo must not wear that repo's
  tagline. Baked text is trusted; nothing on the banner path reads
  repo-controlled bytes at runtime, so termsafe's contract is untouched.
- **ANSI tables beside the palette.** `internal/livery` gains pinned
  256- and 16-colour tables keyed identically to the hex palette, with a
  key-set parity test; truecolor renders `38;2;r;g;b` straight from hex.
  The 'k' substitution rule (TransparentEyeColor) is honoured by the
  terminal renderer exactly as documented on `Palette()`.
- **Pure renderer, thin emitter.** Grid → composed-lines rendering
  (half-block pairing, render-time padding of the five-row strip to even
  height, panel background painting, glyph selection for transparent
  halves) is a pure function returning `[]string`, testable without a
  terminal; the CLI root owns the emission decision. The canonical grids
  are never mutated (itd-133's geometry and drift gates stay green).
- **Ladder and seam.** Colour detection is a pure resolver over an explicit
  environment map — table-tested for the ruled precedence (`--no-color` >
  NO_COLOR present-and-non-empty > TERM dumb/unset > COLORTERM > 256 > 16)
  — and the TTY check is one exported stdout-facing seam (a package-var
  seam per the repo's established pattern), consolidating the two existing
  hand-rolled stdin checks rather than adding a third copy. Both are
  exported, banner-independent; itd-110 carries `builds_on: [itd-112]`.
- **Wiring.** The banner renders only in the root command's bare-invocation
  path, gated on args-empty, not-`--json`, and the TTY seam; one test
  asserts the status board's bytes are unchanged when the banner is off,
  and one asserts captured non-TTY output contains no escape bytes (the
  adr-49 machine-stream assertion). Mono renders the five-row shade-block
  form; a non-UTF-8 locale gets the text lines only.
- **Version.** The version segment renders the build-stamped
  `core.Version`; `"dev"`/empty renders `abcd (dev build)`. No repo read.

## Acceptance criteria → how satisfied

1. *Banner over unchanged board* — root-path wiring plus the
   board-bytes-unchanged test; fixed ≤66-column layout.
2. *Machine streams clean* — the TTY seam defaults off under test; the
   no-escape-bytes assertion runs on captured non-TTY output.
3. *Ladder precedence + mono art* — table-tested resolver; mono renders
   the shade-block form; locale check falls back to text-only.
4. *Baked identity + dev build* — generator, drift test, positioning
   surface registration; version render rule.
5. *Tables beside palette, grids immutable* — parity test; render-time
   padding; itd-133 gates untouched.
6. *Records and edges* — adr-49 + invariant 13 (landed with the planning
   interview); ladder/seam exported; itd-110 `builds_on` written.
