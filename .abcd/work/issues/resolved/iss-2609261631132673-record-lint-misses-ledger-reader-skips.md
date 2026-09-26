---
schema_version: 1
id: "iss-2609261631132673"
slug: "record-lint-misses-ledger-reader-skips"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-match"
origin: researcher-authored
production_mode: hand-written
resolution: "The ledger reader's per-file judgement lives in internal/core/issuerecord; capture's scan and record_schema both call issuerecord.Judge, and record_schema reports every issue the reader skips with the reader's layer and reason."
impact: fix
resolved_by:
  commit: "65cce5a5"
---

record-lint does not report an issue record the ledger reader skips for a value its schema check refuses: duplicates: [bogus] or blocked_by: [bogus] is lint-green while capture list skips the record and abcd <id> names the refusal. record_schema re-derives parts of the reader's judgement (required and unknown fields, the filename agreement) and misses the rest, though its store declaration states that a record the reader would refuse is not lint-green.

## Grounds

- pursued: a committed issue carrying duplicates: [bogus], blocked_by: [bogus] or a list naming one record twice is a record_schema blocker in the reader's words while the live ledger stays clean; TestRecordSchemaReportsEveryRecordTheLedgerReaderSkips would fail if the gate reverted to its own validator, and a skip reason the gate does not report would show it wrong
