---
id: spc-2609212141418943
slug: abcd-lab-mechanises-the-lab-conventions-three-hand-run
intent: itd-2609212137128014
origin: researcher-authored
production_mode: hand-written
---
# abcd-lab-mechanises-the-lab-conventions-three-hand-run

## Summary

The design record for itd-2609212137128014: the `abcd lab` verb family, the procedure record and the recording model, from the capstone under the machine-scoped lab store.

## Scope

1. **The store**: `~/.abcd/lab/<lab-id>/` with `index.jsonl`, keyed as the worktree and transcript stores are (criterion 1).
2. **mint / preflight / record / sweep / harvest** in `internal/core/lab`, each writing only under the lab home (criteria 1 to 4).
3. **The discipline record** under `intents/disciplines/`, from the capstone's amendment list (criterion 6).
4. **Halt-and-record**: the preflight and the sweep exit non-zero and write the finding (criterion 5).

## Out of scope

- Auto-merge; labs as a grouping; a next-labs runner.

## Approach

The capstone documents are the source: `series-review.md` for the procedure, `recording-model.md` for the store's shape, the harvests for the harvest template. The verb writes nothing into a repository; its output enters the record only through `capture`.

## Footprint

- packages: internal/core/lab, internal/surface/cli, .abcd/development/intents/disciplines
- tests: each verb over a fixture lab home; the sweep on a planted unapplied correction; the harvest shape

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 mint | scope 1, 2 |
| 2 preflight halts | scope 2, 4 |
| 3 sweep | scope 2 |
| 4 harvest | scope 2 |
| 5 halt-and-record | scope 4 |
| 6 discipline record | scope 3 |
