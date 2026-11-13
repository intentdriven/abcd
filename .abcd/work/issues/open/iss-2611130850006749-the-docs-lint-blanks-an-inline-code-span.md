---
schema_version: 1
id: "iss-2611130850006749"
slug: "the-docs-lint-blanks-an-inline-code-span"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "lane ao build"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
remedy: "On the lines below a backtick left open, to the next blank line, the checks blank no span and read the line, as the page readers do since dcn-2611122109097376."
refines: [iss-2611121843363011]
---

The docs lint blanks an inline code span's backticks on one line; a rendered page can pair a backtick left open on one line with one on a line below, so the lint and the page disagree about which text is prose.
