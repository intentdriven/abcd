---
schema_version: 1
id: "iss-2609262322244502"
slug: "the-site-renderer-s-inline-code-span-closes-on-the-first"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fix3-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mdrender/render.go"
---

The site renderer's inline code span closes on the first occurrence of the opening run's text, not on a run of exactly the same length as CommonMark requires, and it trims every space from a multi-backtick span rather than one from each side. So a span opened by two backticks and holding a, three backticks, b (a CommonMark code span) is refused as an unclosed code span and fails the whole page; this record cannot quote the input literally, because the site renders every ledger record and would refuse it, and a span holding only a space renders empty. termsafe.OpensBalancedCodeSpan and mdrecord's findBacktickRun both pair runs by exact length, so a value a block escaper leaves unescaped as balanced can still be refused by the renderer: loud, not silent, and no text is published wrongly. Confirmed on a scratch copy while fixing iss-2609262309556167; the tree holds three code-span pairers (termsafe closingRun, mdrecord findBacktickRun, mdrender Inline) where one canonical primitive is wanted.
