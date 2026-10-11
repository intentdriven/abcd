---
schema_version: 1
id: "iss-2609251600023777"
slug: "site-renderer-a-top-level-three-backtick"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/markdown.go"
resolution: "A top-level three-backtick fence indented one to three spaces renders as a fence, its lines losing up to the opener's indent (mdrender indentedFence); the tilde and four-backtick twins stay refused, and an indented opener under prose is refused as a fence without a blank line before it. The shape was live in record pages as the fence of a loose list item."
impact: fix
resolved_by:
  commit: "b2fe9f27"
---

site renderer: a top-level three-backtick fence indented one to three spaces still renders silently as a paragraph with inline code (site/markdown.go unrenderedFenceRe covers tildes and four or more backticks only). No page in docs/ or site-src/ has the shape today. Refuse it like the other unsupported fence forms, or render it.

## Grounds

- pursued: every indent 1-3 renders a command block with dedented code and no paragraph (TestRenderBlockRendersAnIndentedThreeBacktickFence); a <p><code> for any of those inputs, or a site-render failure on the record, would show it wrong
