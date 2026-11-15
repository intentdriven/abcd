---
schema_version: 1
id: "iss-2611040554576045"
slug: "05-internals-03-configuration-md-853"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Drop the stale identifier or cite internal/abcdhome."
resolution: "The worktree store section of 05-internals/03-configuration.md no longer cites WorktreeStoreRel; it names internal/core/ahoy/worktree, through which the build loop makes its lanes."
impact: internal
resolved_by:
  spec: "spc-2609301811532881"
---

05-internals/03-configuration.md:853 cites (implement/loop, WorktreeStoreRel) for where a build lane's worktree is created. No Go file under internal/ or cmd/ defines WorktreeStoreRel. The path itself is right, built through internal/abcdhome.
