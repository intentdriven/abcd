---
id: itd-134
slug: every-abcd-managed-cli-greets-like-abcd
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-112]
severity: minor
impact: additive
---

# Every abcd-managed CLI greets like abcd does: one command scaffolds a banner for a managed repo, rendered from that repo's canonical identity block — name, tagline, version — as a committed, language-neutral artefact (the generator emits the rendered banner, not code), so a CLI in any language opens with its own identity without hand-drawing anything. Split out of itd-112 at its 2026-08-21 grill: itd-112 keeps abcd's own banner and the colour ladder; this intent carries the managed-repo generator and the per-repo accent question.

Typed links: `refines` itd-112 — the split parent keeps abcd's own banner
and the colour ladder; this intent carries the managed-repo generator and
the per-repo accent question.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Every abcd-managed CLI greets like abcd does: one command scaffolds a banner for a managed repo, rendered from that repo's canonical identity block — name, tagline, version — as a committed, language-neutral artefact (the generator emits the rendered banner, not code), so a CLI in any language opens with its own identity without hand-drawing anything. Split out of itd-112 at its 2026-08-21 grill: itd-112 keeps abcd's own banner and the colour ladder; this intent carries the managed-repo generator and the per-repo accent question.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md).

- **The colour contract is settled and language-neutral:** suppress colour when `NO_COLOR` is "present and not an empty string (regardless of its value)", flags and user configuration override it, and it covers "only ANSI color, not other styling" (no-color.org); no colour when stdout is not a TTY or `TERM` is `dumb` (clig.dev); "checking `$COLORTERM` is recommended" for truecolor, with the caveat that it "is not forwarded via sudo, ssh, etc." and programs "should fall back to using 8-bit color" (termstandard/colors). itd-112's shipped ladder already encodes this order.
- **ASCII art needs a text alternative:** WCAG technique H86 says ASCII art "can be very confusing to people who are accessing the internet using screen readers" and "should also have a text explanation of what the picture is". The words beside the art (name, version, tagline) are that alternative; the art must never carry meaning the words do not.
- **A rendered artefact beats generated code** for a CLI in any language; no standard for the file's shape was found, so the shape is abcd's to choose.
- **Screen-reader modes drop decoration wholesale** (the host prints no box characters and no colour-only cues in that mode); a managed CLI driven from a session emits no banner at all, as itd-112's rule already requires for a machine-consumed stream.

**Path 2 — native, shaped to the `NO_COLOR` and `COLORTERM` conventions: the generator emits rendered artefacts (plain text, 16-colour, 256-colour and 24-bit variants, since termstandard's fallback from truecolor is 8-bit colour, not 16) with the selection rule written beside them, never code.** The per-repo accent stays open; no convention for deriving a colour from a name was found in this pass. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Whether four committed renderings (plain, 16-colour, 256-colour, 24-bit; termstandard's advised fallback from truecolor is 8-bit colour) or one plain file plus a rule is the right shape for a repository whose CLI may be in any language; and whether the README drift check's marker-and-checksum shape should guard them.
- (SOTA 2026-10-23) Where the per-repo accent comes from: chosen by the owner at setup, derived from the project name, or absent in the first version.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
