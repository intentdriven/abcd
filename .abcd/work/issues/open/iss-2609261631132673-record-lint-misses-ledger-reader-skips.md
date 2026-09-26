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
---

record-lint does not report an issue record the ledger reader skips for a value its schema check refuses: duplicates: [bogus] or blocked_by: [bogus] is lint-green while capture list skips the record and abcd <id> names the refusal. record_schema re-derives parts of the reader's judgement (required and unknown fields, the filename agreement) and misses the rest, though its store declaration states that a record the reader would refuse is not lint-green.
