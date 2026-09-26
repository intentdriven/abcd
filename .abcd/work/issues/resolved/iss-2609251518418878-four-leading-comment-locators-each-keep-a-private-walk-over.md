---
schema_version: 1
id: "iss-2609251518418878"
slug: "four-leading-comment-locators-each-keep-a-private-walk-over"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/sections.go"
resolution: "site's frontmatterLead, lint's and glossary's frontmatterOpen, and reading's displacedFrontmatter all locate leading comments through mdrecord.FirstContent, so there is one reading of a leading comment in the tree and a line carrying prose after a comment's closer is content to all four."
impact: fix
resolved_by:
  commit: "18a89c9f"
---

Four leading-comment locators each keep a private walk over HTML comments instead of mdrecord's reading: site/sections.go frontmatterLead, lint/lint.go frontmatterOpen, glossary/index.go frontmatterOpen and reading's displacedFrontmatter. The first three read any line starting with a comment opener and ending with a closer as one whole comment, so a line holding a comment, then prose, then a closer is skipped as a comment and a frontmatter block below it is found where every renderer sees prose first. reading's walk is fail-closed (it refuses a displaced block) but is still a fourth spelling of the same locator.

## Grounds

- pursued: a line holding a comment, prose and a closer is content to site, lint and glossary (TestFrontmatterLeadReadsContentAfterACommentCloser, TestFrontmatterOpenReadsContentAfterACommentCloser in lint and glossary), a multi-line comment is still skipped, and the site, lint, glossary, reading and mdrecord packages pass; a frontmatter block found below a prose line, or a regression in any of those packages, would show it wrong
