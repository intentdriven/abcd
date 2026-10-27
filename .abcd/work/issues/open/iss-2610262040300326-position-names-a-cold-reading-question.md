---
schema_version: 1
id: "iss-2610262040300326"
slug: "position-names-a-cold-reading-question"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 936 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/reading-position.md:5"
remedy: "none (filed automatically)"
finding_ref: cfk-608abd7ec23ce5fb
---

'position' names a cold-reading question in reading-position and a record's how-soon axis in record-families.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/reading-position.md:5` — "aliases: ["reading position", "position"]"
- End B: `.abcd/development/brief/glossary/core/record-families.md:30` — "The *position* (how soon) is rendered from it as Now / Next / Later and is never stored."

reading-position registers bare 'position' as an alias for the question a cold reading answers (widening, entailment, comparative, detection). record-families uses 'position' for a record's how-soon axis, rendered as Now / Next / Later. Both entries are in the core context, so one word names two concepts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 936.
