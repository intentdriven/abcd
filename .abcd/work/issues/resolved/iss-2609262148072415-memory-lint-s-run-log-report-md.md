---
schema_version: 1
id: "iss-2609262148072415"
slug: "memory-lint-s-run-log-report-md"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/lint.go"
resolution: "memory lint's report.md renders the store path, each finding's code, file, message and suggestion through termsafe.CleanProseLine, and sets the store path and each file off with termsafe.CodeSpan."
impact: fix
resolved_by:
  commit: "bc51fd4ae"
---

memory lint's run-log report.md (renderLintReportMD, internal/core/memory/lint.go) renders a finding's file, message and suggestion through termsafe.Sanitize alone, so a page name or a pii.json pattern name carrying an HTML comment opener or link syntax reaches the markdown report live; the memory renderers fixed for iss-2609020539188868 and the lifeboat renderers fixed for iss-2609251355497247 route the same kind of field through termsafe.CleanProse and set a path off with termsafe.CodeSpan.

## Grounds

- pursued: a finding whose file, message and suggestion carry a comment opener, a script tag and link syntax renders none of them live in report.md and its file inside a code span; any of the three appearing live would show it wrong
