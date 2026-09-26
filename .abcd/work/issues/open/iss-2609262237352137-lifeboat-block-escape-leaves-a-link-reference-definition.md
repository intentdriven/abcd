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
---

The lifeboat markdown renderers' escapeLeadingMarker (internal/core/lifeboat/mdrender.go) leaves a leading left square bracket alone, so a principle, press-release subhead, body or quote shaped like `[label]: http://example.com` is emitted as a CommonMark link reference definition: the value itself renders as nothing, and a `[label]` shortcut reference in any other field (the cleaner breaks only the link-text adjacencies) renders as a live link to the attacker's destination. ideate's blockText already escapes the bracket for the same reason.
