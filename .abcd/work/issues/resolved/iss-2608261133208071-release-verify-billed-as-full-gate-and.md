---
schema_version: 1
id: "iss-2608261133208071"
slug: "release-verify-billed-as-full-gate-and"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: ".github/workflows/release.yml:3"
resolution: "release.yml calls verify the deterministic Linux-only subset of the merge gate and the runbook gains an Enforced-at-merge-only section outside the lockstep parser reach"
impact: internal
resolved_by:
  commit: "9ae6537b"
---

release.yml bills verify as the full verification gate mirroring the ci check and record-lint jobs while it deliberately runs a nine-gate subset; the adjective overclaims and the mirror sentence is stale