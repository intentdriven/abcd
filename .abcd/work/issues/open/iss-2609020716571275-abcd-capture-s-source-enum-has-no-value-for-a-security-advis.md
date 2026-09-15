---
schema_version: 1
id: "iss-2609020716571275"
slug: "abcd-capture-s-source-enum-has-no-value-for-a-security-advis"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/issueschema/issueschema.go"
---

abcd capture's --source enum has no value for a security advisory (an external reviewer's finding on the forge) nor for a handover item (a NEXT.md or lab finding another session left for filing); the autonomous run used review-followup and agent-observation and stated the real origin in each body. Either the enum gains security-advisory and handover, or the surface page documents the mapping so every future run chooses the same values.
