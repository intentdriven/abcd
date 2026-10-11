---
schema_version: 1
id: "iss-2608220142154022"
slug: "update-plan-switch-fails-open-on-unknown"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "bughunt-b-round-7"
found_at: "internal/core/update/update.go"
resolution: "update.Plan has an explicit UpdateTargetFile case and a default named refusal; an unknown kind never falls through to swap"
impact: fix
resolved_by:
  commit: "643c010"
---

update.Plan dispatch switch has no default case, so an unrecognised UpdateTargetKind falls through the return nil permissive branch of a mutating refusal gate to fetch-and-swap, against unrecognized-input-never-writes; UpdateTargetFile is now an explicit case and any other kind a named refusal