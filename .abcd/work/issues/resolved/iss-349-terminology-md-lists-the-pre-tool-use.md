---
schema_version: 1
id: "iss-349"
slug: "terminology-md-lists-the-pre-tool-use"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "docs/reference/terminology.md"
resolution: "terminology Guardrails row states the guard is fail-open-loud per adr-42"
impact: fix
---

terminology.md lists the pre-tool-use shell-hazard guard among deterministic fail-closed gates, asserting the opposite safety property from adr-42's fail-open-loud mistake filter on the one page written for an evaluator
## Evidence

- `docs/reference/terminology.md:56` (Guardrails row): "deterministic, fail-closed gates: lint families, receipt gates, pre-commit guards, and the pre-tool-use shell-hazard guard (`abcd guard`)" — both adjectives distribute over all four members.
- The guard hook is deliberately fail-open-loud: `internal/surface/cli/guard.go:137-139`, `:163-167` (`failOpen`), `hooks/hooks.json:30`, adr-42, and `docs/reference/cli/commands.md:379-404` all state it; spc-16 names the term. The row asserts the opposite safety property on the page written for an evaluator, and breaks the enforcement-claims-are-facts principle the same cell cites.
- Refuter verdict: CONFIRMED substantive. Fix keeps the `guard check` caveat accurate (check does fault closed, exit 2).
