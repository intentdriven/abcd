---
schema_version: 1
id: "iss-2609291118049254"
slug: "capture-resolve-and-capture-wontfix-have-no-way-to-set-the"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous run A 2026-09-29, review-triageMinorC"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/capture.go"
---

capture resolve and capture wontfix have no way to set the typed duplicates link, so a record closed as a duplicate carries the relation only in wontfix_reason prose. A machine reader of the ledger sees a wontfix, not a duplicate (found by review-triageMinorC: four duplicate closures in lane triageMinorC, iss-2608290811463906, iss-2609170640135192, iss-277, iss-91). Add a --duplicates <id> flag (issue or intent id) to wontfix, or a duplicate sub-verb, writing the field the ledger already defines.
