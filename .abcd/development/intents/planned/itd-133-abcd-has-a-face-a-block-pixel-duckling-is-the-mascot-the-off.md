---
id: itd-133
slug: abcd-has-a-face-a-block-pixel-duckling-is-the-mascot-the-off
spec_id: spc-36
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
related_intents: [itd-112, itd-102]
severity: minor
impact: additive
---

# abcd has a face: one pixel-grid identity — duckling mascot, signal-flag logo, lifeboat mark — generated, drift-gated, ready for every surface

Typed links: `refines` itd-112 — supplies the small colour logo its banner
leaves open, and forecloses its object-vs-text-logo open question
(maintainer-ruled 2026-08-21: the object is the flag hoist); `refines`
itd-102 — extends the canonical identity from words to marks; the identity
block schema itself is untouched (extension deferred, see Open Questions).

## Press Release

abcd now has a face, and it is the same face everywhere. A block-pixel
duckling — ducks in a row — is the mascot. The official logo of the terminal
surfaces spells a-b-c-d in international maritime signal flags — alfa, bravo,
charlie, delta — with true geometry at full size, swallowtails and all. A
small lifeboat marks the lifeboat verbs. All of it derives from one
pixel-grid source of truth in the Go tree: a generator emits the committed
SVG assets, a drift gate proves they never diverge, and any surface that
renders the identity draws from the same grids. Alice sees the same duckling
in their terminal that Bob sees on a web page, pixel for pixel, because there
is only one duckling to see.

## Why This Matters

The canonical identity block (itd-102) keeps abcd's words consistent —
title, tagline, pitch — but the project has no visual identity at all beyond
a placeholder image. Art is what makes a CLI recognisable at a glance, and
the banner intent (itd-112) is blocked on exactly this: it needs a small
colour logo to compose. Shipping the assets as one drift-gated source
prevents the failure mode this repo guards against everywhere else — two
hand-maintained copies quietly diverging — and gives itd-112 something
solid to build on.

## Prior Art

- itd-112 (drafts/) — the generated banner; owns all terminal behaviour
  (ANSI rendering, colour detection, `--no-color`/quiet degradation) and
  consumes these assets.
- itd-102 (shipped, spc-19) — the canonical identity block and its drift
  check; this intent is its visual sibling and touches none of its schema.
- `docs/assets/img/logo.png` — the existing forge/web logo; it remains in
  place (maintainer-ruled 2026-08-21, recorded in the decision log).
- Reference prototypes in the local work tier (`.abcd/.work.local/scratch/
  identity/`): a shell ANSI renderer (256-colour, 16-colour, and escape-free
  mono modes) and an SVG generator, both proving the grids render legibly;
  they retire when the Go implementation lands and are referenced by no
  committed file.

## What's In Scope

- One canonical pixel-grid definition of all seven assets (duckling full
  and mini; flag logo strip, 2×2 icon, and compact; lifeboat full and mini) in the Go tree.
- A Go generator deriving the committed SVG assets (dark-panel and
  transparent variants) from the grids; no other toolchain.
- A CI drift gate: regenerating the SVGs from the grids is byte-identical.
- The role-assignment decision recorded in the decision log.

## What's Out of Scope

- All terminal rendering behaviour — the banner, ANSI emission, colour
  detection, fallbacks, quiet modes (itd-112).
- Forge/web page rewiring — the README keeps the existing logo for now.
- Identity-block schema extension for per-repo visual assets (deferred to
  itd-112 planning).

## Scope Conditions

None stated.

## Acceptance Criteria

Adopted by the maintainer 2026-08-21 at the planning interview; criteria 3
and 6 amended in-interview to keep the forge/web logo unchanged for now.

- Given the repository, when Alice inspects the identity assets, then one
  canonical pixel-grid definition exists in the Go tree and every rendered
  artefact (the committed SVGs included) derives from it — no second
  hand-maintained copy.
- Given the committed SVGs, when CI regenerates them from the grids, then
  the output is byte-identical; any drift fails the gate.
- Given the generated SVG assets, when viewed on light and on dark
  backgrounds, then every asset remains fully legible — the panel variants
  on any background, the transparent variants documented as dark-surface
  only. No forge page rewiring ships in this intent.
- Given the full-size logo, when checked against the ICS flag specification,
  then alfa, bravo, charlie, delta geometry is correct — vertically halved
  white/blue swallowtail alfa, all-red swallowtail bravo, five-stripe
  charlie, three-band delta; given the compact variant, then it is labelled
  approximate and carries no geometry claim.
- Given the role assignment — duckling is the mascot, the signal-flag hoist
  is the official logo of the terminal surfaces, the lifeboat marks the
  lifeboat verbs — when Carol consults the record, then the decision is
  discoverable in its durable home (the decision log), linked from this
  intent.
- Given a cut release, when the bundler runs, then the committed SVG assets
  ship via `docs/` as assets referenced by no user-facing page yet, while
  every scratch prototype stays in the untracked local tier, referenced by
  no committed file.

## Open Questions

- Deferred to itd-112 planning: how the itd-102 identity block schema
  extends so a managed repo can declare its own visual assets (itd-112
  already carries the storage question).
- Deferred to itd-112 planning: what the plugin markdown surface receives
  when a banner would render (ANSI art is noise on that surface).
- Resolved at spec build: the asset namespace avoids a third "identity"
  homonym (the git-author gate and the positioning block already share
  nothing but the word) — the spec names the package.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
