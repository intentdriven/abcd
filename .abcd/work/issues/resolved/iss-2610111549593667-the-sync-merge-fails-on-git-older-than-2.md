---
schema_version: 1
id: "iss-2610111549593667"
slug: "the-sync-merge-fails-on-git-older-than-2"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "GitHub issue #894 (external report)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/execpins.go"
remedy: "Pass --diff-algorithm=histogram to merge-file only on a git that accepts it (2.44 or later, read through the existing git version probe), so the driver works on older git while the repository's driver still never starts."
resolution: "The built-in merge driver passes --diff-algorithm=histogram and the %X/%S/%Y labels only on git 2.44 or later, read through the existing git version probe; below 2.44 it runs merge-file's default diff with fixed labels, so a sync merge works on Apple Git 2.39.5 and the repository's driver still never starts. TestTheSyncMergeRunsNoConfiguredMergeDriver passes on git 2.39.5, 2.52 and 2.54."
impact: fix
resolved_by:
  commit: "868bb5482"
---

The sync merge fails on git older than 2.44 (Apple Git 2.39.5) when an attribute selects a merge driver: BuiltinMergeDriver, the merge-file command that replaces every repository merge driver (iss-2610090821510097), passes --diff-algorithm=histogram, which merge-file only accepts from git 2.44, so git merge exits 1 with "unknown option 'diff-algorithm=histogram'" even on a clean merge. The repository's driver still does not start and the merge.default case passes. Reported as GitHub issue #894, pinned at 3b8700ef7.

## Grounds

- pursued: dropping the options only below 2.44 makes the merge work on older git without starting the repository's driver; a sync on old git that merges differently from git's own merge in a case that matters would show it wrong.
