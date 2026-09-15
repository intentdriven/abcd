---
schema_version: 1
id: "iss-2608311234532073"
slug: "the-ingest-verb-s-durable-refusal-record-loses-the-reason-it"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The refusal record carries the whole reason; the redundant outer cap is gone."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest verb's durable refusal record loses the reason it exists to carry: refuse() applies echo() to the composed cause, but every payload-derived substring inside that cause is already echoed at composition, so the outer call only truncates trusted prose. Measured: a 338-rune terminal message becomes a 123-rune refusal.json reason cut mid-word, and for an every-item-refused run the per-item refusals are cut off entirely.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
