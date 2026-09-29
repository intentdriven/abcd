---
id: spc-2609212138241908
slug: the-bare-abcd-status-board-and-the-site-s-status-page-show
intent: itd-2609212103568351
origin: researcher-authored
production_mode: hand-written
---
# the-bare-abcd-status-board-and-the-site-s-status-page-show

## Summary

The design record for itd-2609212103568351: the Now / Next / Later block on the status board and the site, rendered from the lifecycle shelves, the readiness gate and the build's state file.

## Scope

1. **The read** (`internal/core/positioning/status.go`): the planned and draft shelves, `intent.Readiness` per planned record, the build's state file for lanes, and `implement.PickOrder` for the head (criteria 1, 3).
2. **The board block**: rendered after the existing board sections, text and `--json` (criteria 1, 4).
3. **The site page**: `internal/core/site` gains `status.go` calling the same function; the page is opt-in per the site configuration like the others (criterion 2).
4. **No write**: the function is pure over its reads (criterion 3).
5. **The word**: a test greps both renders for "roadmap" (criterion 5).

## Out of scope

- A stored started state; the pick; other site pages.

## Approach

One function in positioning, two renderers; the pick-order head is read through the same entry point `build next` uses so the two never disagree.

## Footprint

- packages: internal/core/positioning, internal/core/site, internal/surface/cli
- tests: the block over a fixture store with and without a state file; the json shape; the site page; the roadmap grep

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 the block | scope 1, 2 |
| 2 the site page | scope 3 |
| 3 no state file | scope 1, 4 |
| 4 json | scope 2 |
| 5 no roadmap | scope 5 |
