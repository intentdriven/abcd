---
schema_version: 1
id: "iss-136"
slug: "provenance-lacks-pass-b-exemption-marker"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "itd-88 fidelity review (2026-07-24 run queue, burst 10)"
found_at: "internal/core/lifeboat/plan.go"
related_intents: [itd-158]
resolution: "Provenance carries a pass_b_exemption marker with its reason; the embark coverage handoff reads it as a declared exemption, and an unmarked record is treated exactly as before"
impact: additive
resolved_by:
  intent: "itd-158"
  spec: "spc-51"
---

itd-88's fidelity gap audit found a missing press-release claim: Pass B is promised to ship as a declared exemption in _provenance.json, never a silent gap, but no exemption field or marker exists anywhere in the lifeboat package or the Provenance struct — a promise with no implementing code, recorded in itd-88's Audit Notes (receipt rcp-4d07032fc6ab)