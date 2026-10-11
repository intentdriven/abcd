---
schema_version: 1
id: "iss-286"
slug: "release-derivation-diagnoses-a-null"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "pr-294-review"
found_at: "internal/core/changelog/shipped.go"
resolution: "newRecord gates ParseImpact on frontmatter.IsNull so every null spelling is classified as missing, agreeing with record-lint (the two messages spell the diagnosis differently)"
impact: fix
resolved_by:
  commit: "a0c6fccb"
---

Release derivation diagnoses a null impact as malformed while record-lint calls it missing

`newRecord` (`internal/core/changelog/shipped.go:277`) calls `ParseImpact`
with no `IsNull` gate, so for a resolved issue carrying `impact: NULL`
record-lint reports "impact must be set explicitly" (missing) while the
release cut records "invalid impact" (malformed) — two diagnoses for one
line. Verdicts never diverge (both paths reject), so this is operator-facing
message inconsistency only; an `IsNull` gate at the call site restores
parity. Evidence: `.abcd/work/reviews/2026-08-19-pr-294-null-predicate/`
(F8).
Forge mirror: #373
