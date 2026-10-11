---
schema_version: 1
id: "iss-2608211432258975"
slug: "lint-engine-fault-exit-1-reads-as"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "internal/surface/cli/lint.go"
resolution: "Map a rule-engine fault to exit 2 (tri-state any-error) instead of 1"
impact: fix
resolved_by:
  commit: "17f3956"
---

abcd lint maps a rule-engine fault to exit 1 which its documented Conftest tri-state reserves for warnings-only, so a CI gate keying on exit>=2 reads a lint that never ran as an advisory pass (fail-open)