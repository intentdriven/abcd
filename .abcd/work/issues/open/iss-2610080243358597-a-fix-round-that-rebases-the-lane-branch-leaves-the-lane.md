---
schema_version: 1
id: "iss-2610080243358597"
slug: "a-fix-round-that-rebases-the-lane-branch-leaves-the-lane"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "overnight drain, personas lane run-2610072316033117, 2026-10-08"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "When a fix round's receipt names new commits, re-derive each resolves entry's commit from the lane's current branch (the fix receipt's commits, or the patch-id equivalent of the original commit after a rebase) before the landing writes resolved_by.commit, and refuse the landing naming the stale sha when it is not an ancestor of the lane head."
---

A fix round that rebases the lane branch leaves the lane receipt's resolves[].commit at the pre-rebase sha, which is no longer reachable from the lane head, so the landing would stamp an unreachable resolved_by.commit that RS003 refuses at preflight. Met overnight on run-2610072316033117 (fab3f1704 rebased to 90f48d5f9); corrected by hand in the run state.
