---
schema_version: 1
id: "iss-2610050429322910"
slug: "an-ai-written-interview-s-role-can-slow-the-change-guard"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: lane e8term-r3c, sweeping treewatch.go for unbounded reads"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "Before each reading runs git, size the files git reads whole on every command in the directories the run's first reading placed (the git directory's HEAD, commondir and config.worktree, the common directory's config, packed-refs and info/exclude) from their Lstat, and refuse past maxHashedBytes naming the largest, so git never reads more than the guard itself would hash. Grounds: the probe above, and the directories are already pinned before git runs (iss-2610050348402910), so the files can be placed without asking git."
resolution: "Before each reading runs git, the files git reads whole on every command in the pinned directories (HEAD, commondir, config.worktree, config, packed-refs, info/exclude) are sized from their Stat against maxHashedBytes, refusing past it naming the largest (TestWhatGitReadsWholeIsSizedBeforeGitRuns)."
impact: fix
resolved_by:
  spec: "spc-2610040847280931"
  commit: "2731f41f1"
---

An AI-written interview's role can slow the change guard through git itself: every reading runs git status before the guard's own bounds apply, and git reads some files in its own directory whole on every command, so a role writing a sparse HEAD, packed-refs or info/exclude there holds the turn while git reads it, before the guard's byte bound refuses the file. A scratch probe of git status with each file grown to 4 GiB sparse: HEAD 0.8 s, packed-refs 2.3 s, info/exclude 0.4 s, growing with the size (so minutes at 128 GiB). The guard fails closed (the file is named once read), and neither the in-tree byte bound (iss-2610050348533437) nor the followed-link budget covers it, since git, not the guard, does the reading. Files git bounds itself are not affected: a tracked file whose size changed is not re-read, a .gitignore or .gitattributes past 100 MiB is skipped, an index of zeros is refused at its header, and a configuration of zeros at its first line.

## Grounds

- pursued: a HEAD, packed refs or info/exclude a role grows is refused before git reads it; a turn held while git reads it would show it wrong
