---
schema_version: 1
id: "iss-2610262036131057"
slug: "the-history-chapter-s-memory-store-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 632 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:394"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290]
finding_ref: cfk-4e9497a27d77e25d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035249588. Same End A line and the same ambiguity; itd-2610091918433290 is the intent-side record that gave the name memory to the notes command."
duplicates: [iss-2610262035249588]
---

The history chapter's 'memory store' is the store whose ingest takes documents and web addresses; itd-2610091918433290 says the name memory was freed into library, and gives /abcd:memory to agents' memory notes.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:394` — "the memory store's ingest takes a document or a web address"
- End B: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:58` — "freed the name `memory` by merging the sources and memory commands into `library`"

11-history says the memory store's ingest takes a document or a web address, and names it as the store that is meant to distil curated knowledge from transcripts. itd-2610091918433290 records that the sources and memory commands were merged into library, which freed the name memory, and uses /abcd:memory for agents' recall notes kept in memory.json. One name now points at two stores, so a reader of the history chapter cannot tell whether the transcript harvest feeds library's store or the agents' notes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 632.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035249588. Same End A line and the same ambiguity; itd-2610091918433290 is the intent-side record that gave the name memory to the notes command.
