---
schema_version: 1
id: "iss-2611030744325032"
slug: "a-loop-run-whose-lane-was-landed-outside"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have the land stage accept a lane whose pull request merged with a head descending from the pinned landing and finish it as landed (worktree, branch and reservation cleaned up), or add a verb that ends a run, releasing what it holds; a test merges a lane's branch past its landing and steps the run to done."
---

A loop run whose lane was landed outside the loop can never be finished: run-2611030127507347's lane merged as #1367 at 85ed321d2 after a hand landing, and implement step now refuses on every call because the branch is not at the records commit 0c466ccb1 the landing pinned, with a remedy to restore the branch, which would rewind a merged branch. Nothing else ends the run, so it keeps its lane worktree (prune keeps it as held by the run) and held its reservation until it was released by hand. iss-2610142233473049 covers the unmerged case.
