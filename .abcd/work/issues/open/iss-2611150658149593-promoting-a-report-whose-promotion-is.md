---
schema_version: 1
id: "iss-2611150658149593"
slug: "promoting-a-report-whose-promotion-is"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report"
remedy: "On a resumed promotion, read the capture the log names from the ledger first; when it is not there, say so and leave the report waiting."
---

Promoting a report whose promotion is already in the log finishes the move without checking that the capture the log names is in the ledger.
