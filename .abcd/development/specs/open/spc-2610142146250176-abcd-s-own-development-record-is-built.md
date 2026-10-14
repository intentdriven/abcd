---
id: spc-2610142146250176
slug: abcd-s-own-development-record-is-built
intent: itd-2610142145236613
origin: researcher-authored
production_mode: hand-written
---
# abcd-s-own-development-record-is-built

## Summary

abcd's own site moves from its Cloudflare deploy onto abcd's own GitHub Pages. It is built by the workflow `site setup --provider github-pages` writes for any project (itd-2610091010348892, spc-2610141709461788). abcdev.app then forwards every address there. The move is planned now and built after itd-2610091010348892 ships, which it is blocked by.

**What abcd's site has that the project workflow does not yet build.** Each gap is closed in the generic route, never as an abcd special case, because the claim is that abcd runs on the route it offers.
- **/docs/**, rendered by MkDocs Material from `docs/`. The project workflow gains an opt-in docs step, declared in `.abcd/site.json` (a `docs` block naming the builder and its requirements file). The step installs the requirements with `pip install --require-hashes` and builds into `site/docs`. abcd's `docs/requirements.txt` becomes a fully hashed lock, closing the version-only pin its own comment flags. MkDocs Material's maintenance window (adr-47 point 4) is unchanged by this: whatever replaces it later plugs into the same step.
- **Security headers.** Cloudflare serves `site-src/headers` as `_headers`; GitHub Pages ignores that file and cannot set response headers (iss-2610142036270003).
  - Every document the site build renders gains its content policy as a `<meta http-equiv="Content-Security-Policy">` element, generated from the same document policy the headers file holds, so the two cannot drift.
  - The MkDocs pages get the same element through a theme override.
  - What a page element cannot carry is recorded in the decision record, not hidden: framing refusal (`frame-ancestors`) and `X-Content-Type-Options: nosniff`.
  - The github-pages provider stops emitting the Cloudflare files `_headers` and `_redirects`, so they are not served as stray pages. This resolves iss-2610142036270003 for every project, not only abcd.
- **The moved docs addresses.** `site-src/redirects` 301s the pre-/docs/ paths (`/reference/`, `/tutorials/`, `/how-to/`, `/explanation/` and the CLI reference pages). On the github-pages provider, the build writes a small page at each old path instead. The page carries a canonical link, a zero-second refresh to its new address and a plain link, which is the state-of-the-art pattern for redirects on a static host without redirect support.
- **The release chain.** abcd's releases are created by `release.yml` with the run's own token, so a `release:` event never fires for them (site.yml's header records why). The generated workflow's `workflow_run` trigger on the release workflow is what deploys abcd. It is checked on abcd's own release chain before the Cloudflare deploy is removed.
- **The preview.** The Cloudflare preview of `main` on every push has no GitHub Pages equivalent, because a repository has one Pages site. It is dropped. `make site-render` and CI's site-render gate still render and check every change.

**abcdev.app forwards.** Per the product thinker's ruling, one dynamic redirect rule at the domain's host sends `abcdev.app/*` and `www.abcdev.app/*` with a 301 to the same path on abcd's Pages address, keeping the query string. This needs no Worker and no deploy. The README's and docs' links to abcdev.app are rewritten to the Pages address in the same change, so they do not depend on the forward. The forward serves links already shared, and leaves abcdev.app free to become the community site later.

**abcdesign.app needs its own repository.** A GitHub repository has exactly one Pages site, and after the move abcd's is the record. So the static abcdesign.app page (spc-2610141709430146) is published from a repository of its own. That repository's workflow downloads the released abcd and runs `abcd session page`, and its Pages carries the abcdesign.app domain. Creating that repository is an outward act at its own cut, with the product thinker's go.

**The cut-over is ordered, and each outward step is confirmed by the person first.** The order is:
1. Pages publishes a release and the result is checked at the Pages address.
2. The forwarding rule is switched on and checked on a sample of old addresses, including a moved docs path.
3. Only then are the Cloudflare Worker, the `site` and `site-preview` environments and their four Cloudflare secrets removed.

A failure at any step leaves the previous host serving.

**Records.** A new decision record supersedes adr-48 in its Cloudflare deploy. adr-48's rule that the site deploys on release, from released bytes, stands and carries over to the Pages route. adr-47's single-source rule, gates and no-Node constraint are unchanged.

## Footprint

- packages: internal/core/site, internal/core/site/setupsrc, internal/adapter/hosting/githubpages, internal/surface/cli, docs/, .github/workflows, .abcd/site.json
- tests: internal/core/site, internal/adapter/hosting/githubpages, internal/surface/cli

## Steps

1. The opt-in docs step in the project workflow, and abcd's docs requirements as a hashed lock
   - packages: internal/core/site, internal/core/site/setupsrc, docs/
   - tests: internal/core/site
2. Page-carried content policy, no Cloudflare files on github-pages, and the redirect pages for moved addresses (resolves iss-2610142036270003)
   - packages: internal/core/site, internal/adapter/hosting/githubpages, docs/
   - tests: internal/core/site, internal/adapter/hosting/githubpages
3. abcd's repository switches to the generated github-pages workflow: the manifest's hosting block, the release chain, README and docs links, and the decision record superseding adr-48's Cloudflare deploy
   - packages: .abcd/site.json, .github/workflows, docs/, internal/surface/cli
   - tests: internal/core/site, internal/surface/cli
4. The cut-over, each outward act confirmed by the person: the first Pages release, the abcdev.app forwarding rule, then removing the Cloudflare Worker, environments and secrets, and the files that configured them
   - packages: .github/workflows, wrangler.jsonc, docs/
   - tests: internal/core/site
