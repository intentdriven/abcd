---
schema_version: 1
id: "iss-2610262041476159"
slug: "abcd-memory-names-the-memory-note"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1025 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/memory-note.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-c5b979a27bb3ac5f
---

/abcd:memory names the memory-note command in the glossary, but itd-36 names the curated knowledge substrate with ingest, ask and lint.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/memory-note.md:18` — "An agent writes one with `/abcd:memory add` when it learns something worth keeping"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:23` — "abcd ships `/abcd:memory` — a per-`voyage` curated knowledge substrate"

The memory-note entry, whose alias is 'memory', says an agent writes a note with `/abcd:memory add`, and the loader recalls it. itd-36's press release presents `/abcd:memory` as the curated knowledge substrate, with `/abcd:memory ask`, `/abcd:memory lint` and `/abcd:memory ingest`. One command name and one word refer to two different concepts, and a reader of the shipped intent will reach for sub-verbs that the command does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1025.
