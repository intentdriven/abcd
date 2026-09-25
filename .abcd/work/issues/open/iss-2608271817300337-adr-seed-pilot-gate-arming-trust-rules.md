---
schema_version: 1
id: "iss-2608271817300337"
slug: "adr-seed-pilot-gate-arming-trust-rules"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-84 decomposition of the pilot-note proposal (2026-08-27)"
found_at: ".abcd/development/research/notes/2026-08-27-security-advisory-handling-pilot.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): File the ADR for the pilot's trust rules (auto-merge arming, human release gates, no verdict transfer) before itd-149 is planned?"
---

ADR seed from the security-advisory pilot's verified trust rules (itd-149 decomposition, part 2): auto-merge arms only after a comprehensive no-bypass required-check set AND an independent adversarial APPROVE, never on CI-green alone (F-N); the two release gates that are human-by-design stay human (F-Q); a verdict is never transferred — every required pass re-runs against the final content commit (F-U). These are trust-boundary rules, not intent scope: file the ADR with its brief invariant, then link itd-149 to it at planning.