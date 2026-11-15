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
resolution: "Fixed in 7d85473b9 and d0b86631b: internal/surface/cli/ahoy_worktree.go:83 passes the labels from registryLabels() (ahoy_worktree.go:432) to worktree.ListAll, and internal/core/ahoy/registry_labels.go:36 adds ahoy.RegistryLabels, which maps each root commit registered in ~/.abcd.noindex/history/index.json to its name and path label through the scrubbing loader. The registry is read from abcdhome.MachineHome, the same home the lanes come from, so inside a walled stage a lane and its name come from the person's home. An absent registry passes no labels and adds no note; one that exists but is malformed or refused also passes none and adds one note line that names no path. Either way each lane stays under its root commit and the listing exits 0. Tests: TestAhoyWorktreeListAllLabelsRegisteredLanes (failed on the code before the fix), TestAhoyWorktreeListAllWithoutARegistry, TestAhoyWorktreeListAllReadsTheRegistryFromTheMachineHome, TestRegistryLabelsMapsRootCommitsToNames, TestRegistryLabelsAbsentOrUnreadable and TestRegistryLabelsReadsTheMachineHome."
impact: fix
resolved_by:
  intent: "itd-2609091014076309"
  spec: "spc-2609301811532881"
  commit: "d0b86631b"
---

abcd ahoy worktree list --all shows every lane under its root commit, even when the repository is registered in ~/.abcd.noindex/history/index.json: the door calls worktree.ListAll with no Labels and package ahoy has no exported reader of the registry's names, so itd-2609091014076309's ac-7 (registered lanes under their index.json names) is not met.
