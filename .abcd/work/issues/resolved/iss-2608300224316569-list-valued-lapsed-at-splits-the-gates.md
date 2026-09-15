---
schema_version: 1
id: "iss-2608300224316569"
slug: "list-valued-lapsed-at-splits-the-gates"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-182 second-round security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (checkIssueRecordShape, lapsed_at)"
resolution: "The lapsed_at absence test is the frontmatter null set alone, so a list-shaped value falls through to the RFC 3339 refusal instead of reading as absent; the gate and the reader now reach the same verdict on lapsed_at: [] for every category."
impact: internal
---

A list-valued lapsed_at ([]) on a non-lapse record is lint-green because the record-shape check tests absence with isAbsentValue, which treats an empty inline list as absent, while capture's reader refuses it as not a string and skips the record silently. Same pre-existing class as found_at; the lapsed_at check should test nullness with the frontmatter null set only and let a list-shaped value fall through to the RFC 3339 finding.
