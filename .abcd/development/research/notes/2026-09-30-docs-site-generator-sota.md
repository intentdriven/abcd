# SOTA survey — the generator that renders `docs/`: Zensical, Hugo with Hextra, or the status quo

Dated 2026-09-30. One host-run research pass (web, primary sources: the
projects' own documentation, package indexes and release pages, each read on
2026-09-30), challenged for fit per [`prefer-sota`](../../principles/prefer-sota.md)
against this repository's stated preferences. It is the research ruling M8
(2026-09-23) asked for before the product thinker chooses the generator as an
ADR, and it serves iss-2608220150157503.

**Why this note.** `docs/` renders through MkDocs with the Material theme
(`mkdocs-material==9.7.7`, pinned in `docs/requirements.txt`). The earlier
site plan ([`../abcdev-site/plan.md`](../abcdev-site/plan.md), Phase 5)
deferred the generator choice to an ADR "before Material's maintenance window
closes (~November 2026)", naming Zensical and Hugo with Hextra as the
candidates and overrides, tags and search as the needs. adr-47 keeps every
generated page outside the docs generator, so the migration touches only
`mkdocs.yml`, the build command and the site workflow.

## What `docs/` actually needs today

Measured at the lane base, not assumed:

- 14 markdown pages under `docs/` in four Diátaxis folders.
- `mkdocs.yml` is 35 lines: `theme: material` with a light/dark palette
  toggle, three theme features (`navigation.sections`, `navigation.expand`,
  `search.suggest`), and three markdown extensions (`admonition`,
  `pymdownx.superfences`, `tables`).
- No `plugins:` block, no `custom_dir` or `overrides/` directory, no tags, no
  hooks. The plan's "overrides, tags" needs are not present in the tree; search
  is the only feature beyond rendering that the site relies on.
- The build runs in `.github/workflows/site.yml` (`pip install`, then
  `mkdocs build -d site/docs`); `lint site` excludes `docs/`, so no gate in
  preflight depends on the generator.

## The ecosystem on 2026-09-30

| Fact | Evidence |
| --- | --- |
| MkDocs 1.x: last release 1.6.1 on 2024-08-30 | PyPI release history (package-index tier) |
| MkDocs 2.0 is in development releases (2.0.dev6, 2026-09-15) and drops plugins | PyPI; Material's 2026-02-18 post "What MkDocs 2.0 means for your documentation projects" (vendor tier) |
| Material for MkDocs: maintenance mode since 2025-11-05, "critical bug fixes and security updates for 12 months at least", no new features; last release 9.7.7 on 2026-07-17 | Material blog (vendor tier); PyPI |
| **Material's critical maintenance is extended to 2027-05-05** | Zensical "Upcoming changes": on 2026-11-05 "Material for MkDocs will receive a six-month end-of-life extension, with critical maintenance continuing until May 5, 2027" (vendor tier) |
| Zensical: MIT, pre-1.0 (0.0.66 on 2026-09-28, releases every few days); 0.1.0 "will begin a dependable release line" on 2026-11-05 and stays on 0.x "while we shape its module system" | Zensical "Upcoming changes"; PyPI (vendor, package-index tiers) |
| Zensical reads `mkdocs.yml` natively and claims "the complete settings surface" of Material, including search, tags and blog, and template overrides through its classic theme variant; plugin support is "a growing list of plugin replacements" | Zensical compatibility page (vendor tier) |
| Zensical's makers launch a commercial "Studio" (Free, Pro, Team) on the same day as 0.1.0 | Zensical "Upcoming changes" (vendor tier) |
| Hugo: v0.167.0 on 2026-09-28, frequent releases; Hextra: MIT, v0.13.0 on 2026-09-29 | GitHub release pages (project tier) |
| Hextra requires Hugo **extended** plus Git, and Go too for the recommended Hugo-module install; built-in search; no Node prerequisite | Hextra getting-started page (project tier) |
| Go CLIs moving off Material: GoReleaser moved to Hugo with Hextra on 2026-03-22, giving no reason and noting only URL breakage ("if you happen to find a 404, its because it slipped through the cracks") | GoReleaser blog (practitioner tier); the site plan records golangci-lint's move in August 2025 |

The due date in the record ("around November 2026") is the end of Material's
original twelve-month promise. The extension moves the hard edge to
2027-05-05. Two dates in November still matter: Zensical's first stable-line
release, and the start of the extension.

## Options against the needs

