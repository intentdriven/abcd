---
schema_version: 1
id: "iss-2610262045126115"
slug: "itd-50-relies-on-a-drainer-that-runs"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1258 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:63"
remedy: "In itd-50's Out of Scope, replace the boundary drainer substrate with itd-53's one bounded command run once by the autonomous run's batch 0, and say the loop is build's last stage."
related_intents: [itd-50, itd-53]
finding_ref: cfk-f8d39777074e221a
---

itd-50 relies on a drainer that runs queued reviews at a safe boundary; itd-53 ruled out any boundary-hooked autodrain

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:63` — "the drainer that actually runs queued reviews at a safe boundary is its substrate"
- End B: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:65` — "run once by the autonomous run's batch 0; no boundary-hooked autodrain."

itd-50 makes the drainer that runs queued reviews at a safe boundary its substrate. itd-53 is that drainer, and its decision 1 replaces the boundary drainer with one bounded command run once by the autonomous run's batch 0, with no boundary-hooked autodrain. As ruled, itd-53 cannot supply what itd-50 says it depends on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1258.
