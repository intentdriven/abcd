---
schema_version: 1
id: "iss-2610161332079474"
slug: "01-ahoy-s-user-scope-tree-marks"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "deferred config-brief fixes, inventory check"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Say the store holds the build loop's lane worktrees today and that its list and reclaim verbs are not built."
resolution: "01-ahoy's user-scope tree now says the build loop creates each lane's worktree in worktrees/<root-sha>/ today and ahoy reports an unlinked one; only the store's own verbs and reclaim are marked NOT BUILT."
impact: internal
---

01-ahoy's user-scope tree marks worktrees/<root-sha>/<name>/ NOT BUILT, but the build loop creates each lane's worktree there today (internal/core/implement/loop/lane.go, WorktreeStoreRel) and ahoy inspects the store (internal/core/ahoy/store_worktrees.go); only the store's verbs and reclaim are unbuilt, as 03-configuration.md's worktree-store section says.
