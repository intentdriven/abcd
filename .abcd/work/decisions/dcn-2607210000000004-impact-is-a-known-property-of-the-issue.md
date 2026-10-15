---
id: dcn-2607210000000004
slug: impact-is-a-known-property-of-the-issue
date: 2026-07-21
---
2026-07-21 — `impact` is a KNOWN property of the issue ledger schema, and
`internal/core/capture` validates it against the shared enum in
`internal/core/changelog` rather than a private copy. The back-fill added
`impact:` to every resolved issue, which `validateStrict`'s
additionalProperties:false allow-list rejected — `abcd capture` reported
"resolved 0" and skipped all 57 records as malformed. Accepting the field
without validating it was rejected: severity/category/source are all
enum-checked on read, and a third definition of the impact enum is exactly what
spc-10 exists to prevent. capture -> changelog is the import direction (no
cycle: changelog imports launch/frontmatter/gitutil only).
