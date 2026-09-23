---
schema_version: 1
id: "iss-130"
slug: "manifest-deletion-disarms-crosscheck-refusals"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "iss-122 implementation review (2026-07-24 run queue, burst 6)"
found_at: ".abcd/development/release-gate/manifest.json"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (sign-off owed: release.yml backstop requiring release-gate manifest for releases after v0.4.0). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

era-gating tradeoff residue from iss-122: deleting release-gate/manifest.json from the content tree disarms the three new procedural refusals (manifest presence IS the era marker, per the decided design F) — the backstop belongs in release.yml (require the manifest to exist for any release newer than v0.4.0), which is CI config and needs maintainer sign-off; until then a committer who removes the manifest silently reverts the gate to pre-manifest rules