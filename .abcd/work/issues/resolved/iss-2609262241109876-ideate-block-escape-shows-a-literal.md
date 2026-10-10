---
schema_version: 1
id: "iss-2609262241109876"
slug: "ideate-block-escape-shows-a-literal"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: review-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/render.go"
resolution: "blockText escapes an ordered-list-shaped idea before its delimiter, so the record shows the idea's own text and opens no list"
impact: fix
resolved_by:
  commit: "08efdae8a"
---

ideate's blockText (internal/core/ideate/render.go) escapes an ordered-list-shaped idea by putting the backslash before the digits, and CommonMark treats a backslash before a non-punctuation character as literal: an idea reading '1. first' renders as a backslash followed by '1. first', so the verdict record shows text the idea never had. The escape belongs before the '.' or ')' delimiter, as the lifeboat block escaper places it. Found while aligning the two block escapers.

## Grounds

- pursued: ideas reading '1. first' and '12) twelve' render through the site renderer as paragraphs carrying exactly that text; TestBlockTextEscapesAnOrderedMarkerFaithfully seeing a list or a literal backslash would show it wrong
