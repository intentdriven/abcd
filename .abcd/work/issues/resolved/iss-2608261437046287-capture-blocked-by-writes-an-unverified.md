---
schema_version: 1
id: "iss-2608261437046287"
slug: "capture-blocked-by-writes-an-unverified"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "internal/core/capture/workflow.go"
resolution: "capture probes blocked-by targets before writing; dangling target refuses with nothing written"
impact: fix
resolved_by:
  commit: "a2aa7780"
---

capture --blocked-by writes an unverified cross-reference its own blocker then refuses