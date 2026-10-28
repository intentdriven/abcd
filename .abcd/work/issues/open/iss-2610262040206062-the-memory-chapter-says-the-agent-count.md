---
schema_version: 1
id: "iss-2610262040206062"
slug: "the-memory-chapter-says-the-agent-count"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 924 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:158"
remedy: "none (filed automatically)"
finding_ref: cfk-66a0ac6932ce1849
---

The memory chapter says the agent count stays at 15; the internals index says eighteen agent prompts ship.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:158` — "The agent count stays at 15"
- End B: `.abcd/development/brief/05-internals/README.md:7` — "The eighteen shipped agent prompts and the design roster still to be built"

07-memory states as a standing fact that the agent count stays at 15, with the curator taking on roles instead of peers. The internals index describes eighteen shipped agent prompts. Both describe the current size of the same roster, so they cannot both be true.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 924.
