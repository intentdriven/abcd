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
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker, jointly with iss-2609251151293535: does the closed --source vocabulary gain members for a security advisory, a handover item and an autonomous bug-hunt finding, or does the capture page document which existing member each maps to?"
---

abcd capture's --source enum has no value for a security advisory (an external reviewer's finding on the forge) nor for a handover item (a NEXT.md or lab finding another session left for filing); the autonomous run used review-followup and agent-observation and stated the real origin in each body. Either the enum gains security-advisory and handover, or the surface page documents the mapping so every future run chooses the same values.
