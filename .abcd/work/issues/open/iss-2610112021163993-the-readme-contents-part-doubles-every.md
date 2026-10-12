---
schema_version: 1
id: "iss-2610112021163993"
slug: "the-readme-contents-part-doubles-every"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "abcd build run-2610110908176735 lane-3 validate round 1 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_derive.go"
remedy: "Keep a backslash that escapes ASCII punctuation as written and escape only a lone backslash (one before a non-punctuation character or at the end) and an unescaped bracket, outside code spans; add the two headings above to TestContentsStripsInlineHTMLFromHeadings."
---

The README contents part doubles every backslash in a heading's text, so a markdown escape renders wrong in the list: `## A \* b` is listed as `[A \\* b](#a--b)`, which the forge shows as 'A \* b' while the heading reads 'A * b'; an escaped backtick (a backslash before a backtick) becomes a literal backslash before a backtick that can then open a code span. escapeLinkText (internal/core/ahoy/readme_derive.go) escapes \ to keep a trailing backslash from escaping the link's closing bracket, but it also re-escapes the backslash of every escape the owner wrote. Found while making the contents part read code spans literally (lane-3 fix round 1, run-2610110908176735).
