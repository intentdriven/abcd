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
---

The lifeboat half of iss-2609020539188868 is still open after the memory renderers were fixed: synthesis_review renders a finding id through termsafe.Sanitize alone, never CleanProse, so it can still carry an HTML comment opener or link syntax, and wraps a severity in its own bracket; the press-release subhead wraps a cleaned value in its own emphasis; synthesis_principles writes a cleaned principle as a bare paragraph with no leading-marker escape. The fix is the one applied to memory: every untrusted field on a markdown line through CleanProse, and no renderer adding delimiters around a cleaned value (termsafe.CodeSpan where a code span is wanted).
