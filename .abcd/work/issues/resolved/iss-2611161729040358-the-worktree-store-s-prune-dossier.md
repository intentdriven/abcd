---
schema_version: 1
id: "iss-2611161729040358"
slug: "the-worktree-store-s-prune-dossier"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/quiet.go"
remedy: "Count the dossier's ignored files with the removal's own reader, store.IgnoredForRemoval's git status of ignored files, keeping the dossier's unread marking."
resolution: "The quiet-candidate dossier now counts ignored files with the removal's own reader, store.IgnoredForRemoval's git status of ignored files, keeping its unread marking, so a file only the default global ignore file matches is named; prune's reclaim rows and the sweep's rows name them too."
impact: fix
resolved_by:
  commit: "aab27ce34fb2bb780e38290996a59a1984a66bc3"
---

The worktree store's prune dossier counts a candidate's ignored files with core.excludesFile turned off, so a file only the default global ignore file matches is not named in its warning, yet the prune's removal deletes it.
