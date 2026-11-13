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
resolution: "The docs lint's link checks, footnote and URL rules and the forbidden-synonym check pair code spans on one line and read whole the lines below a backtick left open, to the next blank line, through mdrecord.SureCodeSpans; the crosswalk rule pairs each row alone (dcn-2611130858310110)."
impact: fix
---

The docs lint blanks an inline code span's backticks on one line; a rendered page can pair a backtick left open on one line with one on a line below, so the lint and the page disagree about which text is prose.
