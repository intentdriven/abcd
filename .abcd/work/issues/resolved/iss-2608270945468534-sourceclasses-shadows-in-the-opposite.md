---
schema_version: 1
id: "iss-2608270945468534"
slug: "sourceclasses-shadows-in-the-opposite"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/memory/schema.go"
resolution: "SourceClasses returns the scalar-plus-plural union, matching the lint's derivedClasses"
impact: internal
resolved_by:
  commit: "4eb192a5"
---

SourceClasses shadows in the opposite direction to the lint derivation — it returns the scalar class alone when both shapes are present, so a both-shapes page renders under its scalar class in the generated index and write log while lint counts the union; display-only and behind the write-path exclusivity check, but the last surviving instance of the shape-shadowing root