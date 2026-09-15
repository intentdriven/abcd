---
schema_version: 1
id: "iss-2609020721142452"
slug: "worktrees-for-parallel-lanes-are-created-one-directory-above"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
---

Worktrees for parallel lanes are created one directory above the repository (../abcd-wt-<lane>, ../abcd-f185, ../abcd-integration), which litters the parent directory with a dozen sibling checkouts named by convention only, leaves them behind when a session dies (21 spent worktrees, 1.4G, were removed by hand on 2026-09-01), and gives nothing a stable place to look for them. Assessment of where they should go: (1) inside the repository under .abcd/.work.local/worktrees/ keeps them with their owner and gitignored, but a worktree inside the main working tree is walked by every tree scan (the payload gate, name-guard, docs-lint), which already flakes during worktree creation (iss-2608261331317889) and would slow every gate; (2) under the user-level home, ~/.abcd/worktrees/<root-sha>/<branch-slug>/, keyed the way the history store already keys a repository by its root commit, keeps them out of every scan, survives a session, and gives an abcd verb a place to list, prune and reattach them; (3) the git default (anywhere) is the status quo. Option 2 fits the existing home layout and the one-canonical-primitive rule (the root-sha keying exists), and it makes the concurrent-sessions convention in AGENTS.md checkable: a session lists its peers' worktrees by reading one directory. It needs: a verb or documented one-liner (abcd worktree add <lane> / list / prune), the AGENTS.md convention updated, the lane brief updated, and a prune rule for worktrees whose branch is merged.
