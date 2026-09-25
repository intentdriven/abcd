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
resolution: "the location is decided by adr-2609091248200336 (the machine-scoped worktree store keyed on the root commit); the store verbs are tracked by the draft itd-2609091014076309"
impact: internal
---

Worktrees for parallel lanes are created one directory above the repository (../abcd-wt-<lane>, ../abcd-f185, ../abcd-integration), which litters the parent directory with a dozen sibling checkouts named by convention only, leaves them behind when a session dies (21 spent worktrees, 1.4G, were removed by hand on 2026-09-01), and gives nothing a stable place to look for them. Assessment of where they should go: (1) inside the repository under .abcd/.work.local/worktrees/ keeps them with their owner and gitignored, but a worktree inside the main working tree is walked by every tree scan (the payload gate, name-guard, docs-lint), which already flakes during worktree creation (iss-2608261331317889) and would slow every gate; (2) under the user-level home, ~/.abcd/worktrees/<root-sha>/<branch-slug>/, keyed the way the history store already keys a repository by its root commit, keeps them out of every scan, survives a session, and gives an abcd verb a place to list, prune and reattach them; (3) the git default (anywhere) is the status quo. Option 2 fits the existing home layout and the one-canonical-primitive rule (the root-sha keying exists), and it makes the concurrent-sessions convention in AGENTS.md checkable: a session lists its peers' worktrees by reading one directory. It needs: a verb or documented one-liner (abcd worktree add <lane> / list / prune), the AGENTS.md convention updated, the lane brief updated, and a prune rule for worktrees whose branch is merged.

**Corroboration (2026-09-20, Gropius session gropiusllm-2b, relayed to
abcd-17).** A third placement, and the one the harness itself defaults to: a
managed repository's sub-agent lanes run in worktrees under
`.claude/worktrees/` INSIDE the checkout, one per intent or issue. That is the
shape option (1) above rejects (every tree scan walks it, and worktree creation
already races the payload gate, iss-2608261331317889), and it is not the
session's choice: the harness's worktree isolation creates them there unless
told otherwise, so adr-2609091248200336 and the machine-scoped store are
unknown to a managed repository until abcd's own worktree verb (itd-148,
spc-42) exists and the lane brief names the path. Until then the convention
in abcd's own AGENTS.md (`git worktree add ~/.abcd/worktrees/<root-sha>/<name>`)
is the documented one-liner, and it reaches no managed repository.

## Grounds

- pursued: parallel-lane worktrees have one ruled home outside the checkout; shown wrong if a lane worktree is created beside or inside a checkout under the current convention
