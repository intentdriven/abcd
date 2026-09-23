---
id: itd-133
shipped_in: v0.6.2
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

<!-- abcd-review: INGESTED receipt=rcp-5f9fa1fdfedf -->
Fidelity review — receipt rcp-5f9fa1fdfedf (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e2548bbe0e21d1d8fa893fef46326e6848b60155d02c12758dbc3d3ab7f8b2ac
Input attestations: diff:tree at de3ba5fa (spc-36 delivered; main after PR #661)@sha256:4e7430d38ef6b7b6bc533fdf0b8b32a6e08a3e2449d0566027d43316036b8178;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the seven grids live only in internal/livery/grids.go, Assets() hands out copies, the gen program renders the SVGs from them, and the in-sync test proves the committed SVGs derive from the grids and nothing else sits in the assets directory
  evidence: internal/livery/grids.go:9 — "'y': "#f0c052", // yellow — duckling body, delta flag"
  evidence: internal/livery/livery.go:33 — "func Assets() []Asset {"
  evidence: internal/livery/gen/main.go:1 — "Command gen writes the committed SVG identity assets"
  evidence: internal/livery/livery_test.go:202 — "func TestSVGAssetsInSync"
- ac-2 — MET: TestSVGAssetsInSync renders every asset in all four variants and compares byte-for-byte with docs/assets/img/livery/, failing on drift or on a stray file; it runs under go test ./... on both CI legs
  evidence: internal/livery/livery_test.go:202 — "func TestSVGAssetsInSync"
  evidence: internal/livery/livery_test.go:205 — "want[a.Name+".svg"] = RenderSVG(a, true)"
  evidence: .github/workflows/ci.yml:282 — "run: go test ./..."
- ac-3 — MET_WITH_CONCERNS: panel variants render a dark rounded panel behind the art and the transparent variants carry a < desc> stating 'for dark surfaces only', with the forge logo.png untouched; the concern is that legibility on a light background is a visual claim no test or gate measures — the evidence is the labelling and the panel, not a rendered check
  evidence: internal/livery/svg.go:16 — "PanelColor is the dark panel behind the panel variants."
  evidence: internal/livery/svg.go:101 — "parts = append(parts, "Transparent variant: for dark surfaces only.")"
  evidence: docs/assets/img/logo.png:0 — "forge/web logo unchanged"
- ac-4 — MET: TestFlagGeometry checks the strip and the 2x2 icon: alfa white hoist / blue fly with a swallowtail, bravo all red with a swallowtail, charlie five stripes b-w-r-w-b, delta three bands y-b-b-b-y, and asserts the compact variant is declared ApproximateGeometry while full and icon are not
  evidence: internal/livery/livery_test.go:131 — "func TestFlagGeometry"
  evidence: internal/livery/livery_test.go:145 — "if !compact.ApproximateGeometry {"
  evidence: internal/livery/livery_test.go:183 — "for i, want := range []rune{'b', 'w', 'r', 'w', 'b'} {"
  evidence: internal/livery/grids.go:80 — "ApproximateGeometry: true,"
- ac-5 — MET_WITH_CONCERNS: the role assignment is recorded in the decision log's 2026-08-21 entry (duckling mascot, signal-flag hoist as the terminal logo, lifeboat for the lifeboat verbs, forge logo kept); the concern is that the intent refers to 'the decision log' in prose only — there is no link to the entry, so discovery is by date
  evidence: .abcd/work/DECISIONS.md:1663 — "2026-08-21 — Visual identity roles (itd-133, maintainer-ruled at the planning"
  evidence: .abcd/development/intents/shipped/itd-133-abcd-has-a-face-a-block-pixel-duckling-is-the-mascot-the-off.md:55 — "place (maintainer-ruled 2026-08-21, recorded in the decision log)."
- ac-6 — MET: the launch payload config includes docs/, so the 28 SVGs under docs/assets/img/livery/ ship as assets; no committed file references them from a user-facing page or references the local scratch prototypes (the only scratch mention in the tree is an unrelated test fixture path)
  evidence: .abcd/config/launch-payload.json:8 — ""docs","
  evidence: docs/assets/img/livery/duckling.svg:1 — "< svg xmlns"
  evidence: internal/core/site/manifest_test.go:154 — ".abcd/.work.local/scratch/identity.md"

Gap audit:
- honoured:
  - one canonical pixel-grid definition of all seven assets in the Go tree
    evidence: internal/livery/livery.go:33 — "func Assets() []Asset {"
  - a Go generator deriving the committed SVGs, no other toolchain
    evidence: internal/livery/gen/main.go:1 — "Command gen writes the committed SVG identity assets"
  - a CI drift gate on byte-identical regeneration
    evidence: internal/livery/livery_test.go:202 — "TestSVGAssetsInSync"
    evidence: .github/workflows/ci.yml:282 — "run: go test ./..."
  - the role-assignment decision recorded in the decision log
    evidence: .abcd/work/DECISIONS.md:1663 — "Visual identity roles (itd-133"
- diverged:
  - the decision is linked from this intent — referenced in prose, not linked
    evidence: .abcd/development/intents/shipped/itd-133-abcd-has-a-face-a-block-pixel-duckling-is-the-mascot-the-off.md:55 — "recorded in the decision log"
- missing: (none)