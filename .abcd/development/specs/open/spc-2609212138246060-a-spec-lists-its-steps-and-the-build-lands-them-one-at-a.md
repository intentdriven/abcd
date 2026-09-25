---
id: spc-2609212138246060
slug: a-spec-lists-its-steps-and-the-build-lands-them-one-at-a
intent: itd-2609212103565953
origin: researcher-authored
production_mode: hand-written
---
# a-spec-lists-its-steps-and-the-build-lands-them-one-at-a

## Summary

The design record for itd-2609212103565953: the `## Steps` section and the loop's per-step lanes.

## Scope

1. **The template**: `## Steps` seeded empty by `intent plan`; the readiness gate reports the section's shape (a list or empty) as advisory (criterion 1).
2. **The parser**: `spec.Steps(spec)` returns the ordered list with footprints, or one implicit step (criterion 1).
3. **The loop**: the lane in the state file gains `step: n/N`; `implement` starts the next step's lane only when the previous lane's pull request is an ancestor of the default branch (criterion 2).
4. **The remainder**: `spec close --remainder` copies steps not marked landed into the new spec (criterion 3).
5. **Briefs and the run record**: the brief renderer prints the step and its predecessors; the record lists `step` per lane (criterion 4).
6. **The page**: `commands/intent.md` and the build page say the word once for both (criterion 5).

## Out of scope

- The build proposing a split; a task family; reordering mid-run.

## Approach

A parser in the spec store, a counter in the state file, one branch in the loop's advance; the first spec to carry steps is the implement spec itself, whose three pieces become its `## Steps`.

## Footprint

- packages: internal/core/spec, internal/core/implement, internal/surface/cli
- tests: the parser over a stepped and an unstepped spec; the loop advancing only after merge with a fake forge; the remainder copy; the brief

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 template and one implicit step | scope 1, 2 |
| 2 lanes in order | scope 3 |
| 3 remainder carries steps | scope 4 |
| 4 briefs and record | scope 5 |
| 5 one word | scope 6 |

## Progress

- Criteria 1 and 3 (scope 1, 2 and 4) are delivered by commit e5dfe0fc: the plan stub carries an empty `## Steps` section, rendered from one list of stub sections; `internal/core/spec/steps.go` parses the section (`- packages:`, `- tests:` and `- landed: <pull request or commit>` indented under each numbered step) and turns a spec listing none into one implicit step; `spec close --remainder` carries the steps not marked landed into the remainder and refuses, writing nothing, on a section it cannot read; `intent ready` reports the section's shape on an advisory `steps` row.
- Criteria 2, 4 and 5 (scope 3, 5 and 6) are owed to the implement-loop lanes of itd-2609201916151817, which close this spec.
