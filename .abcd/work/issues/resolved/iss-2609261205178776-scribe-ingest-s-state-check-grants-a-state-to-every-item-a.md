---
schema_version: 1
id: "iss-2609261205178776"
slug: "scribe-ingest-s-state-check-grants-a-state-to-every-item-a"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
resolution: "a line naming several items gives each only the text from its id to the next item id, so one item's ruling is not granted to another the line mentions; the positional residue is disclosed in 31-scribe.md and commands/scribe.md"
impact: fix
resolved_by:
  commit: "e8c12571"
---

scribe ingest's state check grants a state to every item a line names: lineCarries in internal/core/scribe/ingest.go reads the whole line that mentions the item, so a line such as 'rdi-2: accepted, unlike rdi-1' carries 'accepted' for rdi-1 too, and a payload filing rdi-1 as accepted passes the check the researcher's own line contradicts (probed by review2-scribe: {1}: accepted ... (unlike {0}) wrote {0} accepted). The state must be held to the part of the line that belongs to the item.

## Grounds

- pursued: TestScribeIngestHoldsTheStateToTheItemsPartOfALine refuses rdi-1 accepted from 'rdi-2: accepted (unlike rdi-1)', lands each item's own ruling, holds an admission the same way, and keeps a single-item line whole; a payload granting the mentioned item the other's state landing would show it wrong
