---
schema_version: 1
id: "iss-2609012043439946"
slug: "a-transient-i-o-failure-while-loading-a-definition-becomes-a"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest.go"
---

A transient I/O failure while loading a definition becomes a permanent refusal of the run. In ingest (internal/core/reading/ingest.go), once the run's identity is proven, a LoadDefinition error is routed through refuse and recorded as refusal.json — by design, so a refused run has something durable to find it by (iss-2608311518250688). The consequence: an EACCES on the definition file, the size cap, or a refused symlink — conditions that clear on retry — produce a durable refusal that refuseARerun then treats as the run's outcome, so the same run id cannot be retried; recovery is re-assembling under a new run id. Recorded as a consequence of a deliberate choice, not as a defect; a later change could distinguish a definition that does not resolve from one that could not be read.
