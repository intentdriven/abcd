---
schema_version: 1
id: "iss-2609251509209801"
slug: "the-reading-exclusion-floor-admits-an-excluded-heading"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "The verifier refuses an excluded heading nested in a list item, a blockquote or an indent: nestedHeadingRe matches an ATX heading behind any run of indentation, blockquote markers and list markers, and a match naming an excluded heading refuses the document, naming the line. The disclosure comment in project.go now names only the homoglyph residue."
impact: fix
resolved_by:
  commit: "5221e2a8"
---

The reading exclusion floor admits an excluded heading nested in a list item or a blockquote, and the section travels with a nil error: '- ## Audit Notes', '> ## Audit Notes', and a heading at a list item's content indent of four columns all render as headings, while floorATXRe wants the line's first non-blank character to be '#' within three spaces and the site section walk reads column 0 alone. project.go discloses the residue in a code comment ('Two shapes this floor does NOT see'), and a disclosure in a comment is not a recorded deferral. No committed markdown file carries such a heading naming an excluded title, so the floor can refuse the shape outright.

## Grounds

- pursued: list-prefixed, ordered-list-prefixed, content-indented, blockquoted, list-in-blockquote and tab-indented excluded headings are refused while other nested headings, a bullet naming the title in prose and a fenced nested quote are admitted (TestVerifyRedactionRefusesAHeadingNestedInAListOrBlockquote), and the reading package and the cold-reading eval lane pass over the committed corpus; an admitted nested excluded heading or a refused committed file would show it wrong
