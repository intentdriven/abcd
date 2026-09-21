---
id: itd-2609211913453478
slug: one-page-in-the-glossary-maps-abcd-s-record-families-and-how
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-34, itd-78]
related_intents: [itd-24, itd-42, itd-172]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# One page maps the record families and how they relate, and says whether a phase still is one

## Press Release

> One page in the glossary maps abcd's record families and how they relate: intent, spec, bundle, phase, batch, issue, roadmap and release each get one definition and one line saying what they group, what groups them, and which verb moves them. A product thinker reads it in five minutes and knows which word to use; bundle and batch gain entries; and the page answers whether a phase is still the sequencing layer now that bundles group intents that ship together and a run works in batches.

## Why This Matters

On 2026-09-21 the product thinker asked, mid-interview, whether abcd has an intent that sorts out its own vocabulary (intents, issues, phases, bundles, roadmap, specs) and how they relate, and whether phases are still wanted now that bundles exist. The answer was: a glossary exists (`.abcd/development/brief/glossary/`, with phase, intent, spec, roadmap, record and ledger defined in prose and adr-9 making the phase the product layer between the brief and the intent), but nothing draws the map, `bundle` has no entry, and the autonomous run adds a third grouping, the batch. Three ways of grouping intents is one too many for a vocabulary a product thinker is meant to hold; the evidence that phases have thinned is that no spec carries a phase anchor and phase membership is recorded editorially in the phase document.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## What's In Scope

- **One page**, `glossary/core/record-families.md` or its equivalent, with one row per family: the noun, one definition, what it groups, what groups it, its lifecycle folders, and the verb that moves it; the page is the single source the other entries point at.
- **Two new entries**: `bundle` (intents sharing one spec because they ship as one change; itd-34) and `batch` (the order an autonomous run takes lanes in; the run file).
- **The phase question, answered on the page**: whether a phase remains the sequencing layer, is folded into bundles, or is replaced by the run's batches; the answer is a decision recorded before the page ships, and the retrospective intent (itd-24) and the bundle rule in itd-34 follow it.
- **The lint**: every glossary entry's `not_to_be_confused_with` names a family on the page; a family named in a record's frontmatter that the page does not define is a finding.

## What's Out of Scope

- Renaming any family or any folder.
- The dependency graph between intents (itd-78).

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **Do phases stay?** Kept as the sequencing layer ending in a milestone; folded into bundles (a bundle is the only grouping, and the roadmap orders bundles); or replaced by the run's batches (the run file's routing is the roadmap). Each answer changes itd-24 and itd-34.
- **Where the page lives**: one glossary page, or the brief's mental-model chapter with the glossary pointing at it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
