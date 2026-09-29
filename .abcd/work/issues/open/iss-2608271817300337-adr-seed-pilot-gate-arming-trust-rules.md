---
schema_version: 1
id: "iss-2608271817300337"
slug: "adr-seed-pilot-gate-arming-trust-rules"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-84 decomposition of the pilot-note proposal (2026-08-27)"
found_at: ".abcd/development/research/notes/2026-08-27-security-advisory-handling-pilot.md"
remedy: "Waits on ruling E: if filed before planning, mint the ADR with `go run ./cmd/abcd decide` stating the three trust rules (auto-merge arms only after a no-bypass required-check set and an independent adversarial APPROVE; the two human-by-design release gates stay human; every required pass re-runs against the final content commit), add its brief invariant line, and list it in itd-149's related ADRs, proven by lint-decisions and record-lint green; if left to planning, move the three rules into itd-149's open questions and resolve this record pointing there."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed E): File the ADR for the pilot's trust rules (auto-merge arming, human release gates, no verdict transfer) before itd-149 is planned?"
---

ADR seed from the security-advisory pilot's verified trust rules (itd-149 decomposition, part 2): auto-merge arms only after a comprehensive no-bypass required-check set AND an independent adversarial APPROVE, never on CI-green alone (F-N); the two release gates that are human-by-design stay human (F-Q); a verdict is never transferred — every required pass re-runs against the final content commit (F-U). These are trust-boundary rules, not intent scope: file the ADR with its brief invariant, then link itd-149 to it at planning.

## Remedy grounds (2026-09-29)

- The itd-84 routing sends a trust rule to an ADR plus a brief invariant, the route adr-44 followed, and no ADR at the base carries these three rules.
- No outside-practice check: the rules come from the repository's own hand-run pilot.
- Rejected: folding the rules into itd-149's scope, which the record calls out as the wrong home for a trust boundary.
