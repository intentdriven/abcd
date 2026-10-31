---
schema_version: 1
id: "iss-2610310331124321"
slug: "the-dashboard-chapter-says-exit-1-only"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/36-dashboard.md"
remedy: "Widen exit 1 in 04-surfaces/36-dashboard.md to include the unreadable-managed-state case, or move that case to exit 2 in code."
resolution: "the design brief now states what the binary does, in present tense"
impact: internal
---

The dashboard chapter says exit 1 only follows something started; dashboard start also exits 1 with nothing started when it cannot read whether abcd manages the checkout.
