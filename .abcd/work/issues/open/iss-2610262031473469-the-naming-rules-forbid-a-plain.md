---
schema_version: 1
id: "iss-2610262031473469"
slug: "the-naming-rules-forbid-a-plain"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 176 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:100"
remedy: "none (filed automatically)"
related_intents: [itd-2609091014076309]
finding_ref: cfk-6f8a3600f8767dc2
---

The naming rules forbid a plain unfiltered `<verb> list`, but itd-2609091014076309 makes `abcd ahoy worktree list` render the same thing as the bare verb.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:100` — "`<verb> show`, `<verb> stats`, `<verb> view` and a plain unfiltered `<verb> list`"
- End B: `.abcd/development/intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md:44` — "**Bare `abcd ahoy worktree` and `abcd ahoy worktree list`** render the lane for this repository"

04-naming forbids `<verb> show`, `stats`, `view` and a plain unfiltered `<verb> list` at design time, because each only renames what the bare invocation already renders. The worktree-store intent has bare `abcd ahoy worktree` and `abcd ahoy worktree list` render the same lane, which is exactly the shape the rule forbids. Only `--all` changes the scope.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 176.
