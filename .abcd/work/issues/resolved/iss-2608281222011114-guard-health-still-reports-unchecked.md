---
schema_version: 1
id: "iss-2608281222011114"
slug: "guard-health-still-reports-unchecked"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "batch F guard fail-safe fix (2026-08-28)"
found_at: "internal/core/ahoy/guard_health.go"
resolution: "guard health distinguishes repo-overrides-dropped-bundled-armed from genuinely-unguarded, so it no longer claims commands run unchecked when bundled hazards are armed"
impact: internal
resolved_by:
  commit: "3eb4c53a"
---

the guard health report still says a broken repo guard.json means commands run unchecked, but the fail-safe fix (iss-2608261551087492) keeps the bundled hazards armed in that case: guard_health.go's guardRegistryUnloadableReason and the RegistryLoadable bool model a broken repo layer as fully-unguarded, which is now false. The health check should distinguish 'repo overrides dropped, bundled hazards still armed' (a mild, expected state) from 'no registry at all' (the only genuinely-unguarded state, which the embed makes unreachable). Refine the health model and reword the reason. Follow-up to the fail-safe change.