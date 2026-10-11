---
schema_version: 1
id: "iss-2608270908346617"
slug: "record-filename-grammar-diverges-lint-s"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/lint/schema.go"
resolution: "lint per-store filename match now uses recordid.FilenameNumRe, so a divergent record can no longer pass the gate and be lost by the reader"
impact: fix
resolved_by:
  commit: "b55c5c38"
---

record filename grammar diverges: lint's schema accepts an arbitrary tail and zero ordinal where recordid and capture require a kebab slug and positive ordinal, so a file like iss-5 with an underscore tail is lint-visible but dropped by capture's scanLedger into neither Issues nor Skipped and is a hard CitationError when cited — a silently lost record