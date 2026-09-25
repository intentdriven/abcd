---
schema_version: 1
id: "iss-2609251940383450"
slug: "prose-lines-drops-a-ruled-document-up-to-a-later-rule"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
---

proseLines in internal/core/launch/gates.go treats a document that opens with a '---' horizontal rule as opening YAML frontmatter whenever a second '---' rule appears anywhere later, so every line up to that later rule is dropped from the marker-block and change-narration gates, silently. A rule opens frontmatter only when the block it closes holds key: value lines. Remedy: drop the leading block only when its lines read as YAML frontmatter (key: value lines, their continuations, list items, comments), and read the document whole otherwise.
