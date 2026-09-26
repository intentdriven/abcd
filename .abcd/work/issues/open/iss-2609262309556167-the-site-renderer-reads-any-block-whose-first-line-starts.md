---
schema_version: 1
id: "iss-2609262309556167"
slug: "the-site-renderer-reads-any-block-whose-first-line-starts"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: verify-fix2-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mdrender/render.go"
---

The site renderer reads any block whose first line starts with three backticks as a fence, without the rule mdrecord's block walk applies: a backtick fence's info string may not contain a backtick. So a line that is a balanced code span, not a fence, is mis-rendered: '``` ```' and '```x```' render as an EMPTY command block (the value's text is lost), and '``` x ```' and '```` ``` ````' fail the whole render with an unsupported-construct error. The two block escapers that leave a balanced leading code span unescaped (termsafe.OpensBalancedCodeSpan, CommonMark-correct) inherit the defect: lifeboat's escapeLeadingMarker (introduced on this branch at 4df17e706; at f9b06e720 all four inputs rendered as paragraphs) and ideate's blockText, which has carried the same pre-existing shape since d4b825630. No injection (the info string is attribute-escaped) and no concealment of neighbouring fields. The root is mdrender's fence opener (render.go RenderBlock's first-line and per-line checks, and site compose's code-tab test) spelling its own opener instead of mdrecord's; the fix routes it through mdrecord's one opener predicate.
