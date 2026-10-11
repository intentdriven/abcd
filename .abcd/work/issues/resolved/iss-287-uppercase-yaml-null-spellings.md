---
schema_version: 1
id: "iss-287"
slug: "uppercase-yaml-null-spellings"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "pr-294-review"
found_at: "internal/core/frontmatter/frontmatter.go"
resolution: "frontmatter.IsNull is the YAML 1.2 core null set exactly — empty, ~, null, Null, NULL — and deliberately not a case-insensitive compare, because YAML does not accept nUlL and reading records no parser agrees with is worse than the miss. internal/core/lint's private duplicate, which recognised only the lower-case spelling, now delegates to it: that duplicate was the split where impact: NULL was null to capture and a malformed impact to the lint."
impact: fix
---

Uppercase YAML null spellings unrecognised by the null predicates (reported as GitHub #290)

Retroactive capture of the bug reported as GitHub #290 so the ledger holds
the handle the CHANGELOG entry should cite: `frontmatter.IsNull` and lint's
`isNull` recognised `null` and `~` but not the YAML 1.2 core-schema
spellings `Null`/`NULL`, so records using an uppercase null were misread
(e.g. a superseded ADR's `superseded_by:` treated as a live handle). Fixed
in flight by PR #294, which widens both predicates with regression tests;
resolve this issue when that PR merges, citing its commit. Evidence:
`.abcd/work/reviews/2026-08-19-pr-294-null-predicate/` (F10).
Forge mirror: #374
