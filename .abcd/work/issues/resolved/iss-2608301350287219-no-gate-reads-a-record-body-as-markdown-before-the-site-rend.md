---
schema_version: 1
id: "iss-2608301350287219"
slug: "no-gate-reads-a-record-body-as-markdown-before-the-site-rend"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "itd-189-round-3-remediation"
found_at: "internal/core/lint, cmd/record-lint"
resolution: "record_schema calls site.CheckRecordBody, the record page's own renderer path, on every issue record body and reports the construct it refuses; registered by cmd/record-lint and the CLI. TestRecordSchemaRefusesAnIssueBodyTheSiteCannotRender and TestRecordLintRegistersTheSiteBodyCheck pin it."
impact: internal
resolved_by:
  commit: "a812db1d"
---

no gate reads a record body as markdown before the site render, so a construct the renderer refuses reaches the branch and turns preflight red at the far end

`abcd capture` accepts any body text, `lint-issues` judges the frontmatter and
the resolution trailers, and `record-lint` judges the schema. The first gate that
reads a record BODY as markdown is `site-render`, at the far end of preflight and
only in a checkout where somebody runs it — so a construct the renderer refuses
is committed, pushed, and found by whoever next runs the whole gate rather than
by the author who wrote it.

Observed as an indented code block in a committed issue record, which held
`site-render` red on a branch for two commits (iss-2608301349335095 carries that
instance and its fence). The remedy is a check where the record is written: the
constructs the site renderer reads are a fixed subset, so the same subset can be
asserted at capture time or in `record-lint`, and the author is told in the
sentence they are writing rather than at the push.

## Grounds

- pursued: an issue body the site render refuses fails record-lint in the change that writes it; one that passes record-lint and fails site-render would show it wrong
