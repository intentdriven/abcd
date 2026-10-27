---
schema_version: 1
id: "iss-2610230016561927"
slug: "a-build-run-serialises-every-lane-on-the"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "lane C, 2026-10-22 crash recovery"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Weigh two remedies: count a predecessor ready once it has landed on the run's own branch (lanes stacked on one another, merged together or in order), or default a step's predecessors to the spec's needs: lines rather than every earlier step, so independent steps run in parallel"
resolution: "The cause was authoring: 5 of 169 specs declared needs, and the code already honours them. intent ready and record-lint (spec_steps_serial, at warn) now warn on a multi-step spec with no needs line, and the planning page and the planning interviewer ask for each step's needs. Stacked readiness (a dependent lane starting once its predecessor lands on the run's own branch) and the backfill of needs lines on the open specs are captured as records of their own, named in the resolving commit."
impact: additive
resolved_by:
  commit: "cd6e38c139c138c1ce6e737c9f375195c381cbd9"
---

A build run serialises every lane on the previous lane merging into main. The ready rule for a lane's predecessors (the DR6b default needs, read with landedInRun as being an ancestor of the default branch) holds each step until the step before it has passed the merge queue, so a six-step spec spends roughly 20 to 25 hours in serial queue time at the preflight and merge-queue times seen on 2026-10-22, most of it waiting rather than building; the merge-gate build was the case that night.

## Grounds

- pursued: a spec whose steps declare what they truly need runs its independent steps in parallel, so a build stops spending most of its time waiting on serial merges; the coordinator ruled on 2026-10-23 (dcn-2610230857411093) to make the missing declaration visible where specs are written rather than guess dependencies from file overlap. Shown wrong if warned planners still leave needs undeclared, or if specs that declare them still queue serially.
