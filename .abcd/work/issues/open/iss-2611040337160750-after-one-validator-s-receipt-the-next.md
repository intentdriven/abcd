---
schema_version: 1
id: "iss-2611040337160750"
slug: "after-one-validator-s-receipt-the-next"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Say the agent is out and awaited; a test hands back one of two validators and reads next:."
---

After one validator's receipt, the next: line says start a fresh ruthless-reviewer agent while that agent is still out, which invites a duplicate spend.
