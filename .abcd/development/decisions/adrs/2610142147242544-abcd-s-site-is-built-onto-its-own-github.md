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


# ADR-2610142147242544: abcd's record is built onto its own GitHub Pages, and abcdev.app becomes abcd's documentation site

## Context

adr-48 deploys abcdev.app to a Cloudflare Worker from released bytes at each release, record and docs together, with a preview of `main` on every push. itd-2610091010348892 gives every opted-in project a record built onto its own GitHub Pages by a workflow abcd writes. The product thinker made three rulings:
- On 2026-10-14, that abcd's own repository moves to that route ("so that it no longer has to deploy to cloudflare").
- On 2026-10-15, that a site renders only what abcd writes to a repository anyway, with docs read on GitHub by default.
- Also on 2026-10-15, that for abcd itself "docs are rendered in high-quality, easily accessible way at abcdev.app for now", and that "abcdev.app is the abcd docs, abcdesign.app is the team-brief edits for any abcd-managed repos".

The move is planned in itd-2610142145236613 and built later.

## Decision

Proposed with the plan. It is accepted in the change that lands spc-2610142146250176's third step, when the code that enforces it lands.

1. **abcd's about page and record are built and deployed by the workflow `site setup --provider github-pages` writes for any project,** from what abcd writes and nothing else. abcd's own needs are met inside that route, not beside it.
2. **abcdev.app is abcd's documentation site until a community site replaces it.** Its deploy (adr-48: on release, from released bytes, with the preview of `main`) narrows to the documentation and its landing page. The record explorer leaves it.
3. **The record's old addresses on abcdev.app (`/record/`, `/contributors/`, `/references/`) forward with a 301 to the same path on abcd's GitHub Pages address.** The forwarding is declared in the docs host's redirect file in the repository. abcd's own links point at the Pages address directly.
4. **What GitHub Pages cannot carry is named, not hidden.** On the project route, the document content policy moves into each page as an element. Framing refusal (`frame-ancestors`) and `X-Content-Type-Options: nosniff` are lost there. The record loads nothing from outside itself and holds nothing a framing page could act on, so the loss is accepted for a read-only record. abcdev.app keeps its response headers on its own host.
5. **The static abcdesign.app page is published from a repository of its own,** because a repository has one Pages site and abcd's is the record.

## Alternatives Considered

- **Keep the Cloudflare deploy as it is (adr-48 unchanged).** It keeps everything in one place. But abcd's record would not run on the route abcd offers, against the product thinker's 2026-10-14 ruling.
- **Move everything, docs included, onto abcd's GitHub Pages and forward all of abcdev.app.** This was the plan of 2026-10-14. The single-source ruling of 2026-10-15 rules out rendering hand-written docs on a project site, and the product thinker kept abcdev.app for abcd's docs.
- **Read abcd's docs on GitHub, like any project's.** That is the default for every other repository. The product thinker chose a high-quality docs site for abcd itself, for now.
- **The record on the project route, abcdev.app narrowed to the docs, and the record's old addresses forwarded (chosen).**

## Consequences

- abcd's record depends on GitHub Pages' availability and limits (1 GB per site). The docs stay on their current host.
- Every project on the github-pages provider gains the page-carried content policy that abcd's record needs.
- abcdev.app's build drops the record explorer, and its redirect file gains the record's forwards. Cloudflare keeps serving the docs only.
- abcdesign.app's publishing repository is a new outward act at that intent's cut.
- When a community site replaces abcdev.app, the docs move with it, by a later decision.
