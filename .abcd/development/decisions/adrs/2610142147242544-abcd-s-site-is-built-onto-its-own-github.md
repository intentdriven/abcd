---
id: adr-2610142147242544
slug: abcd-s-site-is-built-onto-its-own-github
status: proposed
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610142145236613, itd-2610091010348892]
related_rfcs: []
related_adrs: [adr-47, adr-48, adr-2610091016270711]
---

# ADR-2610142147242544: abcd's site is built onto its own GitHub Pages and abcdev.app forwards there

## Context

adr-48 deploys abcdev.app to a Cloudflare Worker from released bytes at each release. It also uploads a non-production preview of `main` on every push. itd-2610091010348892 gives every opted-in project a record built onto its own GitHub Pages by a workflow abcd writes. On 2026-10-14 the product thinker ruled that abcd's own repository moves to that route ("so that it no longer has to deploy to cloudflare"), and that abcdev.app then forwards to abcd's GitHub Pages address ("Forward to github.io"). The move is planned in itd-2610142145236613 and built later.

## Decision

Proposed with the plan. It is accepted in the change that lands spc-2610142146250176's third step, when the code that enforces it lands.

1. **abcd's site is built and deployed by the workflow `site setup --provider github-pages` writes for any project.** abcd's own needs are closed in that route, not beside it: the opt-in docs step, page-carried content policy, and redirect pages for moved addresses.
2. **adr-48's rule that the site deploys on release, from released bytes, stands.** Only its Cloudflare deploy and its preview of `main` are superseded. A repository has one Pages site, so there is no preview. Every change is still rendered and checked by the site-render gate.
3. **abcdev.app forwards every address, path and query kept, with a 301 to the same page on abcd's GitHub Pages address.** It uses one redirect rule at the domain's host, with no Worker and no deploy. abcd's own links point at the Pages address directly.
4. **What GitHub Pages cannot carry is named, not hidden.** The document content policy moves into each page as an element. Framing refusal (`frame-ancestors`, which a page element cannot set) and `X-Content-Type-Options: nosniff` (a response header Pages does not let a site set) are lost on this host. The site loads nothing from outside itself and holds nothing a framing page could act on, so the loss is accepted for a read-only record.
5. **The static abcdesign.app page is published from a repository of its own,** because abcd's single Pages site is the record.

## Alternatives Considered

- **Keep the Cloudflare deploy (adr-48 as is).** It keeps the response headers and the preview. But abcd would keep running on a route it does not offer, against the product thinker's ruling.
- **Keep abcd's own bespoke site workflow and swap only its deploy step to Pages.** That is the least change. But abcd's site would not be built "the same way as any other project's", so each release would prove nothing about the route projects get.
- **Point abcdev.app at the Pages site as its custom domain.** Shared links keep working with no forwarding rule. The product thinker chose forwarding instead, leaving abcdev.app free for a community site.
- **The generic route, with abcd's gaps closed in it, and abcdev.app forwarding (chosen).**

## Consequences

- abcd's site depends on GitHub Pages' availability and limits (1 GB per site) instead of Cloudflare's.
- Every project on the github-pages provider gains the docs step, the page-carried policy and the redirect pages, so the work abcd needs is shared.
- Cloudflare is left holding one forwarding rule for abcdev.app and nothing else. Its Worker, the `site` and `site-preview` environments and their secrets are removed once the forward is checked.
- abcdesign.app's publishing repository is a new outward act at that intent's cut.
