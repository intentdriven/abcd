---
schema_version: 1
id: "iss-2608270926037660"
slug: "derivedclasses-takes-the-classes-list-in"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/memory/lint.go"
resolution: "derivedClasses unions the scalar class with the plural list so ML001 and its siblings see every class the page declares"
impact: fix
resolved_by:
  commit: "7cb8aae4"
---

derivedClasses takes the classes list in an else-if that shadows the scalar class, so a memory page declaring class: external_pdf beside classes: [session_memory] and no licence passes ML001 that main raised — the lint path never runs the write-path exclusivity validator, so nothing else catches the combination; the derivation must union the scalar with the list