---
schema_version: 1
id: "iss-2609262241109876"
slug: "ideate-block-escape-shows-a-literal-backslash-before-an-ordered-marker"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: review-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/render.go"
---

ideate's blockText (internal/core/ideate/render.go) escapes an ordered-list-shaped idea by putting the backslash before the digits, and CommonMark treats a backslash before a non-punctuation character as literal: an idea reading '1. first' renders as a backslash followed by '1. first', so the verdict record shows text the idea never had. The escape belongs before the '.' or ')' delimiter, as the lifeboat block escaper places it. Found while aligning the two block escapers.
