---
schema_version: 1
id: "iss-2610231114293603"
slug: "lane-reports-hook-failure-as-not-added"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/lane.go"
remedy: "In the loop's lane stage, when git fails, check whether git registered the worktree at the lane path on the lane's branch and, if it did, adopt it in that same step and report that the checkout hook failed with its status, as the store's add does."
---

The build loop's lane stage reports a lane worktree as not added when the repository's post-checkout hook fails: git runs that hook after it has made and registered the worktree and created the branch, then exits with the hook's status, and the lane stage reads the exit as git could not add the worktree. A rerun adopts the standing worktree, so the lane recovers, but the first message is false and the run spends a step on it. Found while fixing the same case in the worktree store's add on 2026-10-23, which now finishes a worktree git made and says the hook failed.
