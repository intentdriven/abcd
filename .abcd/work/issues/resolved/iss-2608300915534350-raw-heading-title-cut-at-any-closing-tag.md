---
schema_version: 1
id: "iss-2608300915534350"
slug: "raw-heading-title-cut-at-any-closing-tag"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-183 seventh-round ruthless review, 2026-08-30"
found_at: "internal/core/reading/project.go (rawHeadingEndRe, blockCloser, quotedSpanRe, setext loop)"
resolution: "The raw-heading title was cut at the first closing tag of any element, so an anchor before the text emptied it and emphasis on the first word truncated it to that word, and both sections were admitted. The title is bounded at the opening element's own closing tag now, matched case-insensitively, with the next heading open and a blank line as the other bounds — a heading's own close is the only bound that survives the inline markup a heading is allowed to carry. A block closer is read at column 0, so an indented ellipsis inside a block scalar no longer closes the block and refuses the file. And the closing delimiter is not read as a setext underline of the line above it: it closes a block, it underlines nothing."
impact: fix
---

The raw-heading title is cut at the first closing tag of ANY element, so inline markup inside a heading element — an anchor then text, or emphasis on the first word — truncates or empties the title and the excluded section is admitted (refused before the seventh-round commit); bound the title at the opening element's own closing tag, the next heading open, or a blank line. Also: blockCloser trims indentation before testing three dots so an indented ellipsis inside a block scalar closes the block and refuses the file (test at column 0 only); the double-quoted span does not honour an escaped quote; the setext loop reads the frontmatter closer as an underline of a column-0 last frontmatter line.
