---
schema_version: 1
id: "iss-2608301221402131"
slug: "checkissuerecordshape-s-block-spelled-lapsed-at-leg-gates-on"
severity: "nitpick"
category: "ux"
source: "impl-review"
found_during: "itd-189 review round, 2026-08-30"
found_at: "internal/core/lint/schema.go (checkIssueRecordShape, lapsed_at block leg)"
---

checkIssueRecordShape's block-spelled lapsed_at leg gates on the same-line value being empty, so it never sees the blocks entry that a block-scalar HEADER now populates: lapsed_at spelled as a header over an indented instant reports 'not an RFC 3339 instant' naming the header byte, instead of the block-spelled message its sibling shape gets. Both refuse the record, so this is message quality rather than a hole — the reader is sent to fix a format that is not the problem. Read the lapse leg through the same schemaRecord accessor the required-field check uses.
