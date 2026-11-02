---
schema_version: 1
id: "iss-2610311249121083"
slug: "the-shell-guard-refuses-git-worktree"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "post-cut cleanup 2026-10-31"
origin: researcher-authored
production_mode: hand-written
found_at: "the shell guard's worktree-remove proof"
remedy: "Have the guard prove a worktree removal safe when HEAD is on the default branch by ancestry, detached or not, and every ignored file is either disposable (a lock file or an empty directory) or held by an archive in the machine backup store whose listing matches the files. Record that archive proof in the cleanup step of abcd sweep, so the sweep removes a finished worktree with no per-item allow."
---

The shell guard refuses git worktree remove for a worktree whose HEAD is on origin/main and whose status is clean, whenever the worktree holds ignored files: a local tier with notes in it, an empty .abcd/.work.local, or the issue allocator's lock file. It also refuses a detached HEAD that sits on origin/main. In the 31 Oct cleanup it refused all 85 merged, clean worktrees, about 7.2 GB, even though every local tier had first been archived to the machine backup store. Reclaiming them takes one guard allow per worktree from the person.
