---
schema_version: 1
id: "iss-2609170627427239"
slug: "the-status-line-badge-s-default-state-reads-abcd-which-says"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "first run of the v0.9.0 status line after ahoy install, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/badge.go"
---

The status-line badge's default state reads 'abcd', which says the tool is present but not that this repository is managed by it; make the managed-and-nobody-waiting badge read 'abcd-managed' in an abcd-managed repository, so the word on the row states the fact the badge exists to show, alongside the facilitator and product-thinker states.
