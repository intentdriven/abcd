---
id: spc-2610141709461788
slug: anyone-who-opens-abcdev-app-finds-the
intent: itd-2610091010348892
origin: researcher-authored
production_mode: hand-written
---
# anyone-who-opens-abcdev-app-finds-the

## Summary

spc-2610141709461788 delivers itd-2610091010348892: a project that switches it
on at setup has its about page and development record built by its own release
workflow at each release and published on its own GitHub Pages, at
`https://<owner>.github.io/<repo>/` (or the root of a custom domain), with no
secret and nothing run by abcd. It is the existing `abcd site setup` pipeline
with a second hosting provider, not a parallel one: one canonical primitive.
Every word on the site comes from what abcd already writes to the repository,
as the product thinker ruled on 2026-10-15, so the project route has no
composition and no site sources.

- **The provider.** `internal/adapter/hosting/githubpages` implements the
  hosting seam. The repository half holds no secret: the deploy job runs in the
  `github-pages` environment with the run's own short-lived token (`contents:
  read`, `pages: write`, `id-token: write`) and deploys through
  `actions/upload-pages-artifact` v5.0.0 and `actions/deploy-pages` v5.0.1,
  pinned to full commits; there is no host configuration file. The host half
  speaks the GitHub REST API through `gh` as the person who invoked setup, the
  way the forge stage does, so abcd resolves and stores no credential for it:
  it reads the repository's visibility and Pages state, creates a Pages site
  built by the workflow (`build_type: workflow`), sets the custom domain only
  when one was given, and reports the `html_url`. A private repository, a site
  published from a branch, a domain routed elsewhere, a live custom domain when
  setup was given none, and a site served below another path than the one the
  workflow builds for are refused, never moved; setup reads the host before it
  asks or writes, so each is refused before anything is written. An existing
  workflow-built site is named by its address and build type in the switch
  question (or in a question of its own when the composition already holds the
  switch), and in the host outcome. The seam gained `Site.Repo`,
  `State.Existing` and one method, `Profile` (deploy environment and
  permissions, the switch, the public-only rule, whether it publishes a
  project site, the base path and the expected address); cloudflare's profile
  reproduces its workflow byte for byte. Nothing in the adapter names abcd's
  own repository.
- **The switch (criterion 1).** `abcd site setup --provider github-pages` asks
  whether to switch the project site on, naming the address, before anything
  is written. A yes writes `.abcd/site.json` holding the project site's
  settings and nothing else (`"mode": "project"`, a `hosting` block with
  `"publish": true`) and adds `.github/workflows/site.yml`. Those two files are
  the whole repository half: no composition and nothing under `site-src/`.
  This supersedes the coordinator's ruling of 2026-10-14, under which setup
  also wrote the site's inputs under `site-src/` as the site's settings. A
  settings file is decoded strictly and refuses every composition key
  (`identity`, `ui_strings`, `home`, `docs`, `record_pages`, `checks`). A no
  writes nothing. `hosting.publish` is refused on a provider without a switch.
  `.abcd/config.json` was weighed as the switch's home and not taken: it is
  ahoy's layered configuration, rewritten at each install, with a machine
  layer that must not switch a repository's site on, and the workflow already
  reads the switch from `.abcd/site.json`.
- **Single-source content (criterion 3).** `abcd site build --project` builds
  the about page in code from four parts, in order, each only where the
  repository holds it: the identity block `.abcd/positioning.json` points at,
  when abcd wrote its file; the brief's opening, the first section with text in
  `.abcd/development/brief/01-product/01-press-release.md` (the first row of
  the brief↔lifeboat contract, `internal/core/lifeboat/mapping.go`); the press
  releases of the three newest shipped intents whose audit was met, newest id
  first (a timestamped id carries when abcd minted it, and an ordinal id sorts
  before every timestamped one; the date an intent entered `shipped/` is git's
  and plays no part); and the release list in `CHANGELOG.md`. Anything absent is left out; with
  no identity block the page is headed and titled by the repository's name
  on GitHub. Setup, and `site build --project` before it writes anything,
  refuse a repository holding none of the four: the site would be empty. The
  repository's address and name come from the origin remote alone, never from
  `.claude-plugin/plugin.json`, a person's file. The record pages render as on
  every site, from `.abcd/record-lint.json` or, where a managed repository
  carries none, from abcd's standard layout under `.abcd/development/` (the
  issue ledger only where the settings opt in). The references page (it
  quotes `ACKNOWLEDGEMENTS.md`) and every picture (pictures live under
  `docs/`; asking for them is iss-2610150529485165) are left out. A project
  site also carries no git-history details, as the product thinker ruled, and
  the build never reads git history (no log, shortlog or blame): no
  contributors page, no timeline, none of the health page's authorship checks,
  and no `authorship`, `history` or per-record `dates` in `record.json`; a
  record is dated by its own frontmatter alone. abcd's own composed site keeps
  all of them, and every page and `record.json` are byte-identical before and
  after (3097 files at `/` and at `--base-path /abcd/`). The one file that
  changes is the graph script `record.js`: a record exported without git dates
  shows its own date as that date and links its history undated, and a record
  with git dates is shown exactly as before. Whether a project site should show any of it is to be
  revisited. The workflow builds with abcd's latest
  release, so a project site cannot build until a release carries
  `--project`; setup says so in a note on every project run.
  The interface words, stylesheet, scripts and headers policy are the binary's
  (`setupsrc/`), so the repository carries no site sources. A build or gate
  whose `--project` and settings disagree is refused, both ways.
- **Public repositories only (scope condition 1).** Setup reads the
  repository's visibility from the forge first and refuses a private one, or
  one whose visibility it cannot read, in one line before it asks or writes.
  The workflow holds the same line at each release: the switch step reads the
  event's `repository.private` and the deploy job requires it to be `false`,
  so a repository made private later publishes nothing.
- **Each release (criterion 2).** The workflow is the existing
  render-on-release-then-deploy workflow, with the provider's profile: the
  render job builds and gates the site with `--project` and
  `--base-path /<repo>/`, and the deploy job hands it to Pages.
- **The base path.** `abcd site build` and `abcd lint site` take
  `--base-path` (default `/`, normalised to a leading and trailing slash, plain
  path segments only). Every root-absolute URL the build writes goes through
  one helper, `sitePath`; the gate resolves the stylesheet against the prefixed
  root and a new `base-path` check refuses a root-absolute link outside it. The
  default output is byte-identical (golden files unchanged; this repository's
  own 3082-file site identical before and after).
- **The managed-repository line (criterion 4).** Every page opens with "This is
  the record of an abcd-managed repository", linking to the repository. The
  words are a new `managed_record` interface string in the closed `ui.json`
  allowlist; a project site reads it from the binary's allowlist. A composed
  site with no package manifest takes its address from the origin remote on
  github.com, and a project site always does.
- **The provenance gate (criterion 5).** The gate still fails a build on any
  word no source or interface string accounts for. With `--project` it also
  fails any `data-src` span, on any page, the record's own included, whose
  file abcd does not write: `.abcd/**` (but the local tier) and `CHANGELOG.md`
  pass, `docs/`, `README.md` and everything else fail. abcd's own composed
  site is not a project site and is unchanged: its build is byte-identical
  before and after (3091 files at `/` and at `--base-path /abcd/`).
- **No secret (criterion 6).** The workflow reads no `secrets.` value and
  interpolates no expression into a shell.
- **Switching off (criterion 7).** With `"publish"` anything but `true` at the
  released commit or at the head of the default branch (a missing or malformed
  file reads as off), the render job's switch step stops the build and the
  deploy job does not run, so the release publishes nothing, and a later push or
  dispatch rendering an older tag does not undo it. `abcd site setup` then
  touches nothing remote and offers to remove the workflow it added, keeping
  one edited by hand.
- **abcd's own repository** is not moved in this change. Its move was this
  intent's fourth criterion when the spec was drafted; that criterion has
  since moved to its own planned intent, itd-2610142145236613 (with the
  proposed adr-2610142147242544), and this intent's seven criteria no longer
  include it. The move is more than a composition edit. The workflow setup writes renders the composed site only, and builds
  no `/docs/` tree; abcd's composition declares a `docs` block, so its header
  links `/docs/`, which abcd's own `site.yml` fills by running MkDocs into
  `site/docs` (with `site_url` set to `https://abcdev.app` in `mkdocs.yml`).
  Moving abcd onto this route therefore also needs the MkDocs build carried
  into the Pages workflow, or the docs link dropped, and the MkDocs tree built
  below the same base path; and it would leave behind what abcd's own
  workflow does beyond the generic one (the attested `site.tar.gz` attached
  to each release, and the preview deploy on push). None of that is built
  here, and abcd's own `site.yml` is unchanged.

Found and captured, not fixed here: the `_headers` file the build copies is
Cloudflare's, so a project site on GitHub Pages is served without those
security headers (iss-2610142036270003).

Found and fixed here: the provenance gate refused a two-word author beside the
footer's year ("© 2026 Fixture Author") on every site, because a name was
matched only as a whole piece.

## Footprint

- packages: internal/adapter/hosting, internal/adapter/hosting/cloudflare, internal/adapter/hosting/githubpages, internal/adapter/hosting/githubpages/githubpagestest, internal/core/repolint, internal/core/site, internal/surface/cli
- tests: the project route (settings only and refused composition keys, the about page from its four parts and each left out when absent, no person's words on any page, the binary's static files, the gate refusing a span from `docs/` or the README on the about page and a record page, the route named both ways, the managed line on every project page, the base path, a hostile package manifest reaching no page while the managed line links the origin remote, the repository's name heading and titling a site with no identity block, the standard record layout read with no record-lint config and its issue ledger only when opted in, no contributors page, an empty build refused before it writes); bare `abcd lint` gating a project site on its own route; setup writing exactly the settings and the workflow, refusing a repository with nothing abcd wrote, and noting the release a project workflow needs; a two-word author beside the footer's year; the base path (normalising, every root-absolute URL prefixed, default output unchanged, the gate against the prefixed root, both verbs' flag); the managed-repository line on every page and its origin fallback; the github-pages adapter over its in-process fake (profile, create, route, address, refusals, every forge failure); setup switched on, declined, private, visibility unreadable, switched off with the workflow removed, kept when hand-edited, kept when the removal is declined; the Pages workflow holding no secret and gated on the switch and on a public repository, pinned by a golden file, and its switch step run with bash against a checkout (switched off at the head, missing or malformed there, made private); a live custom domain without --domain, a site served below another path and an existing site named before it is published over; the base-path gate denied a pass on an unparsed page; the cloudflare workflow pinned byte for byte by a golden file; the forge's visibility read and the `gh api` transport

## Steps

1. Build and gate a site served below a base path
   - packages: internal/core/site, internal/surface/cli
   - tests: basepath_test.go, site_basepath_test.go, the unchanged golden files
   - landed: ff9eee7b6
2. Open every page with the managed-repository line
   - packages: internal/core/site
   - tests: managedline_test.go, the updated golden files
   - landed: ffa77caf5
3. The github-pages hosting adapter and the seam's profile
   - packages: internal/adapter/hosting, internal/adapter/hosting/cloudflare, internal/adapter/hosting/githubpages
   - tests: githubpages_test.go, cloudflare_test.go
   - landed: 52a12eb24
4. The switch at setup, public repositories only, the provider-aware workflow and switching off
   - packages: internal/core/site, internal/surface/cli
   - tests: setup_pages_test.go, setup_workflow_golden_test.go, forge_pages_test.go, site_setup_test.go
   - landed: 862ace7f8
5. Single-source project site
   - packages: internal/adapter/hosting, internal/adapter/hosting/githubpages, internal/core/repolint, internal/core/site, internal/surface/cli
   - tests: project_test.go, setup_pages_test.go, setup_pages_takeover_test.go, the updated github-pages workflow golden, check_test.go, site_basepath_test.go, githubpages_test.go, rule_site_test.go, the unchanged composed golden files
   - landed: 54f5060ad
   - The step landed in five commits, 8fbb2f3c5, 19a41f006, 916ff24cc, c22868d2f and 54f5060ad, the last named above; the review's fixes followed on the same branch in 6388e1a56, 364470574, 13991c426, dbc8d4384, 7bc9fecff, 063b04710, eafdefac5 and 38d66d63d, and a second round in 3c6ca3d96, 9871ce11e and 3885b24e1.
