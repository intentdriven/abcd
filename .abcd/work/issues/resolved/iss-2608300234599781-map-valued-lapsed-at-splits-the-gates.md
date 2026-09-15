---
schema_version: 1
id: "iss-2608300234599781"
slug: "map-valued-lapsed-at-splits-the-gates"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-182 third-round security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (checkIssueRecordShape, lapsed_at)"
resolution: "checkIssueRecordShape looks ahead for a block-spelled lapsed_at when the same-line value is empty, so a nested mapping is present-and-malformed rather than absent; the look-ahead reuses one generalised frontmatter block walker that blockSequenceAt now reads through, not a second scanner."
impact: internal
---

A lapsed_at carrying a nested map (an empty same-line value followed by an indented block) is read as absent by the committed-ledger gate because the lenient scanner yields an empty value and the null test passes, while capture's reader builds a map and refuses the record as not a string; on a non-lapse category the gate is green and the record is skipped invisibly. Sibling of the closed list case: when the same-line value is empty, look ahead for an indented continuation (blockSequenceAt already exists for the shape) and treat it as present and malformed.
