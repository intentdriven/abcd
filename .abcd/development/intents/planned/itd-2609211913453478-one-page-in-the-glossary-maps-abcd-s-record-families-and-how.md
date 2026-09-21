---
id: itd-2609211913453478
slug: one-page-in-the-glossary-maps-abcd-s-record-families-and-how
spec_id: spc-2609212131112235
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-34, itd-2609212103565953]
related_intents: [itd-24, itd-42, itd-172, itd-78]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_adrs: [adr-2609212115255771]
---

# One page maps the record families, and phase, milestone and roadmap are retired

## Press Release

> **One glossary page maps abcd's record families and how they relate, and three words leave the vocabulary.**
>
> "I kept asking which word to use, and every answer named a different document," said a product thinker who had just approved bundles and wondered whether phases still meant anything. "Now there is one page: intent, spec, step, bundle, issue, release, status. Phase and milestone are on it too, marked superseded, with what replaced them. I read it in five minutes."

## Why This Matters

On 2026-09-21 the product thinker asked, mid-interview, whether abcd has an intent that sorts out its vocabulary and whether phases survive bundles. A glossary existed with phase, intent, spec, roadmap, record and ledger defined in prose and adr-9 making the phase the product layer; nothing drew the map, `bundle` had no entry, and the autonomous run had added a third grouping, the batch. The research pass and the discussion that followed retired phase, milestone and roadmap and named their successors (adr-2609212115255771); this page is where a reader learns that.

## Mechanism

We expect one page that every glossary entry points at to stop new second names for one concept, because a name that must earn a row on a map is a name someone has to justify; shown wrong if a new record family or grouping word appears in the next release without a row on the page.

## Scope Conditions

None stated.

## What's In Scope

- **One page**, `glossary/core/record-families.md`: one row per family, intent, spec, step, bundle, issue, release, status (Now / Next / Later), with the definition, what it groups, what groups it, its lifecycle folders and the verb that moves it; every other glossary entry's `not_to_be_confused_with` may name only a family on the page.
- **Superseded terms**: `phase`, `milestone` and `roadmap` marked `status: superseded` with the successor named (dependencies and the status block; the derived release and each intent's criteria; the status block); the phase documents under the roadmap folder carry a retirement line and stay as history.
- **New entries**: `bundle` (itd-34) and `step` (itd-2609212103565953); `batch` defined on the page as the run's internal order, not a term.
- **The lint**: a glossary entry whose `not_to_be_confused_with` names nothing on the page is refused; a record frontmatter key naming a family the page does not define is reported.
- **The brief's mental-model chapter** reads brief → intent → spec (→ steps), with the bundle as a delivery grouping and the derived release as the checkpoint; adr-9 is superseded by adr-2609212115255771.

## What's Out of Scope

- Renaming any family or folder.
- The dependency graph between intents (itd-78).
- A release press release at the phase's old granularity (named in the decision record's consequences, not ruled).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves; the checkpoint is the derived release plus each intent's criteria (adr-2609212115255771).
2. Now / Next / Later is a rendered status, never stored, and the word roadmap goes with the phase documents.
3. The unit below a spec is the step, a section, not a record family; an issue carries no spec by design; the batch is the run's internal order.
4. One term per concept: the page is the map every entry points at.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** the glossary, **when** the page is read, **then** it holds one row per family (intent, spec, step, bundle, issue, release, status) with the definition, what it groups, what groups it, its lifecycle folders and the verb that moves it, and every other entry points at it.
- **Given** `phase`, `milestone` and `roadmap`, **when** their entries are read, **then** each is marked superseded with its successor named, and the phase documents carry a retirement line.
- **Given** `bundle` and `step`, **when** the glossary is read, **then** each has an entry, and `batch` is defined on the page as the run's internal order.
- **Given** an entry whose `not_to_be_confused_with` names nothing on the page, or a record frontmatter key naming a family the page does not define, **when** the record lint runs, **then** the first is refused and the second reported.
- **Given** the brief's mental-model chapter, **when** it is read, **then** it reads brief → intent → spec (→ steps) with the bundle and the derived release named, and adr-9 reads superseded by adr-2609212115255771.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: three terms are retired today and the map is where a reader learns what replaced them; we expect no new grouping word to appear without a row; shown wrong if one does in the next release
