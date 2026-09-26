---
schema_version: 1
id: "iss-2609261036355193"
slug: "scribe-ingest-never-holds-a-disposition-s-state-to-the"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
---

scribe ingest never holds a disposition's state to the supplied text: a researcher line 'rdi-X: rejected — <ground>' ingests as state accepted with the verbatim ground, so the ruling itself is the one field the scribe can author, against the spec's out-of-scope rule that a state the material does not carry is refused, never supplied
