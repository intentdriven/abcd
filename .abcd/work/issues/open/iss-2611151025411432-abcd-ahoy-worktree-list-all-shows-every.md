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
---

abcd ahoy worktree list --all shows every lane under its root commit, even when the repository is registered in ~/.abcd.noindex/history/index.json: the door calls worktree.ListAll with no Labels and package ahoy has no exported reader of the registry's names, so itd-2609091014076309's ac-7 (registered lanes under their index.json names) is not met.
