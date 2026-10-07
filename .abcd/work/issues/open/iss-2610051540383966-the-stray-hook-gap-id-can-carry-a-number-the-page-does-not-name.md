---
schema_version: 1
id: "iss-2610051540383966"
slug: "the-stray-hook-gap-id-can-carry-a-number-the-page-does-not-name"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "v0.13.1 docs-currency release review (dc-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ahoy.md"
remedy: "Say on commands/ahoy.md that a second abcd hook on the same event is reported as harness.stray_hook.<Event>.<n>."
---

commands/ahoy.md names the stray-hook gap id as harness.stray_hook.<Event>, but internal/core/ahoy/harness_strays.go appends .<n> for a second abcd hook on the same event. The page already tells the host to relay detail and fix_hint as they stand, so no action goes wrong; the id shape is simply incomplete. Deferred from the v0.13.1 cut to keep its content commit stable.
