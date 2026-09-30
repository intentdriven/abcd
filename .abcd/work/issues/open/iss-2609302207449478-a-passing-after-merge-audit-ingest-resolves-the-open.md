---
schema_version: 1
id: "iss-2609302207449478"
slug: "a-passing-after-merge-audit-ingest-resolves-the-open"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "On every passing ingest, ask the ledger to sweep the open carriers of that intent and receipt (one List of the open ledger per pass) whether or not the flag is present, resolving each as the flagged path does; and reword the 05-intent.md row tail to say what the ingest does to the ledger: it files or links one carrier of a failed or undecided check, declining extra carriers as duplicates, and a passing verdict resolves every open carrier of its receipt."
---

A passing after-merge audit ingest resolves the open owed-check carriers of its receipt only when the intent carries the audit-owed flag, so a carrier left open on an unflagged receipt (an ingest that filed it and then failed to write the intent) outlives the passing audit; and brief 05-intent.md's audit-ingest row still ends '(may capture or resolve one issue)', though a failed ingest may decline extra carriers and a passing one resolves every carrier (reverify-dq1cFlag, minor).
