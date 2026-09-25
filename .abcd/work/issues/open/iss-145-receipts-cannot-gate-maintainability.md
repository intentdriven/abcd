---
schema_version: 1
id: "iss-145"
slug: "receipts-cannot-gate-maintainability"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "2026-07-27 external-source review"
found_at: ".abcd/development/principles/"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Adopt 'receipts gate process integrity, not maintainability' as a principle?"
---

receipts gate process integrity, not maintainability: candidate principle surfaced by the 2026-07 software-factories critique (primary observation: RL rewards fast oracles — tests pass in seconds — while bad architecture's cost function is measured in months, so maintainability has no fast oracle and models are not penalised for eroding it). Consequence for abcd: no receipt or deterministic gate can certify design quality; that verdict stays human (verifier-selects-gates-decide), and the maintainer merge on every PR is load-bearing, not ceremony. Candidate homes: a principle file or a brief mental-model line; adoption is the maintainer's call.