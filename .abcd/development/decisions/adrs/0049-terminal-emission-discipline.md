---
id: adr-49
slug: terminal-emission-discipline
status: accepted
date: 2026-08-22
supersedes: null
superseded_by: null
related_intents: [itd-112, itd-2610030810350727]
related_rfcs: []
related_adrs: []
---

# ADR-49: Terminal emission discipline — decoration only on interactive TTYs, machine streams undecorated, untrusted text always sanitised

## Context

Until itd-112, abcd emitted no ANSI at all, and the termsafe primitive's
posture — ESC never reaches a render path — was total: every C0 control in
every rendered string is masked, because rendered text is so often
repo-controlled (commit subjects, refs, prose from possibly hostile or
archived repositories). A banner introduces the first *sanctioned* ANSI
emission: trusted, compiled-in art and attribute sequences the renderer
itself composes. Two rules were being carried as prose inside one intent —
the TTY-only trigger and the termsafe carve-out — and they are the same
boundary: what abcd may emit, onto which stream. A rule that binds every
decorated surface that follows (itd-110 is next) belongs in the decision
record, not in one intent's grill notes.

## Decision

1. **Decoration renders only on an interactive TTY.** ANSI colour,
   half-block or shade-block art, and any other ornament may be written
   only when stdout is an interactive terminal. Machine-consumed streams —
   non-TTY stdout, `--json` output, hook context injections, quiet modes —
   receive no decoration bytes, ever. A subcommand's output is
   machine-consumable by default; decoration is opt-in per surface at an
   explicit decision.
2. **Only trusted-static content may carry ANSI.** Compiled-in art (the
   livery grids and their ANSI tables), attribute sequences the renderer
   composes, and build-time-baked identity text are trusted. Everything
   read at runtime — repo files, environment-derived strings, anything a
   repository controls — remains untrusted and passes `termsafe.Sanitize`
   before it may join a rendered line, exactly as before this ADR.
3. **Colour honours the caller.** `--no-color` and the NO_COLOR convention
   (present and non-empty) strip colour without stripping content; a
   decorated surface degrades loudly to its undecorated form, never to
   blank output.

## Consequences

- The banner (itd-112) implements the first decorated surface under these
  rules and exports the colour ladder and TTY seam as the canonical
  primitives; later styled surfaces (itd-110 first) consume them rather
  than minting parallel copies (one-canonical-primitive).
- termsafe's contract is unchanged: this ADR narrows where its *output* may
  be dressed up, not what it masks.
- Tests for any decorated surface must include the machine-stream
  assertion: captured non-TTY output contains no escape bytes.
- Brief invariant 13 states the boundary; this ADR carries the rationale.

## Amendment 2026-10-03

itd-110 (the grill interview's styled rendering, named above as the next decorated surface) is superseded by itd-2610030810350727 (one question layout); the decorated plain-Terminal surface is itd-2610030810370060, which builds on itd-112. The related intent is re-pointed; the decision itself is unchanged.
