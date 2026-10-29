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
remedy: "Per dcn-2610271112334659: an agent's lesson is kept first as a machine-local memory note and reaches the project only when a person promotes it. Rescope 07-memory's 'most memory is repo-scoped' to curated pages built from ingested sources and promoted notes."
finding_ref: cfk-c43c122782de6446
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The memory chapter makes the in-tree repository store memory's primary home; the glossary says a memory note stays on the person's machine until it is promoted.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:13` — "Project-shaped knowledge — pitfalls, decisions, principles tied to the project being built. Most memory is repo-scoped."
- End B: `.abcd/development/brief/glossary/core/memory-note.md:18` — "A note stays on the person's machine until it is promoted into the project's record, where every agent receives it."

07-memory sends memory to the committed in-tree .abcd/memory/ by default, calling it the primary home and saying most memory is repo-scoped; the user scope is only for personal preferences. The memory-note entry describes the same keyword-recalled lesson as staying on the person's machine and entering the project's record only when promoted. A builder cannot make the default home both the repository and the machine.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 926.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
