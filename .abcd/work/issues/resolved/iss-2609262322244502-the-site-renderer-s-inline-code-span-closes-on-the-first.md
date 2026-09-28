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
resolution: "termsafe.PairCodeSpan is the tree's one code-span pairer (closer: the first later run of exactly the opening length) and termsafe.CodeSpanText applies CommonMark's content rules (a line ending is a space; one space off each side only when both are present and the content is not all spaces). The site renderer, the prose cleaner, OpensBalancedCodeSpan, mdrecord's OpensComment and CodeSpanRanges and the surface appendix pair through it and their private walks are gone; TestNoSecondCodeSpanPairer refuses a new one. One shared table holds the renderer, lifeboat's and ideate's block escapers to the same verdicts. The fence renderer's last-line closer is mdrecord's verdict, not a prefix test. A site build of the same tree differs only by the CommonMark content rules (117 of 2271 files) and every record still renders."
impact: fix
resolved_by:
  commit: "e275afcc5"
---

The site renderer's inline code span closes on the first occurrence of the opening run's text, not on a run of exactly the same length as CommonMark requires, and it trims every space from a multi-backtick span rather than one from each side. So a span opened by two backticks and holding a, three backticks, b (a CommonMark code span) is refused as an unclosed code span and fails the whole page; this record cannot quote the input literally, because the site renders every ledger record and would refuse it, and a span holding only a space renders empty. termsafe.OpensBalancedCodeSpan and mdrecord's findBacktickRun both pair runs by exact length, so a value a block escaper leaves unescaped as balanced can still be refused by the renderer: loud, not silent, and no text is published wrongly. Confirmed on a scratch copy while fixing iss-2609262309556167; the tree holds three code-span pairers (termsafe closingRun, mdrecord findBacktickRun, mdrender Inline) where one canonical primitive is wanted.

## Grounds

- pursued: a value whose leading run termsafe calls balanced renders as that span on the site, and a span opened by two backticks around a run of three renders instead of failing the page; a committed record the site refuses to render, or an escaper and the renderer disagreeing on a row of the shared table, would show it wrong
