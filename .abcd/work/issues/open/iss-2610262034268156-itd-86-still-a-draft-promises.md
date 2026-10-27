---
schema_version: 1
id: "iss-2610262034268156"
slug: "itd-86-still-a-draft-promises"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 495 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-86-cold-reading-surface.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-86]
finding_ref: cfk-597f29e1cd4fcd99
---

itd-86, still a draft, promises cold-reading detections that land as reading records; the capture chapter says the reading producer has shipped and owns that output contract.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:35` — "Each detection lands as a reading record in the issue tier via the cold-reading output contract"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:610` — "this surface and it ships: the reading verb's ingest owns the output contract"

itd-86 still lists as undelivered that each detection lands as a reading record in the issue tier through the cold-reading output contract. 06-capture.md says the producer of reading items has shipped and that the reading verb's ingest owns the output contract. A shipped surface and an open draft claim the same ground.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 495.
