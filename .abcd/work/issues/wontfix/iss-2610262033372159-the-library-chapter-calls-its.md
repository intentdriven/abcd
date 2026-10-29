---
schema_version: 1
id: "iss-2610262033372159"
slug: "the-library-chapter-calls-its"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 425 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md:50"
remedy: "none (filed automatically)"
finding_ref: cfk-3afc5377ee653fc8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033287288. Same root cause as the 07-library line-11 record: the library chapter still calls its pages half the memory store after the surfaces README says the merge freed that name for the notes command."
duplicates: [iss-2610262033287288]
---

The library chapter calls its distilled-pages half the memory store, while the surfaces register says the merge freed the name memory for the notes command.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/07-library.md:50` — "memory store: ingesting a source, asking a question and the store lint."
- End B: `.abcd/development/brief/04-surfaces/README.md:494` — "That merge freed the name `memory`, which now belongs to the notes command"

07-library calls the half of /abcd:library that ingests, asks and lints distilled pages the memory store. The surfaces README says the merge that created library freed the name memory, which now belongs to the notes command and its short lessons. The two documents use the same word for two different stores.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 425.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033287288. Same root cause as the 07-library line-11 record: the library chapter still calls its pages half the memory store after the surfaces README says the merge freed that name for the notes command.
