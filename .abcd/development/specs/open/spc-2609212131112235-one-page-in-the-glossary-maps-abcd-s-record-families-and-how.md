---
id: spc-2609212131112235
slug: one-page-in-the-glossary-maps-abcd-s-record-families-and-how
intent: itd-2609211913453478
origin: researcher-authored
production_mode: hand-written
---
# one-page-in-the-glossary-maps-abcd-s-record-families-and-how

## Summary

The design record for itd-2609211913453478: the record-families page, the three retirements and the lint, from the vocabulary rulings of 2026-09-21 (adr-2609212115255771).

## Scope

1. **The page** `glossary/core/record-families.md`, a table with the seven families and a prose paragraph per axis (lifecycle; position); `glossary/core/README.md` indexes it first (criterion 1).
2. **Superseded entries**: `phase.md`, `roadmap.md` set `status: superseded`, `superseded_by` naming the page and the successor; a `milestone.md` entry is added in the superseded state, since the word had only lived as a forbidden synonym; `roadmap/README.md` and `roadmap/phases/README.md` open with the retirement line (criterion 2).
3. **New entries** `bundle.md`, `step.md` from the template, `status: stable`; the page's batch row (criterion 3).
4. **The lint**: `glossary_terms` gains the two checks beside the existing entry-shape check (criterion 4).
5. **The brief**: `01-product/03-mental-model.md` rewritten to the three layers plus steps, bundle and release; the surface chapters that say "phase" repointed (criterion 5).

## Out of scope

- Renames; itd-78; the release press release.

## Approach

Prose and one lint rule; the lint reads the page's table as the closed set of families. The retirement lines are one sentence each pointing at the decision record.

## Footprint

- packages: internal/core/lint (glossary_terms), .abcd/development/brief/glossary, .abcd/development/brief/01-product
- tests: the two lint checks over fixtures; the page's table parsed; the superseded entries' status

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 the page | scope 1 |
| 2 superseded with successors | scope 2 |
| 3 bundle, step, batch | scope 3 |
| 4 the lint | scope 4 |
| 5 the brief and adr-9 | scope 5 |
