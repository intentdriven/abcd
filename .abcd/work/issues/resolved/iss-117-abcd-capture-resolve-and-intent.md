---
schema_version: 1
id: "iss-117"
slug: "abcd-capture-resolve-and-intent"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "itd-73 phase 1 derived versioning"
found_at: "internal/core/capture/workflow.go"
resolution: "capture resolve and intent seed now stamp a valid impact; the write paths reach the shared changelog enum so the records they mint satisfy issue_impact_valid and intent_impact_valid"
impact: fix
---

abcd capture resolve and intent seedDraft cannot set impact, so the tool's own path produces a record the new issue_impact_valid and intent_impact_valid blockers reject