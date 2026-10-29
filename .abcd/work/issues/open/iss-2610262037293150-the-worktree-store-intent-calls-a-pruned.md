---
schema_version: 1
id: "iss-2610262037293150"
slug: "the-worktree-store-intent-calls-a-pruned"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 726 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md:46"
remedy: "Rename the pruned-tier archive in the worktree-store intent (scope bullet and decisions 4 and 10) to a name without 'notes' (for example a reclaimed-tiers store), and mirror it in the store layout of 04-surfaces/01-ahoy.md and 05-internals/03-configuration.md, before the store is built. The announce intent already retired 'note' for the same clash with the memory note."
related_intents: [itd-2609091014076309]
finding_ref: cfk-5720d4198d0524cc
---

The worktree-store intent calls a pruned worktree's archived local tier its notes and keeps it under ~/.abcd.noindex/notes/; the memory brief's notes are the memory notes in memory.json

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md:46` — "it moves the tier into a sibling store, `~/.abcd.noindex/notes/<root-sha>/<UTC timestamp>-<worktree name>/`"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:26` — "The machine's `~/.abcd.noindex/memory.json` holds the person's own notes"

The memory chapter calls the person's own notes the memory notes kept in `~/.abcd.noindex/memory.json`. The worktree-store intent moves a pruned worktree's local tier into a store named `~/.abcd.noindex/notes/` and calls it the notes archive. Under one abcd home, two different stores would both be called notes, and the one named `notes/` would not hold the memory notes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 726.
