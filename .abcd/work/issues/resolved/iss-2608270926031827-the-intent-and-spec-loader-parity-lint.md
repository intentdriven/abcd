---
schema_version: 1
id: "iss-2608270926031827"
slug: "the-intent-and-spec-loader-parity-lint"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/lint/lint.go"
resolution: "record-lint refuses an intent or spec whose frontmatter does not open on line 1, closing the extraction axis the loaders fail-close"
impact: fix
resolved_by:
  commit: "02ca8f3a"
---

the intent and spec loader-parity lint gate reads frontmatter from the first delimiter while the loaders require it on line 0, so a record led by a blank line or an HTML comment is lint-green yet fail-closes the whole store for every puller — the parity rule must refuse a preamble-led record in the families whose loaders refuse it