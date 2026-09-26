---
schema_version: 1
id: "iss-2609251939472371"
slug: "the-scaffolded-release-workflows-in-repo-audit-covers"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/scaffold/cichecks.go"
---

The scaffolded release workflows' in-repo audit covers injection and duplicate keys for every profile, but it is not a full zizmor stand-in for the bare profile: action pinning, permissions and credential classes are unchecked there, and only the abcd profile is zizmor-audited in CI. Follow-up: run zizmor over a rendered bare profile in CI.
