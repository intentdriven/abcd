---
schema_version: 1
id: "iss-2608271804494867"
slug: "decisions-log-ordering-drifts-and"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/work/DECISIONS.md"
resolution: "DECISIONS.md gated append-only by DA001-DA004 (position, preservation, per-parent merge multiplicity, no-NUL), wired into preflight/pre-push/CI"
impact: fix
resolved_by:
  commit: "154af92a"
---

DECISIONS.md violates its own newest-last ordering in five places (backwards date steps among the 247 dated bullets), and nothing gates it. Do not reorder — reordering is what append-only forbids, and a naive date-monotonicity rule would refuse honest back-dated tail entries. Gate the position instead: a CI check that every line a diff adds to DECISIONS.md lands at the tail, never inserted above an existing entry.