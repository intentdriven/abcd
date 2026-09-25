---
schema_version: 1
id: "iss-2609250915305413"
slug: "record-lint-s-links-resolve-rule-reads-a-markdown-link"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "links_resolve blanks inline code spans through stripInlineCode before matching, so a code span is never read as a link; TestLinksResolveSkipsInlineCodeSpans pins it."
impact: fix
resolved_by:
  commit: "201b68c2"
---

record-lint's links_resolve rule reads a Markdown link inside an inline code span: checkLinks masks fenced blocks but runs linkRe over every other line whole, so a code span such as `Get[T](s, key)` or `Decode[T](raw)` in a record is reported as a BLOCKER 'link target does not resolve'. CommonMark gives a code span precedence over link syntax, so the text is not a link; a record author has to reword code to pass the gate.

## Grounds

- pursued: we expect no record's code span to be reported as an unresolved link while every real link, including one whose text is a span, is still checked; shown wrong if a link written beside or around a code span stops being reported when its target is missing
