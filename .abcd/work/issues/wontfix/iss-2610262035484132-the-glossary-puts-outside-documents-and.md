---
schema_version: 1
id: "iss-2610262035484132"
slug: "the-glossary-puts-outside-documents-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 599 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/memory-note.md:26"
remedy: "none (filed automatically)"
finding_ref: cfk-108f5feb409f5889
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. 11-history.md uses the sanctioned name for the library command's second store, and the library glossary entry already covers the cited notes drawn from sources. Nothing in memory-note.md claims ingest belongs to memory notes."
---

The glossary puts outside documents and links in the library, not memory; the history chapter credits 'the memory store' with an ingest that takes documents and web addresses

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/memory-note.md:26` — "For outside material a decision may need to cite: that belongs in the"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:394` — "the memory store's ingest takes a document or a web address, never a stored transcript"

memory-note.md defines memory as short lessons an agent learned, and says outside material a decision may need to cite belongs in the library. The history chapter says 'the memory store's ingest takes a document or a web address'. Taking in documents and web addresses is what the glossary assigns to the library, so the history chapter uses 'memory' in the sense the glossary gives to 'library'.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 599.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. 11-history.md uses the sanctioned name for the library command's second store, and the library glossary entry already covers the cited notes drawn from sources. Nothing in memory-note.md claims ingest belongs to memory notes.
