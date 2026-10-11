---
schema_version: 1
id: "iss-2608270908346940"
slug: "two-adrs-sharing-a-filename-ordinal-make"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/record/record.go"
resolution: "record-lint reports an id/ordinal collision instead of silently overwriting the second claimant"
impact: internal
resolved_by:
  commit: "b55c5c38"
---

two ADRs sharing a filename ordinal make the second unreachable: describeADR routes on the first matching ordinal and no lint rule checks ADR id uniqueness, the backstop recordid.Resolver's first-in-scan-order comment assumes exists