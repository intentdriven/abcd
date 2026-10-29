---
id: spc-2610142146250176
slug: abcd-s-own-development-record-is-built
intent: itd-2610142145236613
origin: researcher-authored
production_mode: hand-written
---
# abcd-s-own-development-record-is-built

## Summary

abcd's own record moves onto abcd's own GitHub Pages, built by the same workflow `site setup --provider github-pages` writes for any project (itd-2610091010348892), and from what abcd writes to the repository and nothing else (the product thinker's single-source ruling of 2026-10-15). abcdev.app becomes abcd's documentation site until a community site replaces it. abcdesign.app is the team-brief editing page for any abcd-managed repository (itd-2610091016262335). This intent is planned now and built after itd-2610091010348892 ships; it is blocked by that intent.

**The three addresses after the move:**
- **abcd's GitHub Pages address.** It carries abcd's about page and record, built like any project's at each release. No hand-written docs and no file written for the website go into it.
- **abcdev.app.** It carries abcd's documentation, rendered in high quality from `docs/`, from MkDocs Material. abcdev.app's landing page and docs, today composed through `.abcd/site.json` from `docs/`, stay as they are. They are the documentation's own front page, not a project site.
- **abcdesign.app.** It is the static brief-session page, published from a repository of its own (spc-2610141709430146).

**What changes in abcd's repository:**
- **abcdev.app's deploy narrows to the documentation.** The existing deploy (adr-48: on release, from released bytes) keeps building the docs and their landing page. It stops building the record explorer: `/record/`, `/contributors/` and `/references/` leave abcdev.app.
  - Each of those addresses forwards with a 301 to the same path on abcd's GitHub Pages site, query kept. The forwarding goes in the docs host's redirect file (`site-src/redirects`), so no rule is kept outside the repository.
  - The docs' own 301s for the paths moved under /docs/ stay.
  - Links in the README and docs to abcd's record point at the Pages address.
- **abcd's repository switches its record on to the project route.** The switch is set in abcd's settings and the generated workflow is added. Its about page comes from abcd's identity block, the brief, the newest shipped intents' press releases and the changelog, exactly as for any project.
  - abcd's current `.abcd/site.json` composition and `site-src/` stay as the documentation site's own sources.
  - The project route reads neither, and its gate fails on any text not from a file abcd writes.
- **Security headers on GitHub Pages.** GitHub Pages ignores `_headers` (iss-2610142036270003). Every document the project route renders carries its content policy as a page element. Framing refusal and the no-sniff header cannot be carried, and that is named in the decision record. abcdev.app keeps its response headers on its own host.
- **The preview of `main`.** It stays with the documentation deploy on abcdev.app. The record has no preview, because a repository has one Pages site. The site-render gate still renders and checks every change.

**The cut-over is ordered, and each outward step is confirmed by the person first:**
1. The first Pages release is published and checked at the Pages address.
2. The forwarding of the record's addresses is switched on, and a sample of old record links is checked.
3. Only then does the documentation deploy stop building the record.

A failure at any step leaves the record where it was.

**Records.** A new decision record amends adr-48: abcdev.app's deploy narrows to the documentation, and the record moves to the project route. adr-48's rule that a site deploys on release, from released bytes, stands for both.

## Footprint

- packages: internal/core/site, internal/core/site/setupsrc, internal/adapter/hosting/githubpages, site-src, docs/, .github/workflows, .abcd/site.json
- tests: internal/core/site, internal/adapter/hosting/githubpages, internal/surface/cli

## Steps

1. The project route carries its content policy in each page and emits no Cloudflare files (resolves iss-2610142036270003)
   - packages: internal/core/site, internal/adapter/hosting/githubpages
   - tests: internal/core/site, internal/adapter/hosting/githubpages
2. abcd's repository switches its record on to the project route, and its README and docs link the record at the Pages address
   - packages: .abcd/, .github/workflows, docs/
   - tests: internal/core/site, internal/surface/cli
3. abcdev.app's deploy narrows to the documentation and forwards the record's addresses, with the decision record amending adr-48
   - packages: internal/core/site, site-src, .github/workflows, docs/
   - tests: internal/core/site
4. The cut-over, each outward act confirmed by the person: the first Pages release, then the forwarding, then dropping the record from abcdev.app's build
   - packages: .github/workflows, site-src
   - tests: internal/core/site
