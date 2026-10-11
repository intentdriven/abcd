---
schema_version: 1
id: "iss-2609170627427239"
slug: "the-status-line-badge-s-default-state"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "first run of the v0.9.0 status line after ahoy install, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/badge.go"
resolution: "The quiet badge reads abcd-managed in a managed repository, beside the two role labels; no state renders the bare tool name."
impact: fix
resolved_by:
  intent: "itd-2609212130146198"
  spec: "spc-2609212139593041"
  commit: "f31c564a07c9e78db4c44bd0efcaada88597d8b5"
---

The status-line badge's default state reads 'abcd', which says the tool is present but not that this repository is managed by it; make the managed-and-nobody-waiting badge read 'abcd-managed' in an abcd-managed repository, so the word on the row states the fact the badge exists to show, alongside the facilitator and product-thinker states.

## Grounds

- pursued: the badge's word states that this repository is managed, not merely that the tool is present; shown wrong if a managed repository's status line still reads a bare 'abcd' badge
