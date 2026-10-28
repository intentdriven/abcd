---
schema_version: 1
id: "iss-2610262038308994"
slug: "the-memory-note-entry-says-a-promoted"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 799 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/memory-note.md:31"
remedy: "none (filed automatically)"
finding_ref: cfk-a89e833b5cea5c11
---

The memory-note entry says a promoted note goes into the project's rules; the configuration chapter puts promoted notes in the project's own `memory.json` layer.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/memory-note.md:31` — "A note that has been recalled in several sessions is promoted into the project's rules"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:792` — "`memory.json` (the layer the prompt router recalls above the machine's"

memory-note.md's example has a note recalled across sessions promoted into the project's rules. 03-configuration says the project's memory notes live in `.abcd/memory.json`, a layer the prompt router recalls above the machine's `memory.json`, separate from `rules.json`. So promotion has two different destinations.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 799.
