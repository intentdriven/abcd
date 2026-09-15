---
schema_version: 1
id: "iss-2608300235377731"
slug: "condition-marker-lost-on-reflow"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-177 adversarial reviews, 2026-08-30"
found_at: "internal/core/intent/claims.go (condMarkerRe, parseConditions, stampScopeConditions)"
resolution: "A condition's identity is now read anywhere in the bullet — first line or a folded continuation line — and excised from the reported prose, so an editor reflow can no longer orphan it or provoke a second mint; a bullet carrying two well-formed markers is named by the gate rather than silently resolved to one, and only marker-free bullets are stamped."
impact: fix
resolved_by:
  intent: "itd-177"
  spec: "spc-55"
---

A scope-condition identity marker is recognised only when it closes the bullet's first physical line, so an ordinary editor reflow of an 80-column bullet (or prose appended after the marker) moves the marker onto a continuation line, the gate reports the condition unmarked, the named remedy mints a new id, and the old id is folded into the condition text — orphaning any disposition keyed on it. This violates the identity-survives-edits criterion and never-renumbered. Match a well-formed marker anywhere in the bullet including folded continuation lines, strip it from the text, fault on two markers in one bullet, and stamp only marker-free bullets.
