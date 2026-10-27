---
schema_version: 1
id: "iss-2610262039245351"
slug: "the-worktree-store-intent-makes-its-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 857 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md:115"
remedy: "none (filed automatically)"
related_intents: [itd-2609091014076309]
finding_ref: cfk-fb7437d0a5e70412
---

The worktree-store intent makes its verb a sub-verb of `ahoy`, but the surfaces register lists it as a top-level `/abcd:worktree`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md:115` — "The verb is a sub-verb of `ahoy`"
- End B: `.abcd/development/brief/04-surfaces/README.md:36` — "`/abcd:worktree` | staged | Keep session and agent worktrees in a machine-scoped store"

Decision 5 of itd-2609091014076309 makes the verb a sub-verb of `ahoy` (`abcd ahoy worktree add`, `list`, `prune`). The surfaces register carries a staged row for a top-level `/abcd:worktree` that cites the same intent. The register's coverage check expects a shipped row to have its own command page, so once the intent ships the row names a command that will never exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 857.
