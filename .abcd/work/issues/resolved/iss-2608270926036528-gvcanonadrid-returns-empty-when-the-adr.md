---
schema_version: 1
id: "iss-2608270926036528"
slug: "gvcanonadrid-returns-empty-when-the-adr"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/lifeboat/graveyard_abandoned.go"
resolution: "gvCanonADRID canonicalises the ordinal textually, so no unrepresentable input exists and every well-formed adr-N keeps its identity"
impact: fix
resolved_by:
  commit: "2a668ece"
---

gvCanonADRID returns empty when the ADR ordinal overflows Atoi, so an overlong id silently drops the finding the previous regexp path emitted — canonicalise textually by trimming leading zeros so overflow cannot unmake an identity