---
schema_version: 1
id: "iss-2609251606553359"
slug: "reading-ingest-with-route-refuses-an-oversized-output-as"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

reading ingest with --route refuses an oversized output as "names no reading position" instead of giving the size-cap message (internal/surface/cli/reading.go:246-251), so the operator is told the wrong cause (review2-tier2 note b).
