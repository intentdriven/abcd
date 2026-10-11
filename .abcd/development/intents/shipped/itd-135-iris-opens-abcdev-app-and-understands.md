---
id: itd-135
slug: iris-opens-abcdev-app-and-understands
spec_id: spc-37
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Iris opens abcdev.app and understands who abcd is for, what they and their facilitator own, and how to install it — from one page rendered from the repository

## Press Release

> **Iris opens abcdev.app and understands who abcd is for, what they and
> their facilitator own, and how to install it — from one page rendered from
> the repository.** Iris thinks in products, not in repositories. Until now,
> abcdev.app greeted them with a documentation filing system, and the story of
> who abcd serves lived in a README they would only find by already knowing
> where to look. Now the front page is that story: a hero built from the
> rationale page and the repository's canonical Identity block, four chapters
> — Roles, Artefacts, Process, Install — rendered from the four documentation
> pages that carry them, joined by one thread line, and, as the only
> testimonial, the newest shipped intent whose audit verdict is MET, quoted
> verbatim from the record. The install chapter puts the plugin path on the
> left and the CLI group on the right with Iris's own system starred, every
> command a fenced block from the install page. A Beta badge sits by the brand
> for as long as the major version is 0 — a rule on the release version, not
> copy. Not one sentence on the page was written for the website: every span
> is selected from a repository file through `.abcd/site.json`, and the build
> fails on any text it cannot source. "I read one page and knew whose job
> abcd does and whose it protects," said Iris, a product thinker. "And
> nothing on it could be marketing, because nothing on it was written for it."

## Why This Matters

The repository already holds an unusually rich, machine-readable account of
what abcd is — the brief's Identity block, the README's product narrative
moving into `docs/`, a record with audited shipped intents — and none of it
reaches a visitor. A landing page assembled by selection instead of authorship
turns that account into the front door while making marketing drift
structurally impossible: if a sentence would improve the site, it must be
written into the documentation, where it must also read true on GitHub. The
single-source rule and its build gates are recorded in adr-47; the
release-bound deploy that keeps the page describing an installable product is
adr-48. The README→docs migration and the `abcd site build` generator that
this page rides on are plumbing, recorded in the brief rather than filed as
intents.

## Acceptance Criteria

- Given the Identity block under `.abcd/development/brief/01-product/README.md`
  changes, when the site rebuilds, then the hero at `/` renders the new
  tagline and pitch with no template edit — the hero selector in
  `.abcd/site.json` names the block, and `abcd site check` verifies the
  rendered hero against it at build time (the site-hero analogue of the
  `.abcd/positioning.json` surfaces, which check committed files and so
  cannot carry a build-time surface themselves).
- Given any visible text node on `/`, when `abcd site check` runs, then the
  node sits inside an element carrying a `data-src` provenance attribute that
  names a repository file span, or matches an interface string in
  `site-src/ui.json`, a number, a date, a file name or an asset name — and
  the check fails naming any node it cannot source.
- Given the four chapters a–d on `/`, then each is rendered from its
  documentation page (`docs/explanation/roles.md`,
  `docs/explanation/artefacts.md`, `docs/explanation/process.md`,
  `docs/how-to/install.md`) through `.abcd/site.json`, and the only
  testimonial on the page is the newest shipped intent whose audit verdict is
  MET, quoted verbatim.
- Given any `abcd …` snippet on `/`, when the site builds, then the snippet
  matches the generated CLI reference (`docs/reference/cli/commands.md`) or
  the build fails on the stale snippet.
- Given rendered text from any tree, when the site builds, then the docs-lint
  banned-token rules run over it, so record text cannot reintroduce on the
  site what docs-lint keeps out of `docs/`.
- Given the latest release has major version 0, then the Beta badge renders
  beside the brand; given a v1 release, then it is absent — with no copy
  change in between.
- Given a 390 px viewport, when `/` renders, then nothing scrolls
  horizontally and no element is wider than the viewport — the static
  checks (viewport meta, overflow containers on wide elements, max-width
  on images) run in `abcd site check`; the rendered-overflow screenshot
  audit runs as a CI job, since a browserless binary cannot measure
  layout.
