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
  permissions, the switch, the public-only rule, the base path and the
  expected address); cloudflare's profile reproduces its workflow byte for
  byte. Nothing in the adapter names abcd's own repository.
- **The switch (criterion 1).** `abcd site setup --provider github-pages` asks
  whether to switch the project site on, naming the address, before anything
  is written. A yes records `"publish": true` in the `hosting` block of
  `.abcd/site.json` and adds `.github/workflows/site.yml`. Setup also writes
  the site's inputs under `site-src/`, as it does for every provider; those
  count as the site's abcd settings, so nothing outside the site's settings and
  the workflow changes (the coordinator's ruling of 2026-10-14). A no writes
  nothing. `hosting.publish` is refused on a provider without a switch.
- **Public repositories only (scope condition 1).** Setup reads the
  repository's visibility from the forge first and refuses a private one, or
  one whose visibility it cannot read, in one line before it asks or writes.
  The workflow holds the same line at each release: the switch step reads the
  event's `repository.private` and the deploy job requires it to be `false`,
  so a repository made private later publishes nothing.
- **Each release (criterion 2).** The workflow is the existing
  render-on-release-then-deploy workflow, with the provider's profile: the
  render job builds and gates the site with `--base-path /<repo>/`, and the
  deploy job hands it to Pages.
- **The base path.** `abcd site build` and `abcd lint site` take
  `--base-path` (default `/`, normalised to a leading and trailing slash, plain
  path segments only). Every root-absolute URL the build writes goes through
  one helper, `sitePath`; the gate resolves the stylesheet against the prefixed
  root and a new `base-path` check refuses a root-absolute link outside it. The
  default output is byte-identical (golden files unchanged; this repository's
  own 3082-file site identical before and after).
- **The managed-repository line (criterion 3).** Every page opens with "This is
  the record of an abcd-managed repository", linking to the repository. The
  words are a new `managed_record` interface string in the closed `ui.json`
  allowlist, so an older `ui.json` gains it with its default words and the
  provenance gate still fails a build on any word no repository file accounts
  for (criterion 5). A repository with no package manifest takes its address
  from the origin remote on github.com.
- **No secret (criterion 6).** The workflow reads no `secrets.` value and
  interpolates no expression into a shell.
- **Switching off (criterion 7).** With `"publish"` anything but `true` at the
  released commit or at the head of the default branch (a missing or malformed
  file reads as off), the render job's switch step stops the build and the
  deploy job does not run, so the release publishes nothing, and a later push or
  dispatch rendering an older tag does not undo it. `abcd site setup` then
  touches nothing remote and offers to remove the workflow it added, keeping
  one edited by hand.
- **abcd's own repository (criterion 4)** is not moved in this change (the
  product thinker's "at some point"), and the move is more than a composition
  edit. The workflow setup writes renders the composed site only, and builds
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

## Footprint

- packages: internal/adapter/hosting, internal/adapter/hosting/cloudflare, internal/adapter/hosting/githubpages, internal/adapter/hosting/githubpages/githubpagestest, internal/core/site, internal/surface/cli
- tests: the base path (normalising, every root-absolute URL prefixed, default output unchanged, the gate against the prefixed root, both verbs' flag); the managed-repository line on every page and its origin fallback; the github-pages adapter over its in-process fake (profile, create, route, address, refusals, every forge failure); setup switched on, declined, private, visibility unreadable, switched off with the workflow removed, kept when hand-edited, kept when the removal is declined; the Pages workflow holding no secret and gated on the switch and on a public repository, pinned by a golden file, and its switch step run with bash against a checkout (switched off at the head, missing or malformed there, made private); a live custom domain without --domain, a site served below another path and an existing site named before it is published over; the base-path gate denied a pass on an unparsed page; the cloudflare workflow pinned byte for byte by a golden file; the forge's visibility read and the `gh api` transport

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
