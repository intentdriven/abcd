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
---

The worktree store's prune dossier counts a candidate's ignored files with core.excludesFile turned off, so a file only the default global ignore file matches is not named in its warning, yet the prune's removal deletes it.
