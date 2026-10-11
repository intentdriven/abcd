---
schema_version: 1
id: "iss-2608270908339164"
slug: "glossary-index-tests-presence-with"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/glossary/index.go"
resolution: "glossary readTerm treats a null field as missing, so term: NULL no longer mints a phantom term"
impact: internal
resolved_by:
  commit: "4eb192a5"
---

glossary index tests presence with TrimSpace-empty only, so term: NULL or status: ~ is accepted as a literal value and mints a term named NULL — a YAML null becoming data rather than a diagnosis