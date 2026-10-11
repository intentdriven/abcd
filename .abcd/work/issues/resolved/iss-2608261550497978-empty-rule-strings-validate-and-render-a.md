---
schema_version: 1
id: "iss-2608261550497978"
slug: "empty-rule-strings-validate-and-render-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/core/rules/rules.go"
resolution: "rules Validate refuses an empty/whitespace rule body; renderDomain emits no contentless bullet"
impact: internal
resolved_by:
  commit: "ee9e48cb"
---

An empty or all-whitespace rule string passes rules.json validation and renders as a bare contentless bullet in the injected block: the loader validates domain names and states but not rule bodies. Refuse empty or whitespace-only rules at load, or drop them at render loudly.