---
schema_version: 1
id: "iss-2610290741142392"
slug: "a-branch-cleanup-keeps-no-backup-of-what"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker ruling relayed by the coordinator session, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sweep/apply.go"
remedy: "Add a backups store at ~/.abcd.noindex/backups/<root-sha>/: before a branch cleanup deletes any unmerged branch it writes a git bundle of every such branch with a manifest (branch, tip sha, date), and a retention sweep removes bundles older than a configurable age, thirty days by default, naming each one it removes."
---

A branch cleanup keeps no backup of what it deletes in a place abcd manages. The product thinker decided on 2026-10-29 that the backup taken before a cleanup is kept on purpose in a store of its own under ~/.abcd.noindex/backups/, keyed on the repository's root commit like abcd's other machine stores, so it lasts beyond the cleanup and can be cleared away later, for instance once it is thirty days old. Today no such store exists: an unmerged branch the cleanup removes survives only as long as git's reflog keeps it, and nothing names where to find it or when it goes.
