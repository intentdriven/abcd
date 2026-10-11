---
schema_version: 1
id: "iss-2608211432254405"
slug: "scaffold-derivebranch-worktree-gitfile"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "internal/core/launch/scaffold/scaffold.go"
resolution: "Resolve the .git gitfile so deriveBranch reads HEAD from the worktree gitdir and origin/HEAD from commondir"
impact: fix
resolved_by:
  commit: "4618e20"
---

launch scaffold deriveBranch assumes .git is a directory and reads HEAD/origin-HEAD with raw os.ReadFile, so in a linked worktree or submodule (where .git is a gitfile) both reads fail and it stamps the fallback branch main into the generated release workflows, leaving the release gate silently inert on a non-main-default repo