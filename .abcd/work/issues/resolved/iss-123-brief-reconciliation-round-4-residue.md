---
schema_version: 1
id: "iss-123"
slug: "brief-reconciliation-round-4-residue"
severity: "minor"
impact: internal
category: "observation"
source: "agent-finding"
found_during: "iss-121 reconciliation, round 4"
found_at: ".abcd/development/brief/04-surfaces/"
resolution: "All nine residue findings fixed and verified against binary and source in a final targeted pass; the live-bundle finding corrected upward during verification (four bundle-members, itd-20/24/63/69, not two) with the stale intents/README.md bundles table fixed alongside. Fifth-run crosscheck counts are iss-122 calibration data by maintainer decision, not a merge gate."
---

Round-4 residue of the brief reconciliation: nine precision-grade discrepancies remain after three fix rounds (102 -> 52 -> 25 -> 9), all empirically verified by the checkers and none of the judgement-flip-flop class. Examples: bare ahoy prints a next-step line only for unmanaged kinds but the criterion says any folder kind; embark's manifest verification excludes the post-pack synthesis layer while the chapter claims blanket tamper-evidence; a live bundle (spc-83-operator-surfaces, itd-20+itd-24) contradicts the no-live-bundles claim; the persona_registry lint checks personas.json, not the markdown page named as SSOT; history capture idempotency keys on (sha256, session, kind), not content hash alone. Full findings in the local scratch tier (2026-07-22-iss35-crosscheck-round4 file). The geometric tail also suggests the detector at 22-checker depth has a nonzero noise floor per fresh run — the pass threshold question belongs to iss-122.