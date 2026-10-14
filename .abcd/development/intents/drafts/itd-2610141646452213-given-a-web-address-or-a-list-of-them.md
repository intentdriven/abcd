---
id: itd-2610141646452213
slug: given-a-web-address-or-a-list-of-them
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610141646126492, itd-2610141646454347, itd-2610141646456128]
related_issues: [iss-2610141647294219]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Ideate starts from a source through the librarian

## Press Release

> **A person gives ideate a web address, a list of them, or files, and gets the
> ideas in them tested without rewriting them by hand.** Ideate has the
> librarian read the sources into a private reading pack, shows the ideas it
> found, ranked, and the person picks which to test; in an automatic mode it
> takes the top few and tests them one at a time. Every verdict records which
> pack items it tested and which ideas were offered.
>
> "I pasted the address of an engineering write-up and got five ideas back, each
> with its sentence," said Carol, a tech lead. "I picked two. Each verdict says
> which paragraph it came from."

## Why This Matters

Ideate takes one idea already worded. Given a source, the runner improvises the
extraction, and its choices are recorded nowhere (iss-2610141647294219). The
product thinker ruled the selection on 2026-10-14: the person picks from a
ranked list, or an automatic mode ranks and tests one at a time.

## What's In Scope

- Web addresses, lists of them and local files as ideate input, through the
  librarian's `library pack` step (itd-2610141646126492).
- The pick, the automatic mode and its cap, and the record of both.
- What the leg-3 reviewers receive from a pack.

## What's Out of Scope

- The pack itself (itd-2610141646126492) and the verdict schema
  (itd-2610141646454347).

## Mechanism

We expect source-to-verdict runs to be auditable because the extraction is moved
out of the runner's head into a recorded pack, and the pick (ideas offered,
ideas chosen) is recorded with each verdict. Shown wrong if a verdict from a
source run cannot be traced to the pack items it tested.

## Scope Conditions

- A typed idea keeps working without a librarian run, recorded under the
  version 2 verdict schema.
- A verdict whose idea came from a private pack is always private (the product
  thinker's ruling of 2026-10-14), so pack citations never enter a committed
  record.
- Leg 3 stays off-policy: reviewers receive the pack's quoted items only, never
  its ranking, lens readings or project fit.

## Acceptance Criteria

- **Given** a web address, a list of them or a local file as ideate's input,
  **when** the run starts, **then** a reading pack is recorded and the person is
  shown its ranked ideas to pick from.
- **Given** automatic mode, **when** the run starts, **then** the top ideas up to
  the cap are tested one after another with no question asked, and automatic
  mode refuses to start unless private or public is given.
- **Given** a verdict from a source run, **when** it is recorded, **then** it
  carries the ideas offered and the ideas picked, and cites the pack items it
  tested, each resolving in the local pack store.
- **Given** a verdict whose idea came from a private pack, **when** it is
  recorded, **then** it is recorded privately, and recording it publicly is
  refused.
- **Given** a source run, **when** the leg-3 reviewers are briefed, **then** the
  brief holds the pack's quoted items and nothing of its ranking, lens readings
  or project fit.
- **Given** a typed idea, **when** ideate runs, **then** no librarian run is
  required.

## Prior Art

- itd-104 (shipped): ideate's admission gauntlet, which this feeds.
- iss-2610141647294219 (open): the lab finding this delivers.

## Open Questions

- The automatic mode's flag name and its default cap (technical facilitator, at
  planning).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
