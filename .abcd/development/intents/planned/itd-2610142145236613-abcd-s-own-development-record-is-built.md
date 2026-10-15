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

> abcd's own development record is built the way any opted-in project's is. At each release, abcd's release workflow renders its about page and record onto abcd's own GitHub Pages address, from what abcd writes to the repository and nothing else, and GitHub builds and hosts it. abcdev.app becomes abcd's documentation site, rendered in high quality until a community site takes its place, and a link to the record shared before the move still lands on the same page at its new address. abcdesign.app is where a team edits its brief together, for any abcd-managed repository.

## Why This Matters

abcd offers every managed project a record built onto its own GitHub Pages at each release (itd-2610091010348892). abcd's own record should be built the same way, so the route abcd offers is the route abcd runs on, and abcd stops deploying to a host of its own. The product thinker, 2026-10-14, verbatim: "if a user opts-in, abcd automatically builds a github.io site with each release. We should test that with the abcd repo at one point so that it no longer has to deploy to cloudflare. The cloudflare abcdev.app will then become a polished version specifically for the abcd community (if there's a need for that)".

## Mechanism

We expect the move to hold because abcd's site is then built by the same workflow every opted-in project gets, so each abcd release exercises the route abcd offers. The claim fails if abcd's site needs a step no other project could declare, or if links shared before the move stop landing on their pages.

## Scope Conditions

- abcd's repository stays public, which the project-site route requires. <!-- cond: cond-2610142146255098 -->
- The abcdev.app domain stays registered, at a host that builds and serves abcd's documentation and can forward the record's old addresses. <!-- cond: cond-2610142146253871 -->
- GitHub Pages stays free for public repositories, and abcd's site fits its limits (at most 1 GB). <!-- cond: cond-2610142146250921 -->

## Acceptance Criteria

- **Given** abcd publishes a release, **when** its site workflow runs, **then** abcd's about page and record appear at abcd's own GitHub Pages address, as of that release, built by the workflow that `site setup` writes for any project, from what abcd writes to the repository and nothing else.
- **Given** abcdev.app, **when** a visitor opens it, **then** it shows abcd's documentation, rendered in high quality and easy to find one's way around, until a community site replaces it.
- **Given** a link to abcd's record on abcdev.app shared before the move, **when** a visitor opens it, **then** they land on the same page of abcd's GitHub Pages site, with the path kept.
- **Given** a link to a docs address that moved under /docs/ before, **when** a visitor opens it, **then** it still lands on its page on abcdev.app.
- (technical) **Given** the move has landed, **when** the next release publishes, **then** abcdev.app's deploy builds the documentation only, and the record, about page and their pages are built only by the GitHub Pages workflow.
- (technical) **Given** any document on abcd's GitHub Pages site, **when** a browser loads it, **then** it carries the content policy the Cloudflare headers carry today, as an element in the page, and the protections GitHub Pages cannot carry (framing refusal, the no-sniff header) are named in the decision record.
- (technical) **Given** the move, **when** it lands, **then** the decision record that set up the Cloudflare deploy (adr-48) is amended by a recorded decision: abcdev.app's deploy narrows to the documentation.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-14: this intent was split out of itd-2610091010348892, whose press release ended "abcd's own record moves to the same route, built exactly the way any other project's is" and whose fourth criterion covered the move. The product thinker had said the move happens "at one point", and on 2026-10-14 asked, verbatim: "keep working so that we have this intent planned and ready to implement. We'll have to delay building it but I want it to be ready". The move is planned here and built later; itd-2610091010348892 ships without it.
- 2026-10-14: asked what abcdev.app shows once abcd's record builds onto its own GitHub Pages (options: the same site on the new host, forward to github.io, park the domain), the product thinker chose "Forward to github.io": abcdev.app forwards every address to the same page of abcd's GitHub Pages site, by one forwarding rule kept at the domain.
- 2026-10-15: the product thinker ruled that a site renders only what abcd writes to a repository anyway (itd-2610091010348892, Decisions of 2026-10-15), and, asked where hand-written docs live, answered verbatim: "docs read on github as default for all abcd-managed repos. For the abcd repo, docs are rendered in high-quality, easily accessible way at abcdev.app for now (unitl the community site arrives)." Then, verbatim: "so: abcdev.app is the abcd docs, abcdesign.app is the team-brief edits for any abcd-managed repos". This replaces the 2026-10-14 "Forward to github.io" answer for abcdev.app as a whole: abcdev.app serves abcd's docs, and only its record addresses forward to abcd's GitHub Pages site. The press release, the criteria on abcdev.app and the docs step are rewritten to say so; the docs step leaves the project workflow, since no project's site renders docs.

## Grounds

- pursued: abcd's site moves onto the route it offers every project because the same workflow then builds both, so each abcd release exercises that route; shown wrong if abcd's site needs a step no other project could declare, or if links shared before the move stop landing
