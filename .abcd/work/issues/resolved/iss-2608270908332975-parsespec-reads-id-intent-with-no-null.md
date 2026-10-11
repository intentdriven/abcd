---
schema_version: 1
id: "iss-2608270908332975"
slug: "parsespec-reads-id-intent-with-no-null"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/spec/store.go"
resolution: "parseSpec runs id/slug/intent through frontmatter.IsNull, so a null id reads as unset rather than a malformed literal"
impact: internal
resolved_by:
  commit: "4eb192a5"
---

parseSpec reads id/intent with no null gate, so id: NULL yields a malformed-shape diagnosis while lint's schema gate reports the field unset — the spec-family sibling of the iss-286 divergence