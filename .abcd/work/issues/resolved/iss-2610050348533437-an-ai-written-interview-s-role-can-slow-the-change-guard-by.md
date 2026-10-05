---
schema_version: 1
id: "iss-2610050348533437"
slug: "an-ai-written-interview-s-role-can-slow-the-change-guard-by"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: lane e8term-r3b, sweeping the guard for directories it reads but does not own"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "Bound the bytes one reading hashes in the tree, beside maxWatched: count each listed path's size from its Lstat before hashing it, and refuse past the bound (some GiB, far past a source tree's changed files) naming the largest path, so a tree the person already holds past it is refused before the first dispatch and one the role grows past it is refused before the file is read. Grounds: hashOne and contentState in treewatch.go read every listed path to its end; maxWatched already refuses a tree by path count in the same way, before the first dispatch."
resolution: "Followed the record's remedy (refuse, not a stat-only judgement): maxHashedBytes (2 GiB) bounds what one reading hashes of what it owns; the paths git lists are sized from their Lstat before any is hashed and a listing past the bound is refused naming its largest path, git's own directory is counted as it is reached, and a file is hashed no further than its Lstat size (TestTheBytesOneReadingHashesAreBounded)."
impact: fix
resolved_by:
  spec: "spc-2610040847280931"
  commit: "2731f41f1"
---

An AI-written interview's role can slow the change guard by writing one large file into the working tree: the reading after its turn hashes every path git status lists whole (hashOne, then contentState's io.Copy), with no byte bound, so an untracked sparse file of many GiB holds the turn for as long as hashing it takes (about 7 s for 16 GiB in the hooks-directory probe, growing with the size) before the new path is reported. The guard fails closed (the path is named), and the budget on what one reading reads outside the tree (iss-2610050315146044) does not cover it, because the path is inside the tree, where a large untracked file can be the person's own.

## Grounds

- pursued: a large file a role writes into the tree or git's own directory is refused promptly, named, before it is hashed; a turn held while the file is hashed, or a refusal naming a smaller path than the largest, would show it wrong
