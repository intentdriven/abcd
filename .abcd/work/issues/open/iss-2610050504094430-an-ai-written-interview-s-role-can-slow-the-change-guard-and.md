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
