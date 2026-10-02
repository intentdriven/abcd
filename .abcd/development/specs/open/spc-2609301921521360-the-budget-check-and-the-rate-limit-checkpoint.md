---
id: spc-2609301921521360
slug: the-budget-check-and-the-rate-limit-checkpoint
intent: itd-2609201925079472
origin: researcher-authored
production_mode: hand-written
---
# the-budget-check-and-the-rate-limit-checkpoint

## Summary

The rest of itd-2609201925079472 after spc-2609202134341288 closed: pieces 4
and 5 of that spec, which wait on a runner that reports its quota and its
rate-limit responses (itd-2609201916056194).

- **The budget check** (criterion 7, from `itd-29`): where the runner reports
  remaining quota, an estimate from the spec's size is compared before the run
  starts, and a run it exceeds is refused naming both numbers with no state
  written; a runner that reports none is named and the check is skipped out
  loud.
- **The rate-limit checkpoint** (criterion 8, from `itd-29`): a runner's
  rate-limit response ends the window early for the whole run, since every
  lane spends the same budget; every lane with work in flight is checkpointed
  to its own branch, `next_eligible_at` is written once, and the record names
  the lane the response came from (spc-2609202134341288, "The pace and the fix
  rounds, per lane").

The pace, the window clock and the ceiling with its parallel lanes
(criteria 1 to 6 and 9) are built; spc-2609202134341288's design sections
remain the record for them.

## Footprint

- packages: internal/core/implement/loop, the runner adapter itd-2609201916056194 delivers
- tests: a fake runner reporting quota over and under the estimate, one reporting none, and a rate-limit response mid-lane with two lanes in flight

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
