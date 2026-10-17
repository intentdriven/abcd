---
schema_version: 1
id: "iss-2610151050427571"
slug: "a-worktree-cut-from-origin-main-tracks"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "v0.15.0 run 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "worktrees cut from origin/main"
remedy: "Document in the CONCURRENCY rules that a lane worktree is made with git worktree add --no-track -b <name> <path> origin/main (or its upstream unset), and have every push abcd or a runbook makes name an explicit refspec to the branch's own name."
---

A lane worktree made with git worktree add -b <name> <path> origin/main tracks origin/main, so a bare git push from it aims at main. On 2026-10-15 a post-preflight push from such a branch was refused only by push.default=simple, because the names differ; under push.default=upstream it would have pushed a feature branch's commits to main.
