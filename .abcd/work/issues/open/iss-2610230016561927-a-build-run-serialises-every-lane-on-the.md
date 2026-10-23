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
---

A build run serialises every lane on the previous lane merging into main. The ready rule for a lane's predecessors (the DR6b default needs, read with landedInRun as being an ancestor of the default branch) holds each step until the step before it has passed the merge queue, so a six-step spec spends roughly 20 to 25 hours in serial queue time at the preflight and merge-queue times seen on 2026-10-22, most of it waiting rather than building; the merge-gate build was the case that night.
