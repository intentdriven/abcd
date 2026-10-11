---
id: spc-37
slug: iris-opens-abcdev-app-and-understands
intent: itd-135
---
# iris-opens-abcdev-app-and-understands-who-abcd-is-for-what-t

## Summary

spc-37 delivers itd-135's landing page: the README→docs migration that puts
every source span into the repository first, the `abcd site build` composition
that renders `/` from `.abcd/site.json` under the single-source rule, and the
`abcd site check` gates that fail the build on unsourced text, drifted
snippets, banned tokens and static mobile violations. The composition rules
are ported from `compose.py` in the migration bundle — the executable spec —
not reinvented. On the itd-140 boundary, everything here is **specific-side**:
opt-in composition declared in this repo's `.abcd/site.json`.

## Settled constraints

- **Single-source rule (adr-47 decisions 2–3).** No text is written for the
  site; every span is selected by path and heading through `.abcd/site.json`;
  the only added words are `site-src/ui.json` interface strings plus numbers,
  dates, file names and asset names. Condensed views are countable-only.
- **Repo-first (interview ruling).** The migration lands as drafted: all
  migrated text enters the repository exactly as the site selects it, before
  the site may select it. The working copy under
  `.abcd/work/site-plan/readme-migration/` is deleted in the same change.
- **The hero is verified by `abcd site check`, not by positioning** (itd-135
  AC 1): the positioning surfaces check committed files and cannot carry a
  build-time surface; the hero selector in `.abcd/site.json` names the
  Identity block and the check compares the rendered hero against it.
- **No new dependencies without sign-off.** The generator is stdlib-only: a
  Markdown-subset renderer, a strict tokenizer over the generator's own HTML,
  and the layout maths are written in-repo and pinned by golden tests. The
  library alternative is recorded in the ledger for a maintainer decision.
- **No Node in the build; no runtime API calls; no analytics, scripts or
  trackers** (adr-47, adr-48, interview ruling).

## Mechanism

### Migration (lands first, on its own PR)

- Promote the bundle to real paths: `docs/explanation/{rationale,roles,
  artefacts,process}.md`, `docs/how-to/install.md`, the two section indexes,
  the six referenced assets under `docs/assets/img/` (the third role
  portrait stays out until a docs page references it), `.abcd/site.json`,
  `site-src/ui.json`, and the slim `README.md` (tagline `<p>` stays the first
  `<p>` so the `readme-strapline` surface keeps resolving). Delete
  `.abcd/work/site-plan/readme-migration/` including `MIGRATION.md` and
  `_sources/`.
- `.abcd/site.json` gains one key the draft lacks: the explicit working-tier
  opt-in (`"record": {"issue_ledger": true}`) ruled in the interview.
- `site-src/redirects` lands as the committed 301 map for the docs URLs on
  today's sitemap (the root stays unmapped — adr-47 gives `/` to the landing
  page). The `mkdocs.yml` `site_url` flip to `https://abcdev.app/docs/`, the
  `site/docs/` output move and the `_redirects` emission land together with
  the deploy workflow, so canonical URLs never precede the serving move.
- `site-src/install.sh.tmpl` enters with the migration: the universal
  one-liner's logic as a script template (no generator wiring yet — spc-40
  serves it). A Go surface test asserts README's one-liner, the template and
  install.md's per-OS forms share the same release URLs, the same checksum
  step and the same install path, with only OS detection resolved.
- docs-lint gates the moved text under `docs/` exactly as it gated README.

### Composition (`abcd site build`)

- New package `internal/core/site`; front door `internal/surface/cli/site.go`
  registered in the root command block, plus `commands/site.md` on the plugin
  surface (commands index, surface registry row, regenerated `surface.json`
  and CLI reference land in the same change).
- `manifest.go` parses `.abcd/site.json` (schema_version 1, unknown fields
  rejected); `ui.go` loads `site-src/ui.json` as the closed allowlist.
- `sections.go` and `mdrender.go` port `compose.py`'s `sections`/`blocks`/
  `slug` walk and render the Markdown subset the corpus uses (headings,
  paragraphs, emphasis, links, images, fenced code, tables, blockquotes,
  lists); anything outside the subset is a loud build error, never silent
  passthrough.
- `compose.go` ports the four layouts (`cards-from-h2`, `lead-in-cards`,
  `prose`, `install`) and the hero: eyebrow/tagline/pitch from the Identity
  block via `positioning.ParseBlock` (the canonical parser), H1 and first
  paragraph from `rationale.md`, the figure from the page's first image.
  Every rendered block carries `data-src="path#heading"`.
- The featured quote is the newest shipped intent whose Audit Notes verdict
  is MET, by entered-`shipped/` date from the one git pass, id descending as
  tie-break — derived, never pinned (interview ruling).
- The Beta badge renders while the latest CHANGELOG version has major 0,
  via the changelog package's dated-heading reader; no copy change at v1.
- SVG assets are inlined (their `var(--token, fallback)` colours follow the
  theme); rasters are copied verbatim — optimisation is deferred to a
  ledger-recorded dependency decision, and the build never draws.
- The page head, footer tag/commit line, `_redirects` and `_headers` are
  emitted from committed inputs; footer content is file names, links and
  build metadata only.

### Checks (`abcd site check`)

- **Provenance**: a strict tokenizer walks the generated HTML; every visible
  text node must sit inside a `data-src` element or match a `ui.json` string,
  a number, a date, a file name or an asset name; failures name the node.
- **Banned tokens over composed text**: a new exported seam in
  `internal/core/lint` (`LintText`-shaped, compiled from the docs-lint
  config) runs over every composed span, whichever tree it came from —
  scoped exactly as adr-47 decision 3 states: composed surfaces only.
- **Snippet pinning**: every `abcd …` fenced block on `/` must appear in the
  generated CLI reference or the check fails naming the stale snippet.
- **Hero-vs-Identity**: the rendered hero's three spans must equal the
  Identity block's title, tagline and pitch.
- **Static mobile checks**: viewport meta present, wide elements inside
  `overflow-x` containers, `max-width` on images — the rendered-overflow
  screenshot audit is CI's optional job, not the binary's.
- Loop-figure labels: every label in `process-loop.svg` is a phrase on
  `process.md` (ported from `compose.py`'s closing assertion).

## Acceptance-criteria mapping

- AC 1 (Identity change re-renders the hero; check verifies) → hero
  composition + hero-vs-Identity check.
- AC 2 (every text node sourced or allowlisted; failures named) →
  provenance check.
- AC 3 (four chapters from their four pages; the only testimonial is the
  newest shipped MET intent, verbatim) → composition layouts + featured
  quote derivation.
- AC 4 (snippets match the CLI reference) → snippet pinning.
- AC 5 (docs-lint banned tokens over rendered text from any tree) →
  banned-token seam.
- AC 6 (Beta badge is a rule on the release version) → changelog-driven
  badge.
- AC 7 (390 px: no horizontal scroll; static checks in the binary,
  screenshots in CI) → static mobile checks.
- AC 8 (every picture a committed asset; SVGs inlined with token colours) →
  migration assets + inline/copy pipeline.

## Out of scope

- The record explorer pages, `record.json` and the baseline ratchet
  (spc-38), the chart and genealogy (spc-39), serving `/install.sh`
  (spc-40).
- The deploy workflow (adr-48 plumbing, recorded in the brief).
- Raster optimisation and an OpenGraph crop asset (ledger-recorded,
  maintainer decisions).
- Any genericity claim for `abcd site build` (itd-140 rule 3; itd-139 holds
  the demonstration and stays in drafts).
