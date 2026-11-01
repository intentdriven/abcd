---
id: itd-2610040754440360
slug: any-team-that-manages-its-project-with
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610032150577708]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The private dashboard for every abcd-managed project

## Press Release

> Any team that manages its project with abcd opens the same private dashboard on their own project, and it works there as it does on abcd's own, shown on a second, sparse sample project before it is promised.

_Filed by the facilitator on 2026-10-04 at the product thinker's request: "a follow-up intent to make the web dashboard available and workable for all abcd-managed repos". Routing confirmed by the product thinker: one draft, with the proof inside._

## Why This Matters

_Facilitator-written._

The dashboard (itd-2610032150577708) is promised for abcd's own record only: its decision 13 gave up the promise that the record works for any project, because the proof (a second, sparse sample project) went with the superseded itd-139. This follow-up brings that promise back for the private dashboard, and earns it the way the discipline itd-140 requires: it is not done until it runs on a second, sparse sample project.

Typed links: builds on itd-2610032150577708; carries the demonstration itd-140 names.

## Decisions

1. 2026-10-04, the product thinker, asked how to file it (one idea, proof inside; two ideas; decide later): one idea, proof inside. It is done only when the dashboard runs on a second, sparse sample project.
2. 2026-10-04, the facilitator: a follow-up, for later; itd-2610032150577708 decision 13 stands for the first version (flagged as a later return of the given-up promise, confirmed by the product thinker's request).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md). Most of this intent is internal plumbing; the one outside convention is where the proof lives.

- **The dashboard already reads a project, not abcd:** its listener, gate and control files under abcd's home "name no project content", and pages come from the record site's renderer by record id. A second project needs a tree the renderer and the status read can open, not a second dashboard.
- **A fixture tree is the Go convention for exactly this:** "The go tool will ignore a directory named "testdata", making it available to hold ancillary data needed by the tests" (cmd/go). A sparse sample project is a committed `testdata` tree with a `.abcd/` of a few records, opened by a test that starts the dashboard against it.
- **A separate sample repository** is the honest alternative and a second checkout to keep current; Hugo's theme-submission page names `theme.toml` and screenshots but the example-site requirement often associated with it was not on the page opened, so it is not claimed.

**Path 2 — native, shaped to the `testdata` fixture convention: the proof is a test that starts the dashboard on a committed sparse tree; an adopting project starts it with the same verb and no abcd-specific setup, and the test is what shows that claim true or false.** The independent fit-challenge runs at plan time.

## Open Questions

- What the sparse sample project holds, and where it lives.
- How an adopting project starts the dashboard, and what of abcd's own setup it needs.
- (SOTA 2026-10-23) Whether the discipline itd-140 names accepts a committed fixture tree as "a second, sparse sample project", or needs a real checkout a team could manage.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
