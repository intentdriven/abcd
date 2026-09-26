---
schema_version: 1
id: "iss-2609260926130217"
slug: "abcd-intent-consistency-ingest-files-every-finding-in-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/consistency.go"
---

abcd intent consistency ingest files every finding in the ledger before it creates the report, and when the report create or write fails the error names only the report path, not the issue records already filed with that report as their evidence, unlike the filing-failure error beside it, which does name them.
