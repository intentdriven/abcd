---
id: adr-48
slug: website-deploys-on-release-not-on-merge
status: accepted
date: 2026-08-22
supersedes: null
superseded_by: null
related_intents: [itd-135, itd-136, itd-137, itd-138]
related_rfcs: []
related_adrs: [adr-47, adr-37, adr-38]
---

# ADR-48: The website deploys on release, not on merge

## Context

[adr-47](0047-abcdev-app-rendered-from-this-repository.md) makes
abcdev.app a rendered surface of this repository. The remaining question is
the trigger. Every hosted platform's default is deploy-on-push, and the
record is the most alive part of this repository — 1,188 commits in 45 days
against 10 releases at the 2026-08-21 snapshot — so a release-bound site
shows a record days stale. The adversarial pass on both options is recorded
in the implementation prompt's Part 2
([research/abcdev-site/](../../research/abcdev-site/abcdev-implementation-prompt.md)).

## Decision

1. **Production deploys per release, from the tag.** The site workflow
   checks out the tag, downloads the released binary, checksum-verifies it
   exactly as the install one-liner does, renders the site with that
   binary, builds the docs into `site/docs/`, runs `abcd site check`,
   attests `site.tar.gz` as a release asset, and deploys with wrangler from
   a protected GitHub Environment. The site is produced by the same bytes
   users run, and abcdev.app becomes one verifiable statement: *this is
   abcd at that tag*.
2. **The trigger is the release chain, not a `release:` event.** Releases
   here are created by `release.yml`'s `gh release create` under the
   workflow's own `GITHUB_TOKEN`, and GitHub Actions never fires event
   triggers for actions taken with that token — `auto-release.yml`'s
   header documents exactly this semantic and already works around it with
   `workflow_call`. The site deploy is therefore a reusable workflow
   invoked from the release chain as a **separate, non-gating job after
   the release job**: a site failure can neither block nor taint the
   release (the isolation is job dependency, not workflow-file
   separation), running after `gh release create` completes removes the
   race against asset uploads, and the job is idempotent (re-running
   deploys the same tag). Pre-releases never deploy production.
3. **Every push to main deploys to a clearly labelled preview** (an
   "unreleased — main@sha" label rendered from build metadata), replacing
   the Cloudflare branch builds that currently repeat only the version
   command. The preview is built by Actions with a **source-built** binary
   — main is ahead of any release, so the released-bytes rule cannot apply
   — and deployed as a non-production version; the "unreleased" label is
   precisely the disclosure of that difference. The team gets the live
   record; the public gets the released one.
4. **Emergencies redeploy from the latest tag by `workflow_dispatch`** (a
   tag input), never from main — the invariant survives the emergency. A
   genuine content emergency is a CHANGELOG line and a patch release under
   [adr-37](0037-changelog-driven-releases.md): the fix gets a version and
   a date.
5. **Cloudflare's automatic production builds are turned off**; deploys come
   from Actions with wrangler, and the dashboard's build command becomes a
   comment in `wrangler.jsonc`, where the repo already documents it.

## Alternatives Considered

- **Deploy production on every merge to main**: the platform default, and
  the record explorer stays current to the hour — rejected because the
  site's job is to describe a product someone can install: following main
  lets the site document a verb no downloadable binary has, and the install
  command, version badge, changelog and docs can disagree for days.
  Versioned docs would repair that and are far too heavy for a project this
  size. The staleness cost is served by the labelled preview instead.
- **Emergency redeploys from main**: rejected — it silently converts the
  emergency path into deploy-on-merge; dispatch rebuilds from the latest
  tag only.
- **Rendering production with a source-built binary**: rejected — the site
  must be produced by the released, checksum-verified bytes, or the
  attestation says nothing about what visitors actually read.

## Consequences

- The cadence of the website becomes the cadence of releases, making small,
  frequent releases slightly more valuable than they already are — and ten
  releases in 45 days suggests that is the cadence anyway.
- The Cloudflare deploy token is scoped to one Worker and lives in a
  protected GitHub Environment inside the release workflow's existing trust
  chain; it never touches release signing.
- The runtime posture matches [adr-38](0038-implicit-checks-are-disk-only.md):
  everything the site knows is injected at build time; the published pages
  make no API calls.
