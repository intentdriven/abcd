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
---

scribe ingest's state check grants a state to every item a line names: lineCarries in internal/core/scribe/ingest.go reads the whole line that mentions the item, so a line such as 'rdi-2: accepted, unlike rdi-1' carries 'accepted' for rdi-1 too, and a payload filing rdi-1 as accepted passes the check the researcher's own line contradicts (probed by review2-scribe: {1}: accepted ... (unlike {0}) wrote {0} accepted). The state must be held to the part of the line that belongs to the item.
