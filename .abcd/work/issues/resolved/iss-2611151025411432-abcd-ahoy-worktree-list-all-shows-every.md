---
schema_version: 1
id: "iss-2611151025411432"
slug: "abcd-ahoy-worktree-list-all-shows-every"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "intent audit rcp-01d317e17525"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ahoy_worktree.go"
remedy: "Add ahoy.RegistryLabels, which reads the registry through the scrubbing loader and maps each root commit to its name and path label, and pass its answer to ListAll from the list --all door; an absent or unreadable registry passes no labels, so each lane stays under its key and the listing does not fail."
resolution: "Fixed: internal/surface/cli/ahoy_worktree.go:82 now passes registryLabels() (ahoy_worktree.go:424) to worktree.ListAll, and internal/core/ahoy/registry_labels.go:25 adds ahoy.RegistryLabels, which maps each root commit registered in ~/.abcd.noindex/history/index.json to its name and path label through the scrubbing loader. An absent, malformed or refused registry passes no labels, so each lane stays under its root commit and the listing does not fail. Tests: TestAhoyWorktreeListAllLabelsRegisteredLanes (failed on the code before the fix), TestAhoyWorktreeListAllWithoutARegistry, TestRegistryLabelsMapsRootCommitsToNames and TestRegistryLabelsAbsentOrUnreadable."
impact: fix
resolved_by:
  intent: "itd-2609091014076309"
  spec: "spc-2609301811532881"
  commit: "7d85473b9"
---

abcd ahoy worktree list --all shows every lane under its root commit, even when the repository is registered in ~/.abcd.noindex/history/index.json: the door calls worktree.ListAll with no Labels and package ahoy has no exported reader of the registry's names, so itd-2609091014076309's ac-7 (registered lanes under their index.json names) is not met.
