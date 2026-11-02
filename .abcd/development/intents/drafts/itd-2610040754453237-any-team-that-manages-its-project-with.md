---
id: itd-2610040754453237
slug: any-team-that-manages-its-project-with
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The public record site for every abcd-managed project

## Press Release

> Any team that manages its project with abcd publishes the public record site for their own project, the timeline, the decisions and the announcements, as abcd publishes its own.

_Filed by the facilitator on 2026-10-04 at the product thinker's request: "a follow-up intent to make the public web dashboard available for all abcd-managed repos"; the product thinker, verbatim: "this should be a separate intent for later, not now"._

## Why This Matters

_Facilitator-written._

abcd publishes its own record as a public site. Other teams managing their project with abcd cannot yet do the same with confidence: the public half of the team-site idea (itd-139) was retired on 2026-10-03 when the private dashboard replaced it. This draft brings the public site back, separately and for later. Its promise to work for any project is held to the discipline itd-140: a second, sparse sample project proves it, shared with itd-2610040754440360.

Typed links: builds on the record site; refines the public half of superseded itd-139; sibling of itd-2610040754440360.

## Decisions

1. 2026-10-04, the product thinker: a separate intent, for later, not now.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md), shared with itd-2610091010348892 (the opted-in project's record on its own GitHub Pages), which builds the route this draft generalises.

- **Hosting is settled by the sibling's ruling and the forge's own actions:** the project's workflow runs `configure-pages`, `upload-pages-artifact` (a single tar, no symlinks, under 1 GB supported) and `deploy-pages` (`pages: write`, `id-token: write`, the `github-pages` environment); a site may be no larger than 1 GB, deploys time out at 10 minutes, and the ten-builds-an-hour limit does not apply to a custom workflow (GitHub Pages docs and action READMEs).
- **A project site lives under a path,** `<owner>.github.io/<repositoryname>`; `configure-pages` outputs `base_path`, so the renderer must take one.
- **What stays private is decided by the plan, not by abcd:** an access-controlled Pages site needs Enterprise Cloud; elsewhere a site built from a private repository is public (Pages visibility; the partners note). A private repository is not offered the switch until a later intent; a public one publishes the committed tree only (`.abcd/development/` and `.abcd/work/`), and the local tier is never in the repository, so never on the site.
- **Pages' terms** forbid sensitive transactions and commercial SaaS and log visitor addresses on GitHub's side; a record site is informational, and the "report" link to GitHub matches who holds the logs.

**Path 1 for hosting — the project's own GitHub Pages through its own workflow, as ruled for the sibling — with the renderer native and base-path-aware; this draft adds only the default: public repositories opt in, a private one is told its site would be public and gets no switch yet.** The independent fit-challenge runs at plan time.

## Open Questions

- Whether it shares the sample project with the private dashboard's follow-up.
- What an adopting project publishes by default, and what stays private.
- (SOTA 2026-10-23) Whether the site workflow is copied into each project (then kept current by setup) or called as a reusable workflow from abcd's repository (then every project's build depends on abcd's default branch); the ruling's "the project's own workflow builds it" favours the copy.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
