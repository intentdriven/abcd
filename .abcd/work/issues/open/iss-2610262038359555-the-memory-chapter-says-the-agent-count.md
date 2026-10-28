---
schema_version: 1
id: "iss-2610262038359555"
slug: "the-memory-chapter-says-the-agent-count"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 805 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:158"
remedy: "none (filed automatically)"
finding_ref: cfk-965eacd5c171d8a5
---

The memory chapter says the agent count stays at 15; the agent catalog ships eighteen.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:158` — "The agent count stays at 15"
- End B: `.abcd/development/brief/05-internals/01-agents.md:18` — "Eighteen agent prompts ship in `agents/` today, in five groups:"

07-memory says the agent count stays at 15 and that the curator gains roles rather than peers. 01-agents says eighteen agent prompts ship in `agents/`. The two counts cannot both describe the roster.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 805.
