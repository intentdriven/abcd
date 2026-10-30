---
schema_version: 1
id: "iss-2610262034384038"
slug: "the-library-chapter-says-the-memory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 511 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md:11"
remedy: "none (filed automatically)"
related_intents: [itd-39]
finding_ref: cfk-59780892e699c1bf
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033287288. Same End A (07-library.md:11) and same claim as the 37-memory variant; itd-39 line 78 is simply the intent-side statement that memory is now the notes command."
duplicates: [iss-2610262033287288]
---

The library chapter says the memory command merged into library; itd-39 uses memory as the live name of the agents' notes command.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/07-library.md:11` — "The command merges the two that came before it, `source` and `memory`"
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:78` — "Since the verb split, `memory` is the agents' notes command"

07-library states that library merges the two commands before it, source and memory, the memory store's command among them, and that neither old name survives as an alias. itd-39's decision says that since the verb split memory is the agents' notes command, a different store (memory notes) from the memory pages library serves. One verb name thus denotes the retired memory-store command in one document and a live notes command in the other, beside two stores that are both called memory.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 511.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033287288. Same End A (07-library.md:11) and same claim as the 37-memory variant; itd-39 line 78 is simply the intent-side statement that memory is now the notes command.
