---
id: spc-2609212139587510
slug: abcd-s-verbs-consolidate-ahoy-s-three-modes-become-flags
intent: itd-2609212130136102
origin: researcher-authored
production_mode: hand-written
---
# abcd-s-verbs-consolidate-ahoy-s-three-modes-become-flags

## Summary

The design record for itd-2609212130136102: the verb consolidation, one breaking change beside the four renames.

## Scope

1. **Flags on `ahoy`** with the three sub-verbs kept one release as deprecated stubs that print the flag form and exit 2 (criterion 1).
2. **`--version`** on the root; `update --check`; `intent new` deleted (criterion 2).
3. **`lint` targets**: `lint` gains sub-verbs `docs`, `outbound`, `site`, `identity` that call the existing functions; the old verbs become stubs for one release; `docs` keeps `cite`, `site` keeps `build` (criterion 3).
4. **The snapshot** gains `moved_to` per stub; pages and brief chapters regenerated and edited; the intent's `impact: breaking` derives the cut (criterion 4).
5. **The count test** over the rendered default help (criterion 5).

## Out of scope

- The agent block; record-family verbs.

## Approach

Stubs for one release keep muscle memory from failing silently; the release after removes them (a follow-on remainder spec, minted at close).

## Footprint

- packages: internal/surface/cli, internal/core/surface, commands/, .abcd/development/brief/04-surfaces
- tests: each stub's message and exit; each flag and target's behaviour equal to the old verb; the snapshot's moved_to; the count

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 ahoy flags | scope 1 |
| 2 --version, update --check, no intent new | scope 2 |
| 3 one lint | scope 3 |
| 4 snapshot, pages, breaking | scope 4 |
| 5 fourteen at most | scope 5 |
