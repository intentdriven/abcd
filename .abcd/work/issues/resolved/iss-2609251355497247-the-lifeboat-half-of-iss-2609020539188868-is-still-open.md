---
schema_version: 1
id: "iss-2609251355497247"
slug: "the-lifeboat-half-of-iss-2609020539188868-is-still-open"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat"
resolution: "Every markdown file the lifeboat writes renders through one discipline (internal/core/lifeboat/mdrender.go): untrusted fields through termsafe.CleanProse, a delimiter only through termsafe.CodeSpan (severity, finding id, evidence refs, packed source paths), and a leading-marker escape on every value that begins a block (principle, subhead, body, quote). The review severity bracket and the press-release subhead emphasis are gone."
impact: fix
resolved_by:
  commit: "b156bb1bd"
---

The lifeboat half of iss-2609020539188868 is still open after the memory renderers were fixed: synthesis_review renders a finding id through termsafe.Sanitize alone, never CleanProse, so it can still carry an HTML comment opener or link syntax, and wraps a severity in its own bracket; the press-release subhead wraps a cleaned value in its own emphasis; synthesis_principles writes a cleaned principle as a bare paragraph with no leading-marker escape. The fix is the one applied to memory: every untrusted field on a markdown line through CleanProse, and no renderer adding delimiters around a cleaned value (termsafe.CodeSpan where a code span is wanted).

## Grounds

- pursued: review, principles, press-release and brief-section renders driven with comment openers, script tags, link syntax and every block-marker lead carry none of them live and open no line with the raw marker; a raw comment opener, script tag or live link in any of the four renders, or a line opening with an untrusted marker, would show it wrong
