---
schema_version: 1
id: "iss-2610262040221402"
slug: "the-memory-chapter-makes-the-in-tree"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 926 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:13"
remedy: "none (filed automatically)"
finding_ref: cfk-c43c122782de6446
---

The memory chapter makes the in-tree repository store memory's primary home; the glossary says a memory note stays on the person's machine until it is promoted.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:13` — "Project-shaped knowledge — pitfalls, decisions, principles tied to the project being built. Most memory is repo-scoped."
- End B: `.abcd/development/brief/glossary/core/memory-note.md:18` — "A note stays on the person's machine until it is promoted into the project's record, where every agent receives it."

07-memory sends memory to the committed in-tree .abcd/memory/ by default, calling it the primary home and saying most memory is repo-scoped; the user scope is only for personal preferences. The memory-note entry describes the same keyword-recalled lesson as staying on the person's machine and entering the project's record only when promoted. A builder cannot make the default home both the repository and the machine.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 926.
