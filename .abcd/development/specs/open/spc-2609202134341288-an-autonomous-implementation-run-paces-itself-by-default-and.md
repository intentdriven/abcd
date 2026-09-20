---
id: spc-2609202134341288
slug: an-autonomous-implementation-run-paces-itself-by-default-and
intent: itd-2609201925079472
origin: researcher-authored
production_mode: hand-written
---
# an-autonomous-implementation-run-paces-itself-by-default-and

## Summary

The design record for itd-2609201925079472, from the six decisions on the
intent (2026-09-20).

## Scope

1. **The pace configuration**: `pace.work_minutes`, `pace.pause_minutes`,
   `pace.sub_agents` read through the layering the model-tier intent uses
   (flag, repository `.abcd/config.json`, machine `~/.abcd/config.json`,
   bundled 120/300/2); one resolver reports the value and the layer it
   came from (criteria 1 to 3, 9).
2. **The window clock** in the implement verb's state file: the window's
   start, `next_eligible_at`, the lanes alive; written by the loop at every
   step (criteria 4, 5).
3. **The ceiling**: the loop counts lanes and validators alive from the
   state and starts nothing above the ceiling, exiting with the lanes named
   and the wait accumulated (criterion 6).
4. **The budget check** (from `itd-29`): where the runner reports remaining
   quota, an estimate from the spec's size is compared before the run
   starts; a runner that reports none is named and the check is skipped
   out loud (criterion 7).
5. **The rate-limit checkpoint** (from `itd-29`): a runner's rate-limit
   response ends the window early with the lane checkpointed to its branch
   and `next_eligible_at` set (criterion 8).

## Out of scope

A ceiling across runs (the register's); telemetry and hand verbs.

## Approach

One lane, test-first, on the implement verb's state file; the resolver is
the one canonical layering primitive shared with the model tier. The bundled
default lives in one constant the run record names.

## How the criteria are satisfied

1 to 3 and 9 by piece 1; 4 and 5 by piece 2; 6 by piece 3; 7 by piece 4;
8 by piece 5.

