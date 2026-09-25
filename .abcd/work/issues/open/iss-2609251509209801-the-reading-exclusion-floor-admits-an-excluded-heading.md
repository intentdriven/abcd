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
---

The reading exclusion floor admits an excluded heading nested in a list item or a blockquote, and the section travels with a nil error: '- ## Audit Notes', '> ## Audit Notes', and a heading at a list item's content indent of four columns all render as headings, while floorATXRe wants the line's first non-blank character to be '#' within three spaces and the site section walk reads column 0 alone. project.go discloses the residue in a code comment ('Two shapes this floor does NOT see'), and a disclosure in a comment is not a recorded deferral. No committed markdown file carries such a heading naming an excluded title, so the floor can refuse the shape outright.
