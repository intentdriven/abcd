---
schema_version: 1
id: "iss-2609251004336500"
slug: "links-resolve-can-miss-a-real-broken-link-on-a-line-that"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "stripInlineCode skips a backslash-escaped backtick outside a span, so a link between escaped backticks is read and a broken one reported"
impact: fix
resolved_by:
  commit: "db597610c7e9fcbf552acb595ab5127c5d91bc67"
---

links_resolve can miss a real broken link on a line that carries a backslash-escaped backtick: stripInlineCode (internal/core/lint/lint.go) pairs every backtick it meets, so an escaped backtick, which CommonMark reads as a literal character and never as a code-span delimiter, opens a span that the next backtick closes, and a link between them is blanked before the link pattern runs. A line reading a literal backtick, then text with a link to a missing file, then another literal backtick reports no finding, so the records gate passes a broken link. Found by the tier1 review (review-tier1 finding 4).

## Grounds

- pursued: links_resolve reports a broken link on a line carrying escaped backticks and still blanks real code spans; shown wrong if a link inside a real code span is reported, or a broken link beside an escaped backtick is not
