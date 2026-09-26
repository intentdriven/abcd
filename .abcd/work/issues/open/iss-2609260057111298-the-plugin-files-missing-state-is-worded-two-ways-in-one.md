---
schema_version: 1
id: "iss-2609260057111298"
slug: "the-plugin-files-missing-state-is-worded-two-ways-in-one"
severity: "nitpick"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/guard_health.go"
---

The plugin-files-missing state is worded two ways in one abcd ahoy render: the plugin.root_missing gap speaks plainly, but the guard-health reason still says: plugin root not resolvable, so the hook manifest cannot be read and the guard wiring is unknown.
