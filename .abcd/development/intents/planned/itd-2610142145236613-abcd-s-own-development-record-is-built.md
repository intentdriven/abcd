---
id: itd-2610142145236613
slug: abcd-s-own-development-record-is-built
spec_id: spc-2610142146250176
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610091010348892]
severity: minor
impact: additive
origin: researcher-authored
related_adrs: [adr-47, adr-48, adr-2610142147242544]
production_mode: hand-written
blocked_by: [itd-2610091010348892]
---

# abcd's own record is built onto its GitHub Pages like any project's

## Press Release

> abcd's own development record is built the way any opted-in project's is. At each release, abcd's release workflow renders its about page, docs and record onto abcd's own GitHub Pages address, and abcdev.app forwards every address there, so a link shared before the move still lands on the same page. Nothing abcd runs serves the site any more: GitHub builds and hosts it, as it does for every project that switched its site on.

## Why This Matters

abcd offers every managed project a record built onto its own GitHub Pages at each release (itd-2610091010348892). abcd's own record should be built the same way, so the route abcd offers is the route abcd runs on, and abcd stops deploying to a host of its own. The product thinker, 2026-10-14, verbatim: "if a user opts-in, abcd automatically builds a github.io site with each release. We should test that with the abcd repo at one point so that it no longer has to deploy to cloudflare. The cloudflare abcdev.app will then become a polished version specifically for the abcd community (if there's a need for that)".

## Mechanism

We expect the move to hold because abcd's site is then built by the same workflow every opted-in project gets, so each abcd release exercises the route abcd offers. The claim fails if abcd's site needs a step no other project could declare, or if links shared before the move stop landing on their pages.

## Scope Conditions

- abcd's repository stays public, which the project-site route requires. <!-- cond: cond-2610142146255098 -->
- The abcdev.app domain stays registered, at a host that can forward addresses by a rule without running code. <!-- cond: cond-2610142146253871 -->
- GitHub Pages stays free for public repositories, and abcd's site fits its limits (at most 1 GB). <!-- cond: cond-2610142146250921 -->

## Acceptance Criteria

- **Given** abcd publishes a release, **when** its site workflow runs, **then** abcd's about page, docs and record appear at abcd's own GitHub Pages address, as of that release, built by the workflow that `site setup` writes for any project.
- **Given** any address on abcdev.app, **when** a visitor opens it, **then** they land on the same page of abcd's GitHub Pages site, with the path kept.
- **Given** a link to a docs address that moved under /docs/ before the move, **when** a visitor opens it, **then** it still lands on its page.
- **Given** the move has landed, **when** the next release publishes, **then** nothing is deployed to Cloudflare, and the Cloudflare deploy, its configuration and its secrets are gone from the repository.
- (technical) **Given** abcd's docs, **when** the workflow builds the site, **then** /docs/ is built in the same run, through an opt-in docs step any project can declare, with every dependency pinned by hash.
- (technical) **Given** any document on the site, **when** a browser loads it, **then** it carries the content policy the Cloudflare headers carry today, as an element in the page, and the protections GitHub Pages cannot carry (framing refusal, the no-sniff header) are named in the decision record.
- (technical) **Given** the move, **when** it lands, **then** the decision record that set up the Cloudflare deploy (adr-48) is superseded in that part by a recorded decision.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-14: this intent was split out of itd-2610091010348892, whose press release ended "abcd's own record moves to the same route, built exactly the way any other project's is" and whose fourth criterion covered the move. The product thinker had said the move happens "at one point", and on 2026-10-14 asked, verbatim: "keep working so that we have this intent planned and ready to implement. We'll have to delay building it but I want it to be ready". The move is planned here and built later; itd-2610091010348892 ships without it.
- 2026-10-14: asked what abcdev.app shows once abcd's record builds onto its own GitHub Pages (options: the same site on the new host, forward to github.io, park the domain), the product thinker chose "Forward to github.io": abcdev.app forwards every address to the same page of abcd's GitHub Pages site, by one forwarding rule kept at the domain.

## Grounds

- pursued: abcd's site moves onto the route it offers every project because the same workflow then builds both, so each abcd release exercises that route; shown wrong if abcd's site needs a step no other project could declare, or if links shared before the move stop landing
