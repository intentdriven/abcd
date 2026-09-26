---
schema_version: 1
id: "iss-2609262237415400"
slug: "lifeboat-block-escape-breaks-a-balanced-leading-code-span"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: review-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/mdrender.go"
resolution: "escapeLeadingMarker escapes a leading backtick only when its run is unbalanced, so a cleaned value's balanced leading code span keeps sheltering what it quotes"
impact: fix
resolved_by:
  commit: "4df17e706"
---

The lifeboat markdown renderers' escapeLeadingMarker (internal/core/lifeboat/mdrender.go) backslash-escapes a leading backtick unconditionally, the defect ideate's blockText was fixed for. termsafe's HTML-tag rule exempts a code span, so a value opening with a balanced span that quotes a details tag keeps the tag unbroken; escaping the opening backtick kills the span and republishes the tag as live inline HTML in principles.md or press-release.md, concealing what follows (the site renderer refuses the result as an unclosed code span). Only an unbalanced leading run opens a fence and needs the escape.

## Grounds

- pursued: a block value opening with a balanced span quoting a details tag renders the tag inside a code element, while an unbalanced leading run is still escaped; TestBlockValueKeepsABalancedLeadingCodeSpan seeing the site renderer refuse the value, or an unbalanced run left unescaped, would show it wrong
