---
schema_version: 1
id: "iss-2608300210588414"
slug: "ready-detail-strings-mislead-on-terminal-and-scaffold-records"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-177 build review, 2026-08-30"
found_at: "internal/core/intent/ready.go, internal/core/intent/create.go"
resolution: "The claim checks report as not applicable on shipped and superseded records, carrying no remedy, so the gate never names a backfill spc-55 rules out of scope; and an untouched create-path prompt is reported as unanswered rather than as a stated claim, asking create.go — which owns the templates — whether the bytes are still its own."
impact: internal
resolved_by:
  intent: "itd-177"
  spec: "spc-55"
---

intent ready reports a scope_conditions failure with a backfill remedy on shipped and superseded intents (18 shipped records), which spc-55's Out of scope forbids; and an untouched Mechanism scaffold line is reported as a stated mechanism claim, contradicting the plugin surface's line that the scaffold is not a claim. Detail strings on checks whose verdicts are already settled by the bucket check, so misleading output rather than a wrong gate.
