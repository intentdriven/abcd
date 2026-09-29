---
schema_version: 1
id: "iss-2609262237352137"
slug: "lifeboat-block-escape-leaves-a-link-reference-definition"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/mdrender.go"
resolution: "escapeLeadingMarker escapes a leading bracket, so no lifeboat block value is parsed as a link reference definition"
impact: fix
resolved_by:
  commit: "70fefd58c"
---

The lifeboat markdown renderers' escapeLeadingMarker (internal/core/lifeboat/mdrender.go) leaves a leading left square bracket alone, so a principle, press-release subhead, body or quote shaped like `[label]: http://example.com` is emitted as a CommonMark link reference definition: the value itself renders as nothing, and a `[label]` shortcut reference in any other field (the cleaner breaks only the link-text adjacencies) renders as a live link to the attacker's destination. ideate's blockText already escapes the bracket for the same reason.

## Grounds

- pursued: a definition-shaped principle, subhead, body or quote renders as its own text and no line of principles.md or press-release.md opens with it, so a shortcut reference elsewhere stays plain text; TestBlockValueNeverDefinesALinkReference seeing the definition consumed by the site renderer, or a line opening with it, would show it wrong
