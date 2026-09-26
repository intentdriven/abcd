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
---

The lifeboat markdown renderers' escapeLeadingMarker (internal/core/lifeboat/mdrender.go) backslash-escapes a leading backtick unconditionally, the defect ideate's blockText was fixed for. termsafe's HTML-tag rule exempts a code span, so a value opening with a balanced span that quotes a details tag keeps the tag unbroken; escaping the opening backtick kills the span and republishes the tag as live inline HTML in principles.md or press-release.md, concealing what follows (the site renderer refuses the result as an unclosed code span). Only an unbalanced leading run opens a fence and needs the escape.
