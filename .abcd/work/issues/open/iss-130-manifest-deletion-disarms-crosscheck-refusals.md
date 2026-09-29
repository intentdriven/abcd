---
schema_version: 1
id: "iss-130"
slug: "manifest-deletion-disarms-crosscheck-refusals"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "iss-122 implementation review (2026-07-24 run queue, burst 6)"
found_at: ".abcd/development/release-gate/manifest.json"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Sign off a release.yml step that refuses a release after v0.4.0 that carries no release-gate manifest?"
remedy: "If the person signs off the release.yml step: add a step before the receipts gate that refuses a release newer than v0.4.0 whose tree lacks .abcd/development/release-gate/manifest.json; if not: put the same refusal in record-lint --release-gate itself, so no CI configuration changes. Prove it with a test that deletes the manifest from a post-v0.4.0 fixture and expects the refusal."
---

era-gating tradeoff residue from iss-122: deleting release-gate/manifest.json from the content tree disarms the three new procedural refusals (manifest presence IS the era marker, per the decided design F) — the backstop belongs in release.yml (require the manifest to exist for any release newer than v0.4.0), which is CI config and needs maintainer sign-off; until then a committer who removes the manifest silently reverts the gate to pre-manifest rules

## Remedy grounds (2026-09-29)

- Either placement closes the silent disarm the body names; the second is offered because it needs no CI-configuration sign-off and is covered by the existing receipt tests in internal/core/lint.
- Rejected: treating the manifest's absence as an I/O error unconditionally, which would refuse the pre-manifest history the era marker exists to admit.
