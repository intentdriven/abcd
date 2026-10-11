---
schema_version: 1
id: "iss-2609251939472371"
slug: "the-scaffolded-release-workflows-in-repo"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/scaffold/cichecks.go"
resolution: "CI's zizmor job renders every scaffolded profile (scaffold.AuditProfiles, the set the in-repo audit also reads) with cmd/scaffold-render and runs the same pinned zizmor over them, so the bare and gate profiles are audited for action pinning, permissions and credential handling. zizmor 1.26.1 run locally over the six rendered profiles and the committed workflows reports no findings."
impact: internal
resolved_by:
  commit: "541daa9d0"
---

The scaffolded release workflows' in-repo audit covers injection and duplicate keys for every profile, but it is not a full zizmor stand-in for the bare profile: action pinning, permissions and credential classes are unchecked there, and only the abcd profile is zizmor-audited in CI. Follow-up: run zizmor over a rendered bare profile in CI.

## Grounds

- pursued: a pinning, permission or credential defect in a non-abcd profile now fails CI's zizmor job; shown wrong if a profile template regresses on one of those audits and the job stays green
