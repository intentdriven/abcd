---
schema_version: 1
id: "iss-2610231125086511"
slug: "store-add-adopts-concurrent-worktree"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/worktree.go"
remedy: "Adopt after a failure only a worktree this run's git registered: compare the registration with what existed before the run (no entry at that path beforehand) and the branch's creation with this run, and refuse a branch-exists failure as a taken name without adopting."
related_intents: [itd-2609091014076309]
---

Two concurrent runs of the worktree store's add for the same name race: the second run's git fails because the branch already exists, and the adopt-after-failure path then claims the first run's worktree, reports that the checkout hook failed with git's branch-exists message, which is false, and lays the local tier inside a worktree the first run may still be checking out. Found by the Opus review of the store's step 2 on 2026-10-23.
