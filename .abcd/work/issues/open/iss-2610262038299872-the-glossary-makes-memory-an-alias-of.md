---
schema_version: 1
id: "iss-2610262038299872"
slug: "the-glossary-makes-memory-an-alias-of"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 798 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/memory-note.md:5"
remedy: "Reword the 'Memory locations to keep straight' paragraph in 03-configuration.md: drop 'the brief says memory unqualified means the repo-scope store' and say the brief writes 'memory store' for the curated .abcd/memory/ pages and 'memory notes' for memory.json."
finding_ref: cfk-0bc3faa2a8a3c4a4
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The glossary makes "memory" an alias of memory note; the configuration chapter says unqualified "memory" means the repo-scope `.abcd/memory/` page store.

A terminology drift (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/memory-note.md:5` — "aliases: ["memory"]"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1175` — "the brief says "memory" unqualified, it means the repo-scope store"

memory-note.md gives "memory" as the alias of a memory note, a short lesson kept in `memory.json`. 03-configuration says that when the brief writes "memory" unqualified it means the repo-scope store, the curated `.abcd/memory/` pages that `abcd library ingest` writes, and calls the notes a separate thing. A builder reading "memory" unqualified gets opposite stores depending on which page governs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 798.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
