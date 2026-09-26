---
schema_version: 1
id: "iss-2609261935343851"
slug: "intent-audit-ingest-outside-the-mint-lock"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainA1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

intent audit ingest is not under the intent mint lock: IngestVerdictBytes, reingestVerdict and deadLetter (internal/core/intent/audit.go) resolve the receipt, validate and write the shipped intent as a read-modify-write guarded only by a per-file atomic write, and deadLetter writes two files (the retained payload and the record); every other intent writer holds withIntentMintLock. Two concurrent ingests on one intent, or an ingest beside a condition disposition, each write the bytes they read: the later write erases the earlier one and both exit 0 (a verdict replaced while reporting a fresh ingest, or a dead-letter written over a verdict that had just landed).
