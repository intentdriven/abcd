---
schema_version: 1
id: "iss-2610020730426636"
slug: "the-build-loop-opens-a-lane-for-every-ready-spec-step"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/schedule.go"
remedy: "Per ruling DR6d-1: open a lane for a ready step only while a slot is left for its implementer beside the lanes opened whose implementer is not out yet and fewer step worktrees than the ceiling are on disk; a waiting step has no worktree; amend C4 in the open remainder spec spc-2609301921521360."
---

The build loop opens a lane for every ready spec step whatever the ceiling, making its worktree and brief at once (schedule.go move, the eager opening the parallel-lanes landing shipped), so a spec whose steps declare needs: none puts every step's worktree on disk before a helper is free to take it; ruling DR6d-1 (the product thinker, 2026-10-02) says a step's worktree is made just before an agent takes it, at most the agent ceiling's worth on disk.
