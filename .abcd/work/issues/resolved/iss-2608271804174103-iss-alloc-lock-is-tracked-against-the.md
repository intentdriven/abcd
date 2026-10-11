---
schema_version: 1
id: "iss-2608271804174103"
slug: "iss-alloc-lock-is-tracked-against-the"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/work/issues/.iss-alloc.lock"
resolution: "the tracked gitignored allocator lock removed with git rm --cached"
impact: internal
resolved_by:
  commit: "8db72381"
---

the capture allocator lock is git-tracked against the repo's own ignore rule: .abcd/work/issues/.iss-alloc.lock is committed as the empty blob although .gitignore declares it out of the tree, unlike its sibling .abcd/work/.decisions.lock which is correctly untracked. Fix is git rm --cached — nothing on disk changes and the allocator keeps flocking the same path.