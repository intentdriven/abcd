---
schema_version: 1
id: "iss-2609250846525896"
slug: "record-moving-verbs-other-than-spec-close-leave-links-to-the-moved-record-dead"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
---

capture resolve, capture wontfix and intent plan move a record between status folders and leave every relative markdown link that named its old path pointing at nothing, the same gap iss-2609091732329046 records for spec close. Lane records1 of run A closed two issues that three ADRs and two draft intents linked by path, and record-lint refused seven links_resolve blockers until the links were repointed by hand. Every verb that moves a record is the one place that knows the old and the new path, so each should repoint the links through one shared primitive and report what it rewrote.
