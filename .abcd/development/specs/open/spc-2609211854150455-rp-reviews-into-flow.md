---
id: spc-2609211854150455
slug: rp-reviews-into-flow
intent: itd-28
origin: researcher-authored
production_mode: hand-written
---
# rp-reviews-into-flow

## Summary

The design record for itd-28 as re-scoped on 2026-09-21: every review names
the commit it read, and the status board says how stale it has become. No
new store, no new scrubber.

## Scope

1. **The pin**: every path in abcd that files a review folder under
   `.abcd/work/reviews/<date>-<scope>/` writes `review_of_commit: <full sha>`
   into `00-summary.md`'s frontmatter, taken from the tree it reviewed; the
   gate receipts keyed by sha already carry the pin in their directory name
   (criterion 1).
2. **The lint**: the reviews-charter rule (RD001's sibling) refuses a review
   folder written after this ships without the key, and names a folder from
   before it as legacy rather than refusing (criterion 1).
3. **The board**: the bare `abcd` status board gains a reviews block: scope
   or spec id, `review_of_commit` short sha, commits since (`git rev-list
   --count <sha>..<default>`), and a flag past twenty; ordered stalest first
   (criteria 2, 5).
4. **The name**: a review of a spec is filed as `<date>-<spc-N>-<slug>/`, so
   the spec id is in the folder name; the board reads it from there
   (criterion 3).
5. **The scrub**: unchanged; the scanner's store-before-commit redactor and
   the pre-commit name-guard already run on the reviews tree (criterion 4).

## Out of scope

- A review store of abcd's own, a JSON sidecar, a two-stage redaction, a
  pre-commit verifier: dropped by decision 1.
- Re-running a stale review: the board flags, a person or a run re-runs.

## Approach

`internal/core/record` already reads the reviews tree for the charter check;
the pin is one more frontmatter key it reads and the lint one more rule
beside RD001. The board's block is computed in `internal/core/positioning`
(where the status board's other blocks live) from the same read plus one
`rev-list --count` per folder through `gitutil`. The writers are the few
places abcd itself files a review (the semantic-gate receipts already pin;
the intent-audit ingest and the reviews-charter template gain the key).

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 pin written, lint refuses its absence | scope 1, 2 |
| 2 board lists staleness, flags past twenty | scope 3 |
| 3 spec id in the folder name | scope 4 |
| 4 scrub is the existing scanner | scope 5 |
| 5 json carries the rows | scope 3 |
