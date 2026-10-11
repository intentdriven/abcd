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
---

The sync merge fails on git older than 2.44 (Apple Git 2.39.5) when an attribute selects a merge driver: BuiltinMergeDriver, the merge-file command that replaces every repository merge driver (iss-2610090821510097), passes --diff-algorithm=histogram, which merge-file only accepts from git 2.44, so git merge exits 1 with "unknown option 'diff-algorithm=histogram'" even on a clean merge. The repository's driver still does not start and the merge.default case passes. Reported as GitHub issue #894, pinned at 3b8700ef7.
