---
schema_version: 1
id: "iss-2610262040599781"
slug: "the-glossary-sends-a-promoted-memory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 969 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/memory-note.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290]
finding_ref: cfk-f26a2fb9e23c9c33
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038308994. Identical End A (memory-note.md:31 promoted into the project's rules) and the same destination fact from itd-2610091918433290."
duplicates: [iss-2610262038308994]
---

The glossary sends a promoted memory note into the project's rules; the intent keeps promoted notes in `.abcd/memory.json`, apart from the rules people curate.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/memory-note.md:31` — "is promoted into the project's rules, so a second contributor's agent receives it too"
- End B: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64` — "the project's `.abcd/memory.json` holds promoted notes"

memory-note.md's example says a note recalled across sessions is promoted into the project's rules. itd-2610091918433290 puts promoted notes in the project's `.abcd/memory.json`, states that agents write memory.json while people curate rules.json, and keeps the two files separate. A promoted note cannot end up in both places.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 969.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038308994. Identical End A (memory-note.md:31 promoted into the project's rules) and the same destination fact from itd-2610091918433290.
