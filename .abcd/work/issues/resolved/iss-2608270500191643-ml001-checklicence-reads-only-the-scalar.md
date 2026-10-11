---
schema_version: 1
id: "iss-2608270500191643"
slug: "ml001-checklicence-reads-only-the-scalar"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/memory/lint.go"
resolution: "ML001 derives the class set via derivedClasses so plural source.classes pages are licence-checked"
impact: fix
resolved_by:
  commit: "898c194e"
---

ML001 checkLicence reads only the scalar source.class, never the plural classes: [external_pdf], so a plural-list licence passes with zero blockers. GitHub mirror: #320