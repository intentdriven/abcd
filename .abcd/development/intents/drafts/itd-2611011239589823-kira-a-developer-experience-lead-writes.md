---
id: itd-2611011239589823
slug: kira-a-developer-experience-lead-writes
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611011151475128, itd-2611011151512956]
severity: minor
refines: [itd-2611011151475128]
related_adrs: [adr-2611011153058085]
origin: researcher-authored
production_mode: hand-written
---

# A project's how-to guides and learning material stay current with the product

## Press Release

> Kira, a developer-experience lead, writes the tutorials for a product their team did not build. A repository managed by abcd may have a second authored surface beside the brief: the how. Where it exists, it holds the how-to guides, which the product team owns, and, if the project wants it, learning material such as tutorials, exercises and videos, which educators and trainers may own. It lives in the same repository and the same site as the rest of the documentation, edited with the same files, reviews and gates as the brief, so an educator needs no second tool. The how stays current with the product: every step a guide or tutorial tells a reader to take is checked against the product on every change, a scripted recording is regenerated from its script, versions and flags are filled in from the release instead of typed, a hand edit to a rendered page is caught and offered back, and every change that alters behaviour says whether the how needs updating, so a release cannot ship with a needed update left undone.

## Why This Matters

_Written by the agent on 2026-11-01 from the product thinker's rulings that
day (dcn-2611011248594647, dcn-2611011551216958) and two state-of-the-art
reviews, restated; the planning interview confirms or rewrites it._

Rendering every documentation surface from its one home
(itd-2611011151475128) leaves the pages no generator can write: how-to guides
and tutorials follow the user's goals, not the product's structure, so a
person writes them. The product thinker ruled on 2026-11-01 that an
abcd-managed repository has two surfaces people write in, restated:

- **The brief**, for what the product is and why, written by the product
  thinker and the team.
- **The how**, for how-to guides, tutorials and other learning material
  (exercises, videos, depending on the project), often written by a separate
  team of educators or trainers working from the brief and the prototype.
  Not every repository needs it, but abcd is prepared for it: keeping it
  current, hand editing with the two-way lane (itd-2611011151512956), and
  collaborative editing by its own stakeholders, all through the standard
  abcd toolchain, as for the brief.
- **Everything else is generated**: reference from the command tree and
  schemas, install facts from the release, badges, citation and links.

What the state-of-the-art review (2026-11-01) changed. The product thinker
confirmed the routing that carried these changes the same day
(dcn-2611011551216958 records the rulings); a bullet the decision does not
name is the agent's proposal for the planning interview, and says so:

- **Separate the owners, not the destination.** The how has its own owners
  in the same repository, site and navigation, never a second site: a project
  that split its learning material into a second resource merged it back
  when readers found two confusing, and learning material hosted on an
  outside platform stopped working when the platform closed (the research
  note's review 2).
- **How-to guides are the product team's; learning material may be the
  educators'.** A how-to guide serves a competent user at work and changes
  with every product change, so it drifts first under an optional owner. The
  how is optional; where it exists, it holds both kinds, owned apart through
  review routing, with the how-to guides the product team's and the learning
  material the project's choice.
- **Tests keep it current, not review alone.** Every step a guide or
  tutorial tells a reader to take is checked against the built product as a
  test, in whatever form the product takes (a command run, a page visited, a
  call made), exercises are checked the way exercise tracks check theirs, a
  scripted recording is regenerated from its script on every change, and a
  hand-recorded video carries the version it shows and a review date.
  Proposed by the agent: a repository that adopts the how adopts its tests as
  gates, so the choice to have the how is the opt-in.
- **Facts are filled in, never typed.** Proposed by the agent: versions,
  command names and flags in an authored page come from the release, never
  written as a version into a committed page (adr-19), so a tutorial cannot go
  on naming a release long since replaced. Where they are filled in is open in
  itd-2611011151475128, since a managed repository's docs are read on the
  forge with no site build (itd-2610091010348892).
- **Every change that alters behaviour says what it does to the how.** A
  docs-impact state, needed, done or not needed, is recorded on the change, so
  "not needed" is a choice rather than a silence, and the release cut refuses
  while a needed update is still open.
- **Editing starts with the forge's own editor and previews.** A content
  system for non-developers is added only once a real team of them exists.

Links to the record, typed:

- **refines** itd-2611011151475128: its how-to and tutorial pages come from
  this surface.
- **builds on** itd-2611011151512956 and adr-2611011153058085: the two-way
  lane and the never-discard rule cover the how as they cover the brief.

## SOTA

`.abcd/development/research/notes/2026-11-01-documentation-sources-sota.md`
(review 2): the model is supported with changes. One site and toolchain with
separate owners; how-to guides owned by the product team; tutorials and
exercises runnable and run in CI; release facts filled in rather than typed; a
three-state docs-impact mark with a release check; scripted recordings
regenerated; the forge's editor before any content system.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **The how's home in the tree.** Where the folder lives (under the
  development record, or beside `docs/` as the authored source it renders
  from), and how its owners are declared.
- **The docs-impact state's carrier.** A pull-request label, a commit
  trailer, a field on the record the change lands with, or the record lint's
  own check; and which changes count as altering behaviour.
- **Docs-impact state as its own intent.** Whether the docs-impact state and
  its release check stay in this intent or become their own, since they serve
  the brief-rendered pages as well as the how.
- **Educators as stakeholders.** Whether educators and trainers become a role
  of their own beside the product thinker and the technical facilitator, with
  a glossary entry and a persona, or stay a stakeholder group the how names.
- **Video and exercises.** Which kinds abcd supports first, and how a
  hand-recorded video's review date is enforced.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