| | Status quo (Material 9.7.x) | Zensical | Hugo with Hextra |
| --- | --- | --- | --- |
| Config change | none | none expected: `mkdocs.yml` is read as is (to be proved by one build) | rewrite: `hugo.toml`, the menu, and front matter on 14 pages |
| Toolchain | Python + pip (present) | Python + pip (present); a compiled wheel | Hugo extended binary (new to CI and to every contributor who previews docs); Go and Git present |
| Search | built in | built in | built in |
| Admonitions, superfences, tables | yes | claimed | Hextra's own callout shortcodes; page bodies need editing |
| Maintenance horizon | critical fixes until 2027-05-05, then none | active; 0.x, "dependable release line" from 2026-11-05 | active, mature (Hugo since 2013); Hextra a single-maintainer-led theme |
| Dependency change needing sign-off | none | swap one pinned Python package | a new binary toolchain, plus a theme module |
| Reversal cost | n/a | low: `mkdocs.yml` stays valid for Material | high: a second configuration format and edited page bodies |

## Adversary filter (fit against this repository's preferences)

- **No new dependency without sign-off.** Every option except the status quo
  is a dependency change. Zensical swaps one pinned Python package for
  another in the same toolchain; Hugo adds a binary to CI and to the local
  preview path. The sign-off is owed either way, and the Zensical ask is the
  smaller one.
- **Host-agnostic, single binary, Go.** The Go-native argument for Hugo is
  weaker than it looks: abcd does not build Hugo, it downloads it, so it is a
  second toolchain beside Python rather than a replacement for one — unless
  the Python step is removed entirely, which is the one real gain Hugo
  offers (one fewer language runtime in the site job).
- **Script-first MVP.** The cheapest test of the Zensical claim is to run it:
  `zensical build` against the unchanged `mkdocs.yml` in a scratch copy, and
  `abcd lint site` over the output. No design work precedes that.
- **Vendor risk.** Zensical is pre-1.0, and its makers are launching a paid
  product. The core is MIT, and the migration keeps `mkdocs.yml`, so a
  Zensical that stalls leaves the repository exactly where it is today. The
  Hextra theme is led largely by one maintainer; Hugo itself is not a risk.
- **Pinning.** Zensical publishes a release every few days. The site
  workflow pins the direct dependency only, and `docs/requirements.txt`
  already names that gap; a move to Zensical should pin the whole resolved
  set with hashes, or builds drift weekly.
- **What the record claimed and the tree does not hold.** "Overrides, tags"
  are not in use, so they cannot decide the choice. If the product thinker
  wants them later, both candidates have them.

## Verdict (a proposal; the ruling is the product thinker's)

1. **Stay on Material now.** Nothing breaks before 2027-05-05, and nothing in
   `docs/` uses a feature Material is losing.
2. **Take Zensical as the successor, on 0.1.0 or later** (from 2026-11-05),
   by one change that swaps the pinned package and the build command and
   pins the full resolved set. Precondition: a scratch build of the unchanged
   `mkdocs.yml` renders all 14 pages, search works, and `abcd lint site`
   passes over the output. Ground: the same toolchain, no config rewrite, the
   lowest reversal cost, and search matching what the site uses today.
3. **Hugo with Hextra is the fallback**, chosen only if the Zensical build
   fails the precondition or 0.1.0 slips past 2027-02, leaving too little of
   the extension for a rewrite.
4. **Record it as an ADR** citing this note; the ADR is where the dependency
   swap is signed off.

What would show this wrong: a Zensical build of the unchanged `mkdocs.yml`
that needs config edits beyond the build command, or a Zensical 0.1.0 that
lands with a licence or telemetry change affecting the open-source core.

## Where no evidence was found

- No independent report (outside the vendor) of a Zensical migration of a
  project this size was located; the compatibility claims are vendor-tier
  until the scratch build runs.
- GoReleaser's post gives no reason for its move, so it is evidence of
  direction, not of cause.

## Review record

2026-09-30: authored in one pass by an implementer in autonomous run A (lane
drainResearch), with the fit-challenge run in-pass by the author. No
independent adversarial reviewer has read this note yet; per the
[research protocol](2026-08-22-sota-research-protocol.md) that review is owed
before the ADR cites it.

## Sources (all accessed 2026-09-30)

- [Zensical — Upcoming changes](https://zensical.org/upcoming-changes/)
- [Zensical — Compatibility: MkDocs features](https://zensical.org/compatibility/features/)
- [Zensical — a modern static site generator (Material blog, 2025-11-05)](https://squidfunk.github.io/mkdocs-material/blog/2025/11/05/zensical/)
- [Material for MkDocs blog index (maintenance mode; MkDocs 2.0 post of 2026-02-18)](https://squidfunk.github.io/mkdocs-material/blog/)
- [PyPI JSON — zensical](https://pypi.org/pypi/zensical/json)
- [PyPI JSON — mkdocs-material](https://pypi.org/pypi/mkdocs-material/json)
- [PyPI JSON — mkdocs](https://pypi.org/pypi/mkdocs/json)
- [Hextra — Getting started](https://imfing.github.io/hextra/docs/getting-started/)
- [Hextra releases](https://github.com/imfing/hextra/releases)
- [Hugo releases](https://github.com/gohugoio/hugo/releases)
- [GoReleaser — Goodbye, Mkdocs (2026-03-22)](https://goreleaser.com/blog/new-site/)
