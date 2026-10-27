---
schema_version: 1
id: "iss-2610262033478703"
slug: "memory-names-two-things-the-library-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 437 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/memory-note.md:18"
remedy: "none (filed automatically)"
finding_ref: cfk-e165534b0e2013d5
---

'memory' names two things: the library's in-repo memory store, whose old memory command the library chapter says was merged away, and the glossary's memory note, written with /abcd:memory add and kept on the person's machine.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/memory-note.md:18` — "An agent writes one with `/abcd:memory add` when it learns something worth keeping"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:11` — "The command merges the two that came before it, `source` and `memory`, under"

07-library says the library command merged the earlier source and memory commands, with neither old name surviving as an alias, and calls its in-repo .abcd/memory/ pages 'the memory store'. glossary/core/memory-note.md, whose alias is 'memory', says an agent writes a memory note with /abcd:memory add and that the note stays on the person's machine until promoted. One word names a repository store of distilled source pages and a machine-local store of agent lessons, and the two pages disagree on whether a memory command exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 437.
