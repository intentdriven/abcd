---
schema_version: 1
id: "iss-2609261036354146"
slug: "scribe-ingest-authenticates-nothing-about-the-parked-context"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
---

scribe ingest authenticates nothing about the parked context and manifest pair: a scribe that rewrites context.json's supplied dispositions and recomputes the manifest's context hash passes the verbatim check against text it wrote itself, while manifest.supplied.dispositions_sha256 is recorded and never read