- Given every picture on `/`, then it is a committed asset under
  `docs/assets/img/` referenced from a documentation page; inlined SVGs use
  `var(--token, fallback)` colours so they follow the site theme while GitHub
  renders the fallbacks.

## Open Questions

- Which rule picks the featured intent when several shipped intents carry a
  MET audit from the same day (the plan's §7 leaves this to the team).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-0070560a7803 -->
Fidelity review — receipt rcp-0070560a7803 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:5429c3934de08dc635589d0da1621a890e5b1f382585288b9dbe9c7c63cd245e
Input attestations: diff:internal/core/site, site-src, .abcd/site.json and .github/workflows/site-screenshots.yml at chore/audit-run-a-1 80b44890 (git ls-tree -r; spc-37 closed, itd-135 shipped)@sha256:f15c4593a450e6c1d01a0517bd88786c849d49f0e6a0da97087ddf05a2b1e352;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the manifest's identity selector names the brief file and heading, the composer renders the hero from that block with no template prose, and checkHero holds the rendered .hero paragraphs to the block with tests for a drifted, a pitch-less and a missing hero; the concern is that the verb the criterion names, `abcd site check`, does not exist at BASE — the gate is `abcd lint site` (renamed by itd-2609212130136102) and it runs as a separate step chained after `site build` rather than inside the build
  evidence: .abcd/site.json:5 — ""file": ".abcd/development/brief/01-product/README.md","
  evidence: internal/core/site/compose.go:477 — "func (c *composer) hero() (string, error) {"
  evidence: internal/core/site/check.go:1030 — "func (c *checker) checkHero() {"
  evidence: internal/core/site/check_test.go:393 — "func TestCheckRefusesADriftedHero(t *testing.T) {"
  evidence: Makefile:261 — "go run ./cmd/abcd lint site --out"
- ac-2 — MET: checkProvenance walks every visible text node on the composed pages, fails naming any node outside a data-src element unless generatorWords covers it, and covers is the ui.json strings plus number, date and commit tokens and file names; a test refuses an unsourced node
  evidence: internal/core/site/check.go:598 — "func (c *checker) checkProvenance() {"
  evidence: internal/core/site/check.go:620 — "which sits in no data-src element"
  evidence: internal/core/site/check.go:943 — "func (g generatorWords) covers(text string) bool {"
  evidence: internal/core/site/check_test.go:225 — "func TestCheckRefusesAnUnsourcedTextNode(t *testing.T) {"
- ac-3 — MET: the manifest composes chapters a-d from roles.md, artefacts.md, process.md and install.md, and the only feature is kind shipped-intent-press-release picked newest-with-audit-MET; featureBlock quotes the newest MET intent's press release and first criterion verbatim through auditIsMet, which tests hold to the real Audit Notes rollup
  evidence: .abcd/site.json:17 — ""page": "docs/explanation/roles.md","
  evidence: .abcd/site.json:26 — ""pick": "newest-with-audit-MET","
  evidence: internal/core/site/compose.go:1111 — "func (c *composer) featureBlock(f *Feature) (string, error) {"
  evidence: internal/core/site/compose_test.go:519 — "func TestAuditIsMetOnTheCommittedIntents(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: checkSnippets pins every `abcd …` snippet on the site to the command paths parsed from docs.cli, fails on one it cannot pin, and a test refuses a stale snippet; the concern is that `abcd site build` itself exits 0 on a stale snippet — the failure lands in `abcd lint site`, which the site-render target, ci.yml and site.yml each chain after the build, so the promise holds for the pipeline rather than for the build verb
  evidence: internal/core/site/check.go:1189 — "func (c *checker) checkSnippets() {"
  evidence: internal/core/site/check_test.go:538 — "func TestCheckRefusesAStaleSnippet(t *testing.T) {"
  evidence: Makefile:260 — "go run ./cmd/abcd site build --out"
  evidence: .github/workflows/site.yml:499 — "./abcd lint site --out site"
- ac-5 — MET_WITH_CONCERNS: checkBannedTokens runs the docs-lint TokenChecker built from .abcd/docs-lint.json over every visible span of the composed pages, whichever tree the span came from, and a test refuses a banned token in a composed span; the concern is the scope adr-47 decision 3 rules — the verbatim record rendering under /record/** is exempt, so 'any tree' holds for the spans the manifest selects, not for the record pages
  evidence: internal/core/site/check.go:313 — "tokens, err := lint.NewTokenChecker(docsCfg.BannedTokens)"
  evidence: internal/core/site/check.go:1103 — "func (c *checker) checkBannedTokens() {"
  evidence: internal/core/site/check.go:30 — "adr-47 decision 3 scopes the banned-token gate"
  evidence: internal/core/site/check_test.go:449 — "func TestCheckRefusesABannedTokenInAComposedSpan(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: isPreOne reads the major component of the newest release and headerFor renders the ui.json Beta string only while it is 0, so no copy changes at v1; a test asserts the badge at a 0.x release and its absence with no release, but no test builds against a v1 release, so the 'absent at v1' half rests on the predicate alone
  evidence: internal/core/site/build.go:572 — "func isPreOne(version string) bool {"
  evidence: internal/core/site/compose.go:320 — "b.WriteString(`<span class="beta">` + escapeText(c.ui.Beta) + `</span>`)"
  evidence: internal/core/site/build_test.go:1087 — "if !strings.Contains(html, `<span class="beta">Beta</span>`) {"
  evidence: internal/core/site/build_test.go:1138 — "if strings.Contains(html, `class="beta"`) {"
- ac-7 — MET: checkMobile runs the static half over every page the build writes — viewport meta, overflow containers on table and pre, img max-width, inline widths against the 390px constant — with tests for a missing viewport and an unconstrained image; the rendered audit is the site-screenshots CI job driving overflow-audit.js at widths including 390
  evidence: internal/core/site/check.go:1435 — "func (c *checker) checkMobile() {"
  evidence: internal/core/site/check.go:1427 — "const maxInlineWidthPx = 390"
  evidence: internal/core/site/check_test.go:817 — "func TestCheckRefusesAMissingViewport(t *testing.T) {"
  evidence: site-src/audit/overflow-audit.js:40 — "const WIDTHS = [360, 390, 768, 1360];"
  evidence: .github/workflows/site-screenshots.yml:135 — "- name: Audit every route for horizontal overflow"
- ac-8 — MET: the asset pipe refuses a remote or data: image outright, resolves every reference relative to the docs page that wrote it, refuses an SVG that styles rather than draws, and a test holds every committed SVG asset to the var(--token, fallback) drawing rule
  evidence: internal/core/site/assets.go:401 — "every picture is a committed asset under docs/assets/img/"
  evidence: internal/core/site/assets.go:355 — "a drawing's colours are var(--token, fallback) values on its paint attributes"
  evidence: internal/core/site/assets_test.go:153 — "func TestAssetsRefuseStyledSVG(t *testing.T) {"
  evidence: internal/core/site/assets_test.go:313 — "func TestCommittedSVGAssetsAreDrawings(t *testing.T) {"

Gap audit:
- honoured:
  - the hero is the Identity block read through the manifest and verified after render
    evidence: internal/core/site/check.go:1030 — "func (c *checker) checkHero() {"
  - every visible text node is sourced or a generator word, and the check names the offender
    evidence: internal/core/site/check.go:620 — "which sits in no data-src element"
  - the only testimonial is the newest audit-MET intent quoted verbatim
    evidence: internal/core/site/compose.go:1111 — "func (c *composer) featureBlock(f *Feature) (string, error) {"
  - mobile layout is gated statically in the binary and by a browser audit in CI
    evidence: site-src/audit/overflow-audit.js:40 — "const WIDTHS = [360, 390, 768, 1360];"
  - pictures are committed assets and inline SVGs are themed drawings
    evidence: internal/core/site/assets_test.go:313 — "func TestCommittedSVGAssetsAreDrawings(t *testing.T) {"
- diverged:
  - the gate verb is `abcd site check` and runs at build time
    evidence: Makefile:261 — "go run ./cmd/abcd lint site --out"
    evidence: .github/workflows/site.yml:499 — "./abcd lint site --out site"
  - banned-token rules run over rendered text from any tree
    evidence: internal/core/site/check.go:30 — "adr-47 decision 3 scopes the banned-token gate"
- missing:
  - a test that the Beta badge is absent at a v1 release
    evidence: internal/core/site/build_test.go:1138 — "if strings.Contains(html, `class="beta"`) {"
