---
schema_version: 1
id: "iss-2609261036355918"
slug: "scribe-ingest-promotes-the-manifest-beside-the-run-even-when"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
---

scribe ingest promotes the manifest beside the run even when no record landed, so a payload of refusals or outstanding items alone locks the run against every later scribe session over it
