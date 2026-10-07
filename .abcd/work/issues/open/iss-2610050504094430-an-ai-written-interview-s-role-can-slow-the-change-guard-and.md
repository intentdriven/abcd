---
schema_version: 1
id: "iss-2610050504094430"
slug: "an-ai-written-interview-s-role-can-slow-the-change-guard-and"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: lane e8term-r3d, the guard check of e8term-r3c (finding 1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "Before git runs, size every submodule git directory's HEAD, commondir, config.worktree, config, packed-refs and info/exclude under the common directory's modules/ against what is left of maxHashedBytes, beside the checkout's own (sizeGitReads), refusing past it naming the largest; and read each submodule git directory as the common directory is read (HEAD, packed-refs, refs and info besides its hooks and config), so a write there is named. Grounds: the probe above; git status recurses into a populated submodule by default (diff.ignoreSubmodules unset), so its git directory is read on every reading."
---

An AI-written interview's role can slow the change guard, and change a submodule's refs unseen, through a tracked submodule's git directory: git status in the checkout runs status inside each populated submodule, which reads that submodule's git directory under the common directory's modules/ (its HEAD, config, packed-refs, info/exclude) whole, and the guard neither sizes those files before git runs nor hashes the submodule's HEAD and refs (readModules hashes its hooks and config only). A check's probe grew .git/modules/sub/packed-refs to 128 GiB sparse and held the reading past 20 s; a small write to the same file was unseen (changedSince empty).

## Remainder after the 2026-10-07 review

The branch that sized and watched submodule git directories (fix/guard-commondir-bound) was reviewed adversarially before it merged (verdict SHIP-WITH-FIXES). Fixed on that branch: a write to a submodule's `commondir` is now named, and a pipe or device where git reads a file whole is refused before git runs. Two placements are still not pinned, so this record stays open for them:

- A submodule's `commondir` is named only after git has run, not pinned before it: a write pointing it at a foreign directory is caught at that reading, but git reads the foreign directory's configuration and refs (unsized) during that same reading. Remedy: pin every `modules/**/commondir` beside the checkout's own pointers at the first reading, refusing a changed one before git runs (the redirectedError path).
- A submodule's own `.git` pointer file (`<path>/.git`, `gitdir: …`) is never read: rewriting it moves the submodule's git directory anywhere, and the superproject's status does not list it. Remedy: enumerate gitlinks (mode 160000 in `git ls-files -s`, or `.gitmodules`) at the first reading and pin each `<path>/.git` as a pointer, refusing a changed one before git runs.
