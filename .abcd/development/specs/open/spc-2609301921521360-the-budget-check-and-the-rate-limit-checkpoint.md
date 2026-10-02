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
remain the record for them, as amended below.

## Amendments to spc-2609202134341288

The closed spec is not reopened; these rulings amend its design and its
criterion C4 from 2026-10-02, and the loop is built to them.

- **A step's worktree is made only when a helper is free (ruling DR6d-1).**
  The product thinker, verbatim: "preparing parallel steps' worktrees: ONLY
  WHEN A HELPER IS FREE — a step's worktree is made just before an agent takes
  it; at most the agent ceiling's worth on disk." It reverses the eager opening
  "The count" and C4 carried ("A stage the binary performs itself proceeds at
  the ceiling", read as a ready step's worktree and brief made whatever the
  ceiling). A lane opens for a ready spec step only while a slot is left for
  its implementer beside every lane opened whose implementer is not out yet,
  and fewer step worktrees than the ceiling are on disk (a lane holds one from
  its worktree stage until it lands or is discarded); the opening call makes
  its worktree, and its brief and implementer follow. A step waiting for a
  helper has no worktree and waits under `waiting` as `step <n>`. C4 reads: a
  step made ready at the ceiling waits for a helper with no worktree, and its
  lane opens at the next free slot (as C6 already says of `- needs: none`).
  The stages the binary performs on a lane already open (its brief, a round's
  close, a landing step, a sync, a hold) still take no slot and are never held
  by the ceiling.
- **A missing preflight receipt holds only its lane (ruling DR6d-2).** The
  product thinker, verbatim: "missing preflight receipt for one lane: CARRY
  ON, WAIT SHOWN — other lanes continue; the lane waits for its receipt like
  the merge wait (contend, not refuse), and the status shows 'waiting for its
  full check (since HH:MM)'." The landing's push without a receipt naming the
  lane's head is a contention, as the forge's merge is: the call moves another
  lane and names the wait under `blocked`, and gives the wait (exit 3) only
  when nothing else moves. The time the wait began is written once, on the
  lane's landing as `check_wait_since` (a version-9 field), with a record
  entry, kept across calls and cleared by the push; the status shows it in
  UTC. The remedy still names how the receipt is minted. Landing stays one lane
  at a time: a landing waiting for its full check is a landing under way, so a
  sibling's landing waits behind it as it waits behind a merge. Who starts the
  check (ruling DR6d-2b) is a separate intent and is not built here.

## Footprint

- packages: internal/core/implement/loop, the runner adapter itd-2609201916056194 delivers
- tests: a fake runner reporting quota over and under the estimate, one reporting none, and a rate-limit response mid-lane with two lanes in flight

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
