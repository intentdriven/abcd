---
schema_version: 1
id: "iss-2611022247131866"
slug: "ahoy-worktree-prune-keeps-merged-clean"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/notes.go"
remedy: "Leave the local tier's build products (bin/) out of the notes the archive collects, so a merged, clean worktree is reclaimable; a test prunes a worktree whose tier holds a large bin/abcd."
---

ahoy worktree prune keeps merged, clean worktrees it should reclaim when their notes hold the locally built bin/abcd: on 2026-11-02 about ten merged worktrees were kept because that binary is over the notes archive's 4 MiB per-file cap, though it is a build product, not a note.
